package variable

import (
	"context"

	"github.com/databricks/databricks-sdk-go"
)

type resolveClusterPolicy struct {
	name string
}

func (l resolveClusterPolicy) Resolve(ctx context.Context, w *databricks.WorkspaceClient) (string, error) {
	entity, err := w.ClusterPolicies.GetByName(ctx, l.name)
	if err != nil {
		return "", err
	}
	return entity.PolicyId, nil
}

func (l resolveClusterPolicy) String() string {
	return "cluster-policy: " + l.name
}
