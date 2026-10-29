package run

import (
	"encoding/json"
	"fmt"
	"testing"

	flag "github.com/spf13/pflag"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestPipelineOptionsPathSelectionsPayload(t *testing.T) {
	tests := []struct {
		name     string
		opts     PipelineOptions
		expected string
	}{
		{
			name: "file selections for both refresh modes",
			opts: PipelineOptions{
				RefreshByPath:     []string{"/Workspace/Shared/source.sql"},
				FullRefreshByPath: []string{"/Workspace/Shared/notebook"},
			},
			expected: `{
				"refresh_selection_by_path": ["/Workspace/Shared/source.sql"],
				"full_refresh_selection_by_path": ["/Workspace/Shared/notebook"]
			}`,
		},
		{
			name: "table refresh and file full-refresh",
			opts: PipelineOptions{
				Refresh:           []string{"catalog.schema.table1"},
				FullRefreshByPath: []string{"/Workspace/Shared/notebook"},
			},
			expected: `{
				"refresh_selection": ["catalog.schema.table1"],
				"full_refresh_selection_by_path": ["/Workspace/Shared/notebook"]
			}`,
		},
		{
			name: "file refresh and table full-refresh",
			opts: PipelineOptions{
				RefreshByPath: []string{"/Workspace/Shared/source.sql"},
				FullRefresh:   []string{"catalog.schema.table2"},
			},
			expected: `{
				"refresh_selection_by_path": ["/Workspace/Shared/source.sql"],
				"full_refresh_selection": ["catalog.schema.table2"]
			}`,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			payload, err := tt.opts.toPayload(nil, "pipeline-id", false)
			require.NoError(t, err)
			assert.Equal(t, "pipeline-id", payload.PipelineId)
			data, err := json.Marshal(payload)
			require.NoError(t, err)
			assert.JSONEq(t, tt.expected, string(data))
		})
	}
}

func TestPipelineOptionsRejectMixedSelectors(t *testing.T) {
	tests := []struct {
		name string
		opts PipelineOptions
	}{
		{
			name: "--refresh",
			opts: PipelineOptions{
				Refresh:       []string{"table1"},
				RefreshByPath: []string{"/Workspace/Shared/source.sql"},
			},
		},
		{
			name: "--full-refresh",
			opts: PipelineOptions{
				FullRefresh:       []string{"table1"},
				FullRefreshByPath: []string{"/Workspace/Shared/source.sql"},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			payload, err := tt.opts.toPayload(nil, "pipeline-id", false)
			require.ErrorContains(t, err, "cannot mix table names and source files in "+tt.name)
			assert.Nil(t, payload)
		})
	}
}

func TestPipelineOptionsValidatePathSelections(t *testing.T) {
	for _, full := range []bool{false, true} {
		for _, exclusive := range []string{"refresh-all", "full-refresh-all", "validate-only"} {
			t.Run(fmt.Sprintf("full=%t/%s", full, exclusive), func(t *testing.T) {
				opts := PipelineOptions{
					RefreshAll:     exclusive == "refresh-all",
					FullRefreshAll: exclusive == "full-refresh-all",
					ValidateOnly:   exclusive == "validate-only",
				}
				if full {
					opts.FullRefreshByPath = []string{"/Workspace/Shared/source.sql"}
				} else {
					opts.RefreshByPath = []string{"/Workspace/Shared/source.sql"}
				}
				_, err := opts.toPayload(nil, "pipeline-id", false)
				assert.ErrorContains(t, err, "cannot use --refresh or --full-refresh together with --"+exclusive)
			})
		}
	}
}

func setupPipelineOptions(t *testing.T) (*flag.FlagSet, *PipelineOptions) {
	var fs flag.FlagSet
	var opts PipelineOptions
	opts.Define(&fs)
	return &fs, &opts
}

func TestPipelineOptionsRefreshAll(t *testing.T) {
	fs, opts := setupPipelineOptions(t)
	err := fs.Parse([]string{`--refresh-all`})
	require.NoError(t, err)
	assert.True(t, opts.RefreshAll)
}

func TestPipelineOptionsRefresh(t *testing.T) {
	fs, opts := setupPipelineOptions(t)
	err := fs.Parse([]string{`--refresh=arg1,arg2,arg3`})
	require.NoError(t, err)
	assert.Equal(t, []string{"arg1", "arg2", "arg3"}, opts.Refresh)
}

func TestPipelineOptionsFullRefreshAll(t *testing.T) {
	fs, opts := setupPipelineOptions(t)
	err := fs.Parse([]string{`--full-refresh-all`})
	require.NoError(t, err)
	assert.True(t, opts.FullRefreshAll)
}

func TestPipelineOptionsFullRefresh(t *testing.T) {
	fs, opts := setupPipelineOptions(t)
	err := fs.Parse([]string{`--full-refresh=arg1,arg2,arg3`})
	require.NoError(t, err)
	assert.Equal(t, []string{"arg1", "arg2", "arg3"}, opts.FullRefresh)
}

func TestPipelineOptionsValidateOnly(t *testing.T) {
	fs, opts := setupPipelineOptions(t)
	err := fs.Parse([]string{`--validate-only`})
	require.NoError(t, err)
	assert.True(t, opts.ValidateOnly)
}

func TestPipelineOptionsValidateSuccessWithSingleOption(t *testing.T) {
	args := []string{
		`--refresh-all`,
		`--refresh=arg1,arg2,arg3`,
		`--full-refresh-all`,
		`--full-refresh=arg1,arg2,arg3`,
		`--validate-only`,
	}
	for _, arg := range args {
		fs, opts := setupPipelineOptions(t)
		err := fs.Parse([]string{arg})
		require.NoError(t, err)
		err = opts.Validate(nil)
		assert.NoError(t, err)
	}
}

func TestPipelineOptionsValidateSuccessWithRefreshAndFullRefresh(t *testing.T) {
	fs, opts := setupPipelineOptions(t)
	err := fs.Parse([]string{`--refresh=arg1,arg2`, `--full-refresh=arg3,arg4`})
	require.NoError(t, err)
	err = opts.Validate(nil)
	assert.NoError(t, err)
	assert.Equal(t, []string{"arg1", "arg2"}, opts.Refresh)
	assert.Equal(t, []string{"arg3", "arg4"}, opts.FullRefresh)
}

func TestPipelineOptionsValidateFailureWithMultipleOptions(t *testing.T) {
	args := []string{
		`--refresh-all`,
		`--full-refresh-all`,
		`--validate-only`,
	}
	for i := range args {
		for j := range args {
			if i == j {
				continue
			}
			fs, opts := setupPipelineOptions(t)
			err := fs.Parse([]string{args[i], args[j]})
			require.NoError(t, err)
			err = opts.Validate(nil)
			assert.ErrorContains(t, err, "pipeline run flags are mutually exclusive")
		}
	}
}

func TestPipelineOptionsValidateFailureWithRefreshAndMutuallyExclusiveFlags(t *testing.T) {
	args := []string{
		`--refresh=arg1`,
		`--full-refresh=arg1`,
	}
	mutuallyExclusiveArgs := []string{
		`--refresh-all`,
		`--full-refresh-all`,
		`--validate-only`,
	}

	for _, refreshArg := range args {
		for _, exclusiveArg := range mutuallyExclusiveArgs {
			fs, opts := setupPipelineOptions(t)
			err := fs.Parse([]string{refreshArg, exclusiveArg})
			require.NoError(t, err)
			err = opts.Validate(nil)
			assert.ErrorContains(t, err, "cannot use --refresh or --full-refresh together with")
		}
	}
}
