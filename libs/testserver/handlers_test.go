package testserver

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

// routeKeys returns the method and pattern of every route registered on server.
func routeKeys(server *Server) map[string]bool {
	keys := map[string]bool{}
	for path, methods := range server.exact {
		for method := range methods {
			keys[method+" "+path] = true
		}
	}
	for pattern := range server.wildcard {
		keys[pattern] = true
	}
	return keys
}

// The router keeps the first registration of a route, so a route registered by
// two APIs would be served by whichever AddDefaultHandlers happens to run first.
func TestDefaultRoutesDoNotOverlap(t *testing.T) {
	base := routeKeys(New(t))

	owners := map[string]string{}
	for _, r := range sortedRouteRegistrations() {
		server := New(t)
		r.register(server)
		keys := routeKeys(server)
		assert.Greater(t, len(keys), len(base), "%s registers no routes", r.key)
		for key := range keys {
			if base[key] {
				continue
			}
			if owner, ok := owners[key]; ok {
				t.Errorf("%s is registered by both %s and %s", key, owner, r.key)
			}
			owners[key] = r.key
		}
	}

	server := New(t)
	AddDefaultHandlers(server)
	assert.Len(t, routeKeys(server), len(base)+len(owners))
}
