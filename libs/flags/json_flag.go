package flags

import (
	"fmt"
	"os"

	"github.com/databricks/databricks-sdk-go/marshal"
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
	return marshal.UnmarshalCustom(j.raw, v, marshal.UnmarshalOptions{
		UnmarshalTopLevelIgnoredFields: true,
	})
}

func (j *JsonFlag) Type() string {
	return "JSON"
}
