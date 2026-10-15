package mutator

import (
	"fmt"
	"testing"

	"github.com/databricks/cli/bundle"
	"github.com/databricks/cli/bundle/config"
	"github.com/databricks/cli/bundle/config/variable"
	"github.com/databricks/cli/libs/env"
	"github.com/databricks/cli/libs/testserver"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
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

	server := testserver.New(t)
	server.Handle("GET", "/api/2.1/clusters/list", func(req testserver.Request) any {
		assert.Equal(t, []string{"API", "UI"}, req.URL.Query()["filter_by.cluster_sources"])
		return map[string]any{"clusters": []map[string]any{
			{"cluster_id": "1234-5678-abcd", "cluster_name": clusterRef1},
			{"cluster_id": "9876-5432-xywz", "cluster_name": clusterRef2},
		}}
	})
	b.SetWorkpaceClient(newTestClients(t, server))

	diags := bundle.Apply(t.Context(), b, ResolveLookupVariables())
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

	server := testserver.New(t)
	server.Handle("GET", "/api/2.1/clusters/list", func(req testserver.Request) any {
		assert.Equal(t, []string{"API", "UI"}, req.URL.Query()["filter_by.cluster_sources"])
		return map[string]any{"clusters": []map[string]any{
			{"cluster_id": "1234-5678-abcd", "cluster_name": "some other cluster"},
		}}
	})
	b.SetWorkpaceClient(newTestClients(t, server))

	diags := bundle.Apply(t.Context(), b, ResolveLookupVariables())
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

	b.SetWorkpaceClient(newTestClients(t, testserver.New(t)))

	err := b.Config.Variables["my-cluster-id"].Set("random value")
	require.NoError(t, err)

	diags := bundle.Apply(t.Context(), b, ResolveLookupVariables())
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

	server := testserver.New(t)
	server.Handle("GET", "/api/2.0/preview/scim/v2/ServicePrincipals", func(req testserver.Request) any {
		assert.Equal(t, fmt.Sprintf("displayName eq '%s'", spName), req.URL.Query().Get("filter"))
		return map[string]any{"Resources": []map[string]any{{"applicationId": "app-1234"}}}
	})
	b.SetWorkpaceClient(newTestClients(t, server))

	diags := bundle.Apply(t.Context(), b, ResolveLookupVariables())
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

	server := testserver.New(t)
	server.Handle("GET", "/api/2.1/clusters/list", func(req testserver.Request) any {
		assert.Equal(t, []string{"API", "UI"}, req.URL.Query()["filter_by.cluster_sources"])
		return map[string]any{"clusters": []map[string]any{
			{"cluster_id": "1234-5678-abcd", "cluster_name": "cluster-bar-dev"},
			{"cluster_id": "9876-5432-xywz", "cluster_name": "some other cluster"},
		}}
	})
	b.SetWorkpaceClient(newTestClients(t, server))

	diags := bundle.ApplySeq(t.Context(), b, ResolveVariableReferencesInLookup(), ResolveLookupVariables())
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

	b.SetWorkpaceClient(newTestClients(t, testserver.New(t)))

	diags := bundle.ApplySeq(t.Context(), b, ResolveVariableReferencesInLookup(), ResolveLookupVariables())
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

	b.SetWorkpaceClient(newTestClients(t, testserver.New(t)))

	ctx := t.Context()
	ctx = env.Set(ctx, "BUNDLE_VAR_lookup", "1234-5678-abcd")

	diags := bundle.ApplySeq(ctx, b, SetVariables(), ResolveVariableReferencesInLookup(), ResolveLookupVariables())
	require.NoError(t, diags.Error())
	require.Equal(t, "1234-5678-abcd", b.Config.Variables["lookup"].Value)
}
