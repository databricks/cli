package scripts

import (
	"bytes"
	"context"
	"fmt"
	"os/exec"
	"strings"

	"github.com/databricks/cli/bundle"
	"github.com/databricks/cli/bundle/config"
	"github.com/databricks/cli/libs/cmdio"
)

func Execute(hook config.ScriptHook) bundle.Mutator {
	return &script{
		scriptHook: hook,
	}
}

type script struct {
	scriptHook config.ScriptHook
}

func (m *script) Name() string {
	return fmt.Sprintf("scripts.%s", m.scriptHook)
}

func (m *script) Apply(ctx context.Context, b *bundle.Bundle) error {
	out, err := executeHook(ctx, b, m.scriptHook)
	cmdio.LogString(ctx, bytes.NewBuffer(out).String())
	return err
}

func executeHook(ctx context.Context, b *bundle.Bundle, hook config.ScriptHook) ([]byte, error) {
	command := getCommmand(b, hook)
	if command == "" {
		return nil, nil
	}
	commands := strings.Split(strings.ReplaceAll(string(command), "\r\n", "\n"), "\n")

	out := make([][]byte, 0)
	for _, command := range commands {
		if command == "" {
			continue
		}

		subcommands := strings.Split(command, " && ")
		for _, subcommand := range subcommands {
			buildParts := strings.Split(subcommand, " ")
			cmd := exec.CommandContext(ctx, buildParts[0], buildParts[1:]...)
			res, err := cmd.CombinedOutput()
			if err != nil {
				return res, err
			}
			out = append(out, res)
		}
	}
	return bytes.Join(out, []byte{}), nil
}

func getCommmand(b *bundle.Bundle, hook config.ScriptHook) config.Command {
	if b.Config.Experimental == nil || b.Config.Experimental.Scripts == nil {
		return ""
	}

	return b.Config.Experimental.Scripts[hook]
}
