package scripts

import (
	"bufio"
	"bytes"
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
	command := getCommand(b, m.scriptHook)
	if command == "" {
		log.Debugf(ctx, "No script defined for %s, skipping", m.scriptHook)
		return nil
	}

	executor, err := exec.NewCommandExecutor(b.BundleRootPath)
	if err != nil {
		return diag.FromErr(err)
	}

	cmd, err := executeHook(ctx, executor, command)
	if err != nil {
		return diag.FromErr(fmt.Errorf("failed to execute script: %w", err))
	}

	cmdio.LogString(ctx, fmt.Sprintf("Executing '%s' script", m.scriptHook))

	// Spool stderr to memory while stdout is being streamed. Reading the two
	// pipes sequentially (stdout to EOF, then stderr) deadlocks once the
	// script writes more than the OS pipe buffer (~64KiB) to stderr while
	// stdout is still open: the script blocks on the stderr write and never
	// closes stdout. Spooling keeps stderr drained while preserving the
	// stdout-then-stderr output order.
	var stderr bytes.Buffer
	stderrDone := make(chan struct{})
	go func() {
		defer close(stderrDone)
		_, _ = io.Copy(&stderr, cmd.Stderr())
	}()

	logOutput(ctx, cmd.Stdout())
	<-stderrDone
	logOutput(ctx, &stderr)

	err = cmd.Wait()
	if err != nil {
		return diag.FromErr(fmt.Errorf("failed to execute script: %w", err))
	}

	return nil
}

// logOutput logs output line by line, including a final line without a
// trailing newline.
func logOutput(ctx context.Context, out io.Reader) {
	reader := bufio.NewReader(out)
	for {
		line, err := reader.ReadString('\n')
		if line != "" {
			cmdio.LogString(ctx, strings.TrimSpace(line))
		}
		if err != nil {
			break
		}
	}
}

func executeHook(ctx context.Context, executor *exec.Executor, command config.Command) (exec.Command, error) {
	// Don't run any arbitrary code when restricted execution is enabled.
	if _, ok := env.RestrictedExecution(ctx); ok {
		return nil, errors.New("running scripts is not allowed when DATABRICKS_BUNDLE_RESTRICTED_CODE_EXECUTION is set")
	}

	return executor.StartCommand(ctx, string(command))
}

func getCommand(b *bundle.Bundle, hook config.ScriptHook) config.Command {
	if b.Config.Experimental == nil || b.Config.Experimental.Scripts == nil {
		return ""
	}

	return b.Config.Experimental.Scripts[hook]
}
