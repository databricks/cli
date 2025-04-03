package project

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"path/filepath"
	"strings"

	"github.com/databricks/cli/libs/cmdio"
	"github.com/databricks/cli/libs/log"
	"github.com/databricks/cli/libs/process"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
)

type proxy struct {
	Entrypoint    `yaml:",inline"`
	Name          string `yaml:"name"`
	Description   string `yaml:"description"`
	TableTemplate string `yaml:"table_template,omitempty"`
	Flags         []flag `yaml:"flags,omitempty"`
}

func (cp *proxy) register(parent *cobra.Command) {
	cmd := &cobra.Command{
		Use:   cp.Name,
		Short: cp.Description,
		RunE:  cp.runE,
	}
	parent.AddCommand(cmd)
	flags := cmd.Flags()
	for _, flag := range cp.Flags {
		flag.register(flags)
	}
}

func (cp *proxy) runE(cmd *cobra.Command, _ []string) error {
	err := cp.checkUpdates(cmd)
	if err != nil {
		return err
	}
	args, err := cp.commandInput(cmd)
	if err != nil {
		return err
	}
	envs, err := cp.Prepare(cmd)
	if err != nil {
		return fmt.Errorf("entrypoint: %w", err)
	}
	ctx := cmd.Context()
	log.Debugf(ctx, "Forwarding subprocess: %s", strings.Join(args, " "))
	if cp.TableTemplate != "" {
		return cp.renderJsonAsTable(cmd, args, envs)
	}
	err = process.Forwarded(ctx, args,
		cmd.InOrStdin(),
		cmd.OutOrStdout(),
		cmd.ErrOrStderr(),
		process.WithEnvs(envs))
	if errors.Is(err, fs.ErrNotExist) && cp.IsPythonProject() {
		msg := "cannot find Python %s. Please re-run: databricks labs install %s"
		return fmt.Errorf(msg, cp.MinPython, cp.Name)
	}
	return err
}

// [EXPERIMENTAL] this interface contract may change in the future.
// See https://github.com/databricks/cli/issues/994
func (cp *proxy) renderJsonAsTable(cmd *cobra.Command, args []string, envs map[string]string) error {
	var buf bytes.Buffer
	ctx := cmd.Context()
	err := process.Forwarded(ctx, args,
		cmd.InOrStdin(),
		&buf,
		cmd.ErrOrStderr(),
		process.WithEnvs(envs))
	if err != nil {
		return err
	}
	var anyVal any
	err = json.Unmarshal(buf.Bytes(), &anyVal)
	if err != nil {
		return err
	}
	// IntelliJ eagerly replaces tabs with spaces, even though we're not asking for it
	fixedTemplate := strings.ReplaceAll(cp.TableTemplate, "\\t", "\t")
	return cmdio.RenderWithTemplate(ctx, anyVal, "", fixedTemplate)
}

func (cp *proxy) commandInput(cmd *cobra.Command) ([]string, error) {
	flags := cmd.Flags()
	commandInput := struct {
		Command    string         `json:"command"`
		Flags      map[string]any `json:"flags"`
		OutputType string         `json:"output_type"`
	}{
		Command: cp.Name,
		Flags:   map[string]any{},
	}
	for _, f := range cp.Flags {
		v, err := f.get(flags)
		if err != nil {
			return nil, fmt.Errorf("get %s flag: %w", f.Name, err)
		}
		commandInput.Flags[f.Name] = v
	}
	logLevelFlag := flags.Lookup("log-level")
	if logLevelFlag != nil {
		commandInput.Flags["log_level"] = logLevelFlag.Value.String()
	}
	var args []string
	ctx := cmd.Context()
	if cp.IsPythonProject() {
		args = append(args, cp.virtualEnvPython(ctx))
		libDir := cp.EffectiveLibDir()
		entrypoint := filepath.Join(libDir, cp.Main)
		args = append(args, entrypoint)
	}
	raw, err := json.Marshal(commandInput)
	if err != nil {
		return nil, fmt.Errorf("command input: %w", err)
	}
	args = append(args, string(raw))
	return args, nil
}

type flag struct {
	Name        string `yaml:"name"`
	Description string `yaml:"description"`
	Default     any    `yaml:"default,omitempty"`
}

func (f *flag) register(pf *pflag.FlagSet) {
	var dflt string
	if f.Default != nil {
		dflt = fmt.Sprint(f.Default)
	}
	pf.String(f.Name, dflt, f.Description)
}

func (f *flag) get(pf *pflag.FlagSet) (any, error) {
	return pf.GetString(f.Name)
}
