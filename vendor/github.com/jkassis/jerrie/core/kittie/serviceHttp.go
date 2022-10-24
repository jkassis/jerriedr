package kittie

import (
	"context"
	"crypto/tls"
	fmt "fmt"
	"io/ioutil"
	"net"
	"net/http"
	"net/http/httputil"
	"os"
	"strconv"
	"time"

	sentryhttp "github.com/getsentry/sentry-go/http"
	"github.com/gorilla/mux"
	"github.com/jkassis/jerrie/core"
	reuse "github.com/libp2p/go-reuseport"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/rs/cors"
	"github.com/sirupsen/logrus"
)

// RequestLoggingMiddleware : logs requests
func RequestLoggingMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var err error
		if _, err = strconv.ParseBool(os.Getenv("DUMP_REQUESTS")); err == nil {
			if requestDump, err := httputil.DumpRequest(r, true); err != nil {
				core.Log.Error(err)
			} else {
				core.Log.Info(string(requestDump))
			}
		}

		// Call the next handler, which can be another middleware in the chain, or the final handler.
		next.ServeHTTP(w, r)
	})
}

// ServiceHTTP is a basic HTTP server with logging, graceful shutdown, metrics, etc.
type ServiceHTTP struct {
	corsHandler          http.Handler
	playing              bool
	ln                   net.Listener
	Router               *mux.Router
	Server               *http.Server
	ShutdownGracePeriod  int
	Addr                 string
	HTTPServerResTimeout time.Duration
	Context              context.Context
	TLSEnable            bool
	TLSCertFilePath      string
	TLSKeyFilePath       string
}

// Init setups up basic routes for health checks and metrics
func (s *ServiceHTTP) Init() {
	s.Router = mux.NewRouter()
	s.Router.Use(RequestLoggingMiddleware)
	s.Router.HandleFunc("/ping", s.pingHandler)
	s.Router.HandleFunc("/log/level", s.logLevelPut)
	// See https://godoc.org/github.com/prometheus/client_golang/prometheus#pkg-examples
	// for examples of metrics
	s.Router.Handle("/metrics", promhttp.Handler())
	s.Router.Handle("/debug/vars", http.DefaultServeMux)

	// Server.Server
	s.corsHandler = cors.Default().Handler(s.Router)
	s.Server = &http.Server{
		Handler:      s.corsHandler,
		Addr:         s.Addr,
		WriteTimeout: 15 * time.Second,
		ReadTimeout:  15 * time.Second,
	}
}

// Play starts the server and begins listening for requests
func (s *ServiceHTTP) Play() error {
	if s.playing {
		return nil
	}
	core.Log.Warn("LISTENING ON " + s.Addr + " (serviceHTTP)")
	ln, err := reuse.Listen("tcp", s.Addr)
	if err != nil {
		core.Log.Error(fmt.Sprintf("error in reuseport listener: %s", s.Addr), err)
		return err
	}
	s.ln = ln
	s.playing = true

	go func() {
		var err error
		defer core.SentryRecover("ServiceHTTP.Play")

		if s.TLSEnable {
			var cw *core.CertWatcher
			if cw, err = core.CertWatcherNew(s.TLSCertFilePath, s.TLSKeyFilePath, core.Log); err != nil {
				core.Log.Fatal(err)
			}
			if err = cw.Watch(); err != nil {
				core.Log.Fatal(err)
			}

			s.Server.TLSConfig = &tls.Config{
				GetCertificate: cw.GetCertificate,
			}
			if err = s.Server.ServeTLS(ln, "", ""); err != nil {
				core.Log.Fatal(fmt.Sprintf("error in https Server: %s", s.Addr), err)
			}
		} else {
			if err = s.Server.Serve(ln); err != nil {
				core.Log.Fatal(fmt.Sprintf("error in http Server: %s", s.Addr), err)
			}
		}
	}()

	return nil
}

// Stop stops the service
func (s *ServiceHTTP) Stop() (err error) {
	if s.playing {
		return nil
	}

	err = s.Server.Close()
	if err != nil {
		return err
	}

	s.playing = false
	return nil
}

func (s *ServiceHTTP) logLevelPut(w http.ResponseWriter, r *http.Request) {
	level, err := logrus.ParseLevel(r.URL.Query().Get("level"))
	if err != nil {
		w.WriteHeader(400)
		w.Write([]byte("LOG_LEVEL Invalid"))
		return
	}
	core.Log.SetLevel(level)
	w.Write(core.ResponseOK)
}

func (s *ServiceHTTP) pingHandler(w http.ResponseWriter, r *http.Request) {
	w.Write([]byte("pong"))
}

// httpSentryHandler for panic reporting
// https://docs.sentry.io/platforms/go/http/
var httpSentryHandler = sentryhttp.New(sentryhttp.Options{
	Repanic:         true,
	WaitForDelivery: false,
})

// RouteAdd registers a path and handler
func (s *ServiceHTTP) RouteAdd(path string, handler Handler) {
	httpHandler := func(w http.ResponseWriter, r *http.Request) {
		var res []byte
		body, err := ioutil.ReadAll(r.Body)
		if err == nil {
			ctx, ctxCancelFn := context.WithTimeout(context.Background(), s.HTTPServerResTimeout)
			defer ctxCancelFn()
			res, err = handler(ctx, body)
		}
		if err != nil {
			w.WriteHeader(500)
			errTxt := err.Error()
			w.Write([]byte(errTxt))
			core.Log.Errorf("%s: %s", path, errTxt)
			return
		}
		// https://stackoverflow.com/questions/39415827/how-to-check-if-responsewriter-has-been-written
		w.Write(res)
		w.WriteHeader(200)
	}

	s.Router.PathPrefix("/" + path).HandlerFunc(httpSentryHandler.HandleFunc(httpHandler))
}

// RoutesAddN registers multiple routes
func (s *ServiceHTTP) RoutesAddN(routes map[string]Handler) {
	for path, handler := range routes {
		s.RouteAdd(path, handler)
	}
}
