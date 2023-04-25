package flags

import (
	"encoding/json"
	"fmt"
	"os"
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

func (j *JsonFlag) Unmarshall(v any) error {
	if j.raw == nil {
		return nil
	}
	return json.Unmarshal(j.raw, v)
}

func (j *JsonFlag) Type() string {
	return "JSON"
}
