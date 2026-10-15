package trampoline

import (
	"testing"

	"github.com/databricks/cli/libs/sdktypes/jobs/v2"

	"github.com/databricks/cli/bundle"
	"github.com/databricks/cli/bundle/config"
	"github.com/databricks/cli/bundle/config/resources"
	"github.com/databricks/cli/bundle/internal/bundletest"
	"github.com/databricks/cli/libs/testserver"
	"github.com/stretchr/testify/require"
)

func TestIncompatibleWheelTasksWithNewCluster(t *testing.T) {
	b := &bundle.Bundle{
		Config: config.Root{
			Resources: config.Resources{
				Jobs: map[string]*resources.Job{
					"job1": {
						JobSettings: jobs.JobSettings{
							Tasks: []jobs.TaskSettings{
								{
									TaskKey:         new("key1"),
									PythonWheelTask: &jobs.PythonWheelTask{},
									NewCluster: &jobs.ClusterSpec_NewCluster{
										SparkVersion: new("12.2.x-scala2.12"),
									},
									Libraries: []jobs.Library{
										{Whl: new("./dist/test.whl")},
									},
								},
								{
									TaskKey:         new("key2"),
									PythonWheelTask: &jobs.PythonWheelTask{},
									NewCluster: &jobs.ClusterSpec_NewCluster{
										SparkVersion: new("13.1.x-scala2.12"),
									},
									Libraries: []jobs.Library{
										{Whl: new("./dist/test.whl")},
									},
								},
							},
						},
					},
				},
			},
		},
	}

	diags := hasIncompatibleWheelTasks(t.Context(), b)
	require.NotEmpty(t, diags)
}

func TestIncompatibleWheelTasksWithJobClusterKey(t *testing.T) {
	b := &bundle.Bundle{
		Config: config.Root{
			Resources: config.Resources{
				Jobs: map[string]*resources.Job{
					"job1": {
						JobSettings: jobs.JobSettings{
							JobClusters: []jobs.JobCluster{
								{
									JobClusterKey: new("cluster1"),
									NewCluster: &jobs.ClusterSpec_NewCluster{
										SparkVersion: new("12.2.x-scala2.12"),
									},
								},
								{
									JobClusterKey: new("cluster2"),
									NewCluster: &jobs.ClusterSpec_NewCluster{
										SparkVersion: new("13.1.x-scala2.12"),
									},
								},
							},
							Tasks: []jobs.TaskSettings{
								{
									TaskKey:         new("key1"),
									PythonWheelTask: &jobs.PythonWheelTask{},
									JobClusterKey:   new("cluster1"),
									Libraries: []jobs.Library{
										{Whl: new("./dist/test.whl")},
									},
								},
								{
									TaskKey:         new("key2"),
									PythonWheelTask: &jobs.PythonWheelTask{},
									JobClusterKey:   new("cluster2"),
									Libraries: []jobs.Library{
										{Whl: new("./dist/test.whl")},
									},
								},
							},
						},
					},
				},
			},
		},
	}

	diags := hasIncompatibleWheelTasks(t.Context(), b)
	require.NotEmpty(t, diags)

	diags = bundle.Apply(t.Context(), b, WrapperWarning())
	require.ErrorContains(t, diags.Error(), "uses incompatible DBR version 12.2.x-scala2.12")
}

func TestIncompatibleWheelTasksWithExistingClusterId(t *testing.T) {
	b := &bundle.Bundle{
		Config: config.Root{
			Resources: config.Resources{
				Jobs: map[string]*resources.Job{
					"job1": {
						JobSettings: jobs.JobSettings{
							Tasks: []jobs.TaskSettings{
								{
									TaskKey:           new("key1"),
									PythonWheelTask:   &jobs.PythonWheelTask{},
									ExistingClusterId: new("test-key-1"),
									Libraries: []jobs.Library{
										{Whl: new("./dist/test.whl")},
									},
								},
								{
									TaskKey:           new("key2"),
									PythonWheelTask:   &jobs.PythonWheelTask{},
									ExistingClusterId: new("test-key-2"),
									Libraries: []jobs.Library{
										{Whl: new("./dist/test.whl")},
									},
								},
							},
						},
					},
				},
			},
		},
	}

	setClusters(t, b, map[string]string{
		"test-key-1": "12.2.x-scala2.12",
		"test-key-2": "12.2.x-scala2.12",
	})

	diags := hasIncompatibleWheelTasks(t.Context(), b)
	require.NotEmpty(t, diags)
	require.ErrorContains(t, diags.Error(), "uses cluster with incompatible DBR version 12.2.x-scala2.12")
}

