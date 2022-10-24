package kittie

import "github.com/jkassis/jerrie/core"

type (
	// ServiceRouter inspects requests and routes to handlers/services that are registered
	ServiceRouter struct {
		Routes map[string]Handler
	}
)

// RoutesAdd registers a path and handler
func (server *ServiceRouter) RoutesAdd(path string, handler Handler) {
	server.Routes[path] = handler
}

// RoutesAddN registers multiple routes
func (server *ServiceRouter) RoutesAddN(routes map[string]Handler) {
	core.Log.Warnf("ServiceRouter: RoutesAddN: adding %d routes", len(routes))
	for path, handler := range routes {
		server.RoutesAdd(path, handler)
		core.Log.Warnf("ServiceRouter: RoutesAddN: adding '%s'", path)
	}
}
