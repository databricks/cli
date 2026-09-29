package mutator_test

import (
	"context"
	"runtime"
	"testing"

	"github.com/databricks/cli/bundle"
	"github.com/databricks/cli/bundle/config"
	"github.com/databricks/cli/bundle/config/mutator"
	"github.com/databricks/cli/bundle/config/resources"
	"github.com/databricks/cli/bundle/internal/bundletest"
	"github.com/databricks/cli/libs/dbr"
	"github.com/databricks/cli/libs/dyn"
	"github.com/databricks/databricks-sdk-go/service/jobs"
	"github.com/stretchr/testify/require"
)

func TestApplyPresetsSourceLinkedDeployment(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("this test is not applicable on Windows because source-linked mode works only in the Databricks Workspace")
	}

	testContext := t.Context()
	enabled := true
	disabled := false
	workspacePath := "/Workspace/user.name@company.com"

	tests := []struct {
		name            string
		ctx             context.Context
		mutateBundle    func(b *bundle.Bundle)
		initialValue    *bool
		expectedValue   *bool
		expectedWarning string
		expectedError   string
	}{
		{
			name:          "preset enabled, bundle in Workspace, databricks runtime",
			ctx:           dbr.MockRuntime(testContext, dbr.Environment{IsDbr: true, Version: "15.4"}),
			initialValue:  &enabled,
			expectedValue: &enabled,
		},
		{
			name: "preset enabled, bundle not in Workspace, databricks runtime",
			ctx:  dbr.MockRuntime(testContext, dbr.Environment{IsDbr: true, Version: "15.4"}),
			mutateBundle: func(b *bundle.Bundle) {
				b.SyncRootPath = "/Users/user.name@company.com"
			},
			initialValue:    &enabled,
			expectedValue:   &disabled,
			expectedWarning: "source-linked deployment is available only in the Databricks Workspace",
		},
		{
			name:            "preset enabled, bundle in Workspace, not databricks runtime",
			ctx:             dbr.MockRuntime(testContext, dbr.Environment{}),
			initialValue:    &enabled,
			expectedValue:   &disabled,
			expectedWarning: "source-linked deployment is available only in the Databricks Workspace",
		},
		{
			name:          "preset disabled, bundle in Workspace, databricks runtime",
			ctx:           dbr.MockRuntime(testContext, dbr.Environment{IsDbr: true, Version: "15.4"}),
			initialValue:  &disabled,
			expectedValue: &disabled,
		},
		{
			name:          "preset nil, bundle in Workspace, databricks runtime",
			ctx:           dbr.MockRuntime(testContext, dbr.Environment{IsDbr: true, Version: "15.4"}),
			initialValue:  nil,
			expectedValue: nil,
		},
		{
			name: "preset nil, dev mode true, bundle in Workspace, databricks runtime",
			ctx:  dbr.MockRuntime(testContext, dbr.Environment{IsDbr: true, Version: "15.4"}),
			mutateBundle: func(b *bundle.Bundle) {
				b.Config.Bundle.Mode = config.Development
			},
			initialValue:  nil,
			expectedValue: &enabled,
		},
		{
			name: "preset enabled, workspace.file_path is defined by user",
			ctx:  dbr.MockRuntime(testContext, dbr.Environment{IsDbr: true, Version: "15.4"}),
			mutateBundle: func(b *bundle.Bundle) {
				b.Config.Workspace.FilePath = "file_path"
				b.Config.Bundle.Mode = config.Development
			},
			initialValue:    &enabled,
			expectedValue:   &enabled,
			expectedWarning: "workspace.file_path setting will be ignored in source-linked deployment mode",
		},
		{
			name: "preset enabled, apps is defined by user",
			ctx:  dbr.MockRuntime(testContext, dbr.Environment{IsDbr: true, Version: "15.4"}),
			mutateBundle: func(b *bundle.Bundle) {
				b.Config.Resources.Apps = map[string]*resources.App{
					"app": {},
				}
				b.Config.Bundle.Mode = config.Development
			},
			initialValue:  &enabled,
			expectedValue: &enabled,
		},
		{
			name: "preset nil, dev mode true, immutable folder, bundle in Workspace, databricks runtime",
			ctx:  dbr.MockRuntime(testContext, dbr.Environment{IsDbr: true, Version: "15.4"}),
			mutateBundle: func(b *bundle.Bundle) {
				b.Config.Bundle.Mode = config.Development
				b.Config.Experimental = &config.Experimental{ImmutableFolder: true}
			},
			initialValue:  nil,
			expectedValue: nil,
		},
		{
			name: "preset enabled, immutable folder, bundle in Workspace, databricks runtime",
			ctx:  dbr.MockRuntime(testContext, dbr.Environment{IsDbr: true, Version: "15.4"}),
			mutateBundle: func(b *bundle.Bundle) {
				b.Config.Bundle.Mode = config.Development
				b.Config.Experimental = &config.Experimental{ImmutableFolder: true}
			},
			initialValue:  &enabled,
			expectedValue: &enabled,
			expectedError: "presets.source_linked_deployment cannot be enabled when experimental.immutable_folder is enabled",
		},
		{
			name: "preset disabled, immutable folder, bundle in Workspace, databricks runtime",
			ctx:  dbr.MockRuntime(testContext, dbr.Environment{IsDbr: true, Version: "15.4"}),
			mutateBundle: func(b *bundle.Bundle) {
				b.Config.Bundle.Mode = config.Development
				b.Config.Experimental = &config.Experimental{ImmutableFolder: true}
			},
			initialValue:  &disabled,
			expectedValue: &disabled,
		},
		{
			name: "preset enabled, production mode, bundle in Workspace, databricks runtime",
			ctx:  dbr.MockRuntime(testContext, dbr.Environment{IsDbr: true, Version: "15.4"}),
			mutateBundle: func(b *bundle.Bundle) {
				b.Config.Bundle.Mode = config.Production
			},
			initialValue:    &enabled,
			expectedValue:   &enabled,
			expectedWarning: "source-linked deployment in non-development mode is deprecated and will not be supported in a future release",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			b := &bundle.Bundle{
				SyncRootPath: workspacePath,
				Config: config.Root{
					Presets: config.Presets{
						SourceLinkedDeployment: tt.initialValue,
					},
				},
			}

			if tt.mutateBundle != nil {
				tt.mutateBundle(b)
			}

			bundletest.SetLocation(b, "presets.source_linked_deployment", []dyn.Location{{File: "databricks.yml"}})
			bundletest.SetLocation(b, "workspace.file_path", []dyn.Location{{File: "databricks.yml"}})

			diags := bundle.Apply(tt.ctx, b, mutator.ApplySourceLinkedDeploymentPreset())
			if diags.HasError() && tt.expectedError == "" {
				t.Fatalf("unexpected error: %v", diags)
			}

			if tt.expectedWarning != "" {
				require.Equal(t, tt.expectedWarning, diags[0].Summary)
				require.NotEmpty(t, diags[0].Locations)
			}

			if tt.expectedError != "" {
				require.Equal(t, tt.expectedError, diags[0].Summary)
				require.NotEmpty(t, diags[0].Locations)
			}

			require.Equal(t, tt.expectedValue, b.Config.Presets.SourceLinkedDeployment)
		})
	}
}

