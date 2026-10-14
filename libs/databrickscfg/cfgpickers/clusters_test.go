package cfgpickers

import (
	"testing"

	"github.com/databricks/cli/libs/cmdio"
	"github.com/databricks/cli/libs/httpfixtures"
	"github.com/databricks/cli/libs/sdkclients"
	compute "github.com/databricks/sdk-go/clusters/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestIsCompatible(t *testing.T) {
	require.True(t, IsCompatibleWithUC(compute.ClusterInfo{
		SparkVersion:     new("13.2.x-aarch64-scala2.12"),
		DataSecurityMode: compute.DataSecurityMode_UserIsolation,
	}, "13.0"))
	require.False(t, IsCompatibleWithUC(compute.ClusterInfo{
		SparkVersion:     new("13.2.x-aarch64-scala2.12"),
		DataSecurityMode: compute.DataSecurityMode_None,
	}, "13.0"))
	require.False(t, IsCompatibleWithUC(compute.ClusterInfo{
		SparkVersion:     new("9.1.x-photon-scala2.12"),
		DataSecurityMode: compute.DataSecurityMode_None,
	}, "13.0"))
	require.False(t, IsCompatibleWithUC(compute.ClusterInfo{
		SparkVersion:     new("9.1.x-photon-scala2.12"),
		DataSecurityMode: compute.DataSecurityMode_None,
	}, "10.0"))
	require.False(t, IsCompatibleWithUC(compute.ClusterInfo{
		SparkVersion:     new("custom-9.1.x-photon-scala2.12"),
		DataSecurityMode: compute.DataSecurityMode_None,
	}, "14.0"))
}

func TestGetRuntimeVersion(t *testing.T) {
	tests := []struct {
		sparkVersion string
		want         string
		wantOk       bool
	}{
		{"13.3.x-scala2.12", "13.3", true},
		// Newer clusters report the runtime without a minor version, e.g. 19.x.
		{"19.x-scala2.13", "19", true},
		{"14.x-snapshot-cpu-ml-scala2.12", "14.999", true},
		{"custom-9.1.x-photon-scala2.12", "", false},
	}
	for _, tc := range tests {
		got, ok := GetRuntimeVersion(compute.ClusterInfo{SparkVersion: new(tc.sparkVersion)})
		assert.Equal(t, tc.wantOk, ok, tc.sparkVersion)
		assert.Equal(t, tc.want, got, tc.sparkVersion)
	}
}

func TestIsCompatibleWithMajorOnlyVersion(t *testing.T) {
	require.True(t, IsCompatibleWithUC(compute.ClusterInfo{
		SparkVersion:     new("19.x-scala2.13"),
		DataSecurityMode: compute.DataSecurityMode_UserIsolation,
	}, "13.0"))
}

func TestIsCompatibleWithSnapshots(t *testing.T) {
	require.True(t, IsCompatibleWithUC(compute.ClusterInfo{
		SparkVersion:     new("14.x-snapshot-cpu-ml-scala2.12"),
		DataSecurityMode: compute.DataSecurityMode_UserIsolation,
	}, "14.0"))
}

func TestWithoutSystemClusters(t *testing.T) {
	fn := WithoutSystemClusters()

	// Sources to exclude.
	for _, v := range []string{
		"JOB",
		"PIPELINE",
		"SOME_UNKNOWN_VALUE",
	} {
		assert.False(t, fn(&Cluster{Source: v}, nil))
	}

	// Sources to include.
	for _, v := range []string{
		"UI",
		"API",
	} {
		assert.True(t, fn(&Cluster{Source: v}, nil))
	}
}

const clustersListResource = "/api/2.1/clusters/list?filter_by.cluster_sources=API&filter_by.cluster_sources=UI&page_size=100"

func clusterFixtures(clustersJSON string) httpfixtures.Fixtures {
	return httpfixtures.Fixtures{
		{Method: "GET", Resource: clustersListResource, Response: `{"clusters":` + clustersJSON + `}`},
		{Method: "GET", Resource: "/api/2.0/preview/scim/v2/Me", Response: `{"userName":"serge"}`},
		{Method: "GET", Resource: "/api/2.1/clusters/spark-versions", Response: `{"versions":[{"key":"14.5.x-whatever","name":"14.5 (Awesome)"}]}`},
	}
}

func TestFirstCompatibleCluster(t *testing.T) {
	cfg, _ := clusterFixtures(`[
		{"cluster_id":"abc-id","cluster_name":"first shared","cluster_source":"API","data_security_mode":"USER_ISOLATION","spark_version":"12.2.x-whatever","state":"RUNNING"},
		{"cluster_id":"bcd-id","cluster_name":"second personal","cluster_source":"UI","data_security_mode":"SINGLE_USER","spark_version":"14.5.x-whatever","state":"RUNNING","single_user_name":"serge"}
	]`).Config(t)
	w, err := sdkclients.New(t.Context(), cfg)
	require.NoError(t, err)

	ctx := cmdio.MockDiscard(t.Context())

	clusterID, err := AskForCluster(ctx, w, WithDatabricksConnect("13.1"))
	require.NoError(t, err)
	require.Equal(t, "bcd-id", clusterID)
}

func TestNoCompatibleClusters(t *testing.T) {
	cfg, _ := clusterFixtures(`[
		{"cluster_id":"abc-id","cluster_name":"first shared","data_security_mode":"USER_ISOLATION","spark_version":"12.2.x-whatever","state":"RUNNING"}
	]`).Config(t)
	w, err := sdkclients.New(t.Context(), cfg)
	require.NoError(t, err)

	ctx := cmdio.MockDiscard(t.Context())
	_, err = AskForCluster(ctx, w, WithDatabricksConnect("13.1"))
	require.Equal(t, ErrNoCompatibleClusters, err)
}