func TestNoIncompatibleWheelTasks(t *testing.T) {
	b := &bundle.Bundle{
		Config: config.Root{
			Resources: config.Resources{
				Jobs: map[string]*resources.Job{
					"job1": {
						JobSettings: jobs.JobSettings{
							JobClusters: []jobs.JobCluster{
								{
									JobClusterKey: new("cluster1"),
									NewCluster: &jobs.ClusterSpec_NewCluster{
										SparkVersion: new("12.2.x-scala2.12"),
									},
								},
								{
									JobClusterKey: new("cluster2"),
									NewCluster: &jobs.ClusterSpec_NewCluster{
										SparkVersion: new("13.1.x-scala2.12"),
									},
								},
							},
							Tasks: []jobs.TaskSettings{
								{
									TaskKey:         new("key1"),
									PythonWheelTask: &jobs.PythonWheelTask{},
									NewCluster: &jobs.ClusterSpec_NewCluster{
										SparkVersion: new("12.2.x-scala2.12"),
									},
									Libraries: []jobs.Library{
										{Whl: new("/Workspace/Users/me@me.com/dist/test.whl")},
									},
								},
								{
									TaskKey:         new("key2"),
									PythonWheelTask: &jobs.PythonWheelTask{},
									NewCluster: &jobs.ClusterSpec_NewCluster{
										SparkVersion: new("13.3.x-scala2.12"),
									},
									Libraries: []jobs.Library{
										{Whl: new("./dist/test.whl")},
									},
								},
								{
									TaskKey:         new("key3"),
									PythonWheelTask: &jobs.PythonWheelTask{},
									NewCluster: &jobs.ClusterSpec_NewCluster{
										SparkVersion: new("12.2.x-scala2.12"),
									},
									Libraries: []jobs.Library{
										{Whl: new("dbfs:/dist/test.whl")},
									},
								},
								{
									TaskKey:         new("key4"),
									PythonWheelTask: &jobs.PythonWheelTask{},
									JobClusterKey:   new("cluster1"),
									Libraries: []jobs.Library{
										{Whl: new("/Workspace/Users/me@me.com/dist/test.whl")},
									},
								},
								{
									TaskKey:         new("key5"),
									PythonWheelTask: &jobs.PythonWheelTask{},
									JobClusterKey:   new("cluster2"),
									Libraries: []jobs.Library{
										{Whl: new("./dist/test.whl")},
									},
								},
								{
									TaskKey:           new("key6"),
									PythonWheelTask:   &jobs.PythonWheelTask{},
									ExistingClusterId: new("test-key-2"),
									Libraries: []jobs.Library{
										{Whl: new("./dist/test.whl")},
									},
								},
								{
									TaskKey:           new("key7"),
									PythonWheelTask:   &jobs.PythonWheelTask{},
									ExistingClusterId: new("test-key-2"),
									Libraries: []jobs.Library{
										{Whl: new("signol_lib-0.4.4-20240822+prod-py3-none-any.whl")},
										{Pypi: &jobs.PythonPyPiLibrary{
											Package: new("requests==2.25.1"),
										}},
									},
								},
							},
						},
					},
				},
			},
		},
	}

	setClusters(t, b, map[string]string{
		"test-key-2": "13.2.x-scala2.12",
	})

	diags := hasIncompatibleWheelTasks(t.Context(), b)
	require.Empty(t, diags)
}

