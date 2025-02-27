package mutator

import (
	"context"
	"testing"

	"github.com/databricks/cli/bundle"
	"github.com/databricks/cli/bundle/config"
	"github.com/databricks/cli/bundle/config/variable"
	"github.com/databricks/cli/libs/env"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"github.com/databricks/databricks-sdk-go/experimental/mocks"
	"github.com/databricks/databricks-sdk-go/service/compute"
	"github.com/databricks/databricks-sdk-go/service/iam"
)

func TestResolveClusterReference(t *testing.T) {
	clusterRef1 := "Some Custom Cluster"
	clusterRef2 := "Some Other Name"
	justString := "random string"
	b := &bundle.Bundle{
		Config: config.Root{
			Variables: map[string]*variable.Variable{
				"my-cluster-id-1": {
					Lookup: &variable.Lookup{
						Cluster: clusterRef1,
					},
				},
				"my-cluster-id-2": {
					Lookup: &variable.Lookup{
						Cluster: clusterRef2,
					},
				},
				"some-variable": {
					Value: justString,
				},
			},
		},
	}

	m := mocks.NewMockWorkspaceClient(t)
	b.SetWorkpaceClient(m.WorkspaceClient)
	clusterApi := m.GetMockClustersAPI()
	clusterApi.EXPECT().ListAll(mock.Anything, compute.ListClustersRequest{
		FilterBy: &compute.ListClustersFilterBy{
			ClusterSources: []compute.ClusterSource{compute.ClusterSourceApi, compute.ClusterSourceUi},
		},
	}).Return([]compute.ClusterDetails{
		{ClusterId: "1234-5678-abcd", ClusterName: clusterRef1},
		{ClusterId: "9876-5432-xywz", ClusterName: clusterRef2},
	}, nil)

	diags := bundle.Apply(context.Background(), b, ResolveResourceReferences())
	require.NoError(t, diags.Error())
	require.Equal(t, "1234-5678-abcd", b.Config.Variables["my-cluster-id-1"].Value)
	require.Equal(t, "9876-5432-xywz", b.Config.Variables["my-cluster-id-2"].Value)
}

func TestResolveNonExistentClusterReference(t *testing.T) {
	clusterRef := "Random"
	justString := "random string"
	b := &bundle.Bundle{
		Config: config.Root{
			Variables: map[string]*variable.Variable{
				"my-cluster-id": {
					Lookup: &variable.Lookup{
						Cluster: clusterRef,
					},
				},
				"some-variable": {
					Value: justString,
				},
			},
		},
	}

	m := mocks.NewMockWorkspaceClient(t)
	b.SetWorkpaceClient(m.WorkspaceClient)
	clusterApi := m.GetMockClustersAPI()
	clusterApi.EXPECT().ListAll(mock.Anything, compute.ListClustersRequest{
		FilterBy: &compute.ListClustersFilterBy{
			ClusterSources: []compute.ClusterSource{compute.ClusterSourceApi, compute.ClusterSourceUi},
		},
	}).Return([]compute.ClusterDetails{
		{ClusterId: "1234-5678-abcd", ClusterName: "some other cluster"},
	}, nil)

	diags := bundle.Apply(context.Background(), b, ResolveResourceReferences())
	require.ErrorContains(t, diags.Error(), "failed to resolve cluster: Random, err: cluster named 'Random' does not exist")
}

func TestNoLookupIfVariableIsSet(t *testing.T) {
	clusterRef := "donotexist"
	b := &bundle.Bundle{
		Config: config.Root{
			Variables: map[string]*variable.Variable{
				"my-cluster-id": {
					Lookup: &variable.Lookup{
						Cluster: clusterRef,
					},
				},
			},
		},
	}

	m := mocks.NewMockWorkspaceClient(t)
	b.SetWorkpaceClient(m.WorkspaceClient)

	err := b.Config.Variables["my-cluster-id"].Set("random value")
	require.NoError(t, err)

	diags := bundle.Apply(context.Background(), b, ResolveResourceReferences())
	require.NoError(t, diags.Error())
	require.Equal(t, "random value", b.Config.Variables["my-cluster-id"].Value)
}

