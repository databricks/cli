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
	"github.com/databricks/cli/libs/diag"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestApplySyncMode(t *testing.T) {
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
		initialValue    config.SyncMode
		expectedValue   config.SyncMode
		expectedWarning string
		expectedError   string
	}{
		{
			name:          "in-place, bundle in Workspace, databricks runtime",
			ctx:           dbr.MockRuntime(testContext, dbr.Environment{IsDbr: true, Version: "15.4"}),
			initialValue:  config.SyncModeInPlace,
			expectedValue: config.SyncModeInPlace,
		},
		{
			name: "in-place, bundle not in Workspace, databricks runtime",
			ctx:  dbr.MockRuntime(testContext, dbr.Environment{IsDbr: true, Version: "15.4"}),
			mutateBundle: func(b *bundle.Bundle) {
				b.SyncRootPath = "/Users/user.name@company.com"
			},
			initialValue:    config.SyncModeInPlace,
			expectedValue:   config.SyncModeOverwrite,
			expectedWarning: "source-linked deployment is available only in the Databricks Workspace",
		},
		{
			name:            "in-place, bundle in Workspace, not databricks runtime",
			ctx:             dbr.MockRuntime(testContext, dbr.Environment{}),
			initialValue:    config.SyncModeInPlace,
			expectedValue:   config.SyncModeOverwrite,
			expectedWarning: "source-linked deployment is available only in the Databricks Workspace",
		},
		{
			name:          "overwrite, bundle in Workspace, databricks runtime",
			ctx:           dbr.MockRuntime(testContext, dbr.Environment{IsDbr: true, Version: "15.4"}),
			initialValue:  config.SyncModeOverwrite,
			expectedValue: config.SyncModeOverwrite,
		},
		{
			name:          "unset, bundle in Workspace, databricks runtime",
			ctx:           dbr.MockRuntime(testContext, dbr.Environment{IsDbr: true, Version: "15.4"}),
			initialValue:  "",
			expectedValue: "",
		},
		{
			name: "unset, dev mode true, bundle in Workspace, databricks runtime",
			ctx:  dbr.MockRuntime(testContext, dbr.Environment{IsDbr: true, Version: "15.4"}),
			mutateBundle: func(b *bundle.Bundle) {
				b.Config.Bundle.Mode = config.Development
			},
			initialValue:  "",
			expectedValue: config.SyncModeInPlace,
		},
		{
			name: "overwrite, dev mode true, bundle in Workspace, databricks runtime",
			ctx:  dbr.MockRuntime(testContext, dbr.Environment{IsDbr: true, Version: "15.4"}),
			mutateBundle: func(b *bundle.Bundle) {
				b.Config.Bundle.Mode = config.Development
			},
			initialValue:  config.SyncModeOverwrite,
			expectedValue: config.SyncModeOverwrite,
		},
		{
			name: "immutable, dev mode true, bundle in Workspace, databricks runtime",
			ctx:  dbr.MockRuntime(testContext, dbr.Environment{IsDbr: true, Version: "15.4"}),
			mutateBundle: func(b *bundle.Bundle) {
				b.Config.Bundle.Mode = config.Development
			},
			initialValue:  config.SyncModeImmutable,
			expectedValue: config.SyncModeImmutable,
		},
		{
			name: "in-place, workspace.file_path is defined by user",
			ctx:  dbr.MockRuntime(testContext, dbr.Environment{IsDbr: true, Version: "15.4"}),
			mutateBundle: func(b *bundle.Bundle) {
				b.Config.Workspace.FilePath = "file_path"
				b.Config.Bundle.Mode = config.Development
			},
			initialValue:    config.SyncModeInPlace,
			expectedValue:   config.SyncModeInPlace,
			expectedWarning: "workspace.file_path setting will be ignored in source-linked deployment mode",
		},
		{
			name: "in-place, apps is defined by user",
			ctx:  dbr.MockRuntime(testContext, dbr.Environment{IsDbr: true, Version: "15.4"}),
			mutateBundle: func(b *bundle.Bundle) {
				b.Config.Resources.Apps = map[string]*resources.App{
					"app": {},
				}
				b.Config.Bundle.Mode = config.Development
			},
			initialValue:  config.SyncModeInPlace,
			expectedValue: config.SyncModeInPlace,
		},
		{
			name: "in-place, production mode, bundle in Workspace, databricks runtime",
			ctx:  dbr.MockRuntime(testContext, dbr.Environment{IsDbr: true, Version: "15.4"}),
			mutateBundle: func(b *bundle.Bundle) {
				b.Config.Bundle.Mode = config.Production
			},
			initialValue:    config.SyncModeInPlace,
			expectedValue:   config.SyncModeInPlace,
			expectedWarning: "source-linked deployment in non-development mode is deprecated and will not be supported in a future release",
		},
		{
			name:          "invalid value",
			ctx:           dbr.MockRuntime(testContext, dbr.Environment{IsDbr: true, Version: "15.4"}),
			initialValue:  "inplace",
			expectedValue: "inplace",
			expectedError: `invalid value "inplace" for workspace.sync_mode (expected one of [overwrite in-place immutable])`,
		},
		{
			name: "deprecated source_linked_deployment enabled, bundle in Workspace, databricks runtime",
			ctx:  dbr.MockRuntime(testContext, dbr.Environment{IsDbr: true, Version: "15.4"}),
			mutateBundle: func(b *bundle.Bundle) {
				b.Config.Presets.SourceLinkedDeployment = &enabled
			},
			expectedValue:   config.SyncModeInPlace,
			expectedWarning: `presets.source_linked_deployment is deprecated, please set workspace.sync_mode to "in-place" instead`,
		},
		{
			name: "deprecated source_linked_deployment enabled, not databricks runtime",
			ctx:  dbr.MockRuntime(testContext, dbr.Environment{}),
			mutateBundle: func(b *bundle.Bundle) {
				b.Config.Presets.SourceLinkedDeployment = &enabled
			},
			expectedValue:   config.SyncModeOverwrite,
			expectedWarning: `presets.source_linked_deployment is deprecated, please set workspace.sync_mode to "in-place" instead`,
		},
		{
			name: "deprecated source_linked_deployment disabled, dev mode true, bundle in Workspace, databricks runtime",
			ctx:  dbr.MockRuntime(testContext, dbr.Environment{IsDbr: true, Version: "15.4"}),
			mutateBundle: func(b *bundle.Bundle) {
				b.Config.Presets.SourceLinkedDeployment = &disabled
				b.Config.Bundle.Mode = config.Development
			},
			expectedValue:   config.SyncModeOverwrite,
			expectedWarning: `presets.source_linked_deployment is deprecated, please set workspace.sync_mode to "overwrite" instead`,
		},
		{
			name: "deprecated immutable_folder",
			ctx:  dbr.MockRuntime(testContext, dbr.Environment{IsDbr: true, Version: "15.4"}),
			mutateBundle: func(b *bundle.Bundle) {
				b.Config.Experimental = &config.Experimental{ImmutableFolder: true}
			},
			expectedValue:   config.SyncModeImmutable,
			expectedWarning: `experimental.immutable_folder is deprecated, please set workspace.sync_mode to "immutable" instead`,
		},
		{
			name: "deprecated immutable_folder, dev mode true, bundle in Workspace, databricks runtime",
			ctx:  dbr.MockRuntime(testContext, dbr.Environment{IsDbr: true, Version: "15.4"}),
			mutateBundle: func(b *bundle.Bundle) {
				b.Config.Experimental = &config.Experimental{ImmutableFolder: true}
				b.Config.Bundle.Mode = config.Development
			},
			expectedValue:   config.SyncModeImmutable,
			expectedWarning: `experimental.immutable_folder is deprecated, please set workspace.sync_mode to "immutable" instead`,
		},
		{
			name: "deprecated immutable_folder, deprecated source_linked_deployment disabled",
			ctx:  dbr.MockRuntime(testContext, dbr.Environment{IsDbr: true, Version: "15.4"}),
			mutateBundle: func(b *bundle.Bundle) {
				b.Config.Experimental = &config.Experimental{ImmutableFolder: true}
				b.Config.Presets.SourceLinkedDeployment = &disabled
			},
			expectedValue:   config.SyncModeImmutable,
			expectedWarning: "presets.source_linked_deployment is deprecated and ignored because experimental.immutable_folder is set, please remove it",
		},
		{
			name: "deprecated immutable_folder, deprecated source_linked_deployment enabled",
			ctx:  dbr.MockRuntime(testContext, dbr.Environment{IsDbr: true, Version: "15.4"}),
			mutateBundle: func(b *bundle.Bundle) {
				b.Config.Experimental = &config.Experimental{ImmutableFolder: true}
				b.Config.Presets.SourceLinkedDeployment = &enabled
			},
			expectedValue:   config.SyncModeImmutable,
			expectedWarning: "presets.source_linked_deployment is deprecated and ignored because experimental.immutable_folder is set, please remove it",
		},
		{
			name: "deprecated source_linked_deployment with sync_mode",
			ctx:  dbr.MockRuntime(testContext, dbr.Environment{IsDbr: true, Version: "15.4"}),
			mutateBundle: func(b *bundle.Bundle) {
				b.Config.Presets.SourceLinkedDeployment = &enabled
			},
			initialValue:    config.SyncModeOverwrite,
			expectedValue:   config.SyncModeOverwrite,
			expectedWarning: "presets.source_linked_deployment is deprecated and ignored because workspace.sync_mode is set, please remove it",
		},
		{
			name: "deprecated immutable_folder with sync_mode",
			ctx:  dbr.MockRuntime(testContext, dbr.Environment{IsDbr: true, Version: "15.4"}),
			mutateBundle: func(b *bundle.Bundle) {
				b.Config.Experimental = &config.Experimental{ImmutableFolder: true}
			},
			initialValue:    config.SyncModeOverwrite,
			expectedValue:   config.SyncModeOverwrite,
			expectedWarning: "experimental.immutable_folder is deprecated and ignored because workspace.sync_mode is set, please remove it",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			b := &bundle.Bundle{
				SyncRootPath: workspacePath,
				Config: config.Root{
					Workspace: config.Workspace{
						SyncMode: tt.initialValue,
					},
				},
			}

			if tt.mutateBundle != nil {
				tt.mutateBundle(b)
			}

			bundletest.SetLocation(b, "workspace.sync_mode", []diag.Location{{File: "databricks.yml"}})
			bundletest.SetLocation(b, "presets.source_linked_deployment", []diag.Location{{File: "databricks.yml"}})
			bundletest.SetLocation(b, "experimental.immutable_folder", []diag.Location{{File: "databricks.yml"}})
			bundletest.SetLocation(b, "workspace.file_path", []diag.Location{{File: "databricks.yml"}})

			diags := bundle.Apply(tt.ctx, b, mutator.ApplySyncMode())
			if diags.HasError() && tt.expectedError == "" {
				t.Fatalf("unexpected error: %v", diags)
			}

			if tt.expectedWarning != "" {
				requireDiagnosticWithLocation(t, diags, diag.Warning, tt.expectedWarning)
			}

			if tt.expectedError != "" {
				requireDiagnosticWithLocation(t, diags, diag.Error, tt.expectedError)
			}

			require.Equal(t, tt.expectedValue, b.SyncMode())
		})
	}
}

