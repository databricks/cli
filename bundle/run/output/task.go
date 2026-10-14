package output

import (
	"encoding/json"

	"github.com/databricks/cli/libs/sdktypes/jobs/v2"
	"github.com/databricks/cli/libs/utils"
)

type (
	NotebookOutput struct {
		// The value passed to dbutils.notebook.exit().
		Result string `json:"result,omitempty"`
		// Whether or not the result was truncated.
		Truncated bool `json:"truncated,omitempty"`
	}
	DbtOutput struct {
		// An optional map of headers to send when retrieving the artifact from
		// the `artifacts_link`.
		ArtifactsHeaders map[string]string `json:"artifacts_headers,omitempty"`
		// A pre-signed URL to download the (compressed) dbt artifacts.
		ArtifactsLink string `json:"artifacts_link,omitempty"`
	}
	SqlOutput  jobs.SqlTask_SqlOutput
	LogsOutput struct {
		Logs          string `json:"logs"`
		LogsTruncated bool   `json:"logs_truncated"`
	}
)

func structToString(val any) (string, error) {
	b, err := json.MarshalIndent(val, "", "  ")
	if err != nil {
		return "", err
	}
	return string(b), nil
}

func (out *NotebookOutput) String() (string, error) {
	if out.Truncated {
		return out.Result + "\n[truncated...]\n", nil
	}
	return out.Result, nil
}

func (out *DbtOutput) String() (string, error) {
	outputString, err := structToString(out)
	if err != nil {
		return "", err
	}

	// We add this prefix to make this output non machine readable.
	// JSON is used because it's a convenient representation.
	// If user needs machine parsable output, they can use the --output json
	// flag
	return "Dbt Task Output:\n" + outputString, nil
}

func (out *SqlOutput) String() (string, error) {
	outputString, err := structToString(out)
	if err != nil {
		return "", err
	}

	// We add this prefix to make this output non machine readable.
	// JSON is used because it's a convenient representation.
	// If user needs machine parsable output, they can use the --output json
	// flag
	return "SQL Task Output:\n" + outputString, nil
}

func (out *LogsOutput) String() (string, error) {
	if out.LogsTruncated {
		return out.Logs + "\n[truncated...]\n", nil
	}
	return out.Logs, nil
}

func toRunOutput(output *jobs.GetRunOutputResponse) RunOutput {
	logs := utils.Deref(output.Logs)
	logsTruncated := utils.Deref(output.LogsTruncated)
	switch {
	case output.NotebookOutput != nil:
		result := NotebookOutput{
			Result:    utils.Deref(output.NotebookOutput.Result),
			Truncated: utils.Deref(output.NotebookOutput.Truncated),
		}
		if result.Result == "" && !result.Truncated && logs != "" {
			return &LogsOutput{
				Logs:          logs,
				LogsTruncated: logsTruncated,
			}
		}
		return &result
	case output.DbtOutput != nil:
		return &DbtOutput{
			ArtifactsHeaders: output.DbtOutput.ArtifactsHeaders,
			ArtifactsLink:    utils.Deref(output.DbtOutput.ArtifactsLink),
		}

	case output.SqlOutput != nil:
		result := SqlOutput(*output.SqlOutput)
		return &result
	// Corresponds to JAR, python script and python wheel tasks
	case logs != "":
		return &LogsOutput{
			Logs:          logs,
			LogsTruncated: logsTruncated,
		}
	default:
		return nil
	}
}
