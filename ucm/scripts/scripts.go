// Package scripts runs user-defined shell commands at ucm phase boundaries.
// Mirrors bundle/scripts, with the hook set narrowed to the phases UCM
// surfaces to users (init, deploy, destroy).
package scripts

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"strings"

	"github.com/databricks/cli/libs/cmdio"
	"github.com/databricks/cli/libs/diag"
	"github.com/databricks/cli/libs/exec"
	"github.com/databricks/cli/libs/log"
	"github.com/databricks/cli/ucm"
	"github.com/databricks/cli/ucm/config"
)

// Execute returns a mutator that runs the script bound to hook, if any.
// No-op when the hook is unset.
func Execute(hook config.ScriptHook) ucm.Mutator {
	return &script{scriptHook: hook}
}

type script struct {
	scriptHook config.ScriptHook
}

func (m *script) Name() string {
	return fmt.Sprintf("scripts.%s", m.scriptHook)
}

func (m *script) Apply(ctx context.Context, u *ucm.Ucm) diag.Diagnostics {
	command := getCommand(u, m.scriptHook)
	if command == "" {
		log.Debugf(ctx, "No script defined for %s, skipping", m.scriptHook)
		return nil
	}

	executor, err := exec.NewCommandExecutor(u.RootPath)
	if err != nil {
		return diag.FromErr(err)
	}

	cmd, out, err := executeHook(ctx, executor, command)
	if err != nil {
		return diag.FromErr(fmt.Errorf("failed to execute script: %w", err))
	}

	cmdio.LogString(ctx, fmt.Sprintf("Executing '%s' script", m.scriptHook))

	reader := bufio.NewReader(out)
	line, err := reader.ReadString('\n')
	for err == nil {
		cmdio.LogString(ctx, strings.TrimSpace(line))
		line, err = reader.ReadString('\n')
	}

	err = cmd.Wait()
	if err != nil {
		return diag.FromErr(fmt.Errorf("failed to execute script: %w", err))
	}

	return nil
}

func executeHook(ctx context.Context, executor *exec.Executor, command string) (exec.Command, io.Reader, error) {
	cmd, err := executor.StartCommand(ctx, command)
	if err != nil {
		return nil, nil, err
	}
	return cmd, io.MultiReader(cmd.Stdout(), cmd.Stderr()), nil
}

func getCommand(u *ucm.Ucm, hook config.ScriptHook) string {
	s, ok := u.Config.Scripts[hook]
	if !ok {
		return ""
	}
	return s.Content
}
