package dresources

import (
	"context"
	"errors"
	"io"
	"math"
	"net/http"
	"regexp"
	"slices"
	"strings"
	"sync"
	"testing"

	"github.com/databricks/cli/libs/env"
	"github.com/databricks/cli/libs/retry"
	"github.com/databricks/cli/libs/testserver"
	"github.com/databricks/databricks-sdk-go"
	"github.com/databricks/databricks-sdk-go/apierr"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// createRecoveryOptOut lists the resources whose DoCreate does not recover from a failed create
// request, with the reason. TestCreateRecoveryCoverage fails for a resource that neither recovers
// nor is listed here, and for a listed resource that recovers.
//
// A resource recovers in one of two ways:
//   - its create is idempotent (a PUT, or a set of the desired value), so DoCreate wraps the error
//     with retrySafe and DeploymentUnit.Create repeats the request;
//   - the bundle chooses its name and the API keeps it unique, so DoCreate calls recoverCreate,
//     which looks the resource up by name and uses it if it was created after the first request.
//
// Neither works for a resource whose id the backend assigns, unless the create request takes an
// idempotency token, or for one whose read response has no create time: a resource found by name
// could then predate the deploy, and taking it over would let two bundles share it.
var createRecoveryOptOut = map[string]string{
	"jobs":             "the backend assigns the job id, names are not unique, and the create request takes no idempotency token",
	"pipelines":        "the backend assigns the pipeline id, names are not unique, and the create request takes no idempotency token",
	"clusters":         "the backend assigns the cluster id, names are not unique, and the create request takes no idempotency token",
	"sql_warehouses":   "the backend assigns the warehouse id, and the create request takes no idempotency token",
	"instance_pools":   "the backend assigns the pool id, and the create request takes no idempotency token",
	"alerts":           "the backend assigns the alert id, display names are not unique, and the create request takes no idempotency token",
	"genie_spaces":     "the backend assigns the space id, titles are not unique, and the create request takes no idempotency token",
	"feature_specs":    "the read response has no create time",
	"secret_scopes":    "the read response has no create time",
	"quality_monitors": "the read response has no create time",

	"data_quality_monitors":  "the read response has no create time",
	"vector_search_indexes":  "the read response has no create time",
	"synced_database_tables": "the read response has no create time; the Unity Catalog table it registers would have to be read for one",

	"internal_immutable_snapshots": "the snapshot is an immutable upload, and it is not known whether a second upload to the same path is accepted",

	// Candidates: these could recover, each with a change of its own.
	"job_runs":             "candidate: DoCreate mints the run-now idempotency token per request; repeating the request with the same token would return the run it started",
	"materialization_sets": "candidate: DoCreate mints the request_id per request; repeating the request with the same request_id would return the set it created",
	"cluster_policies":     "candidate: policy names are unique and the read response has created_at_timestamp, but the backend assigns the id, so the lookup has to go by name",
	"apps":                 "candidate: app names are unique and the read response has create_time, but DoCreate already handles an app that is being deleted, and recovery has to fit with that",
	"database_instances":   "candidate: instance names are unique and the read response has creation_time; not done yet",
	"experiments":          "candidate: experiment names are unique and the read response has creation_time, but libs/testserver has neither get-by-name nor creation_time",
	"models":               "candidate: model names are unique and the read response has creation_timestamp, but libs/testserver does not set it",
	"postgres_data_apis":   "candidate: a database has one Data API, but the read response takes its spec from the status, which can be empty while it is being enabled, so a recovered Data API needs a wait the create operation would otherwise do",
}

// mutatingRequest is the method and path of a request that is not a read.
type mutatingRequest struct {
	method string
	path   string
}

// idSegment matches a path segment that holds an id the backend assigned, such as a job id or a
// UUID: those differ between the test servers.
var idSegment = regexp.MustCompile(`/[0-9a-f-]*[0-9][0-9a-f-]*(/|$)`)

// withoutIDs returns the request with the ids in its path replaced by a placeholder.
func (r mutatingRequest) withoutIDs() mutatingRequest {
	return mutatingRequest{r.method, idSegment.ReplaceAllString(r.path, "/{id}$1")}
}

// faultTransport records the mutating requests sent once armed, and fails the one at index failAt
// with a 504 that carries an empty message, as the gateway's timeouts do. With applied, the request
// reaches the server first, so its effect is kept.
type faultTransport struct {
	mu       sync.Mutex
	armed    bool
	failAt   int
	applied  bool
	injected *mutatingRequest
	requests []mutatingRequest
}

func (f *faultTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	inject := false
	if req.Method != http.MethodGet {
		f.mu.Lock()
		if f.armed {
			r := mutatingRequest{req.Method, req.URL.Path}
			inject = len(f.requests) == f.failAt
			if inject {
				f.injected = &r
			}
			f.requests = append(f.requests, r)
		}
		f.mu.Unlock()
	}
	if !inject {
		return http.DefaultTransport.RoundTrip(req)
	}
	if f.applied {
		resp, err := http.DefaultTransport.RoundTrip(req)
		if err != nil {
			return nil, err
		}
		_, _ = io.Copy(io.Discard, resp.Body)
		_ = resp.Body.Close()
	}
	return &http.Response{
		StatusCode: http.StatusGatewayTimeout,
		Status:     "504 Gateway Timeout",
		Header:     http.Header{"Content-Type": []string{"application/json"}},
		Body:       io.NopCloser(strings.NewReader(`{"error_code": "DEADLINE_EXCEEDED", "message": ""}`)),
		Request:    req,
	}, nil
}

// arm starts recording, and failing the request at index failAt if it is not negative.
func (f *faultTransport) arm(failAt int, applied bool) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.armed = true
	f.failAt = failAt
	f.applied = applied
}

