package kittie

import (
	context "context"
	fmt "fmt"
	"net"
	"sync"

	"github.com/jkassis/jerrie/core"
	log "github.com/sirupsen/logrus"
	grpc "google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type (
	// ServiceGRPC is a websocket server for mac that just polls sockets
	ServiceGRPC struct {
		sync.Mutex
		playing bool
		ln      net.Listener
		Addr    string
		Server  *grpc.Server
		Routes  map[string]Handler
		Context context.Context
	}
)

// MakeGRPCServer creates an event poller (for mac in this case which ignores epolling cause mac doesn't support)
func MakeGRPCServer() (*ServiceGRPC, error) {
	return &ServiceGRPC{}, nil
}

// Init creates ther server
func (s *ServiceGRPC) Init() {
	s.Server = grpc.NewServer()
	s.Routes = make(map[string]Handler, 0)
	RegisterGRPCServer(s.Server, s)
}

// Play starts the GRPCServer
func (s *ServiceGRPC) Play() error {
	if s.playing {
		return nil
	}
	core.Log.Warn("LISTENING ON " + s.Addr + " (serviceGRPC)")
	ln, err := net.Listen("tcp", s.Addr)
	if err != nil {
		core.Log.Error(fmt.Sprintf("error in reuseport listener: %s", s.Addr), err)
		return err
		// return err
	}
	s.ln = ln
	s.playing = true

	// start the server
	go func() {
		defer core.SentryRecover("ServiceGRPC.Play")

		if err := s.Server.Serve(ln); err != nil {
			log.Error(err)
		}
	}()

	return nil
}

// Stop stops the service
func (s *ServiceGRPC) Stop() error {
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

// RouteAdd registers a path and handler
func (s *ServiceGRPC) RouteAdd(path string, handler Handler) error {
	s.Lock()
	s.Routes[path] = handler
	s.Unlock()
	return nil
}

// HandlePost handles a request
func (s *ServiceGRPC) HandlePost(ctx context.Context, in *ServiceGRPCPostRequest) (*ServiceGRPCPostResponse, error) {
	out, err := s.Routes[in.GetRoute()](s.Context, in.GetData())
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}

	response := &ServiceGRPCPostResponse{}
	response.Data = out
	return response, nil
}
