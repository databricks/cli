package python

import (
	"context"
	"testing"

	"github.com/databricks/cli/bundle"
	"github.com/databricks/cli/bundle/config"
	"github.com/databricks/cli/bundle/config/resources"
	"github.com/databricks/databricks-sdk-go/experimental/mocks"
	"github.com/databricks/databricks-sdk-go/service/compute"
	"github.com/databricks/databricks-sdk-go/service/jobs"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func TestIncompatibleWheelTasksWithNewCluster(t *testing.T) {
	b := &bundle.Bundle{
		Config: config.Root{
			Resources: config.Resources{
				Jobs: map[string]*resources.Job{
					"job1": {
						JobSettings: &jobs.JobSettings{
							Tasks: []jobs.Task{
								{
									TaskKey:         "key1",
									PythonWheelTask: &jobs.PythonWheelTask{},
									NewCluster: &compute.ClusterSpec{
										SparkVersion: "12.2.x-scala2.12",
									},
									Libraries: []compute.Library{
										{Whl: "./dist/test.whl"},
									},
								},
								{
									TaskKey:         "key2",
									PythonWheelTask: &jobs.PythonWheelTask{},
									NewCluster: &compute.ClusterSpec{
										SparkVersion: "13.1.x-scala2.12",
									},
									Libraries: []compute.Library{
										{Whl: "./dist/test.whl"},
									},
								},
							},
						},
					},
				},
			},
		},
	}

	require.True(t, hasIncompatibleWheelTasks(context.Background(), b))
}

func TestIncompatibleWheelTasksWithJobClusterKey(t *testing.T) {
	b := &bundle.Bundle{
		Config: config.Root{
			Resources: config.Resources{
				Jobs: map[string]*resources.Job{
					"job1": {
						JobSettings: &jobs.JobSettings{
							JobClusters: []jobs.JobCluster{
								{
									JobClusterKey: "cluster1",
									NewCluster: &compute.ClusterSpec{
										SparkVersion: "12.2.x-scala2.12",
									},
								},
								{
									JobClusterKey: "cluster2",
									NewCluster: &compute.ClusterSpec{
										SparkVersion: "13.1.x-scala2.12",
									},
								},
							},
							Tasks: []jobs.Task{
								{
									TaskKey:         "key1",
									PythonWheelTask: &jobs.PythonWheelTask{},
									JobClusterKey:   "cluster1",
									Libraries: []compute.Library{
										{Whl: "./dist/test.whl"},
									},
								},
								{
									TaskKey:         "key2",
									PythonWheelTask: &jobs.PythonWheelTask{},
									JobClusterKey:   "cluster2",
									Libraries: []compute.Library{
										{Whl: "./dist/test.whl"},
									},
								},
							},
						},
					},
				},
			},
		},
	}

	require.True(t, hasIncompatibleWheelTasks(context.Background(), b))

	err := bundle.Apply(context.Background(), b, WrapperWarning())
	require.ErrorContains(t, err, "python wheel tasks with local libraries require compute with DBR 13.1+.")
}

func TestIncompatibleWheelTasksWithExistingClusterId(t *testing.T) {
	b := &bundle.Bundle{
		Config: config.Root{
			Resources: config.Resources{
				Jobs: map[string]*resources.Job{
					"job1": {
						JobSettings: &jobs.JobSettings{
							Tasks: []jobs.Task{
								{
									TaskKey:           "key1",
									PythonWheelTask:   &jobs.PythonWheelTask{},
									ExistingClusterId: "test-key-1",
									Libraries: []compute.Library{
										{Whl: "./dist/test.whl"},
									},
								},
								{
									TaskKey:           "key2",
									PythonWheelTask:   &jobs.PythonWheelTask{},
									ExistingClusterId: "test-key-2",
									Libraries: []compute.Library{
										{Whl: "./dist/test.whl"},
									},
								},
							},
						},
					},
				},
			},
		},
	}

	m := mocks.NewMockWorkspaceClient(t)
	b.SetWorkpaceClient(m.WorkspaceClient)
	clustersApi := m.GetMockClustersAPI()
	clustersApi.EXPECT().GetByClusterId(mock.Anything, "test-key-1").Return(&compute.ClusterDetails{
		SparkVersion: "12.2.x-scala2.12",
	}, nil)

	require.True(t, hasIncompatibleWheelTasks(context.Background(), b))
}

func TestNoIncompatibleWheelTasks(t *testing.T) {
	b := &bundle.Bundle{
		Config: config.Root{
			Resources: config.Resources{
				Jobs: map[string]*resources.Job{
					"job1": {
						JobSettings: &jobs.JobSettings{
							JobClusters: []jobs.JobCluster{
								{
									JobClusterKey: "cluster1",
									NewCluster: &compute.ClusterSpec{
										SparkVersion: "12.2.x-scala2.12",
									},
								},
								{
									JobClusterKey: "cluster2",
									NewCluster: &compute.ClusterSpec{
										SparkVersion: "13.1.x-scala2.12",
									},
								},
							},
							Tasks: []jobs.Task{
								{
									TaskKey:         "key1",
									PythonWheelTask: &jobs.PythonWheelTask{},
									NewCluster: &compute.ClusterSpec{
										SparkVersion: "12.2.x-scala2.12",
									},
									Libraries: []compute.Library{
										{Whl: "/Workspace/Users/me@me.com/dist/test.whl"},
									},
								},
								{
									TaskKey:         "key2",
									PythonWheelTask: &jobs.PythonWheelTask{},
									NewCluster: &compute.ClusterSpec{
										SparkVersion: "13.3.x-scala2.12",
									},
									Libraries: []compute.Library{
										{Whl: "./dist/test.whl"},
									},
								},
								{
									TaskKey:         "key3",
									PythonWheelTask: &jobs.PythonWheelTask{},
									NewCluster: &compute.ClusterSpec{
										SparkVersion: "12.2.x-scala2.12",
									},
									Libraries: []compute.Library{
										{Whl: "dbfs:/dist/test.whl"},
									},
								},
								{
									TaskKey:         "key4",
									PythonWheelTask: &jobs.PythonWheelTask{},
									JobClusterKey:   "cluster1",
									Libraries: []compute.Library{
										{Whl: "/Workspace/Users/me@me.com/dist/test.whl"},
									},
								},
								{
									TaskKey:         "key5",
									PythonWheelTask: &jobs.PythonWheelTask{},
									JobClusterKey:   "cluster2",
									Libraries: []compute.Library{
										{Whl: "./dist/test.whl"},
									},
								},
								{
									TaskKey:           "key6",
									PythonWheelTask:   &jobs.PythonWheelTask{},
									ExistingClusterId: "test-key-2",
									Libraries: []compute.Library{
										{Whl: "./dist/test.whl"},
									},
								},
							},
						},
					},
				},
			},
		},
	}

	m := mocks.NewMockWorkspaceClient(t)
	b.SetWorkpaceClient(m.WorkspaceClient)
	clustersApi := m.GetMockClustersAPI()
	clustersApi.EXPECT().GetByClusterId(mock.Anything, "test-key-2").Return(&compute.ClusterDetails{
		SparkVersion: "13.2.x-scala2.12",
	}, nil)

	require.False(t, hasIncompatibleWheelTasks(context.Background(), b))
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
						JobSettings: &jobs.JobSettings{
							Tasks: []jobs.Task{
								{
									TaskKey:         "key1",
									PythonWheelTask: &jobs.PythonWheelTask{},
									NewCluster: &compute.ClusterSpec{
										SparkVersion: "12.2.x-scala2.12",
									},
									Libraries: []compute.Library{
										{Whl: "./dist/test.whl"},
									},
								},
								{
									TaskKey:         "key2",
									PythonWheelTask: &jobs.PythonWheelTask{},
									NewCluster: &compute.ClusterSpec{
										SparkVersion: "13.1.x-scala2.12",
									},
									Libraries: []compute.Library{
										{Whl: "./dist/test.whl"},
									},
								},
							},
						},
					},
				},
			},
		},
	}

	diags := bundle.Apply(context.Background(), b, WrapperWarning())
	require.Empty(t, diags)

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
		"10.4.x-aarch64-photon-scala2.12": true,
		"10.4.x-scala2.12":                true,
		"13.0.x-scala2.12":                true,
		"5.0.x-rc-gpu-ml-scala2.11":       true,
	}

	for k, v := range testCases {
		result := lowerThanExpectedVersion(context.Background(), k)
		require.Equal(t, v, result, k)
	}
}
