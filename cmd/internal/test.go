package internal

import (
	"bytes"
	"context"

	"github.com/databricks/cli/cmd"
)

func RunGetOutput(ctx context.Context, args ...string) ([]byte, error) {
	root := cmd.New()
	args = append(args, "--log-level", "debug")
	root.SetArgs(args)
	var buf bytes.Buffer
	root.SetOut(&buf)
	err := root.ExecuteContext(ctx)
	if err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}
