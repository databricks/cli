package flags

import (
	"fmt"
	"os"

	"github.com/databricks/cli/libs/dyn/convert"
	"github.com/databricks/cli/libs/dyn/jsonloader"
)

type JsonFlag struct {
	raw []byte
}

func (j *JsonFlag) String() string {
	return fmt.Sprintf("JSON (%d bytes)", len(j.raw))
}

// TODO: Command.MarkFlagFilename()
func (j *JsonFlag) Set(v string) error {
	// Load request from file if it starts with '@' (like curl).
	if v[0] != '@' {
		j.raw = []byte(v)
		return nil
	}
	buf, err := os.ReadFile(v[1:])
	if err != nil {
		return fmt.Errorf("read %s: %w", v, err)
	}
	j.raw = buf
	return nil
}

func (j *JsonFlag) Unmarshal(v any) error {
	if j.raw == nil {
		return nil
	}

	dv, err := jsonloader.LoadJSON(j.raw)
	if err != nil {
		return err
	}

	err = convert.ToTyped(v, dv)
	if err != nil {
		return err
	}

	_, diags := convert.Normalize(v, dv)
	if len(diags) > 0 {
		summary := ""
		for _, diag := range diags {
			summary += fmt.Sprintf("- %s\n", diag.Summary)
		}
		return fmt.Errorf("json input error:\n%v", summary)
	}

	return nil
}

func (j *JsonFlag) Type() string {
	return "JSON"
}
