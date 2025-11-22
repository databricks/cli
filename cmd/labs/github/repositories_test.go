package github

import (
	"context"
	"fmt"
	"maps"
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestRepositories(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/users/databrickslabs/repos" {
			_, err := w.Write([]byte(`[{"name": "x"}]`))
			assert.NoError(t, err)
			return
		}
		t.Logf("Requested: %s", r.URL.Path)
		panic("stub required")
	}))
	defer server.Close()

	ctx := context.Background()
	ctx = WithApiOverride(ctx, server.URL)

	r := NewRepositoryCache("databrickslabs", t.TempDir())
	all, err := r.Load(ctx)
	assert.NoError(t, err)
	assert.NotEmpty(t, all)
}

func TestRepositoriesPagination(t *testing.T) {
	var requestedURLs []string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requestedURLs = append(requestedURLs, r.URL.String())
		switch r.URL.Path {
		case "/users/databrickslabs/repos":
			requestedQueryValues := r.URL.Query()
			linkTo := func(page int, rel string) string {
				nextQueryValues := maps.Clone(requestedQueryValues)
				nextQueryValues.Set("page", strconv.Itoa(page))
				builder := *r.URL
				builder.Scheme = "http"
				builder.Host = r.Host
				builder.RawQuery = nextQueryValues.Encode()
				return fmt.Sprintf(`<%s>; rel="%s"`, builder.String(), rel)
			}
			page := requestedQueryValues.Get("page")
			var link string
			var body string
			switch page {
			// Pagination logic with next and prev links for 3 pages of results.
			case "", "1":
				link = linkTo(2, "next")
				body = `[{"name": "repo1"}, {"name": "repo2"}]`
			case "2":
				link = linkTo(1, "prev") + ", " + linkTo(3, "next")
				body = `[{"name": "repo3"}, {"name": "repo4"}]`
			case "3":
				link = linkTo(2, "prev")
				body = `[{"name": "repo5"}]`
			}
			w.Header().Set("link", link)
			_, err := w.Write([]byte(body))
			assert.NoError(t, err)
			return
		}
		t.Logf("Requested: %s", r.URL.String())
		panic("stub required")
	}))
	defer server.Close()

	ctx := context.Background()
	ctx = WithApiOverride(ctx, server.URL)

	repos, err := getRepositories(ctx, "databrickslabs")
	assert.NoError(t, err)
	var names []string
	for _, repo := range repos {
		names = append(names, repo.Name)
	}
	assert.Equal(t, []string{
		"/users/databrickslabs/repos?per_page=100",
		"/users/databrickslabs/repos?page=2&per_page=100",
		"/users/databrickslabs/repos?page=3&per_page=100",
	}, requestedURLs)
	assert.Equal(t, []string{"repo1", "repo2", "repo3", "repo4", "repo5"}, names)
}
