package bundle_test

import (
	"context"
	"testing"

	"github.com/databricks/cli/internal/acc"
	"github.com/databricks/cli/internal/testutil"
	"github.com/databricks/cli/libs/env"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

func runSparkJarTestCommon(t *testing.T, ctx context.Context, sparkVersion, artifactPath string) {
	nodeTypeId := testutil.GetCloud(t).NodeTypeID()
	tmpDir := t.TempDir()
	instancePoolId := env.Get(ctx, "TEST_INSTANCE_POOL_ID")
	bundleRoot, err := initTestTemplateWithBundleRoot(t, ctx, "spark_jar_task", map[string]any{
		"node_type_id":     nodeTypeId,
		"unique_id":        uuid.New().String(),
		"spark_version":    sparkVersion,
		"root":             tmpDir,
		"artifact_path":    artifactPath,
		"instance_pool_id": instancePoolId,
	}, tmpDir)
	require.NoError(t, err)

	err = deployBundle(t, ctx, bundleRoot)
	require.NoError(t, err)

	t.Cleanup(func() {
		err := destroyBundle(t, ctx, bundleRoot)
		require.NoError(t, err)
	})

	out, err := runResource(t, ctx, bundleRoot, "jar_job")
	require.NoError(t, err)
	require.Contains(t, out, "Hello from Jar!")
}

func runSparkJarTestFromVolume(t *testing.T, sparkVersion string) {
	ctx, wt := acc.UcWorkspaceTest(t)
	volumePath := acc.TemporaryVolume(wt)
	ctx = env.Set(ctx, "DATABRICKS_BUNDLE_TARGET", "volume")
	runSparkJarTestCommon(t, ctx, sparkVersion, volumePath)
}

func runSparkJarTestFromWorkspace(t *testing.T, sparkVersion string) {
	ctx, _ := acc.WorkspaceTest(t)
	ctx = env.Set(ctx, "DATABRICKS_BUNDLE_TARGET", "workspace")
	runSparkJarTestCommon(t, ctx, sparkVersion, "n/a")
}

func TestSparkJarTaskDeployAndRunOnVolumes(t *testing.T) {
	testutil.RequireJDK(t, context.Background(), "1.8.0")

	// Failure on earlier DBR versions:
	//
	//   JAR installation from Volumes is supported on UC Clusters with DBR >= 13.3.
	//   Denied library is Jar(/Volumes/main/test-schema-ldgaklhcahlg/my-volume/.internal/PrintArgs.jar)
	//

	versions := []string{
		"13.3.x-scala2.12", // 13.3 LTS (includes Apache Spark 3.4.1, Scala 2.12)
		"14.3.x-scala2.12", // 14.3 LTS (includes Apache Spark 3.5.0, Scala 2.12)
		"15.4.x-scala2.12", // 15.4 LTS Beta (includes Apache Spark 3.5.0, Scala 2.12)
	}

	for _, version := range versions {
		t.Run(version, func(t *testing.T) {
			t.Parallel()
			runSparkJarTestFromVolume(t, version)
		})
	}
}

func TestSparkJarTaskDeployAndRunOnWorkspace(t *testing.T) {
	testutil.RequireJDK(t, context.Background(), "1.8.0")

	// Failure on earlier DBR versions:
	//
	//   Library from /Workspace is not allowed on this cluster.
	//   Please switch to using DBR 14.1+ No Isolation Shared or DBR 13.1+ Shared cluster or 13.2+ Assigned cluster to use /Workspace libraries.
	//

	versions := []string{
		"14.3.x-scala2.12", // 14.3 LTS (includes Apache Spark 3.5.0, Scala 2.12)
		"15.4.x-scala2.12", // 15.4 LTS Beta (includes Apache Spark 3.5.0, Scala 2.12)
	}

	for _, version := range versions {
		t.Run(version, func(t *testing.T) {
			t.Parallel()
			runSparkJarTestFromWorkspace(t, version)
		})
	}
}
