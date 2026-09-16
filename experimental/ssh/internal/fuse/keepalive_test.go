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
					assert.Equal(t, testNotebookDir, body["notebookDir"])
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
				}, "12345", testNotebookDir)
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
				time.Sleep(59 * time.Second)
				synctest.Wait()
				mu.Lock()
				assert.Len(t, requests, initialRequests)
				assert.Equal(t, 1, tokenCalls)
				mu.Unlock()
				time.Sleep(time.Second)
				synctest.Wait()
				mu.Lock()
				refreshes := 1
				switch initialFailure {
				case "token", "both", "workspace":
					refreshes = 2
				}
				assert.Len(t, requests, initialRequests+refreshes)
				assert.Equal(t, 2, tokenCalls, "a failed startup must still start the refresh loop")

				before := len(requests)
				mu.Unlock()
				time.Sleep(3 * fuse.RefreshInterval)
				synctest.Wait()
				mu.Lock()
				assert.Len(t, requests, before+3, "steady state must only refresh volumes")
				for _, req := range requests[before:] {
					assert.Equal(t, "first.test:1015", req.Host)
				}
				before = len(requests)

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

func TestKeepRegisteredRecoversVolumesRegistrationLoss(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		ctx, cancel := context.WithCancel(t.Context())
		defer cancel()
		c, err := fuse.NewClient(registration)
		require.NoError(t, err)
		var mu sync.Mutex
		registered := map[string]bool{}
		requests := map[string]int{}
		httpClient := fuse.HTTPClient(c)
		httpClient.Transport = roundTripFunc(func(r *http.Request) (*http.Response, error) {
			mu.Lock()
			defer mu.Unlock()
			registered[r.URL.Port()] = true
			requests[r.URL.Port()]++
			return &http.Response{StatusCode: http.StatusOK, Body: io.NopCloser(strings.NewReader(""))}, nil
		})
		fuse.ConfigureTestClient(c, httpClient, testHosts, func() (int, error) { return registration.PID, nil })
		require.NoError(t, fuse.KeepRegistered(ctx, c, func(context.Context) (string, error) {
			return "fixed-bootstrap-token", nil
		}, "12345", testNotebookDir))
		synctest.Wait()

		for tick := range 3 {
			mu.Lock()
			// A UC-FUSE restart loses its mapping while WSFS still reports the server PID.
			registered["1015"] = false
			mu.Unlock()
			time.Sleep(fuse.RefreshInterval)
			synctest.Wait()
			mu.Lock()
			assert.True(t, registered["1015"], "restore volumes registration with an unchanged credential")
			assert.Equal(t, 1, requests["1021"], "healthy WSFS must not invalidate its caches")
			assert.Equal(t, tick+2, requests["1015"])
			mu.Unlock()
		}
	})
}

type keepaliveTransport struct {
	http.RoundTripper
	closed chan struct{}
}

func (transport *keepaliveTransport) CloseIdleConnections() {
	close(transport.closed)
}

func TestKeepRegisteredBlockedPIDProbe(t *testing.T) {
	for _, scenario := range []string{"cancel", "recover"} {
		t.Run(scenario, func(t *testing.T) {
			synctest.Test(t, func(t *testing.T) {
				ctx, cancel := context.WithCancel(t.Context())
				defer cancel()
				client, err := fuse.NewClient(registration)
				require.NoError(t, err)
				var mu sync.Mutex
				requests := map[string]int{}
				transport := &keepaliveTransport{
					closed: make(chan struct{}),
					RoundTripper: roundTripFunc(func(request *http.Request) (*http.Response, error) {
						mu.Lock()
						defer mu.Unlock()
						requests[request.URL.Port()]++
						return &http.Response{StatusCode: http.StatusOK, Body: http.NoBody}, nil
					}),
				}
				httpClient := fuse.HTTPClient(client)
				httpClient.Transport = transport
				releaseProbe := make(chan struct{}, 1)
				defer close(releaseProbe)
				probeCalls := 0
				fuse.ConfigureTestClient(client, httpClient, testHosts, func() (int, error) {
					mu.Lock()
					probeCalls++
					firstProbe := probeCalls == 1
					mu.Unlock()
					if firstProbe {
						<-releaseProbe
					}
					return registration.PID, nil
				})
				require.NoError(t, fuse.KeepRegistered(ctx, client, func(context.Context) (string, error) {
					return "fixed-bootstrap-token", nil
				}, "12345", testNotebookDir))
				synctest.Wait()
				start := time.Now()
				for tick := range 3 {
					time.Sleep(time.Until(start.Add(time.Duration(tick+1) * fuse.RefreshInterval)))
					synctest.Wait()
					mu.Lock()
					assert.Equal(t, tick+1, requests["1015"], "wait for the probe before refreshing")
					mu.Unlock()
					time.Sleep(fuse.PIDProbeTimeout)
					synctest.Wait()
					mu.Lock()
					assert.Equal(t, tick+2, requests["1015"], "keep refreshing volumes while WSFS is blocked")
					assert.Equal(t, tick+2, requests["1021"], "force WSFS registration after a probe timeout")
					assert.Equal(t, 1, probeCalls, "reuse the outstanding probe across refreshes")
					mu.Unlock()
				}

				if scenario == "recover" {
					releaseProbe <- struct{}{}
					synctest.Wait()
					time.Sleep(2 * fuse.RefreshInterval)
					synctest.Wait()
					mu.Lock()
					assert.Equal(t, 6, requests["1015"])
					assert.Equal(t, 4, requests["1021"], "stop forcing WSFS registration after probe recovery")
					assert.Equal(t, 2, probeCalls, "resume probing after the outstanding read completes")
					mu.Unlock()
				} else {
					time.Sleep(fuse.RefreshInterval - fuse.PIDProbeTimeout)
					synctest.Wait()
				}

				cancel()
				synctest.Wait()
				select {
				case <-transport.closed:
				default:
					assert.Fail(t, "refresh loop must exit on cancellation even with a blocked probe")
				}
				mu.Lock()
				callsAtCancel := probeCalls
				requestsAtCancel := requests["1015"]
				mu.Unlock()
				time.Sleep(2 * fuse.RefreshInterval)
				synctest.Wait()
				mu.Lock()
				assert.Equal(t, callsAtCancel, probeCalls)
				assert.Equal(t, requestsAtCancel, requests["1015"])
				mu.Unlock()
			})
		})
	}
}