// requireDiagnosticWithLocation requires diags to contain a diagnostic with the given severity
// and summary that points at a location.
func requireDiagnosticWithLocation(t *testing.T, diags diag.Diagnostics, severity diag.Severity, summary string) {
	t.Helper()
	for _, d := range diags {
		if d.Severity == severity && d.Summary == summary {
			require.NotEmpty(t, d.Locations)
			return
		}
	}
	t.Fatalf("expected diagnostic %q, got %v", summary, diags)
}

func TestApplySyncModeDeprecatedSourceLinkedDeploymentLocations(t *testing.T) {
	enabled := true
	b := &bundle.Bundle{
		SyncRootPath: "/Workspace/user.name@company.com",
		Config: config.Root{
			Presets: config.Presets{
				SourceLinkedDeployment: &enabled,
			},
		},
	}
	bundletest.SetLocation(b, "presets.source_linked_deployment", []diag.Location{{File: "databricks.yml", Line: 5}})

	diags := bundle.Apply(dbr.MockRuntime(t.Context(), dbr.Environment{}), b, mutator.ApplySyncMode())
	require.Len(t, diags, 2)

	// The in-place warning points at the deprecated setting that enabled it.
	assert.Equal(t, "source-linked deployment is available only in the Databricks Workspace", diags[1].Summary)
	assert.Equal(t, "presets.source_linked_deployment", diags[1].Paths[0].String())
	assert.Equal(t, []diag.Location{{File: "databricks.yml", Line: 5}}, diags[1].Locations)
}
