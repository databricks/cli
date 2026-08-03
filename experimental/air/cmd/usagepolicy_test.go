package aircmd

import (
	"net/url"
	"strconv"
	"testing"

	"github.com/databricks/cli/libs/testserver"
	"github.com/databricks/databricks-sdk-go"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// policyServer serves /api/2.0/serverless-policies from the given pages,
// returning one page per request and recording each request's query.
func policyServer(t *testing.T, pages ...usagePoliciesResponse) (*databricks.WorkspaceClient, *[]url.Values) {
	server := testserver.New(t)
	t.Cleanup(server.Close)

	var queries []url.Values
	var n int
	server.Handle("GET", "/api/2.0/serverless-policies", func(req testserver.Request) any {
		queries = append(queries, req.URL.Query())
		page := pages[min(n, len(pages)-1)]
		n++
		return page
	})
	testserver.AddDefaultHandlers(server)
	w, err := databricks.NewWorkspaceClient(&databricks.Config{Host: server.URL, Token: "token"})
	require.NoError(t, err)
	return w, &queries
}

func TestListUsagePoliciesSendsFilterAndPageSize(t *testing.T) {
	w, queries := policyServer(t, usagePoliciesResponse{
		Policies: []usagePolicy{{PolicyID: "id-1", PolicyName: "alpha"}},
	})

	policies, err := listUsagePolicies(t.Context(), w, "alpha")
	require.NoError(t, err)
	assert.Equal(t, []usagePolicy{{PolicyID: "id-1", PolicyName: "alpha"}}, policies)

	require.Len(t, *queries, 1)
	q := (*queries)[0]
	// The filter must arrive under its flattened proto path, not as a nested map.
	assert.Equal(t, "alpha", q.Get("filter_by.policy_name"))
	assert.Equal(t, strconv.Itoa(maxPolicyPageSize), q.Get("page_size"))
}

func TestListUsagePoliciesOmitsFilterWhenNoName(t *testing.T) {
	w, queries := policyServer(t, usagePoliciesResponse{})

	_, err := listUsagePolicies(t.Context(), w, "")
	require.NoError(t, err)

	require.Len(t, *queries, 1)
	assert.Empty(t, (*queries)[0].Get("filter_by.policy_name"))
}

func TestListUsagePoliciesPaginates(t *testing.T) {
	w, queries := policyServer(t,
		usagePoliciesResponse{Policies: []usagePolicy{{PolicyID: "id-1", PolicyName: "a"}}, NextPageToken: "tok"},
		usagePoliciesResponse{Policies: []usagePolicy{{PolicyID: "id-2", PolicyName: "b"}}},
	)

	policies, err := listUsagePolicies(t.Context(), w, "")
	require.NoError(t, err)
	assert.Equal(t, []usagePolicy{
		{PolicyID: "id-1", PolicyName: "a"},
		{PolicyID: "id-2", PolicyName: "b"},
	}, policies)

	require.Len(t, *queries, 2)
	assert.Equal(t, "tok", (*queries)[1].Get("page_token"))
}

// A page token that repeats itself must not spin forever.
func TestListUsagePoliciesStopsOnRepeatedToken(t *testing.T) {
	w, _ := policyServer(t, usagePoliciesResponse{
		Policies:      []usagePolicy{{PolicyID: "id-1", PolicyName: "a"}},
		NextPageToken: "same",
	})

	policies, err := listUsagePolicies(t.Context(), w, "")
	require.NoError(t, err)
	// Second response repeats "same", which ends the loop.
	assert.Len(t, policies, 2)
}

func TestResolveUsagePolicyIDByName(t *testing.T) {
	const id = "12345678-90ab-cdef-1234-567890abcdef"

	t.Run("exact match", func(t *testing.T) {
		w, _ := policyServer(t, usagePoliciesResponse{
			Policies: []usagePolicy{{PolicyID: id, PolicyName: "team-a"}},
		})
		got, err := resolveUsagePolicyIDByName(t.Context(), w, "team-a")
		require.NoError(t, err)
		assert.Equal(t, id, got)
	})

	// The server filter is partial; only the exact name (case-insensitively) wins.
	t.Run("case-insensitive exact match wins over partial", func(t *testing.T) {
		w, _ := policyServer(t, usagePoliciesResponse{
			Policies: []usagePolicy{
				{PolicyID: "other", PolicyName: "team-a-staging"},
				{PolicyID: id, PolicyName: "Team-A"},
			},
		})
		got, err := resolveUsagePolicyIDByName(t.Context(), w, "team-a")
		require.NoError(t, err)
		assert.Equal(t, id, got)
	})

	t.Run("no match suggests candidates", func(t *testing.T) {
		w, _ := policyServer(t, usagePoliciesResponse{
			Policies: []usagePolicy{{PolicyID: "x", PolicyName: "team-a-staging"}},
		})
		_, err := resolveUsagePolicyIDByName(t.Context(), w, "team-a")
		require.ErrorContains(t, err, `no usage policy named "team-a"`)
		require.ErrorContains(t, err, `Did you mean one of: "team-a-staging"?`)
	})

	t.Run("no match and no candidates omits the hint", func(t *testing.T) {
		w, _ := policyServer(t, usagePoliciesResponse{})
		_, err := resolveUsagePolicyIDByName(t.Context(), w, "team-a")
		require.ErrorContains(t, err, `no usage policy named "team-a"`)
		assert.NotContains(t, err.Error(), "Did you mean")
	})

	t.Run("suggestions are capped", func(t *testing.T) {
		var policies []usagePolicy
		for i := range maxPolicySuggestions + 5 {
			// Zero-padded so lexical order matches numeric order.
			policies = append(policies, usagePolicy{PolicyID: strconv.Itoa(i), PolicyName: "team-a-" + strconv.Itoa(100+i)})
		}
		w, _ := policyServer(t, usagePoliciesResponse{Policies: policies})
		_, err := resolveUsagePolicyIDByName(t.Context(), w, "team-a")
		require.ErrorContains(t, err, `"team-a-109", ...?`)
		assert.NotContains(t, err.Error(), "team-a-110")
	})

	t.Run("ambiguous match refuses to guess", func(t *testing.T) {
		w, _ := policyServer(t, usagePoliciesResponse{
			Policies: []usagePolicy{
				{PolicyID: "id-1", PolicyName: "team-a"},
				{PolicyID: "id-2", PolicyName: "TEAM-A"},
			},
		})
		_, err := resolveUsagePolicyIDByName(t.Context(), w, "team-a")
		require.ErrorContains(t, err, "multiple usage policies match")
		require.ErrorContains(t, err, `"id-1", "id-2"`)
	})

	t.Run("match without an id is an error", func(t *testing.T) {
		w, _ := policyServer(t, usagePoliciesResponse{
			Policies: []usagePolicy{{PolicyName: "team-a"}},
		})
		_, err := resolveUsagePolicyIDByName(t.Context(), w, "team-a")
		require.ErrorContains(t, err, "has no policy_id")
	})

	// An empty name would drop the server-side filter and list every policy in the
	// workspace, so it is rejected without a round-trip.
	t.Run("blank name is rejected", func(t *testing.T) {
		w, queries := policyServer(t, usagePoliciesResponse{})
		_, err := resolveUsagePolicyIDByName(t.Context(), w, "   ")
		require.ErrorContains(t, err, "must be a non-empty string")
		assert.Empty(t, *queries)
	})
}
