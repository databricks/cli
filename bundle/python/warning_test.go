package python

import (
	"context"
	"testing"

	"github.com/databricks/cli/bundle"
	"github.com/databricks/cli/bundle/config"
	"github.com/databricks/cli/bundle/config/resources"
	"github.com/databricks/databricks-sdk-go/service/compute"
	"github.com/databricks/databricks-sdk-go/service/jobs"
	"github.com/stretchr/testify/require"
)

type MockClusterService struct{}

// ChangeOwner implements compute.ClustersService.
func (MockClusterService) ChangeOwner(ctx context.Context, request compute.ChangeClusterOwner) error {
	panic("unimplemented")
}

// Create implements compute.ClustersService.
func (MockClusterService) Create(ctx context.Context, request compute.CreateCluster) (*compute.CreateClusterResponse, error) {
	panic("unimplemented")
}

// Delete implements compute.ClustersService.
func (MockClusterService) Delete(ctx context.Context, request compute.DeleteCluster) error {
	panic("unimplemented")
}

// Edit implements compute.ClustersService.
func (MockClusterService) Edit(ctx context.Context, request compute.EditCluster) error {
	panic("unimplemented")
}

// Events implements compute.ClustersService.
func (MockClusterService) Events(ctx context.Context, request compute.GetEvents) (*compute.GetEventsResponse, error) {
	panic("unimplemented")
}

// Get implements compute.ClustersService.
func (MockClusterService) Get(ctx context.Context, request compute.GetClusterRequest) (*compute.ClusterDetails, error) {
	clusterDetails := map[string]*compute.ClusterDetails{
		"test-key-1": {
			SparkVersion: "12.2.x-scala2.12",
		},
		"test-key-2": {
			SparkVersion: "13.2.x-scala2.12",
		},
	}

	return clusterDetails[request.ClusterId], nil
}

// GetPermissionLevels implements compute.ClustersService.
func (MockClusterService) GetPermissionLevels(ctx context.Context, request compute.GetClusterPermissionLevelsRequest) (*compute.GetClusterPermissionLevelsResponse, error) {
	panic("unimplemented")
}

// GetPermissions implements compute.ClustersService.
func (MockClusterService) GetPermissions(ctx context.Context, request compute.GetClusterPermissionsRequest) (*compute.ClusterPermissions, error) {
	panic("unimplemented")
}

// List implements compute.ClustersService.
func (MockClusterService) List(ctx context.Context, request compute.ListClustersRequest) (*compute.ListClustersResponse, error) {
	panic("unimplemented")
}

// ListNodeTypes implements compute.ClustersService.
func (MockClusterService) ListNodeTypes(ctx context.Context) (*compute.ListNodeTypesResponse, error) {
	panic("unimplemented")
}

// ListZones implements compute.ClustersService.
func (MockClusterService) ListZones(ctx context.Context) (*compute.ListAvailableZonesResponse, error) {
	panic("unimplemented")
}

// PermanentDelete implements compute.ClustersService.
func (MockClusterService) PermanentDelete(ctx context.Context, request compute.PermanentDeleteCluster) error {
	panic("unimplemented")
}

// Pin implements compute.ClustersService.
func (MockClusterService) Pin(ctx context.Context, request compute.PinCluster) error {
	panic("unimplemented")
}

// Resize implements compute.ClustersService.
func (MockClusterService) Resize(ctx context.Context, request compute.ResizeCluster) error {
	panic("unimplemented")
}

// Restart implements compute.ClustersService.
func (MockClusterService) Restart(ctx context.Context, request compute.RestartCluster) error {
	panic("unimplemented")
}

// SetPermissions implements compute.ClustersService.
func (MockClusterService) SetPermissions(ctx context.Context, request compute.ClusterPermissionsRequest) (*compute.ClusterPermissions, error) {
	panic("unimplemented")
}

// SparkVersions implements compute.ClustersService.
func (MockClusterService) SparkVersions(ctx context.Context) (*compute.GetSparkVersionsResponse, error) {
	panic("unimplemented")
}

// Start implements compute.ClustersService.
func (MockClusterService) Start(ctx context.Context, request compute.StartCluster) error {
	panic("unimplemented")
}

// Unpin implements compute.ClustersService.
func (MockClusterService) Unpin(ctx context.Context, request compute.UnpinCluster) error {
	panic("unimplemented")
}

// UpdatePermissions implements compute.ClustersService.
func (MockClusterService) UpdatePermissions(ctx context.Context, request compute.ClusterPermissionsRequest) (*compute.ClusterPermissions, error) {
	panic("unimplemented")
}

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

	b.WorkspaceClient().Clusters.WithImpl(MockClusterService{})

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

	b.WorkspaceClient().Clusters.WithImpl(MockClusterService{})

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

	err := bundle.Apply(context.Background(), b, WrapperWarning())
	require.NoError(t, err)
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
