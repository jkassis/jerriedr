package kittie

import (
	context "context"
	fmt "fmt"
	math "math"
	"net"
	"net/http"
	"sync"
	"time"

	sentryfasthttp "github.com/getsentry/sentry-go/fasthttp"
	"github.com/jkassis/jerrie/core"
	"github.com/lab259/cors"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/valyala/fasthttp"
	reuse "github.com/libp2p/go-reuseport"
)

var handlerTimeMetrc prometheus.Histogram

func init() {
	Buckets := []float64{}
	for i := 0; i < 30; i++ {
		Buckets = append(Buckets, 1000*math.Pow(2, float64(i)))
	}
	handlerTimeMetrc = prometheus.NewHistogram(
		prometheus.HistogramOpts{
			Name:    "fasthttp_handler_time",
			Help:    "time to handle request in fasthttp",
			Buckets: Buckets,
		})
	core.PromRegisterCollector(handlerTimeMetrc)
}

// ServiceHTTPFastRouter collects routs and handlers in a lockable map
type ServiceHTTPFastRouter struct {
	sync.RWMutex
	Routes map[string]func(ct *fasthttp.RequestCtx)
}

// Init does what it says
func (s *ServiceHTTPFastRouter) Init() {
	s.Routes = make(map[string]func(ct *fasthttp.RequestCtx), 0)
}

// RouteAdd adds a path and handler
func (s *ServiceHTTPFastRouter) RouteAdd(path string, handler func(ct *fasthttp.RequestCtx)) {
	s.Lock()
	s.Routes[path] = handler
	s.Unlock()
}

// Handle processes a request by sending it to one of the registered routes
func (s *ServiceHTTPFastRouter) Handle(ctx *fasthttp.RequestCtx) {
	start := time.Now()
	path := ctx.Path()
	core.Log.Tracef("HTTPFast handling %s", string(path))
	handler := s.Routes[string(path)]
	if handler == nil {
		ctx.SetStatusCode(404)
		ctx.Write([]byte("Not Found"))
		return
	}
	handler(ctx)
	duration := time.Since(start)
	handlerTimeMetrc.Observe(float64(duration))
}

// ServiceHTTPFast is a basic HTTP server with logging, graceful shutdown, metrics, etc.
type ServiceHTTPFast struct {
	ln                   net.Listener
	playing              bool
	Router               *ServiceHTTPFastRouter
	Addr                 string
	Context              context.Context
	HTTPServerResTimeout time.Duration
	TLSEnable            bool
	TLSCertFilePath      string
	TLSKeyFilePath       string
}

// Init setups up basic routes for health checks and metrics
func (s *ServiceHTTPFast) Init() {
	s.Router = &ServiceHTTPFastRouter{}
	s.Router.Init()
	s.Router.RouteAdd("/ping", s.pingHandler)
	// See https://godoc.org/github.com/prometheus/client_golang/prometheus#pkg-examples
	// for examples of metrics
	// server.Router.Handle("/metrics", promhttp.Handler())
	// server.Router.Handle("/debug/vars", http.DefaultServeMux)
}

// Play starts the server and begins listening for requests
func (s *ServiceHTTPFast) Play() error {
	if s.playing {
		return nil
	}
	core.Log.Warn("LISTENING ON " + s.Addr + " (serviceHTTPFast)")
	ln, err := reuse.Listen("tcp", s.Addr)
	if err != nil {
		core.Log.Error(fmt.Sprintf("error in reuseport listener: %s", s.Addr), err)
		return err
	}
	s.ln = ln
	s.playing = true
	go func() {
		defer core.SentryRecover("ServiceHTTPFast.Play")

		if s.TLSEnable {
			if err = fasthttp.ServeTLS(ln, s.TLSCertFilePath, s.TLSKeyFilePath, cors.Default().Handler(s.Router.Handle)); err != nil {
				core.Log.Fatal(fmt.Sprintf("error in fasthttp Server: %s", s.Addr), err)
			}
		} else {
			if err = fasthttp.Serve(ln, cors.Default().Handler(s.Router.Handle)); err != nil {
				core.Log.Fatal(fmt.Sprintf("error in fasthttp Server: %s", s.Addr), err)
			}
		}
	}()
	return nil
}

// Stop stops the service
func (s *ServiceHTTPFast) Stop() error {
	if !s.playing {
		return nil
	}
	err := s.ln.Close()
	if err != nil {
		return err
	}

	s.playing = false
	return nil
}

func (s *ServiceHTTPFast) pingHandler(ctx *fasthttp.RequestCtx) {
	ctx.Write([]byte("pong"))
}

// fastHTTPSentryHandler for panic reporting
// https://docs.sentry.io/platforms/go/http/
var fastHTTPSentryHandler = sentryfasthttp.New(sentryfasthttp.Options{
	Repanic:         true,
	WaitForDelivery: false,
})

// RouteAdd registers a path and handler
func (s *ServiceHTTPFast) RouteAdd(path string, handler Handler) {
	fastHTTPHandler := func(ctx *fasthttp.RequestCtx) {
		reqContext, cancel := context.WithTimeout(s.Context, s.HTTPServerResTimeout)
		defer cancel()
		res, err := handler(reqContext, ctx.Request.Body())
		if err != nil {
			ctx.SetStatusCode(http.StatusInternalServerError)
			errTxt := err.Error()
			ctx.Write([]byte(errTxt))
			core.Log.Errorf("%s: %s", path, errTxt)
			return
		}
		// https://stackoverflow.com/questions/39415827/how-to-check-if-responsewriter-has-been-written
		ctx.Write(res)
		ctx.SetStatusCode(200)
	}

	s.Router.RouteAdd(path, fastHTTPSentryHandler.Handle(fastHTTPHandler))
}

// RoutesAddN registers multiple routes
func (s *ServiceHTTPFast) RoutesAddN(routes map[string]Handler) {
	for path, handler := range routes {
		s.RouteAdd(path, handler)
	}
}
