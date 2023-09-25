package scripts

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"os/exec"
	"strings"

	"github.com/databricks/cli/bundle"
	"github.com/databricks/cli/bundle/config"
	"github.com/databricks/cli/libs/cmdio"
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

func (m *script) Apply(ctx context.Context, b *bundle.Bundle) error {
	cmd, out, err := executeHook(ctx, b, m.scriptHook)
	if err != nil {
		return err
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

	return cmd.Wait()
}

func executeHook(ctx context.Context, b *bundle.Bundle, hook config.ScriptHook) (*exec.Cmd, io.Reader, error) {
	command := getCommmand(b, hook)
	if command == "" {
		return nil, nil, nil
	}

	interpreter, err := findInterpreter()
	if err != nil {
		return nil, nil, err
	}

	// TODO: switch to process.Background(...)
	cmd := exec.CommandContext(ctx, interpreter, "-c", string(command))
	cmd.Dir = b.Config.Path

	outPipe, err := cmd.StdoutPipe()
	if err != nil {
		return nil, nil, err
	}

	errPipe, err := cmd.StderrPipe()
	if err != nil {
		return nil, nil, err
	}

	return cmd, io.MultiReader(outPipe, errPipe), cmd.Start()
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
