package variable

import (
	"context"
	"github.com/databricks/cli/libs/sdkclients"
	"github.com/databricks/cli/libs/utils"
	"github.com/databricks/sdk-go/clusterpolicies/v2"
)

type resolveClusterPolicy struct {
	name string
}

func (l resolveClusterPolicy) Resolve(ctx context.Context, w *sdkclients.Clients) (string, error) {
	resp, err := w.ClusterPolicies.ListPolicies(ctx, clusterpolicies.ListPoliciesRequest{})
	if err != nil {
		return "", err
	}
	entity, err := findByName(sliceSeq(resp.Policies), l.name, "Policy", func(p *clusterpolicies.Policy) string {
		return utils.Deref(p.Name)
	})
	if err != nil {
		return "", err
	}
	return utils.Deref(entity.PolicyId), nil
}

func (l resolveClusterPolicy) String() string {
	return "cluster-policy: " + l.name
}
