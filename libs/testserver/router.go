package testserver

import (
	"net/http"
	"strings"
)

// HandlerFunc is the test-server handler signature.
type HandlerFunc func(req Request) any

// Router maps method+path to a HandlerFunc. Wildcards use Go 1.22 ServeMux
// placeholder syntax ({name} or {name...}).
//
// # Why a custom router
//
// Go 1.22 added method matching ("GET /path") and {name}/{name...}
// placeholders to http.ServeMux, covering most of what we previously used
// gorilla/mux for. But two ServeMux behaviors make it inconvenient to use
// directly in the test server:
//
//   - Exact and wildcard patterns conflict when they cover the same
//     request under different methods. ServeMux treats `GET /x` as
//     matching both GET and HEAD, so it overlaps with `HEAD /{path...}`
//     and panics at registration. Test fixtures register both kinds of
//     routes side by side, so we keep exact paths in our own map and
//     only delegate wildcards to ServeMux. Exact lookup runs first;
//     misses fall through to ServeMux, which also lets a wildcard
//     handler serve methods that the exact registration doesn't cover.
//
//   - ServeMux panics on duplicate pattern registration. Router silently
//     drops the later registration — first wins. Two callers rely on this:
//     AddDefaultHandlers (libs/testserver/handlers.go) installs fallback
//     handlers that any test stub for the same pattern can override, and
//     startLocalServer (acceptance/internal/prepare_server.go) iterates
//     test.toml stubs in reverse so leaf-directory configs register first
//     and win over inherited parent stubs.
//
// Router also clears req.URL.RawPath before dispatch so percent-encoded
// slashes (%2F) match literal slashes in patterns; workspace file paths
// in tests routinely contain encoded slashes.
type Router struct {
	mux      *http.ServeMux
	exact    map[string]map[string]HandlerFunc
	wildcard map[string]bool

	// Dispatch is invoked when a route matches. vars holds the path values for
	// wildcard routes and is nil for exact routes.
	Dispatch func(w http.ResponseWriter, r *http.Request, h HandlerFunc, vars map[string]string)
	// NotFound is invoked when no route matches.
	NotFound http.HandlerFunc
}

func NewRouter() *Router {
	r := &Router{
		mux:      http.NewServeMux(),
		exact:    map[string]map[string]HandlerFunc{},
		wildcard: map[string]bool{},
	}
	r.mux.HandleFunc("/", func(w http.ResponseWriter, req *http.Request) {
		if r.NotFound != nil {
			r.NotFound(w, req)
		}
	})
	return r
}

// Handle registers a handler for method+path. First registration wins;
// duplicate (method, path) registrations are ignored.
func (r *Router) Handle(method, path string, handler HandlerFunc) {
	if !strings.Contains(path, "{") {
		if r.exact[path] == nil {
			r.exact[path] = map[string]HandlerFunc{}
		}
		if _, ok := r.exact[path][method]; !ok {
			r.exact[path][method] = handler
		}
		return
	}
	pattern := method + " " + path
	if r.wildcard[pattern] {
		return
	}
	r.wildcard[pattern] = true
	names := wildcardNames(path)
	r.mux.HandleFunc(pattern, func(w http.ResponseWriter, req *http.Request) {
		vars := make(map[string]string, len(names))
		for _, name := range names {
			vars[name] = req.PathValue(name)
		}
		if r.Dispatch != nil {
			r.Dispatch(w, req, handler, vars)
		}
	})
}

// ServeHTTP routes a request to the registered handler, falling back to
// NotFound if no route matches.
func (r *Router) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	// Force ServeMux to match against the decoded path; see the type doc.
	req.URL.RawPath = ""
	if methods, ok := r.exact[req.URL.Path]; ok {
		if h, ok := methods[req.Method]; ok {
			if r.Dispatch != nil {
				r.Dispatch(w, req, h, nil)
			}
			return
		}
	}
	r.mux.ServeHTTP(w, req)
}

// wildcardNames extracts wildcard parameter names from a path pattern,
// e.g. "/api/{id}/files/{path...}" returns ["id", "path"].
func wildcardNames(path string) []string {
	var names []string
	for part := range strings.SplitSeq(path, "/") {
		if strings.HasPrefix(part, "{") && strings.HasSuffix(part, "}") {
			name := part[1 : len(part)-1]
			name = strings.TrimSuffix(name, "...")
			names = append(names, name)
		}
	}
	return names
}
