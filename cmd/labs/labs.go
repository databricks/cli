package labs

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/databricks/cli/cmd/labs/feature"
	"github.com/databricks/cli/cmd/root"
	"github.com/databricks/databricks-sdk-go/config"
	"github.com/spf13/cobra"
)

func New() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "labs",
		Short: "Databricks Labs features",
		Long:  `Manage experimental Databricks Labs apps`,
	}

	// TODO: this should be on the top CLI level
	cmd.AddGroup(&cobra.Group{
		ID:    "labs",
		Title: "Databricks Labs",
	})

	cmd.AddCommand(
		newListCommand(),
		newInstallCommand(),
		&cobra.Command{
			Use:   "py",
			Short: "...",
			RunE: func(cmd *cobra.Command, args []string) error {
				return nil
			},
		},
	)

	err := infuse(cmd)
	if err != nil {
		panic(err)
	}

	return cmd
}

type commandInput struct {
	Command    string         `json:"command"`
	Flags      map[string]any `json:"flags"`
	OutputType string         `json:"output_type"`
}

func propagateEnvConfig(cfg *config.Config) error {
	for _, a := range config.ConfigAttributes {
		if a.IsZero(cfg) {
			continue
		}
		for _, ev := range a.EnvVars {
			err := os.Setenv(ev, a.GetString(cfg))
			if err != nil {
				return fmt.Errorf("set %s: %w", a.Name, err)
			}
		}
	}
	return nil
}

func infuse(cmd *cobra.Command) error {
	ctx := cmd.Context()
	all, err := feature.LoadAll(ctx)
	if err != nil {
		return err
	}
	for _, f := range all {
		group := &cobra.Command{
			Use:     f.Name,
			Short:   f.Description,
			GroupID: "labs",
		}
		cmd.AddCommand(group)
		for _, v := range f.Commands {
			l := v
			definedFlags := v.Flags
			vcmd := &cobra.Command{
				Use:   v.Name,
				Short: v.Description,
				RunE: func(cmd *cobra.Command, args []string) error {
					flags := cmd.Flags()
					if f.Context == "workspace" {
						// TODO: context can be on both command and feature level
						err = root.MustWorkspaceClient(cmd, args)
						if err != nil {
							return err
						}
						// TODO: add account-level init as well
						w := root.WorkspaceClient(cmd.Context())
						propagateEnvConfig(w.Config)
					}
					ci := &commandInput{
						Command: l.Name,
						Flags:   map[string]any{},
					}
					for _, flag := range definedFlags {
						v, err := flags.GetString(flag.Name)
						if err != nil {
							return fmt.Errorf("get %s flag: %w", flag.Name, err)
						}
						ci.Flags[flag.Name] = v
					}
					logLevelFlag := flags.Lookup("log-level")
					if logLevelFlag != nil {
						ci.Flags["log_level"] = logLevelFlag.Value.String()
					}
					raw, err := json.Marshal(ci)
					if err != nil {
						return err
					}
					ctx := cmd.Context()
					// actually execute the command
					return f.Run(ctx, raw)
				},
			}
			flags := vcmd.Flags()
			for _, flag := range definedFlags {
				flags.String(flag.Name, "", flag.Description)
			}
			group.AddCommand(vcmd)
		}
	}
	return nil
}
