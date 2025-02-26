package bundle_test

import (
	"context"
	"testing"

	"github.com/databricks/cli/integration/internal/acc"
	"github.com/databricks/cli/internal/testutil"
	"github.com/databricks/cli/libs/env"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

func runSparkJarTestCommon(t *testing.T, ctx context.Context, sparkVersion, artifactPath string) {
	nodeTypeId := testutil.GetCloud(t).NodeTypeID()
	tmpDir := t.TempDir()
	instancePoolId := env.Get(ctx, "TEST_INSTANCE_POOL_ID")
	bundleRoot := initTestTemplateWithBundleRoot(t, ctx, "spark_jar_task", map[string]any{
		"node_type_id":     nodeTypeId,
		"unique_id":        uuid.New().String(),
		"spark_version":    sparkVersion,
		"root":             tmpDir,
		"artifact_path":    artifactPath,
		"instance_pool_id": instancePoolId,
	}, tmpDir)

	deployBundle(t, ctx, bundleRoot)

	t.Cleanup(func() {
		destroyBundle(t, ctx, bundleRoot)
	})

	if testing.Short() {
		t.Log("Skip the job run in short mode")
		return
	}

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
	// Failure on earlier DBR versions:
	//
	//   JAR installation from Volumes is supported on UC Clusters with DBR >= 13.3.
	//   Denied library is Jar(/Volumes/main/test-schema-ldgaklhcahlg/my-volume/.internal/PrintArgs.jar)
	//

	testCases := []struct {
		name                string // Test name
		runtimeVersion      string // The Spark runtime version to test
		requiredJavaVersion string // Java version that can compile jar to pass this test
	}{
		{
			name:                "Databricks Runtime 13.3 LTS",
			runtimeVersion:      "13.3.x-scala2.12", // 13.3 LTS (includes Apache Spark 3.4.1, Scala 2.12)
			requiredJavaVersion: "1.8.0",            // Only JDK 8 is supported
		},
		{
			name:                "Databricks Runtime 14.3 LTS",
			runtimeVersion:      "14.3.x-scala2.12", // 14.3 LTS (includes Apache Spark 3.5.0, Scala 2.12)
			requiredJavaVersion: "1.8.0",            // Only JDK 8 is supported
		},
		{
			name:                "Databricks Runtime 15.4 LTS",
			runtimeVersion:      "15.4.x-scala2.12", // 15.4 LTS (includes Apache Spark 3.5.0, Scala 2.12)
			requiredJavaVersion: "1.8.0",            // Only JDK 8 is supported
		},
		{
			name:                "Databricks Runtime 16.2",
			runtimeVersion:      "16.2.x-scala2.12", // 16.2 (includes Apache Spark 3.5.2, Scala 2.12)
			requiredJavaVersion: "11.0",             // Can run jars compiled by Java 11
		},
	}

	type testRun struct {
		canRun      bool
		javaVersion string
	}

	testRuns := make(map[string]testRun)

	// Identify which tests can run in this environment
	for _, tc := range testCases {
		testRun := testRun{canRun: false}
		if testutil.HasJDK(t, context.Background(), tc.requiredJavaVersion) {
			testRun.canRun = true
			testRun.javaVersion = tc.requiredJavaVersion
		}
		testRuns[tc.name] = testRun
	}

	atLeastOneCanRun := false
	for _, result := range testRuns {
		if result.canRun {
			atLeastOneCanRun = true
			break
		}
	}

	if !atLeastOneCanRun {
		t.Fatal("At least one test is required to pass. All tests were skipped because no compatible Java version was found.")
	}

	// Run the tests that can run
	for _, tc := range testCases {
		tc := tc // Capture range variable for goroutine
		result := testRuns[tc.name]

		t.Run(tc.name, func(t *testing.T) {
			if !result.canRun {
				t.Skipf("Skipping %s: requires Java version %v", tc.name, tc.requiredJavaVersion)
				return
			}

			t.Parallel()
			runSparkJarTestFromVolume(t, tc.runtimeVersion)
		})
	}
}

func TestSparkJarTaskDeployAndRunOnWorkspace(t *testing.T) {
	// Failure on earlier DBR versions:
	//
	//   Library from /Workspace is not allowed on this cluster.
	//   Please switch to using DBR 14.1+ No Isolation Shared or DBR 13.1+ Shared cluster or 13.2+ Assigned cluster to use /Workspace libraries.
	//

	testCases := []struct {
		name                string // Test name
		runtimeVersion      string // The Spark runtime version to test
		requiredJavaVersion string // Java version that can compile jar to pass this test
	}{
		{
			name:                "Databricks Runtime 14.3 LTS",
			runtimeVersion:      "14.3.x-scala2.12", // 14.3 LTS (includes Apache Spark 3.5.0, Scala 2.12)
			requiredJavaVersion: "1.8.0",            // Only JDK 8 is supported
		},
		{
			name:                "Databricks Runtime 15.4 LTS",
			runtimeVersion:      "15.4.x-scala2.12", // 15.4 LTS (includes Apache Spark 3.5.0, Scala 2.12)
			requiredJavaVersion: "1.8.0",            // Only JDK 8 is supported
		},
		{
			name:                "Databricks Runtime 16.2",
			runtimeVersion:      "16.2.x-scala2.12", // 16.2 (includes Apache Spark 3.5.2, Scala 2.12)
			requiredJavaVersion: "11.0",             // Can run jars compiled by Java 11
		},
	}

	type testRun struct {
		canRun      bool
		javaVersion string
	}

	testRuns := make(map[string]testRun)

	// Identify which tests can run in this environment
	for _, tc := range testCases {
		testRun := testRun{canRun: false}
		if testutil.HasJDK(t, context.Background(), tc.requiredJavaVersion) {
			testRun.canRun = true
			testRun.javaVersion = tc.requiredJavaVersion
		}
		testRuns[tc.name] = testRun
	}

	atLeastOneCanRun := false
	for _, result := range testRuns {
		if result.canRun {
			atLeastOneCanRun = true
			break
		}
	}

	if !atLeastOneCanRun {
		t.Fatal("At least one test is required to pass. All tests were skipped because no compatible Java version was found.")
	}

	// Run the tests that can run
	for _, tc := range testCases {
		tc := tc // Capture range variable for goroutine
		result := testRuns[tc.name]

		t.Run(tc.name, func(t *testing.T) {
			if !result.canRun {
				t.Skipf("Skipping %s: requires Java version %v", tc.name, tc.requiredJavaVersion)
				return
			}

			t.Parallel()
			runSparkJarTestFromWorkspace(t, tc.runtimeVersion)
		})
	}
}
