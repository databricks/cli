package run

import (
	"testing"

	"github.com/databricks/cli/bundle/config/resources"
	"github.com/databricks/databricks-sdk-go/service/jobs"
	flag "github.com/spf13/pflag"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func setupJobOptions(t *testing.T) (*flag.FlagSet, *JobOptions) {
	var fs flag.FlagSet
	var opts JobOptions
	opts.DefineJobOptions(&fs)
	opts.DefineTaskOptions(&fs)
	return &fs, &opts
}

func TestJobOptionsDbtCommands(t *testing.T) {
	fs, opts := setupJobOptions(t)
	err := fs.Parse([]string{`--dbt-commands=arg1,arg2,arg3`})
	require.NoError(t, err)
	assert.Equal(t, []string{"arg1", "arg2", "arg3"}, opts.dbtCommands)
}

func TestJobOptionsDbtCommandsWithQuotes(t *testing.T) {
	fs, opts := setupJobOptions(t)
	err := fs.Parse([]string{`--dbt-commands="arg1","arg2,arg3"`})
	require.NoError(t, err)
	assert.Equal(t, []string{"arg1", "arg2,arg3"}, opts.dbtCommands)
}

func TestJobOptionsDbtCommandsMultiple(t *testing.T) {
	fs, opts := setupJobOptions(t)
	err := fs.Parse([]string{
		`--dbt-commands=arg1,arg2`,
		`--dbt-commands=arg3`,
	})
	require.NoError(t, err)
	assert.Equal(t, []string{"arg1", "arg2", "arg3"}, opts.dbtCommands)
}

func TestJobOptionsJarParams(t *testing.T) {
	fs, opts := setupJobOptions(t)
	err := fs.Parse([]string{`--jar-params=arg1,arg2,arg3`})
	require.NoError(t, err)
	assert.Equal(t, []string{"arg1", "arg2", "arg3"}, opts.jarParams)
}

func TestJobOptionsJarParamsWithQuotes(t *testing.T) {
	fs, opts := setupJobOptions(t)
	err := fs.Parse([]string{`--jar-params="arg1","arg2,arg3"`})
	require.NoError(t, err)
	assert.Equal(t, []string{"arg1", "arg2,arg3"}, opts.jarParams)
}

func TestJobOptionsJarParamsMultiple(t *testing.T) {
	fs, opts := setupJobOptions(t)
	err := fs.Parse([]string{
		`--jar-params=arg1,arg2`,
		`--jar-params=arg3`,
	})
	require.NoError(t, err)
	assert.Equal(t, []string{"arg1", "arg2", "arg3"}, opts.jarParams)
}

func TestJobOptionsNotebookParams(t *testing.T) {
	fs, opts := setupJobOptions(t)
	err := fs.Parse([]string{`--notebook-params=arg1=1,arg2=2,arg3=3`})
	require.NoError(t, err)
	assert.Equal(t, map[string]string{"arg1": "1", "arg2": "2", "arg3": "3"}, opts.notebookParams)
}

func TestJobOptionsNotebookParamsWithQuotes(t *testing.T) {
	fs, opts := setupJobOptions(t)
	err := fs.Parse([]string{`--notebook-params="arg1=1","arg2=2,arg3=3"`})
	require.NoError(t, err)
	assert.Equal(t, map[string]string{"arg1": "1", "arg2": "2,arg3=3"}, opts.notebookParams)
}

func TestJobOptionsNotebookParamsMultiple(t *testing.T) {
	fs, opts := setupJobOptions(t)
	err := fs.Parse([]string{
		`--notebook-params=arg1=1,arg2=2`,
		`--notebook-params=arg3=3`,
	})
	require.NoError(t, err)
	assert.Equal(t, map[string]string{"arg1": "1", "arg2": "2", "arg3": "3"}, opts.notebookParams)
}

func TestJobOptionsPythonNamedParams(t *testing.T) {
	fs, opts := setupJobOptions(t)
	err := fs.Parse([]string{`--python-named-params=arg1=1,arg2=2,arg3=3`})
	require.NoError(t, err)
	assert.Equal(t, map[string]string{"arg1": "1", "arg2": "2", "arg3": "3"}, opts.pythonNamedParams)
}

func TestJobOptionsPythonNamedParamsWithQuotes(t *testing.T) {
	fs, opts := setupJobOptions(t)
	err := fs.Parse([]string{`--python-named-params="arg1=1","arg2=2,arg3=3"`})
	require.NoError(t, err)
	assert.Equal(t, map[string]string{"arg1": "1", "arg2": "2,arg3=3"}, opts.pythonNamedParams)
}

func TestJobOptionsPythonNamedParamsMultiple(t *testing.T) {
	fs, opts := setupJobOptions(t)
	err := fs.Parse([]string{
		`--python-named-params=arg1=1,arg2=2`,
		`--python-named-params=arg3=3`,
	})
	require.NoError(t, err)
	assert.Equal(t, map[string]string{"arg1": "1", "arg2": "2", "arg3": "3"}, opts.pythonNamedParams)
}

func TestJobOptionsPythonParams(t *testing.T) {
	fs, opts := setupJobOptions(t)
	err := fs.Parse([]string{`--python-params=arg1,arg2,arg3`})
	require.NoError(t, err)
	assert.Equal(t, []string{"arg1", "arg2", "arg3"}, opts.pythonParams)
}

func TestJobOptionsPythonParamsWithQuotes(t *testing.T) {
	fs, opts := setupJobOptions(t)
	err := fs.Parse([]string{`--python-params="arg1","arg2,arg3"`})
	require.NoError(t, err)
	assert.Equal(t, []string{"arg1", "arg2,arg3"}, opts.pythonParams)
}

func TestJobOptionsPythonParamsMultiple(t *testing.T) {
	fs, opts := setupJobOptions(t)
	err := fs.Parse([]string{
		`--python-params=arg1,arg2`,
		`--python-params=arg3`,
	})
	require.NoError(t, err)
	assert.Equal(t, []string{"arg1", "arg2", "arg3"}, opts.pythonParams)
}

func TestJobOptionsSparkSubmitParams(t *testing.T) {
	fs, opts := setupJobOptions(t)
	err := fs.Parse([]string{`--spark-submit-params=arg1,arg2,arg3`})
	require.NoError(t, err)
	assert.Equal(t, []string{"arg1", "arg2", "arg3"}, opts.sparkSubmitParams)
}

func TestJobOptionsSparkSubmitParamsWithQuotes(t *testing.T) {
	fs, opts := setupJobOptions(t)
	err := fs.Parse([]string{`--spark-submit-params="arg1","arg2,arg3"`})
	require.NoError(t, err)
	assert.Equal(t, []string{"arg1", "arg2,arg3"}, opts.sparkSubmitParams)
}

func TestJobOptionsSparkSubmitParamsMultiple(t *testing.T) {
	fs, opts := setupJobOptions(t)
	err := fs.Parse([]string{
		`--spark-submit-params=arg1,arg2`,
		`--spark-submit-params=arg3`,
	})
	require.NoError(t, err)
	assert.Equal(t, []string{"arg1", "arg2", "arg3"}, opts.sparkSubmitParams)
}

func TestJobOptionsSqlParams(t *testing.T) {
	fs, opts := setupJobOptions(t)
	err := fs.Parse([]string{`--sql-params=arg1=1,arg2=2,arg3=3`})
	require.NoError(t, err)
	assert.Equal(t, map[string]string{"arg1": "1", "arg2": "2", "arg3": "3"}, opts.sqlParams)
}

func TestJobOptionsSqlParamsWithQuotes(t *testing.T) {
	fs, opts := setupJobOptions(t)
	err := fs.Parse([]string{`--sql-params="arg1=1","arg2=2,arg3=3"`})
	require.NoError(t, err)
	assert.Equal(t, map[string]string{"arg1": "1", "arg2": "2,arg3=3"}, opts.sqlParams)
}

func TestJobOptionsSqlParamsMultiple(t *testing.T) {
	fs, opts := setupJobOptions(t)
	err := fs.Parse([]string{
		`--sql-params=arg1=1,arg2=2`,
		`--sql-params=arg3=3`,
	})
	require.NoError(t, err)
	assert.Equal(t, map[string]string{"arg1": "1", "arg2": "2", "arg3": "3"}, opts.sqlParams)
}

func TestJobOptionsValidateIfJobHasJobParameters(t *testing.T) {
	job := &resources.Job{
		JobSettings: jobs.JobSettings{
			Parameters: []jobs.JobParameterDefinition{
				{
					Name:    "param",
					Default: "value",
				},
			},
		},
	}

	{
		// Test error if task parameters are specified.
		fs, opts := setupJobOptions(t)
		err := fs.Parse([]string{`--python-params=arg1`})
		require.NoError(t, err)
		err = opts.Validate(job)
		assert.ErrorContains(t, err, "the job to run defines job parameters; specifying task parameters is not allowed")
	}

	{
		// Test no error if job parameters are specified.
		fs, opts := setupJobOptions(t)
		err := fs.Parse([]string{`--params=arg1=val1`})
		require.NoError(t, err)
		err = opts.Validate(job)
		assert.NoError(t, err)
	}
}

func TestJobOptionsValidateIfJobHasNoJobParameters(t *testing.T) {
	job := &resources.Job{
		JobSettings: jobs.JobSettings{
			Parameters: []jobs.JobParameterDefinition{},
		},
	}

	{
		// Test error if job parameters are specified.
		fs, opts := setupJobOptions(t)
		err := fs.Parse([]string{`--params=arg1=val1`})
		require.NoError(t, err)
		err = opts.Validate(job)
		assert.ErrorContains(t, err, "the job to run does not define job parameters; specifying job parameters is not allowed")
	}

	{
		// Test no error if task parameters are specified.
		fs, opts := setupJobOptions(t)
		err := fs.Parse([]string{`--python-params=arg1`})
		require.NoError(t, err)
		err = opts.Validate(job)
		assert.NoError(t, err)
	}
}
