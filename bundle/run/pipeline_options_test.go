package run

import (
	"testing"

	flag "github.com/spf13/pflag"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

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

func TestPipelineOptionsValidateFailureWithMultipleOptions(t *testing.T) {
	args := []string{
		`--refresh-all`,
		`--refresh=arg1,arg2,arg3`,
		`--full-refresh-all`,
		`--full-refresh=arg1,arg2,arg3`,
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
			assert.ErrorContains(t, err, "pipeline run arguments are mutually exclusive")
		}
	}
}
