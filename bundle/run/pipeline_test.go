package run

import (
	"errors"
	"testing"
	"time"

	"github.com/databricks/cli/bundle"
	"github.com/databricks/cli/bundle/config"
	"github.com/databricks/cli/bundle/config/resources"
	"github.com/databricks/cli/libs/cmdio"
	sdk_config "github.com/databricks/databricks-sdk-go/config"
	"github.com/databricks/databricks-sdk-go/experimental/mocks"
	"github.com/databricks/databricks-sdk-go/service/pipelines"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func TestPipelineRunnerCancel(t *testing.T) {
	pipeline := &resources.Pipeline{
		ID: "123",
	}

	b := &bundle.Bundle{
		Config: config.Root{
			Resources: config.Resources{
				Pipelines: map[string]*resources.Pipeline{
					"test_pipeline": pipeline,
				},
			},
		},
	}

	runner := pipelineRunner{key: "test", bundle: b, pipeline: pipeline}

	m := mocks.NewMockWorkspaceClient(t)
	b.SetWorkpaceClient(m.WorkspaceClient)

	mockWait := &pipelines.WaitGetPipelineIdle[struct{}]{
		Poll: func(time.Duration, func(*pipelines.GetPipelineResponse)) (*pipelines.GetPipelineResponse, error) {
			return nil, nil
		},
	}

	pipelineApi := m.GetMockPipelinesAPI()
	pipelineApi.EXPECT().Stop(t.Context(), pipelines.StopRequest{
		PipelineId: "123",
	}).Return(mockWait, nil)

	err := runner.Cancel(t.Context())
	require.NoError(t, err)
}

func TestPipelineRunnerRestart(t *testing.T) {
	pipeline := &resources.Pipeline{
		ID: "123",
	}

	b := &bundle.Bundle{
		Config: config.Root{
			Resources: config.Resources{
				Pipelines: map[string]*resources.Pipeline{
					"test_pipeline": pipeline,
				},
			},
		},
	}

	runner := pipelineRunner{key: "test", bundle: b, pipeline: pipeline}

	m := mocks.NewMockWorkspaceClient(t)
	m.WorkspaceClient.Config = &sdk_config.Config{
		Host: "https://test.test",
	}
	b.SetWorkpaceClient(m.WorkspaceClient)

	ctx := cmdio.MockDiscard(t.Context())

	mockWait := &pipelines.WaitGetPipelineIdle[struct{}]{
		Poll: func(time.Duration, func(*pipelines.GetPipelineResponse)) (*pipelines.GetPipelineResponse, error) {
			return nil, nil
		},
	}

	pipelineApi := m.GetMockPipelinesAPI()
	pipelineApi.EXPECT().Stop(mock.Anything, pipelines.StopRequest{
		PipelineId: "123",
	}).Return(mockWait, nil)

	// Mock runner starting a new update
	pipelineApi.EXPECT().StartUpdate(mock.Anything, pipelines.StartUpdate{
		PipelineId: "123",
	}).Return(&pipelines.StartUpdateResponse{
		UpdateId: "456",
	}, nil)

	// Mock runner polling for events
	pipelineApi.EXPECT().ListPipelineEventsAll(mock.Anything, pipelines.ListPipelineEventsRequest{
		Filter:     `update_id = '456'`,
		MaxResults: 100,
		PipelineId: "123",
	}).Return([]pipelines.PipelineEvent{}, nil)

	// Mock runner polling for update status
	pipelineApi.EXPECT().GetUpdateByPipelineIdAndUpdateId(mock.Anything, "123", "456").
		Return(&pipelines.GetUpdateResponse{
			Update: &pipelines.UpdateInfo{
				State: pipelines.UpdateInfoStateCompleted,
			},
		}, nil)

	_, err := runner.Restart(ctx, &Options{})
	require.NoError(t, err)
}

func TestPipelineRunnerQuotedTables(t *testing.T) {
	tests := []struct {
		name   string
		mode   pipelines.PublishingMode
		schema string
		legacy bool
	}{
		{name: "DPM with schema", mode: pipelines.PublishingModeDefaultPublishingMode, schema: "default"},
		{name: "DPM with target", mode: pipelines.PublishingModeDefaultPublishingMode},
		{name: "legacy", mode: pipelines.PublishingModeLegacyPublishingMode, legacy: true},
		{name: "unset mode with schema", schema: "default"},
		{name: "unset mode with target", legacy: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fs, opts := setupPipelineOptions(t)
			require.NoError(t, fs.Parse([]string{"--refresh", "table:`a`` b,c` orders", "--full-refresh", "`daily rollup`"}))
			require.NoError(t, opts.ResolveRefreshSelectors(t.Context(), nil))
			pipeline := &resources.Pipeline{ID: "123"}
			pipeline.Schema = tt.schema
			b := &bundle.Bundle{}
			m := mocks.NewMockWorkspaceClient(t)
			m.WorkspaceClient.Config = &sdk_config.Config{Host: "https://test.test"}
			b.SetWorkpaceClient(m.WorkspaceClient)
			spec := &pipelines.PipelineSpec{Schema: tt.schema}
			if tt.schema == "" {
				spec.Target = "default"
			}
			api := m.GetMockPipelinesAPI()
			api.EXPECT().GetByPipelineId(mock.Anything, "123").Return(&pipelines.GetPipelineResponse{
				Spec: spec, EffectivePublishingMode: tt.mode,
			}, nil).Once()
			want := pipelines.StartUpdate{
				PipelineId: "123", RefreshSelection: []string{"`a`` b,c`", "orders"}, FullRefreshSelection: []string{"`daily rollup`"},
			}
			if tt.legacy {
				want.RefreshSelection = []string{"a` b,c", "orders"}
				want.FullRefreshSelection = []string{"daily rollup"}
			}
			api.EXPECT().StartUpdate(mock.Anything, want).Return(&pipelines.StartUpdateResponse{UpdateId: "456"}, nil).Once()
			runner := pipelineRunner{key: "test", bundle: b, pipeline: pipeline}
			_, err := runner.Run(cmdio.MockDiscard(t.Context()), &Options{Pipeline: *opts, NoWait: true})
			require.NoError(t, err)
			require.Equal(t, []string{"`a`` b,c`", "orders"}, opts.Refresh)
		})
	}
}