// TestSourceLinkedDeploymentWorkspaceFilePathResolution runs the mutators that decide
// what ${workspace.file_path} in a resource resolves to, for a development-mode bundle
// in /Workspace on Databricks Runtime (where source-linked deployment is auto-enabled).
func TestSourceLinkedDeploymentWorkspaceFilePathResolution(t *testing.T) {
	syncRootPath := "/Workspace/Users/user.name@company.com/my_bundle"
	ctx := dbr.MockRuntime(t.Context(), dbr.Environment{IsDbr: true, Version: "15.4"})
	enabled := true

	tests := []struct {
		name                 string
		immutableFolder      bool
		expectedSourceLinked *bool
		expectedPythonFile   string
	}{
		{
			name:                 "source-linked deployment resolves to the sync root",
			immutableFolder:      false,
			expectedSourceLinked: &enabled,
			expectedPythonFile:   syncRootPath + "/src/main.py",
		},
		{
			name:                 "immutable folder disables source-linked deployment and resolves to the snapshot",
			immutableFolder:      true,
			expectedSourceLinked: nil,
			expectedPythonFile:   resources.SnapshotFullPathRef + "/files/src/main.py",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			b := &bundle.Bundle{
				SyncRootPath: syncRootPath,
				Config: config.Root{
					Bundle: config.Bundle{
						Mode: config.Development,
					},
					Experimental: &config.Experimental{
						ImmutableFolder: tt.immutableFolder,
					},
					Workspace: config.Workspace{
						RootPath: "/Workspace/Users/user.name@company.com/.bundle/my_bundle/dev",
					},
					Resources: config.Resources{
						Jobs: map[string]*resources.Job{
							"job": {
								JobSettings: jobs.JobSettings{
									Tasks: []jobs.Task{
										{
											TaskKey: "task",
											SparkPythonTask: &jobs.SparkPythonTask{
												PythonFile: "${workspace.file_path}/src/main.py",
											},
										},
									},
								},
							},
						},
					},
				},
			}

			diags := bundle.ApplySeq(ctx, b,
				mutator.ApplySourceLinkedDeploymentPreset(),
				mutator.DefineDefaultWorkspacePaths(),
				mutator.ResolveVariableReferencesOnlyResources("workspace"),
			)
			require.NoError(t, diags.Error())

			require.Equal(t, tt.expectedSourceLinked, b.Config.Presets.SourceLinkedDeployment)
			require.Equal(t, tt.expectedPythonFile, b.Config.Resources.Jobs["job"].Tasks[0].SparkPythonTask.PythonFile)
		})
	}
}