func TestTasksWithPyPiPackageAreCompatible(t *testing.T) {
	b := &bundle.Bundle{
		Config: config.Root{
			Resources: config.Resources{
				Jobs: map[string]*resources.Job{
					"job1": {
						JobSettings: jobs.JobSettings{
							JobClusters: []jobs.JobCluster{
								{
									JobClusterKey: new("cluster1"),
									NewCluster: &jobs.ClusterSpec_NewCluster{
										SparkVersion: new("12.2.x-scala2.12"),
									},
								},
							},
							Tasks: []jobs.TaskSettings{
								{
									TaskKey:           new("key1"),
									PythonWheelTask:   &jobs.PythonWheelTask{},
									ExistingClusterId: new("test-key-2"),
									Libraries: []jobs.Library{
										{Pypi: &jobs.PythonPyPiLibrary{
											Package: new("requests==2.25.1"),
										}},
									},
								},
							},
						},
					},
				},
			},
		},
	}

	setClusters(t, b, nil)

	diags := hasIncompatibleWheelTasks(t.Context(), b)
	require.Empty(t, diags)
}

func TestNoWarningWhenPythonWheelWrapperIsOn(t *testing.T) {
	b := &bundle.Bundle{
		Config: config.Root{
			Experimental: &config.Experimental{
				PythonWheelWrapper: true,
			},
			Resources: config.Resources{
				Jobs: map[string]*resources.Job{
					"job1": {
						JobSettings: jobs.JobSettings{
							Tasks: []jobs.TaskSettings{
								{
									TaskKey:         new("key1"),
									PythonWheelTask: &jobs.PythonWheelTask{},
									NewCluster: &jobs.ClusterSpec_NewCluster{
										SparkVersion: new("12.2.x-scala2.12"),
									},
									Libraries: []jobs.Library{
										{Whl: new("./dist/test.whl")},
									},
								},
								{
									TaskKey:         new("key2"),
									PythonWheelTask: &jobs.PythonWheelTask{},
									NewCluster: &jobs.ClusterSpec_NewCluster{
										SparkVersion: new("13.1.x-scala2.12"),
									},
									Libraries: []jobs.Library{
										{Whl: new("./dist/test.whl")},
									},
								},
							},
						},
					},
				},
			},
		},
	}

	diags := bundle.Apply(t.Context(), b, WrapperWarning())
	require.NoError(t, diags.Error())
}

func TestSparkVersionLowerThanExpected(t *testing.T) {
	testCases := map[string]bool{
		"13.1.x-scala2.12":                false,
		"13.2.x-scala2.12":                false,
		"13.3.x-scala2.12":                false,
		"14.0.x-scala2.12":                false,
		"14.1.x-scala2.12":                false,
		"13.x-snapshot-scala-2.12":        false,
		"13.x-rc-scala-2.12":              false,
		"client.1.10-scala2.12":           false,
		"latest-stable-gpu-scala2.11":     false,
		"1.":                              false,
		"10.4.x-aarch64-photon-scala2.12": true,
		"10.4.x-scala2.12":                true,
		"13.0.x-scala2.12":                true,
		"5.0.x-rc-gpu-ml-scala2.11":       true,
	}

	for k, v := range testCases {
		result := lowerThanExpectedVersion(k)
		require.Equal(t, v, result, k)
	}
}

// setClusters serves clusters/get for the given clusters (id -> spark version).
func setClusters(t *testing.T, b *bundle.Bundle, sparkVersions map[string]string) {
	w, server := bundletest.NewClients(t)
	b.SetWorkpaceClient(w)
	server.Handle("GET", "/api/2.1/clusters/get", func(req testserver.Request) any {
		id := req.URL.Query().Get("cluster_id")
		version, ok := sparkVersions[id]
		if !ok {
			t.Errorf("unexpected cluster id %q", id)
		}
		return map[string]string{"cluster_id": id, "spark_version": version}
	})
}
