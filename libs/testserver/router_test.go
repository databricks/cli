package testserver_test

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/databricks/cli/libs/testserver"
	"github.com/stretchr/testify/assert"
)

type capture struct {
	handler  string
	vars     map[string]string
	notFound bool
}

func newRouter(t *testing.T) (*testserver.Router, *capture) {
	t.Helper()
	c := &capture{}
	r := testserver.NewRouter()
	r.Dispatch = func(w http.ResponseWriter, req *http.Request, h testserver.HandlerFunc, vars map[string]string) {
		c.vars = vars
		c.handler = h(testserver.Request{}).(string)
	}
	r.NotFound = func(w http.ResponseWriter, req *http.Request) {
		c.notFound = true
	}
	return r, c
}

func handlerNamed(name string) testserver.HandlerFunc {
	return func(req testserver.Request) any { return name }
}

func TestRouterExactMatch(t *testing.T) {
	r, c := newRouter(t)
	r.Handle("GET", "/foo", handlerNamed("foo-get"))

	r.ServeHTTP(httptest.NewRecorder(), httptest.NewRequest(http.MethodGet, "/foo", nil))
	assert.Equal(t, "foo-get", c.handler)
	assert.Nil(t, c.vars)
}

func TestRouterWildcardMatch(t *testing.T) {
	r, c := newRouter(t)
	r.Handle("GET", "/items/{id}", handlerNamed("item-get"))

	r.ServeHTTP(httptest.NewRecorder(), httptest.NewRequest(http.MethodGet, "/items/42", nil))
	assert.Equal(t, "item-get", c.handler)
	assert.Equal(t, map[string]string{"id": "42"}, c.vars)
}

func TestRouterCatchAllWildcard(t *testing.T) {
	r, c := newRouter(t)
	r.Handle("GET", "/files/{path...}", handlerNamed("files-get"))

	r.ServeHTTP(httptest.NewRecorder(), httptest.NewRequest(http.MethodGet, "/files/a/b/c", nil))
	assert.Equal(t, "files-get", c.handler)
	assert.Equal(t, map[string]string{"path": "a/b/c"}, c.vars)
}

func TestRouterMultipleWildcards(t *testing.T) {
	r, c := newRouter(t)
	r.Handle("GET", "/items/{id}/files/{path...}", handlerNamed("nested"))

	r.ServeHTTP(httptest.NewRecorder(), httptest.NewRequest(http.MethodGet, "/items/42/files/a/b", nil))
	assert.Equal(t, "nested", c.handler)
	assert.Equal(t, map[string]string{"id": "42", "path": "a/b"}, c.vars)
}

func TestRouterExactBeforeWildcard(t *testing.T) {
	r, c := newRouter(t)
	r.Handle("GET", "/foo", handlerNamed("exact"))
	r.Handle("HEAD", "/{path...}", handlerNamed("wildcard-head"))

	r.ServeHTTP(httptest.NewRecorder(), httptest.NewRequest(http.MethodGet, "/foo", nil))
	assert.Equal(t, "exact", c.handler)

	c.handler = ""
	r.ServeHTTP(httptest.NewRecorder(), httptest.NewRequest(http.MethodHead, "/foo", nil))
	assert.Equal(t, "wildcard-head", c.handler)
}

func TestRouterFirstRegistrationWins(t *testing.T) {
	t.Run("exact", func(t *testing.T) {
		r, c := newRouter(t)
		r.Handle("GET", "/foo", handlerNamed("first"))
		r.Handle("GET", "/foo", handlerNamed("second"))

		r.ServeHTTP(httptest.NewRecorder(), httptest.NewRequest(http.MethodGet, "/foo", nil))
		assert.Equal(t, "first", c.handler)
	})

	t.Run("wildcard", func(t *testing.T) {
		r, c := newRouter(t)
		r.Handle("GET", "/items/{id}", handlerNamed("first"))
		r.Handle("GET", "/items/{id}", handlerNamed("second"))

		r.ServeHTTP(httptest.NewRecorder(), httptest.NewRequest(http.MethodGet, "/items/42", nil))
		assert.Equal(t, "first", c.handler)
	})
}

func TestRouterNotFound(t *testing.T) {
	r, c := newRouter(t)
	r.ServeHTTP(httptest.NewRecorder(), httptest.NewRequest(http.MethodGet, "/missing", nil))
	assert.True(t, c.notFound)
}

func TestRouterMethodNotAllowed(t *testing.T) {
	t.Run("exact", func(t *testing.T) {
		r, c := newRouter(t)
		r.Handle("GET", "/foo", handlerNamed("foo-get"))
		r.ServeHTTP(httptest.NewRecorder(), httptest.NewRequest(http.MethodPost, "/foo", nil))
		assert.True(t, c.notFound, "wrong method on exact path should hit NotFound")
		assert.Empty(t, c.handler)
	})

	t.Run("wildcard", func(t *testing.T) {
		r, c := newRouter(t)
		r.Handle("GET", "/items/{id}", handlerNamed("item-get"))
		r.ServeHTTP(httptest.NewRecorder(), httptest.NewRequest(http.MethodPost, "/items/42", nil))
		assert.True(t, c.notFound, "wrong method on wildcard path should hit NotFound")
		assert.Empty(t, c.handler)
	})
}

func TestRouterPercentEncodedSlash(t *testing.T) {
	r, c := newRouter(t)
	r.Handle("GET", "/files/{path...}", handlerNamed("files-get"))

	req := httptest.NewRequest(http.MethodGet, "/files/a%2Fb%2Fc", nil)
	r.ServeHTTP(httptest.NewRecorder(), req)
	assert.Equal(t, "files-get", c.handler)
	assert.Equal(t, "a/b/c", c.vars["path"])
}
