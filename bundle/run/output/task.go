package output

import (
	"encoding/json"

	"github.com/databricks/databricks-sdk-go/service/jobs"
)

type (
	NotebookOutput jobs.NotebookOutput
	DbtOutput      jobs.DbtOutput
	SqlOutput      jobs.SqlOutput
	LogsOutput     struct {
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

func toRunOutput(output *jobs.RunOutput) RunOutput {
	switch {
	case output.NotebookOutput != nil:
		result := NotebookOutput(*output.NotebookOutput)
		return &result
	case output.DbtOutput != nil:
		result := DbtOutput(*output.DbtOutput)
		return &result

	case output.SqlOutput != nil:
		result := SqlOutput(*output.SqlOutput)
		return &result
	// Corresponds to JAR, python script and python wheel tasks
	case output.Logs != "":
		result := LogsOutput{
			Logs:          output.Logs,
			LogsTruncated: output.LogsTruncated,
		}
		return &result
	default:
		return nil
	}
}
