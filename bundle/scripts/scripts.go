package scripts

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"strings"

	"github.com/databricks/cli/bundle"
	"github.com/databricks/cli/bundle/config"
	"github.com/databricks/cli/libs/cmdio"
	"github.com/databricks/cli/libs/diag"
	"github.com/databricks/cli/libs/exec"
	"github.com/databricks/cli/libs/log"
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

func (m *script) Apply(ctx context.Context, b *bundle.Bundle) diag.Diagnostics {
	executor, err := exec.NewCommandExecutor(b.Config.Path)
	if err != nil {
		return diag.FromErr(err)
	}

	cmd, out, err := executeHook(ctx, executor, b, m.scriptHook)
	if err != nil {
		return diag.FromErr(err)
	}
	if cmd == nil {
		log.Debugf(ctx, "No script defined for %s, skipping", m.scriptHook)
		return nil
	}

	cmdio.LogString(ctx, fmt.Sprintf("Executing '%s' script", m.scriptHook))

	reader := bufio.NewReader(out)
	line, err := reader.ReadString('\n')
	for err == nil {
		cmdio.LogString(ctx, strings.TrimSpace(line))
		line, err = reader.ReadString('\n')
	}

	return diag.FromErr(cmd.Wait())
}

func executeHook(ctx context.Context, executor *exec.Executor, b *bundle.Bundle, hook config.ScriptHook) (exec.Command, io.Reader, error) {
	command := getCommmand(b, hook)
	if command == "" {
		return nil, nil, nil
	}

	cmd, err := executor.StartCommand(ctx, string(command))
	if err != nil {
		return nil, nil, err
	}

	return cmd, io.MultiReader(cmd.Stdout(), cmd.Stderr()), nil
}

func getCommmand(b *bundle.Bundle, hook config.ScriptHook) config.Command {
	if b.Config.Experimental == nil || b.Config.Experimental.Scripts == nil {
		return ""
	}

	return b.Config.Experimental.Scripts[hook]
}
