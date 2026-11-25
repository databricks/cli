package testserver

import (
	"slices"
	"strings"
	"sync"

	"github.com/databricks/databricks-sdk-go/service/catalog"
)

// TestWorkspaceID is the id of the fake workspace.
const TestWorkspaceID = 900800700600

var TestMetastore = catalog.MetastoreAssignment{
	DefaultCatalogName: "hive_metastore",
	MetastoreId:        "120efa64-9b68-46ba-be38-f319458430d2",
	WorkspaceId:        TestWorkspaceID,
}

// routeRegistration holds the default routes of one fake API.
type routeRegistration struct {
	key      string
	register func(server *Server)
}

var routeRegistrations []routeRegistration

// registerRoutes adds the default routes of one fake API. Call it from the init
// function of the API's file, with the file name as key.
func registerRoutes(key string, register func(server *Server)) {
	routeRegistrations = append(routeRegistrations, routeRegistration{key: key, register: register})
}

// sortedRouteRegistrations orders the registrations by key, so the order does not
// depend on the order init functions run in. Registrations with the same key keep
// the order they were made in.
var sortedRouteRegistrations = sync.OnceValue(func() []routeRegistration {
	sorted := slices.Clone(routeRegistrations)
	slices.SortStableFunc(sorted, func(a, b routeRegistration) int {
		return strings.Compare(a.key, b.key)
	})
	return sorted
})

func AddDefaultHandlers(server *Server) {
	for _, r := range sortedRouteRegistrations() {
		r.register(server)
	}
}
