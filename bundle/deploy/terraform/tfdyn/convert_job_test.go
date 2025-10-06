package tfdyn

import (
	"context"
	"testing"

	"github.com/databricks/cli/bundle/config/resources"
	"github.com/databricks/cli/bundle/internal/tf/schema"
	"github.com/databricks/cli/libs/dyn"
	"github.com/databricks/cli/libs/dyn/convert"
	"github.com/databricks/databricks-sdk-go/service/compute"
	"github.com/databricks/databricks-sdk-go/service/jobs"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestConvertJob(t *testing.T) {
	src := resources.Job{
		JobSettings: jobs.JobSettings{
			Name: "my job",
			JobClusters: []jobs.JobCluster{
				{
					JobClusterKey: "key",
					NewCluster: compute.ClusterSpec{
						SparkVersion: "10.4.x-scala2.12",
					},
				},
			},
			GitSource: &jobs.GitSource{
				GitProvider: jobs.GitProviderGitHub,
				GitUrl:      "https://github.com/foo/bar",
			},
			Parameters: []jobs.JobParameterDefinition{
				{
					Name:    "param1",
					Default: "default1",
				},
				{
					Name:    "param2",
					Default: "default2",
				},
			},
			Tasks: []jobs.Task{
				{
					TaskKey:       "task_key_b",
					JobClusterKey: "job_cluster_key_b",
					Libraries: []compute.Library{
						{
							Pypi: &compute.PythonPyPiLibrary{
								Package: "package",
							},
						},
						{
							Whl: "/path/to/my.whl",
						},
					},
				},
				{
					TaskKey:       "task_key_a",
					JobClusterKey: "job_cluster_key_a",
				},
				{
					TaskKey:       "task_key_c",
					JobClusterKey: "job_cluster_key_c",
				},
				{
					Description: "missing task key 😱",
				},
			},
		},
		Permissions: []resources.JobPermission{
			{
				Level:    "CAN_VIEW",
				UserName: "jane@doe.com",
			},
		},
	}

	vin, err := convert.FromTyped(src, dyn.NilValue)
	require.NoError(t, err)

	ctx := context.Background()
	out := schema.NewResources()
	err = jobConverter{}.Convert(ctx, "my_job", vin, out)
	require.NoError(t, err)

	// Assert equality on the job
	assert.Equal(t, map[string]any{
		"name": "my job",
		"job_cluster": []any{
			map[string]any{
				"job_cluster_key": "key",
				"new_cluster": map[string]any{
					"spark_version": "10.4.x-scala2.12",
				},
			},
		},
		"git_source": map[string]any{
			"provider": "gitHub",
			"url":      "https://github.com/foo/bar",
		},
		"parameter": []any{
			map[string]any{
				"name":    "param1",
				"default": "default1",
			},
			map[string]any{
				"name":    "param2",
				"default": "default2",
			},
		},
		"task": []any{
			map[string]any{
				"description": "missing task key 😱",
			},
			map[string]any{
				"task_key":        "task_key_a",
				"job_cluster_key": "job_cluster_key_a",
			},
			map[string]any{
				"task_key":        "task_key_b",
				"job_cluster_key": "job_cluster_key_b",
				"library": []any{
					map[string]any{
						"pypi": map[string]any{
							"package": "package",
						},
					},
					map[string]any{
						"whl": "/path/to/my.whl",
					},
				},
			},
			map[string]any{
				"task_key":        "task_key_c",
				"job_cluster_key": "job_cluster_key_c",
			},
		},
	}, out.Job["my_job"])

	// Assert equality on the permissions
	assert.Equal(t, &schema.ResourcePermissions{
		JobId: "${databricks_job.my_job.id}",
		AccessControl: []schema.ResourcePermissionsAccessControl{
			{
				PermissionLevel: "CAN_VIEW",
				UserName:        "jane@doe.com",
			},
		},
	}, out.Permissions["job_my_job"])
}

func TestConvertJobApplyPolicyDefaultValues(t *testing.T) {
	src := resources.Job{
		JobSettings: jobs.JobSettings{
			Name: "my job",
			JobClusters: []jobs.JobCluster{
				{
					JobClusterKey: "key",
					NewCluster: compute.ClusterSpec{
						ApplyPolicyDefaultValues: true,
						PolicyId:                 "policy_id",
						GcpAttributes: &compute.GcpAttributes{
							Availability:  "SPOT",
							LocalSsdCount: 2,
						},
					},
				},
				{
					JobClusterKey: "key2",
					NewCluster: compute.ClusterSpec{
						ApplyPolicyDefaultValues: true,
						PolicyId:                 "policy_id2",
						CustomTags: map[string]string{
							"key": "value",
						},
						InitScripts: []compute.InitScriptInfo{
							{
								Workspace: &compute.WorkspaceStorageInfo{
									Destination: "/Workspace/path/to/init_script1",
								},
							},
							{
								Workspace: &compute.WorkspaceStorageInfo{
									Destination: "/Workspace/path/to/init_script2",
								},
							},
						},
						SparkConf: map[string]string{
							"key": "value",
						},
						SparkEnvVars: map[string]string{
							"key": "value",
						},
						SshPublicKeys: []string{
							"ssh-rsa 1234",
						},
					},
				},
				{
					JobClusterKey: "key3",
					NewCluster: compute.ClusterSpec{
						ApplyPolicyDefaultValues: true,
						SparkVersion:             "16.4.x-scala2.12",
					},
				},
			},
		},
	}

	vin, err := convert.FromTyped(src, dyn.NilValue)
	require.NoError(t, err)

	ctx := context.Background()
	out := schema.NewResources()
	err = jobConverter{}.Convert(ctx, "my_job", vin, out)
	require.NoError(t, err)

	assert.Equal(t, map[string]any{
		"name": "my job",
		"job_cluster": []any{
			map[string]any{
				"job_cluster_key": "key",
				"new_cluster": map[string]any{
					"__apply_policy_default_values_allow_list": []any{
						"apply_policy_default_values",
						"gcp_attributes.availability",
						"gcp_attributes.local_ssd_count",
						"policy_id",
					},
					"apply_policy_default_values": true,
					"policy_id":                   "policy_id",
					"gcp_attributes": map[string]any{
						"availability":    "SPOT",
						"local_ssd_count": int64(2),
					},
				},
			},
			map[string]any{
				"job_cluster_key": "key2",
				"new_cluster": map[string]any{
					"__apply_policy_default_values_allow_list": []any{
						"apply_policy_default_values",
						"custom_tags",
						"init_scripts",
						"policy_id",
						"spark_conf",
						"spark_env_vars",
						"ssh_public_keys",
					},
					"apply_policy_default_values": true,
					"policy_id":                   "policy_id2",
					"custom_tags": map[string]any{
						"key": "value",
					},
					"init_scripts": []any{
						map[string]any{
							"workspace": map[string]any{
								"destination": "/Workspace/path/to/init_script1",
							},
						},
						map[string]any{
							"workspace": map[string]any{
								"destination": "/Workspace/path/to/init_script2",
							},
						},
					},
					"spark_conf": map[string]any{
						"key": "value",
					},
					"spark_env_vars": map[string]any{
						"key": "value",
					},
					"ssh_public_keys": []any{
						"ssh-rsa 1234",
					},
				},
			},
			map[string]any{
				"job_cluster_key": "key3",
				"new_cluster": map[string]any{
					"apply_policy_default_values": true,
					"spark_version":               "16.4.x-scala2.12",
				},
			},
		},
	}, out.Job["my_job"])
}
