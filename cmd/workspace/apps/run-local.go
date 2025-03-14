package apps

import (
	"fmt"
	"os"
	"os/exec"
	"strings"

	"github.com/databricks/cli/cmd/root"
	"github.com/databricks/cli/libs/apps"
	"github.com/databricks/cli/libs/auth"
	"github.com/databricks/cli/libs/cmdio"
	"github.com/databricks/cli/libs/command"
	"github.com/databricks/cli/libs/httpproxy"
	"github.com/spf13/cobra"
)

var (
	debug              bool
	prepareEnvironment bool
	customEnv          []string
)

const DEFAULT_PROXY_ADDR = "127.0.0.1:8001"

func newRunLocal() *cobra.Command {
	cmd := &cobra.Command{}

	cmd.Use = "run-local"
	cmd.Short = `Run an app locally`
	cmd.Long = `Run an app locally.

	  This command starts an app locally.`

	cmd.Flags().BoolVar(&debug, "debug", false, "Enable debug mode")
	cmd.Flags().BoolVar(&prepareEnvironment, "prepare-environment", false, "Prepares the environment for running the app. Requires 'uv' to be installed.")
	cmd.Flags().StringSliceVar(&customEnv, "env", nil, "Set environment variables")

	cmd.PreRunE = root.MustWorkspaceClient
	cmd.RunE = func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		w := command.WorkspaceClient(ctx)
		workspaceId, err := w.CurrentWorkspaceID(ctx)
		if err != nil {
			return err
		}

		cwd, err := os.Getwd()
		if err != nil {
			return err
		}

		config := apps.NewConfig(w.Config.Host, workspaceId, cwd)
		spec, err := apps.ReadAppSpecFile(config)
		if err != nil {
			return err
		}

		env := auth.ProcessEnv(command.ConfigUsed(cmd.Context()))
		profileFlag := cmd.Flag("profile")
		if profileFlag != nil {
			env = append(env, "DATABRICKS_CONFIG_PROFILE="+profileFlag.Value.String())
		}

		for _, envVar := range spec.EnvVars {
			if envVar.Value != nil {
				env = append(env, envVar.Name+"="+*envVar.Value)
			}

			if envVar.ValueFrom != nil {
				found := false
				for _, e := range customEnv {
					if strings.HasPrefix(e, envVar.Name+"=") {
						found = true
						break
					}
				}
				if !found {
					return fmt.Errorf("env var %s needs to be set", envVar.Name)
				}
			}
		}

		app := apps.NewApp(config, spec)
		if prepareEnvironment {
			err := app.PrepareEnvironment()
			if err != nil {
				return err
			}
		}

		specCommand, err := app.GetCommand(debug)
		if err != nil {
			return err
		}

		appCmd := exec.Command(specCommand[0], specCommand[1:]...)
		appCmd.Stdin = cmd.InOrStdin()
		appCmd.Stdout = cmd.OutOrStdout()
		appCmd.Stderr = cmd.ErrOrStderr()

		appEnvs := apps.GetBaseEnvVars(config)
		for _, envVar := range appEnvs {
			env = append(env, envVar.String())
		}
		env = append(env, customEnv...)

		appCmd.Env = env
		appCmd.Dir = config.AppPath

		err = appCmd.Start()
		if err != nil {
			return err
		}

		proxy, err := httpproxy.NewProxy(config.AppURL)
		if err != nil {
			return err
		}

		go func() {
			cmdio.LogString(ctx, "To access your app go to http://"+DEFAULT_PROXY_ADDR)
			err := proxy.Start(DEFAULT_PROXY_ADDR)
			if err != nil {
				cmd.PrintErrln(err)
			}
		}()

		if debug {
			cmdio.LogString(ctx, "To debug your app, attach a debugger to port "+apps.DEBUG_PORT)
		}

		err = appCmd.Wait()
		if err != nil {
			return err
		}

		return nil
	}

	cmd.ValidArgsFunction = cobra.NoFileCompletions

	return cmd
}

func init() {
	cmdOverrides = append(cmdOverrides, func(cmd *cobra.Command) {
		cmd.AddCommand(newRunLocal())
	})
}