func TestPipelineRunnerRestartFileSelections(t *testing.T) {
	tests := []struct {
		name       string
		continuous bool
		full       bool
		wildcard   bool
		getErr     error
		err        string
	}{
		{name: "continuous refresh", continuous: true, err: "file selectors with --restart require a triggered pipeline"},
		{name: "continuous full refresh", continuous: true, full: true, err: "file selectors with --restart require a triggered pipeline"},
		{name: "triggered refresh"},
		{name: "triggered full refresh", full: true},
		{name: "continuous file selection with wildcard", continuous: true, full: true, wildcard: true, err: "file selectors with --restart require a triggered pipeline"},
		{name: "triggered file selection with wildcard", full: true, wildcard: true},
		{name: "cannot read deployed pipeline", getErr: errors.New("cannot read pipeline"), err: "cannot read pipeline"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			pipeline := &resources.Pipeline{ID: "123"}
			b := &bundle.Bundle{}
			m := mocks.NewMockWorkspaceClient(t)
			m.WorkspaceClient.Config = &sdk_config.Config{Host: "https://test.test"}
			b.SetWorkpaceClient(m.WorkspaceClient)
			api := m.GetMockPipelinesAPI()
			api.EXPECT().GetByPipelineId(mock.Anything, "123").Return(&pipelines.GetPipelineResponse{
				Spec: &pipelines.PipelineSpec{Continuous: tt.continuous}, //nolint:staticcheck // Exercise the supported legacy pipeline mode.
			}, tt.getErr).Once()
			opts := &Options{NoWait: true}
			if tt.full {
				opts.Pipeline.FullRefreshByPath = []string{"/Workspace/Shared/orders.py"}
			} else {
				opts.Pipeline.RefreshByPath = []string{"/Workspace/Shared/orders.py"}
			}
			if tt.wildcard {
				opts.Pipeline.Refresh = []string{"*"}
			}
			if tt.err == "" {
				wait := &pipelines.WaitGetPipelineIdle[struct{}]{
					Poll: func(time.Duration, func(*pipelines.GetPipelineResponse)) (*pipelines.GetPipelineResponse, error) {
						return nil, nil
					},
				}
				api.EXPECT().Stop(mock.Anything, pipelines.StopRequest{PipelineId: "123"}).Return(wait, nil).Once()
				api.EXPECT().StartUpdate(mock.Anything, pipelines.StartUpdate{
					PipelineId: "123", RefreshSelection: opts.Pipeline.Refresh,
					RefreshSelectionByPath: opts.Pipeline.RefreshByPath, FullRefreshSelectionByPath: opts.Pipeline.FullRefreshByPath,
				}).Return(&pipelines.StartUpdateResponse{UpdateId: "456"}, nil).Once()
			}
			runner := pipelineRunner{key: "test", bundle: b, pipeline: pipeline}
			_, err := runner.Restart(cmdio.MockDiscard(t.Context()), opts)
			if tt.err != "" {
				require.ErrorContains(t, err, tt.err)
				api.AssertNotCalled(t, "Stop", mock.Anything, mock.Anything)
				api.AssertNotCalled(t, "StartUpdate", mock.Anything, mock.Anything)
			} else {
				require.NoError(t, err)
			}
		})
	}
}
