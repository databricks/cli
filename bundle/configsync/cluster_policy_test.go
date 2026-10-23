package configsync_test

import (
	"encoding/json"
	"errors"
	"strings"
	"testing"

	"github.com/databricks/cli/bundle"
	"github.com/databricks/cli/bundle/configsync"
	"github.com/databricks/cli/bundle/deployplan"
	"github.com/databricks/cli/bundle/direct/dresources"
	"github.com/databricks/cli/libs/dyn"
	"github.com/databricks/cli/libs/dyn/jsonloader"
	"github.com/databricks/cli/libs/structs/structaccess"
	"github.com/databricks/databricks-sdk-go/experimental/mocks"
	"github.com/databricks/databricks-sdk-go/service/compute"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func TestExtractChangesDropsPolicyAppliedValues(t *testing.T) {
	tests := []struct {
		name          string
		attribute     string
		definition    string
		config        any
		remote        any
		applyDefaults bool
		noPolicy      bool
		want          any
	}{
		{
			name:       "fixed S3 log configuration dropped",
			attribute:  "cluster_log_conf",
			definition: `{"cluster_log_conf.type":{"type":"fixed","value":"S3"},"cluster_log_conf.path":{"type":"fixed","value":"s3://logs"},"cluster_log_conf.region":{"type":"fixed","value":"us-west-2"}}`,
			remote:     map[string]any{"s3": map[string]any{"destination": "s3://logs", "region": "us-west-2"}},
		},
		{
			name:          "partial log match keeps complete destination",
			attribute:     "cluster_log_conf",
			definition:    `{"cluster_log_conf.type":{"type":"fixed","value":"S3"},"cluster_log_conf.path":{"type":"fixed","value":"s3://logs"},"cluster_log_conf.region":{"type":"unlimited","defaultValue":"us-west-2"}}`,
			applyDefaults: true,
			remote:        map[string]any{"s3": map[string]any{"destination": "s3://logs", "region": "us-east-1"}},
			want:          map[string]any{"s3": map[string]any{"destination": "s3://logs", "region": "us-east-1"}},
		},
		{
			name:       "fixed scalar dropped when absent from config",
			attribute:  "autotermination_minutes",
			definition: `{"autotermination_minutes":{"type":"fixed","value":60}}`,
			remote:     int64(60),
		},
		{
			name:       "fixed value dropped without apply flag",
			attribute:  "spark_env_vars",
			definition: `{"spark_env_vars.ENV":{"type":"fixed","value":"dev"}}`,
			remote:     map[string]any{"ENV": "dev"},
		},
		{
			name:       "fixed tag dropped and UI tag kept",
			attribute:  "custom_tags",
			definition: `{"custom_tags.Environment":{"type":"fixed","value":"dev"}}`,
			remote:     map[string]any{"Environment": "dev", "Owner": "user"},
			want:       map[string]any{"Owner": "user"},
		},
		{
			name:       "authored field takes remote value",
			attribute:  "spark_env_vars",
			definition: `{"spark_env_vars.ENV":{"type":"fixed","value":"dev"}}`,
			config:     map[string]any{"ENV": "other"},
			remote:     map[string]any{"ENV": "dev"},
			want:       map[string]any{"ENV": "dev"},
		},
		{
			name:       "authored false is kept in whole object changes",
			attribute:  "workload_type",
			definition: `{"workload_type.clients.notebooks":{"type":"fixed","value":false}}`,
			config:     map[string]any{"clients": map[string]any{"notebooks": false}},
			remote:     map[string]any{"clients": map[string]any{"notebooks": false}},
			want:       map[string]any{"clients": map[string]any{"notebooks": false}},
		},
		{
			name:       "authored zero is kept in whole object changes",
			attribute:  "autoscale",
			definition: `{"autoscale.min_workers":{"type":"fixed","value":0}}`,
			config:     map[string]any{"min_workers": int64(0)},
			remote:     map[string]any{"min_workers": int64(0)},
			want:       map[string]any{"min_workers": int64(0)},
		},
		{
			name:          "default dropped with apply flag",
			attribute:     "spark_env_vars",
			definition:    `{"spark_env_vars.ENV":{"type":"allowlist","values":["dev","prod"],"defaultValue":"dev"}}`,
			remote:        map[string]any{"ENV": "dev"},
			applyDefaults: true,
		},
		{
			name:       "default kept without apply flag",
			attribute:  "spark_env_vars",
			definition: `{"spark_env_vars.ENV":{"type":"allowlist","values":["dev","prod"],"defaultValue":"dev"}}`,
			remote:     map[string]any{"ENV": "dev"},
			want:       map[string]any{"ENV": "dev"},
		},
		{
			name:          "non-default value kept",
			attribute:     "spark_env_vars",
			definition:    `{"spark_env_vars.ENV":{"type":"allowlist","values":["dev","prod"],"defaultValue":"dev"}}`,
			remote:        map[string]any{"ENV": "prod"},
			applyDefaults: true,
			want:          map[string]any{"ENV": "prod"},
		},
		{
			name:       "fixed false dropped and empty parents pruned",
			attribute:  "workload_type",
			definition: `{"workload_type.clients.notebooks":{"type":"fixed","value":false}}`,
			remote:     map[string]any{"clients": map[string]any{"notebooks": false}},
		},
		{
			name:          "zero default dropped",
			attribute:     "autoscale",
			definition:    `{"autoscale.min_workers":{"type":"range","minValue":0,"defaultValue":0}}`,
			remote:        map[string]any{"min_workers": int64(0)},
			applyDefaults: true,
		},
		{
			name:       "dotted map key matched as one key",
			attribute:  "spark_conf",
			definition: `{"spark_conf.spark.sql.shuffle.partitions":{"type":"fixed","value":"7"}}`,
			remote:     map[string]any{"spark.sql.shuffle.partitions": "7", "spark.sql.adaptive.enabled": "true"},
			want:       map[string]any{"spark.sql.adaptive.enabled": "true"},
		},
		{
			name:       "indexed rule drops whole array",
			attribute:  "init_scripts",
			definition: `{"init_scripts.0.volumes.destination":{"type":"fixed","value":"/Volumes/c/s/v/init.sh"}}`,
			remote:     []any{map[string]any{"volumes": map[string]any{"destination": "/Volumes/c/s/v/init.sh"}}},
		},
		{
			name:       "partial array match keeps whole array",
			attribute:  "init_scripts",
			definition: `{"init_scripts.0.volumes.destination":{"type":"fixed","value":"/Volumes/c/s/v/policy.sh"}}`,
			remote: []any{
				map[string]any{"volumes": map[string]any{"destination": "/Volumes/c/s/v/policy.sh"}},
				map[string]any{"volumes": map[string]any{"destination": "/Volumes/c/s/v/user.sh"}},
			},
			want: []any{
				map[string]any{"volumes": map[string]any{"destination": "/Volumes/c/s/v/policy.sh"}},
				map[string]any{"volumes": map[string]any{"destination": "/Volumes/c/s/v/user.sh"}},
			},
		},
		{
			name:       "log destination matches policy type and path",
			attribute:  "cluster_log_conf",
			definition: `{"cluster_log_conf.type":{"type":"fixed","value":"DBFS"},"cluster_log_conf.path":{"type":"fixed","value":"dbfs:/logs"}}`,
			remote:     map[string]any{"dbfs": map[string]any{"destination": "dbfs:/logs"}},
		},
		{
			name:          "log destination edits remain managed",
			attribute:     "cluster_log_conf",
			definition:    `{"cluster_log_conf.type":{"type":"fixed","value":"DBFS"},"cluster_log_conf.path":{"type":"unlimited","defaultValue":"dbfs:/logs"}}`,
			applyDefaults: true,
			remote:        map[string]any{"dbfs": map[string]any{"destination": "dbfs:/user-logs"}},
			want:          map[string]any{"dbfs": map[string]any{"destination": "dbfs:/user-logs"}},
		},
		{
			name:      "tags without a policy sync",
			attribute: "custom_tags",
			noPolicy:  true,
			remote:    map[string]any{"Owner": "user"},
			want:      map[string]any{"Owner": "user"},
		},
		{
			name:      "log configuration without a policy syncs",
			attribute: "cluster_log_conf",
			noPolicy:  true,
			remote:    map[string]any{"dbfs": map[string]any{"destination": "dbfs:/user-logs"}},
			want:      map[string]any{"dbfs": map[string]any{"destination": "dbfs:/user-logs"}},
		},
		{
			name:       "unconstrained values sync",
			attribute:  "custom_tags",
			definition: `{}`,
			remote:     map[string]any{"Owner": "user"},
			want:       map[string]any{"Owner": "user"},
		},
	}

	for _, location := range []string{"task", "shared", "for_each", "standalone"} {
		for _, tt := range tests {
			t.Run(location+"/"+tt.name, func(t *testing.T) {
				config := map[string]any{"policy_id": "policy", "apply_policy_default_values": tt.applyDefaults}
				if tt.config != nil {
					config[tt.attribute] = tt.config
				}
				remote := map[string]any{"policy_id": "policy", tt.attribute: tt.remote}
				if tt.noPolicy {
					delete(config, "policy_id")
					delete(remote, "policy_id")
				}
				key, prefix, configRoot := policyClusterLocation(location, config)
				_, _, remoteRoot := policyClusterLocation(location, remote)
				path := prefix + tt.attribute
				remoteState := typedPolicyResource(t, location, remoteRoot)
				remoteValue, err := structaccess.GetByString(remoteState, path)
				require.NoError(t, err)
				entry := &deployplan.PlanEntry{
					RemoteState: remoteState,
					Changes: deployplan.Changes{path: {
						Action: deployplan.Update,
						New:    tt.config,
						Remote: remoteValue,
					}},
				}
				b := policyTestBundle(t, key, configRoot)
				// Only policy retrieval is mocked; the actual comparison and filtering run.
				client := mocks.NewMockWorkspaceClient(t)
				b.SetWorkpaceClient(client.WorkspaceClient)
				if tt.config == nil && !tt.noPolicy {
					client.GetMockClusterPoliciesAPI().EXPECT().GetByPolicyId(mock.Anything, "policy").
						Return(&compute.Policy{Definition: tt.definition}, nil).Once()
				}

				changes, err := configsync.ExtractChanges(t.Context(), b, &deployplan.Plan{Plan: map[string]*deployplan.PlanEntry{key: entry}})
				require.NoError(t, err)
				if tt.want == nil {
					assert.Empty(t, changes)
				} else {
					require.Contains(t, changes, key)
					require.Contains(t, changes[key], path)
					assert.Equal(t, tt.want, changes[key][path].Value)
				}
			})
		}
	}
}

func TestExtractChangesFailsOnPolicyLookupError(t *testing.T) {
	b := policyTestBundle(t, "resources.clusters.test", map[string]any{"policy_id": "policy"})
	client := mocks.NewMockWorkspaceClient(t)
	b.SetWorkpaceClient(client.WorkspaceClient)
	lookupError := errors.New("policy access denied")
	client.GetMockClusterPoliciesAPI().EXPECT().GetByPolicyId(mock.Anything, "policy").Return(nil, lookupError).Once()
	entry := &deployplan.PlanEntry{
		RemoteState: &dresources.ClusterRemote{PolicyId: "policy"},
		Changes: deployplan.Changes{
			"spark_env_vars": {Action: deployplan.Update, Remote: map[string]any{"ENV": "dev"}},
		},
	}
	changes, err := configsync.ExtractChanges(t.Context(), b, &deployplan.Plan{Plan: map[string]*deployplan.PlanEntry{"resources.clusters.test": entry}})
	assert.ErrorIs(t, err, lookupError)
	assert.ErrorContains(t, err, "no configuration was saved")
	assert.Nil(t, changes)
}

func policyClusterLocation(location string, cluster map[string]any) (resourceKey, pathPrefix string, root map[string]any) {
	switch location {
	case "task":
		return "resources.jobs.test", "tasks[task_key='main'].new_cluster.", map[string]any{
			"tasks": []any{map[string]any{"task_key": "main", "new_cluster": cluster}},
		}
	case "shared":
		return "resources.jobs.test", "job_clusters[job_cluster_key='main'].new_cluster.", map[string]any{
			"job_clusters": []any{map[string]any{"job_cluster_key": "main", "new_cluster": cluster}},
		}
	case "for_each":
		return "resources.jobs.test", "tasks[task_key='each'].for_each_task.task.new_cluster.", map[string]any{
			"tasks": []any{map[string]any{"task_key": "each", "for_each_task": map[string]any{
				"task": map[string]any{"task_key": "main", "new_cluster": cluster},
			}}},
		}
	default:
		return "resources.clusters.test", "", cluster
	}
}

func typedPolicyResource(t *testing.T, location string, value map[string]any) any {
	t.Helper()
	var resource any = &dresources.JobRemote{}
	if location == "standalone" {
		resource = &dresources.ClusterRemote{}
	}
	data, err := json.Marshal(value)
	require.NoError(t, err)
	require.NoError(t, json.Unmarshal(data, resource))
	return resource
}

func TestExtractChangesMatchesClustersByKeyAndRemotePolicy(t *testing.T) {
	config := map[string]any{"job_clusters": []any{
		map[string]any{"job_cluster_key": "explicit", "new_cluster": map[string]any{
			"policy_id": "old-policy", "spark_env_vars": map[string]any{"ENV": "authored"},
		}},
		map[string]any{"job_cluster_key": "omitted", "new_cluster": map[string]any{"policy_id": "old-policy"}},
	}}
	remote := map[string]any{"job_clusters": []any{
		map[string]any{"job_cluster_key": "omitted", "new_cluster": map[string]any{
			"policy_id": "new-policy", "spark_env_vars": map[string]any{"ENV": "dev"},
		}},
		map[string]any{"job_cluster_key": "explicit", "new_cluster": map[string]any{
			"policy_id": "new-policy", "spark_env_vars": map[string]any{"ENV": "dev"},
		}},
	}}
	entry := &deployplan.PlanEntry{
		RemoteState: typedPolicyResource(t, "shared", remote),
		Changes: deployplan.Changes{"job_clusters": {
			Action: deployplan.Update, New: config["job_clusters"], Remote: remote["job_clusters"],
		}},
	}
	b := policyTestBundle(t, "resources.jobs.test", config)
	client := mocks.NewMockWorkspaceClient(t)
	b.SetWorkpaceClient(client.WorkspaceClient)
	// The omitted cluster uses the remote policy even though config has a different ID.
	client.GetMockClusterPoliciesAPI().EXPECT().GetByPolicyId(mock.Anything, "new-policy").
		Return(&compute.Policy{Definition: `{"spark_env_vars.ENV":{"type":"fixed","value":"dev"}}`}, nil).Once()

	changes, err := configsync.ExtractChanges(t.Context(), b, &deployplan.Plan{Plan: map[string]*deployplan.PlanEntry{"resources.jobs.test": entry}})
	require.NoError(t, err)
	require.Contains(t, changes, "resources.jobs.test")
	require.Contains(t, changes["resources.jobs.test"], "job_clusters")
	assert.Equal(t, []any{
		map[string]any{"job_cluster_key": "omitted", "new_cluster": map[string]any{"policy_id": "new-policy"}},
		map[string]any{"job_cluster_key": "explicit", "new_cluster": map[string]any{
			"policy_id": "new-policy", "spark_env_vars": map[string]any{"ENV": "dev"},
		}},
	}, changes["resources.jobs.test"]["job_clusters"].Value)
}

func policyTestBundle(t *testing.T, resourceKey string, config map[string]any) *bundle.Bundle {
	t.Helper()
	parts := strings.Split(resourceKey, ".")
	data, err := json.Marshal(map[string]any{"resources": map[string]any{
		parts[1]: map[string]any{parts[2]: config},
	}})
	require.NoError(t, err)
	value, err := jsonloader.LoadJSON(data, "test config")
	require.NoError(t, err)
	b := &bundle.Bundle{}
	require.NoError(t, b.Config.Mutate(func(_ dyn.Value) (dyn.Value, error) {
		return value, nil
	}))
	return b
}

func TestExtractChangesDropsPolicyValueAtDottedLeaf(t *testing.T) {
	key, prefix, config := policyClusterLocation("task", map[string]any{
		"policy_id":  "policy",
		"spark_conf": map[string]any{"spark.sql.adaptive.enabled": "true"},
	})
	_, _, remote := policyClusterLocation("task", map[string]any{
		"policy_id":  "policy",
		"spark_conf": map[string]any{"spark.sql.adaptive.enabled": "true", "spark.sql.shuffle.partitions": "7"},
	})
	b := policyTestBundle(t, key, config)
	client := mocks.NewMockWorkspaceClient(t)
	b.SetWorkpaceClient(client.WorkspaceClient)
	client.GetMockClusterPoliciesAPI().EXPECT().GetByPolicyId(mock.Anything, "policy").
		Return(&compute.Policy{Definition: `{"spark_conf.spark.sql.shuffle.partitions":{"type":"fixed","value":"7"}}`}, nil).Once()
	entry := &deployplan.PlanEntry{
		Action:      deployplan.Skip,
		RemoteState: typedPolicyResource(t, "task", remote),
		Changes: deployplan.Changes{prefix + "spark_conf['spark.sql.shuffle.partitions']": {
			Action: deployplan.Skip, Reason: deployplan.ReasonRemoteAddition, Remote: "7",
		}},
	}
	changes, err := configsync.ExtractChanges(t.Context(), b, &deployplan.Plan{Plan: map[string]*deployplan.PlanEntry{key: entry}})
	require.NoError(t, err)
	assert.Empty(t, changes)
}

func TestExtractChangesKeepsCompleteLogConfigurationWhenStorageChanges(t *testing.T) {
	key, prefix, config := policyClusterLocation("task", map[string]any{
		"policy_id": "policy", "apply_policy_default_values": true,
		"cluster_log_conf": map[string]any{"dbfs": map[string]any{"destination": "dbfs:/old-logs"}},
	})
	newDestination := map[string]any{"destination": "s3://logs", "region": "us-east-1"}
	_, _, remote := policyClusterLocation("task", map[string]any{
		"policy_id": "policy", "cluster_log_conf": map[string]any{"s3": newDestination},
	})
	remoteState := typedPolicyResource(t, "task", remote)
	path := prefix + "cluster_log_conf.s3"
	remoteValue, err := structaccess.GetByString(remoteState, path)
	require.NoError(t, err)
	entry := &deployplan.PlanEntry{
		RemoteState: remoteState,
		Changes: deployplan.Changes{
			path:                             {Action: deployplan.Update, Remote: remoteValue},
			prefix + "cluster_log_conf.dbfs": {Action: deployplan.Update, New: map[string]any{"destination": "dbfs:/old-logs"}},
		},
	}
	b := policyTestBundle(t, key, config)
	client := mocks.NewMockWorkspaceClient(t)
	b.SetWorkpaceClient(client.WorkspaceClient)
	client.GetMockClusterPoliciesAPI().EXPECT().GetByPolicyId(mock.Anything, "policy").
		Return(&compute.Policy{Definition: `{"cluster_log_conf.type":{"type":"fixed","value":"S3"},"cluster_log_conf.path":{"type":"fixed","value":"s3://logs"},"cluster_log_conf.region":{"type":"unlimited","defaultValue":"us-west-2"}}`}, nil).Once()

	changes, err := configsync.ExtractChanges(t.Context(), b, &deployplan.Plan{Plan: map[string]*deployplan.PlanEntry{key: entry}})
	require.NoError(t, err)
	require.Contains(t, changes, key)
	require.Contains(t, changes[key], path)
	assert.Equal(t, newDestination, changes[key][path].Value)
	assert.Equal(t, configsync.OperationRemove, changes[key][prefix+"cluster_log_conf.dbfs"].Operation)
}
