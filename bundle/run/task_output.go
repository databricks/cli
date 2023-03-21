package run

import (
	"encoding/json"
	"fmt"

	"github.com/databricks/databricks-sdk-go/service/jobs"
)

type NotebookOutput jobs.NotebookOutput
type DbtOutput jobs.DbtOutput
type SqlOutput jobs.SqlOutput
type LogsOutput struct {
	Logs          string
	LogsTruncated bool
}

func structToString(val interface{}) (string, error) {
	b, err := json.MarshalIndent(val, "", "  ")
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("==== Task Output ====\n%s", string(b)), nil
}

func (out *NotebookOutput) String() (string, error) {
	if out.Truncated {
		return fmt.Sprintf("%s\n[truncated...]", out.Result), nil
	}
	return out.Result, nil
}

func (out *DbtOutput) String() (string, error) {
	return structToString(out)
}

func (out *SqlOutput) String() (string, error) {
	return structToString(out)
}

func (out *LogsOutput) String() (string, error) {
	if out.LogsTruncated {
		return fmt.Sprintf("%s\n[truncated...]", out.Logs), nil
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
