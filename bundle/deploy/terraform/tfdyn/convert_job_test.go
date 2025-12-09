package tfdyn

import (
	"context"
	"reflect"
	"slices"
	"strings"
	"testing"

	"github.com/databricks/cli/bundle/config/resources"
	"github.com/databricks/cli/bundle/internal/tf/schema"
	"github.com/databricks/cli/libs/dyn"
	"github.com/databricks/cli/libs/dyn/convert"
	"github.com/databricks/cli/libs/textutil"
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

// TestSupportedTypeTasksComplete verifies that supportedTypeTasks includes all task types with a Source field.
func TestSupportedTypeTasksComplete(t *testing.T) {
	// Use reflection to find all task types that have a Source field
	taskType := reflect.TypeOf(jobs.Task{})
	var tasksWithSource []string

	for i := range taskType.NumField() {
		field := taskType.Field(i)

		// Skip non-task fields (like DependsOn, Libraries, etc.)
		if !strings.HasSuffix(field.Name, "Task") {
			continue
		}

		// Get the type of the task field (e.g., *NotebookTask)
		taskFieldType := field.Type
		if taskFieldType.Kind() == reflect.Ptr {
			taskFieldType = taskFieldType.Elem()
		}

		if taskFieldType.Kind() != reflect.Struct {
			continue
		}

		// Recursively search for Source fields in this task type
		// We only search one level deep to catch nested Source fields like sql_task.file
		taskName := textutil.CamelToSnakeCase(field.Name)
		sourcePaths := findSourceFieldsShallow(taskFieldType)
		for _, path := range sourcePaths {
			if path == "" {
				tasksWithSource = append(tasksWithSource, taskName)
			} else {
				tasksWithSource = append(tasksWithSource, taskName+"."+path)
			}
		}
	}

	// Verify that all tasks with Source fields are in supportedTypeTasks
	slices.Sort(tasksWithSource)
	sortedSupported := make([]string, len(supportedTypeTasks))
	copy(sortedSupported, supportedTypeTasks)
	slices.Sort(sortedSupported)

	assert.Equal(t, sortedSupported, tasksWithSource,
		"supportedTypeTasks must include all task types with a Source field. "+
			"If this test fails, update supportedTypeTasks in convert_job.go")
}

// findSourceFieldsShallow searches for Source fields in a struct type, going only one level deep.
// Returns a list of paths to Source fields (e.g., "" for direct Source, "file" for sql_task.file).
func findSourceFieldsShallow(t reflect.Type) []string {
	if t.Kind() == reflect.Ptr {
		t = t.Elem()
	}

	if t.Kind() != reflect.Struct {
		return nil
	}

	var paths []string

	for i := range t.NumField() {
		field := t.Field(i)

		// Check if this field is named "Source"
		if field.Name == "Source" {
			paths = append(paths, "")
			continue
		}

		// Only search one level deep in nested structs
		fieldType := field.Type
		if fieldType.Kind() == reflect.Ptr {
			fieldType = fieldType.Elem()
		}

		if fieldType.Kind() == reflect.Struct {
			// Check if the nested struct has a Source field
			if _, hasSource := fieldType.FieldByName("Source"); hasSource {
				fieldName := textutil.CamelToSnakeCase(field.Name)
				paths = append(paths, fieldName)
			}
		}
	}

	return paths
}
