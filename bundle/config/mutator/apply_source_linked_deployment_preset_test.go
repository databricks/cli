package mutator_test

import (
	"context"
	"runtime"
	"testing"

	"github.com/databricks/cli/bundle"
	"github.com/databricks/cli/bundle/config"
	"github.com/databricks/cli/bundle/config/mutator"
	"github.com/databricks/cli/bundle/internal/bundletest"
	"github.com/databricks/cli/libs/dbr"
	"github.com/databricks/cli/libs/dyn"
	"github.com/stretchr/testify/require"
)

func TestApplyPresetsSourceLinkedDeployment(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("this test is not applicable on Windows because source-linked mode works only in the Databricks Workspace")
	}

	testContext := context.Background()
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
	}{
		{
			name:          "preset enabled, bundle in Workspace, databricks runtime",
			ctx:           dbr.MockRuntime(testContext, true),
			initialValue:  &enabled,
			expectedValue: &enabled,
		},
		{
			name: "preset enabled, bundle not in Workspace, databricks runtime",
			ctx:  dbr.MockRuntime(testContext, true),
			mutateBundle: func(b *bundle.Bundle) {
				b.SyncRootPath = "/Users/user.name@company.com"
			},
			initialValue:    &enabled,
			expectedValue:   &disabled,
			expectedWarning: "source-linked deployment is available only in the Databricks Workspace",
		},
		{
			name:            "preset enabled, bundle in Workspace, not databricks runtime",
			ctx:             dbr.MockRuntime(testContext, false),
			initialValue:    &enabled,
			expectedValue:   &disabled,
			expectedWarning: "source-linked deployment is available only in the Databricks Workspace",
		},
		{
			name:          "preset disabled, bundle in Workspace, databricks runtime",
			ctx:           dbr.MockRuntime(testContext, true),
			initialValue:  &disabled,
			expectedValue: &disabled,
		},
		{
			name:          "preset nil, bundle in Workspace, databricks runtime",
			ctx:           dbr.MockRuntime(testContext, true),
			initialValue:  nil,
			expectedValue: nil,
		},
		{
			name: "preset nil, dev mode true, bundle in Workspace, databricks runtime",
			ctx:  dbr.MockRuntime(testContext, true),
			mutateBundle: func(b *bundle.Bundle) {
				b.Config.Bundle.Mode = config.Development
			},
			initialValue:  nil,
			expectedValue: &enabled,
		},
		{
			name: "preset enabled, workspace.file_path is defined by user",
			ctx:  dbr.MockRuntime(testContext, true),
			mutateBundle: func(b *bundle.Bundle) {
				b.Config.Workspace.FilePath = "file_path"
			},
			initialValue:    &enabled,
			expectedValue:   &enabled,
			expectedWarning: "workspace.file_path setting will be ignored in source-linked deployment mode",
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
			if diags.HasError() {
				t.Fatalf("unexpected error: %v", diags)
			}

			if tt.expectedWarning != "" {
				require.Equal(t, tt.expectedWarning, diags[0].Summary)
				require.NotEmpty(t, diags[0].Locations)
			}

			require.Equal(t, tt.expectedValue, b.Config.Presets.SourceLinkedDeployment)
		})
	}
}
