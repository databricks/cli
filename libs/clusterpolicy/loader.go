package clusterpolicy

import (
	"context"
	"fmt"

	"github.com/databricks/databricks-sdk-go/service/compute"
)

// LookupError wraps a failure to read or parse the policy with the given ID.
type LookupError struct {
	PolicyID string
	Err      error
}

func (e *LookupError) Error() string {
	return fmt.Sprintf("reading cluster policy %s: %v", e.PolicyID, e.Err)
}

func (e *LookupError) Unwrap() error { return e.Err }

// Loader reads policy definitions by ID and caches both definitions and failures for
// its lifetime, so one loader serves a single operation against one workspace.
type Loader struct {
	api         compute.ClusterPoliciesInterface
	definitions map[string]loadedDefinition
}

type loadedDefinition struct {
	rules Definition
	err   error
}

func NewLoader(api compute.ClusterPoliciesInterface) *Loader {
	return &Loader{api: api, definitions: make(map[string]loadedDefinition)}
}

// Get returns the effective definition of a policy, including policy-family overrides.
// Failures are returned as *LookupError.
func (l *Loader) Get(ctx context.Context, policyID string) (Definition, error) {
	if definition, ok := l.definitions[policyID]; ok {
		return definition.rules, definition.err
	}
	var rules Definition
	policy, err := l.api.GetByPolicyId(ctx, policyID)
	if err == nil {
		rules, err = ParseDefinition(policy.Definition)
	}
	if err != nil {
		err = &LookupError{PolicyID: policyID, Err: err}
	}
	l.definitions[policyID] = loadedDefinition{rules: rules, err: err}
	return rules, err
}