func (f *faultTransport) recorded() []mutatingRequest {
	f.mu.Lock()
	defer f.mu.Unlock()
	return slices.Clone(f.requests)
}

// newFaultClient returns a client for a new test server that sends its requests through the
// returned transport.
func newFaultClient(t *testing.T) (*databricks.WorkspaceClient, *faultTransport) {
	server := testserver.New(t)
	testserver.AddDefaultHandlers(server)
	transport := &faultTransport{}
	client, err := databricks.NewWorkspaceClient(&databricks.Config{
		Host:               server.URL,
		Token:              "testtoken",
		RateLimitPerSecond: math.MaxInt,
		HTTPTransport:      transport,
	})
	require.NoError(t, err)
	return client, transport
}

// doCreateLikeApply calls DoCreate and, like DeploymentUnit.Create in bundle/direct/apply.go,
// repeats it while IsRetrySafe. repeated reports whether it did.
func doCreateLikeApply(ctx context.Context, adapter *Adapter, state any) (id string, repeated bool, err error) {
	id, err = retry.With(ctx, func(ctx context.Context, err error) bool {
		safe := IsRetrySafe(ctx, err)
		repeated = repeated || safe
		return safe
	}, func() (string, error) {
		id, _, err := adapter.DoCreate(ctx, state)
		return id, err
	})
	return id, repeated, UnwrapRetrySafe(err)
}

// TestCreateRecoveryCoverage checks that every resource's DoCreate, run as the deploy runs it,
// succeeds when its create request fails with a 504, both when the backend applied the request and
// when it did not. A resource that cannot recover goes in createRecoveryOptOut with the reason.
func TestCreateRecoveryCoverage(t *testing.T) {
	for resourceType, resource := range SupportedResources {
		t.Run(resourceType, func(t *testing.T) {
			ctx := env.Set(t.Context(), retry.IntervalMsVariable, "1")

			// Find the create request: the first mutating request a successful DoCreate sends. DoCreate
			// may send it more than once on its way to success (a dashboard's is repeated after
			// creating its folder), so fail the last of them: the one the backend applies.
			client, transport := newFaultClient(t)
			adapter, err := NewAdapter(resource, resourceType, client)
			require.NoError(t, err)
			state := prepareTestState(t, resourceType, adapter, client)
			transport.arm(-1, false)
			_, _, err = adapter.DoCreate(ctx, state)
			require.NoError(t, err)
			requests := transport.recorded()
			require.NotEmpty(t, requests, "DoCreate sent no mutating request")
			createReq := requests[0]
			failAt, occurrences := 0, 0
			for i, req := range requests {
				if req == createReq {
					failAt = i
					occurrences++
				}
			}

			for _, applied := range []bool{true, false} {
				name := "applied"
				if !applied {
					name = "not_applied"
				}
				t.Run(name, func(t *testing.T) {
					client, transport := newFaultClient(t)
					adapter, err := NewAdapter(resource, resourceType, client)
					require.NoError(t, err)
					state := prepareTestState(t, resourceType, adapter, client)
					transport.arm(failAt, applied)
					id, repeated, err := doCreateLikeApply(ctx, adapter, state)
					require.NotNil(t, transport.injected, "DoCreate sent fewer mutating requests than on success")
					require.Equal(t, createReq.withoutIDs(), transport.injected.withoutIDs(), "DoCreate sent other requests than on success")

					if reason, ok := createRecoveryOptOut[resourceType]; ok {
						apiErr, isAPIErr := errors.AsType[*apierr.APIError](err)
						require.True(t, isAPIErr && apiErr.StatusCode == http.StatusGatewayTimeout,
							"%s recovers from a failed %s %s or fails for another reason (%v); if it recovers, remove it from createRecoveryOptOut (%s)", resourceType, createReq.method, createReq.path, err, reason)
						return
					}
					require.NoError(t, err, "%s does not recover from a failed %s %s; call recoverCreate in DoCreate, or add it to createRecoveryOptOut with the reason", resourceType, createReq.method, createReq.path)
					remote, err := adapter.DoRead(ctx, id)
					require.NoError(t, err)
					require.NotNil(t, remote)

					if applied && !repeated {
						// The resource exists, so the recovery must find it rather than send the request
						// again, which for a backend-assigned id would create a second one.
						sent := 0
						for _, req := range transport.recorded() {
							if req == *transport.injected {
								sent++
							}
						}
						assert.Equal(t, occurrences, sent, "%s sent %s %s again after the backend applied it", resourceType, createReq.method, createReq.path)
					}
				})
			}
		})
	}
}
