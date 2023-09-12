package scripts

import (
	"bytes"
	"context"
	"fmt"
	"os/exec"

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

	interpreter, err := findInterpreter()
	if err != nil {
		return nil, err
	}

	cmd := exec.CommandContext(ctx, interpreter, "-c", string(command))
	return cmd.CombinedOutput()
}

func getCommmand(b *bundle.Bundle, hook config.ScriptHook) config.Command {
	if b.Config.Experimental == nil || b.Config.Experimental.Scripts == nil {
		return ""
	}

	return b.Config.Experimental.Scripts[hook]
}

func findInterpreter() (string, error) {
	// At the moment we just return 'sh' on all platforms and use it to execute scripts
	return "sh", nil
}
