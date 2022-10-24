package kittie

import (
	context "context"
	"net"
	"net/rpc"

	"github.com/jkassis/jerrie/core"
)

// ServiceNetRPC is a websocket server for mac that just polls sockets
type ServiceNetRPC struct {
	Addr    string
	Context context.Context
	ln      net.Listener
	playing bool
	server  *rpc.Server
}

// Init creates ther server
func (s *ServiceNetRPC) Init() {
	s.server = rpc.NewServer()
}

// Play starts the NetRPCServer
func (s *ServiceNetRPC) Play() {
	if s.playing {
		return
	}
	core.Log.Warn("LISTENING ON " + s.Addr + " (serviceNetRPC)")
	ln, err := net.Listen("tcp", s.Addr)
	if err != nil {
		core.Log.Fatal(err)
	}
	s.ln = ln
	s.playing = true

	go func() {
		defer core.SentryRecover("ServiceNetRPC.Play")

		s.server.Accept(ln)
	}()
	return
}

// Stop stops the service
func (s *ServiceNetRPC) Stop() error {
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
func (s *ServiceNetRPC) RouteAdd(path string, handler Handler) {
	route := &ServiceNetRPCRoute{Context: s.Context, Path: path, Handler: handler}
	if err := s.server.RegisterName(path, route); err != nil {
		core.Log.Fatal(err)
	}
}

// ServiceNetRPCRoute can be registered with ServiceNetRPC
type ServiceNetRPCRoute struct {
	Path    string
	Handler Handler
	Context context.Context
}

// ServiceNetRPCPostRequest is an incoming request
type ServiceNetRPCPostRequest struct {
	Data []byte
}

// ServiceNetRPCPostResponse is an outgoing response
type ServiceNetRPCPostResponse struct {
	Data []byte
}

// HandlePost runs a handler
func (s *ServiceNetRPCRoute) HandlePost(req ServiceNetRPCPostRequest, res *ServiceNetRPCPostResponse) error {
	core.Log.Tracef("ServiceNetRPC : handling %s", s.Path)
	handlerRes, err := s.Handler(s.Context, req.Data)
	if err != nil {
		core.Log.Errorf("ServiceNetRPC : %v", err)
		return err
	}
	res.Data = handlerRes
	return nil
}
