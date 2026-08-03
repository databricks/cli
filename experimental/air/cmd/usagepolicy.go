package aircmd

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"slices"
	"strings"

	"github.com/databricks/databricks-sdk-go"
	"github.com/databricks/databricks-sdk-go/client"
)

// serverlessPoliciesPath is the workspace-scoped ListBudgetPolicies endpoint on
// the serverless-policy service. This is called with a raw client.Do because the
// SDK only models the account-scoped /api/2.1/accounts/{id}/budget-policies
// service, which is a different (and unusable here) endpoint.
const serverlessPoliciesPath = "/api/2.0/serverless-policies"

// maxPolicyPageSize is the server's cap: anything larger is coerced down to it.
// Request the max so the common case (a workspace with a handful of policies) is
// a single round-trip.
const maxPolicyPageSize = 1000

// maxPolicySuggestions bounds the candidate names surfaced in a "no exact match"
// error: enough to spot a typo or casing mistake without dumping a huge list.
const maxPolicySuggestions = 10

type usagePolicy struct {
	PolicyID   string `json:"policy_id"`
	PolicyName string `json:"policy_name"`
}

type usagePoliciesResponse struct {
	Policies      []usagePolicy `json:"policies"`
	NextPageToken string        `json:"next_page_token"`
}

// listUsagePolicies pages the serverless-policy index for policies matching
// policyName, which is sent as filter_by.policy_name: a partial,
// case-insensitive server-side filter. Callers must pass a non-empty name.
//
// The filter key is spelled with its dotted proto path rather than as a nested
// map: the SDK only flattens nesting for struct-typed query values, and would
// format a nested map with %v into a useless "map[...]" literal.
func listUsagePolicies(ctx context.Context, w *databricks.WorkspaceClient, policyName string) ([]usagePolicy, error) {
	apiClient, err := client.New(w.Config)
	if err != nil {
		return nil, fmt.Errorf("failed to create API client: %w", err)
	}

	var out []usagePolicy
	// The index can return the same policy on more than one page, and a stuck or
	// cycling cursor can repeat a whole page; dedupe both so an unambiguous name
	// never looks like an ambiguous match downstream.
	seenIDs := map[string]bool{}
	seenTokens := map[string]bool{}
	var pageToken string
	for {
		query := map[string]any{
			"page_size":             maxPolicyPageSize,
			"filter_by.policy_name": policyName,
		}
		if pageToken != "" {
			query["page_token"] = pageToken
		}

		var resp usagePoliciesResponse
		err = apiClient.Do(ctx, http.MethodGet, serverlessPoliciesPath, nil, nil, query, &resp)
		if err != nil {
			return nil, fmt.Errorf("failed to list usage policies: %w", err)
		}

		for _, p := range resp.Policies {
			if seenIDs[p.PolicyID] {
				continue
			}
			seenIDs[p.PolicyID] = true
			out = append(out, p)
		}

		if resp.NextPageToken == "" || seenTokens[resp.NextPageToken] {
			return out, nil
		}
		seenTokens[resp.NextPageToken] = true
		pageToken = resp.NextPageToken
	}
}

// resolveUsagePolicyIDByName resolves a usage policy name to its UUID policy id.
//
// The server-side filter is a partial match, so the exact (but case-insensitive)
// match is re-applied locally; policy names are unique among active policies.
func resolveUsagePolicyIDByName(ctx context.Context, w *databricks.WorkspaceClient, name string) (string, error) {
	target := strings.TrimSpace(name)
	// Guard the contract independently of the YAML validator: an empty filter would
	// otherwise list (then reject against) every policy in the workspace.
	if target == "" {
		return "", errors.New("a usage policy name must be a non-empty string")
	}

	policies, err := listUsagePolicies(ctx, w, target)
	if err != nil {
		return "", err
	}

	var matches []usagePolicy
	for _, p := range policies {
		if strings.EqualFold(strings.TrimSpace(p.PolicyName), target) {
			matches = append(matches, p)
		}
	}

	switch len(matches) {
	case 1:
		if matches[0].PolicyID == "" {
			return "", fmt.Errorf("policy %q has no policy_id in the API response", target)
		}
		return matches[0].PolicyID, nil

	case 0:
		// policies holds the partial-match candidates the server returned for this
		// name; surface a few to help the user fix a typo or casing.
		return "", fmt.Errorf("no usage policy named %q was found in this workspace%s", target, suggestionHint(policies))

	default:
		// Multiple exact (case-insensitive) matches should not happen given name
		// uniqueness, but guard so we never silently pick the wrong policy.
		ids := make([]string, 0, len(matches))
		for _, p := range matches {
			ids = append(ids, fmt.Sprintf("%q", p.PolicyID))
		}
		return "", fmt.Errorf("multiple usage policies match the name %q (ids: %s); please disambiguate with your workspace admin",
			target, strings.Join(ids, ", "))
	}
}

// suggestionHint renders a deduplicated, sorted "did you mean" clause for the
// candidates the partial filter returned, or "" when there are none.
func suggestionHint(candidates []usagePolicy) string {
	names := make([]string, 0, len(candidates))
	for _, p := range candidates {
		if p.PolicyName != "" {
			names = append(names, p.PolicyName)
		}
	}
	slices.Sort(names)
	names = slices.Compact(names)
	if len(names) == 0 {
		return ""
	}

	shown := names
	suffix := ""
	if len(names) > maxPolicySuggestions {
		shown = names[:maxPolicySuggestions]
		suffix = ", ..."
	}
	quoted := make([]string, 0, len(shown))
	for _, n := range shown {
		quoted = append(quoted, fmt.Sprintf("%q", n))
	}
	return fmt.Sprintf(". Did you mean one of: %s%s?", strings.Join(quoted, ", "), suffix)
}
