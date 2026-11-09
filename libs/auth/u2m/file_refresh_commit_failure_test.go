package u2m_test

import (
	"bytes"
	"errors"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/databricks/cli/libs/auth/storage"
	"github.com/databricks/cli/libs/auth/u2m"
	"github.com/databricks/databricks-sdk-go/httpclient/fixtures"
	"golang.org/x/oauth2"
)

type fileCommitFailureTransport struct {
	path      string
	backup    string
	fault     bool
	armed     bool
	spent     bool
	calls     []int
	successes int
	replays   int
}

func (p *fileCommitFailureTransport) RoundTrip(request *http.Request) (*http.Response, error) {
	body, err := io.ReadAll(request.Body)
	if err != nil {
		return nil, errors.New("closed request read failure")
	}
	request.Body = io.NopCloser(bytes.NewReader(body))
	form, err := url.ParseQuery(string(body))
	if err != nil || form.Get("grant_type") != "refresh_token" {
		return nil, errors.New("closed unexpected grant")
	}
	generation := tokenGeneration(form.Get("refresh_token"))
	p.calls = append(p.calls, generation)
	if generation != 0 {
		return nil, errors.New("closed unexpected generation")
	}
	status := http.StatusOK
	response := `{"access_token":"generation-1","refresh_token":"generation-1","token_type":"Bearer","expires_in":3600}`
	if p.spent {
		p.replays++
		status = http.StatusBadRequest
		response = `{"error":"invalid_grant","error_description":"Refresh token is invalid"}`
	}
	transport := fixtures.SliceTransport{{
		Method: "POST", Resource: "/oidc/accounts/account/v1/token",
		Status: status, Response: response,
		ResponseHeaders: map[string][]string{"Content-Type": {"application/json"}},
	}}
	reply, err := transport.RoundTrip(request)
	if err != nil || status != http.StatusOK {
		return reply, err
	}
	p.spent = true
	p.successes++
	if p.fault {
		// Reuse the owning transaction fixture's actual atomic-publication fault.
		if err := os.Rename(p.path, p.backup); err != nil {
			return nil, errors.New("closed fault rename failure")
		}
		if err := os.Mkdir(p.path, 0o700); err != nil {
			return nil, errors.New("closed fault directory failure")
		}
		p.armed = true
	}
	return reply, nil
}

func TestFileRefreshCommitFailure(t *testing.T) {
	for _, fault := range []bool{false, true} {
		name := "healthy"
		if fault {
			name = "commit_failure_fresh_instance"
		}
		t.Run(name, func(t *testing.T) {
			cachePath := filepath.Join(t.TempDir(), "tokens.json")
			probe := &fileCommitFailureTransport{path: cachePath, backup: cachePath + ".saved", fault: fault}
			arg, err := u2m.NewBasicAccountOAuthArgument("https://accounts.databricks.test", "account")
			if err != nil {
				t.Fatal("closed argument setup failure")
			}
			initial, err := storage.NewFileStore(t.Context(), storage.WithFileLocation(cachePath))
			if err != nil {
				t.Fatal("closed initial FileStore setup failure")
			}
			if err := initial.Put(arg.GetCacheKey(), storage.Entry{Token: &oauth2.Token{
				AccessToken: "generation-0", RefreshToken: "generation-0", Expiry: time.Now().Add(-time.Hour),
			}}); err != nil {
				t.Fatal("closed initial generation setup failure")
			}
			original, err := os.ReadFile(cachePath)
			if err != nil {
				t.Fatal("closed cache snapshot failure")
			}
			newAuth := func() *u2m.PersistentAuth {
				store, err := storage.NewFileStore(t.Context(), storage.WithFileLocation(cachePath))
				if err != nil {
					t.Fatal("closed fresh FileStore setup failure")
				}
				auth, err := u2m.NewPersistentAuth(t.Context(), u2m.WithTokenStore(store),
					u2m.WithOAuthArgument(arg), u2m.WithClientID("synthetic-client"),
					u2m.WithOAuthEndpointSupplier(&u2m.BasicOAuthEndpointSupplier{}),
					u2m.WithHttpClient(&http.Client{Transport: probe}))
				if err != nil {
					t.Fatal("closed fresh auth setup failure")
				}
				t.Cleanup(func() { _ = auth.Close() })
				return auth
			}
			first := newAuth()
			firstToken, firstErr := first.Token()
			commitFailed := firstErr != nil && strings.Contains(firstErr.Error(), "cache update")
			if probe.successes != 1 || len(probe.calls) != 1 || probe.calls[0] != 0 {
				t.Fatal("closed first exchange precondition failure")
			}
			if fault {
				if !probe.armed || !commitFailed || firstToken != nil {
					t.Fatal("closed commit-failure precondition failure")
				}
				if err := os.Remove(cachePath); err != nil {
					t.Fatal("closed storage repair remove failure")
				}
				if err := os.Rename(probe.backup, cachePath); err != nil {
					t.Fatal("closed storage repair rename failure")
				}
				probe.armed = false
				restored, err := os.ReadFile(cachePath)
				if err != nil || !bytes.Equal(original, restored) {
					t.Fatal("closed original cache equality failure")
				}
			} else if firstErr != nil || firstToken == nil || tokenGeneration(firstToken.AccessToken) != 1 {
				t.Fatal("closed healthy first result failure")
			}
			_ = first.Close()
			fresh := newAuth()
			freshToken, freshErr := fresh.Token()
			cached, err := initial.Lookup(arg.GetCacheKey())
			if err != nil || cached.Token == nil {
				t.Fatal("closed final cache lookup failure")
			}
			cachedGeneration := tokenGeneration(cached.Token.RefreshToken)
			freshGeneration := -1
			if freshToken != nil {
				freshGeneration = tokenGeneration(freshToken.AccessToken)
			}
			t.Logf("CLOSED_PROBE fault=%t remote_successes=%d commit_failed=%t calls=%d generations=%v replays=%d cached_generation=%d fresh_failed=%t fresh_generation=%d",
				fault, probe.successes, commitFailed, len(probe.calls), probe.calls, probe.replays, cachedGeneration, freshErr != nil, freshGeneration)
			if !fault && (freshErr != nil || freshGeneration != 1 || cachedGeneration != 1) {
				t.Fatal("healthy cache publication contract failed")
			}
			if len(probe.calls) != 1 || probe.replays != 0 {
				t.Fatalf("fresh instance must not replay spent generation: calls=%d replays=%d", len(probe.calls), probe.replays)
			}
		})
	}
}
