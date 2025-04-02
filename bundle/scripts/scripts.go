package scripts

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"io"
	"strings"

	"github.com/databricks/cli/bundle"
	"github.com/databricks/cli/bundle/config"
	"github.com/databricks/cli/bundle/env"
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
	executor, err := exec.NewCommandExecutor(b.BundleRootPath)
	if err != nil {
		return diag.FromErr(err)
	}

	cmd, out, err := executeHook(ctx, executor, b, m.scriptHook)
	if err != nil {
		return diag.FromErr(fmt.Errorf("failed to execute script: %w", err))
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

	err = cmd.Wait()
	if err != nil {
		return diag.FromErr(fmt.Errorf("failed to execute script: %w", err))
	}

	return nil
}

func executeHook(ctx context.Context, executor *exec.Executor, b *bundle.Bundle, hook config.ScriptHook) (exec.Command, io.Reader, error) {
	command := getCommmand(b, hook)
	if command == "" {
		return nil, nil, nil
	}

	// Don't run any arbitrary code when restricted execution is enabled.
	if _, ok := env.RestrictedExecution(ctx); ok {
		return nil, nil, errors.New("Running scripts is not allowed when DATABRICKS_BUNDLE_RESTRICTED_CODE_EXECUTION is set")
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
