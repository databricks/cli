package fuse_test

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strings"
	"sync"
	"testing"
	"testing/synctest"
	"time"

	"github.com/databricks/cli/experimental/ssh/internal/fuse"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestKeepRegistered(t *testing.T) {
	for _, initialFailure := range []string{"", "token", "workspace", "volumes", "both"} {
		t.Run("initialFailure="+initialFailure, func(t *testing.T) {
			synctest.Test(t, func(t *testing.T) {
				ctx, cancel := context.WithCancel(t.Context())
				defer cancel()
				c, err := fuse.NewClient(registration)
				require.NoError(t, err)
				var mu sync.Mutex
				failure := initialFailure
				token := "token-one"
				pid := registration.PID
				var probeErr error
				var requests []recordedRequest
				httpClient := fuse.HTTPClient(c)
				httpClient.Transport = roundTripFunc(func(r *http.Request) (*http.Response, error) {
					mu.Lock()
					defer mu.Unlock()
					var body map[string]any
					if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
						return nil, err
					}
					assert.NotEmpty(t, body["apiToken"], "cancellation must not send a revoke")
					requests = append(requests, recordedRequest{r.Host, r.URL.Path, body})
					status := http.StatusOK
					if failure == "both" || (failure == "workspace" && r.URL.Port() == "1021") || (failure == "volumes" && r.URL.Port() == "1015") {
						status = http.StatusServiceUnavailable
					}
					return &http.Response{StatusCode: status, Body: io.NopCloser(strings.NewReader(""))}, nil
				})
				fuse.ConfigureTestClient(c, httpClient, testHosts, func() (int, error) { mu.Lock(); defer mu.Unlock(); return pid, probeErr })
				tokenCalls := 0
				err = fuse.KeepRegistered(ctx, c, func(context.Context) (string, error) {
					mu.Lock()
					defer mu.Unlock()
					tokenCalls++
					if failure == "token" {
						return "", errors.New("token unavailable")
					}
					return token, nil
				}, "12345")
				if initialFailure == "" {
					require.NoError(t, err)
				} else {
					require.Error(t, err)
				}
				synctest.Wait()
				mu.Lock()
				initialRequests := len(requests)
				failure = ""
				mu.Unlock()
				time.Sleep(fuse.RefreshInterval)
				synctest.Wait()
				mu.Lock()
				retries := 0
				switch initialFailure {
				case "token", "both":
					retries = 2
				case "workspace", "volumes":
					retries = 1
				}
				assert.Len(t, requests, initialRequests+retries)
				assert.Equal(t, 2, tokenCalls, "a failed startup must still start the refresh loop")

				before := len(requests)
				mu.Unlock()
				time.Sleep(3 * fuse.RefreshInterval)
				synctest.Wait()
				mu.Lock()
				assert.Len(t, requests, before, "steady state must not re-register")

				token = "token-two"
				mu.Unlock()
				time.Sleep(fuse.RefreshInterval)
				synctest.Wait()
				mu.Lock()
				require.Len(t, requests, before+2)
				assert.Equal(t, token, requests[before].Body["apiToken"])
				assert.Equal(t, token, requests[before+1].Body["apiToken"])

				pid = registration.PID - 1
				mu.Unlock()
				time.Sleep(fuse.RefreshInterval)
				synctest.Wait()
				mu.Lock()
				assert.Len(t, requests, before+4, "restore registration when a different ancestor is reported")
				pid = registration.PID
				probeErr = errors.New("filesystem metadata unavailable")
				mu.Unlock()
				time.Sleep(fuse.RefreshInterval)
				synctest.Wait()
				mu.Lock()
				assert.Len(t, requests, before+6, "restore registration when the probe fails")
				probeErr = nil
				cancel()
				mu.Unlock()
				synctest.Wait()
				mu.Lock()
				callsAtCancel := tokenCalls
				mu.Unlock()
				time.Sleep(2 * fuse.RefreshInterval)
				synctest.Wait()
				mu.Lock()
				assert.Equal(t, callsAtCancel, tokenCalls)
				assert.Len(t, requests, before+6)
				mu.Unlock()
			})
		})
	}
}