func TestResolveServicePrincipal(t *testing.T) {
	spName := "Some SP name"
	b := &bundle.Bundle{
		Config: config.Root{
			Variables: map[string]*variable.Variable{
				"my-sp": {
					Lookup: &variable.Lookup{
						ServicePrincipal: spName,
					},
				},
			},
		},
	}

	m := mocks.NewMockWorkspaceClient(t)
	b.SetWorkpaceClient(m.WorkspaceClient)
	spApi := m.GetMockServicePrincipalsAPI()
	spApi.EXPECT().GetByDisplayName(mock.Anything, spName).Return(&iam.ServicePrincipal{
		Id:            "1234",
		ApplicationId: "app-1234",
	}, nil)

	diags := bundle.Apply(context.Background(), b, ResolveResourceReferences())
	require.NoError(t, diags.Error())
	require.Equal(t, "app-1234", b.Config.Variables["my-sp"].Value)
}

func TestResolveVariableReferencesInVariableLookups(t *testing.T) {
	s := "bar"
	b := &bundle.Bundle{
		Config: config.Root{
			Bundle: config.Bundle{
				Target: "dev",
			},
			Variables: map[string]*variable.Variable{
				"foo": {
					Value: s,
				},
				"lookup": {
					Lookup: &variable.Lookup{
						Cluster: "cluster-${var.foo}-${bundle.target}",
					},
				},
			},
		},
	}

	m := mocks.NewMockWorkspaceClient(t)
	b.SetWorkpaceClient(m.WorkspaceClient)
	clusterApi := m.GetMockClustersAPI()

	clusterApi.EXPECT().ListAll(mock.Anything, compute.ListClustersRequest{
		FilterBy: &compute.ListClustersFilterBy{
			ClusterSources: []compute.ClusterSource{compute.ClusterSourceApi, compute.ClusterSourceUi},
		},
	}).Return([]compute.ClusterDetails{
		{ClusterId: "1234-5678-abcd", ClusterName: "cluster-bar-dev"},
		{ClusterId: "9876-5432-xywz", ClusterName: "some other cluster"},
	}, nil)

	diags := bundle.ApplySeq(context.Background(), b, ResolveVariableReferencesInLookup(), ResolveResourceReferences())
	require.NoError(t, diags.Error())
	require.Equal(t, "cluster-bar-dev", b.Config.Variables["lookup"].Lookup.Cluster)
	require.Equal(t, "1234-5678-abcd", b.Config.Variables["lookup"].Value)
}

func TestResolveLookupVariableReferencesInVariableLookups(t *testing.T) {
	b := &bundle.Bundle{
		Config: config.Root{
			Variables: map[string]*variable.Variable{
				"another_lookup": {
					Lookup: &variable.Lookup{
						Cluster: "cluster",
					},
				},
				"lookup": {
					Lookup: &variable.Lookup{
						Cluster: "cluster-${var.another_lookup}",
					},
				},
			},
		},
	}

	m := mocks.NewMockWorkspaceClient(t)
	b.SetWorkpaceClient(m.WorkspaceClient)

	diags := bundle.ApplySeq(context.Background(), b, ResolveVariableReferencesInLookup(), ResolveResourceReferences())
	require.ErrorContains(t, diags.Error(), "lookup variables cannot contain references to another lookup variables")
}

func TestNoResolveLookupIfVariableSetWithEnvVariable(t *testing.T) {
	b := &bundle.Bundle{
		Config: config.Root{
			Bundle: config.Bundle{
				Target: "dev",
			},
			Variables: map[string]*variable.Variable{
				"lookup": {
					Lookup: &variable.Lookup{
						Cluster: "cluster-${bundle.target}",
					},
				},
			},
		},
	}

	m := mocks.NewMockWorkspaceClient(t)
	b.SetWorkpaceClient(m.WorkspaceClient)

	ctx := context.Background()
	ctx = env.Set(ctx, "BUNDLE_VAR_lookup", "1234-5678-abcd")

	diags := bundle.ApplySeq(ctx, b, SetVariables(), ResolveVariableReferencesInLookup(), ResolveResourceReferences())
	require.NoError(t, diags.Error())
	require.Equal(t, "1234-5678-abcd", b.Config.Variables["lookup"].Value)
}
