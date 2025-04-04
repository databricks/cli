package apps

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/databricks/cli/cmd/root"
	"github.com/databricks/cli/libs/appproxy"
	"github.com/databricks/cli/libs/apps"
	"github.com/databricks/cli/libs/auth"
	"github.com/databricks/cli/libs/cmdctx"
	"github.com/databricks/cli/libs/cmdio"
	"github.com/spf13/cobra"
)

var (
	port               int
	debug              bool
	prepareEnvironment bool
	entryPoint         string
	customEnv          []string
)

func newRunLocal() *cobra.Command {
	cmd := &cobra.Command{}

	cmd.Use = "run-local"
	cmd.Short = `Run an app locally`
	cmd.Long = `Run an app locally.

	  This command starts an app locally.`

	cmd.Flags().IntVar(&port, "port", 8001, "Port on which to run the app")
	cmd.Flags().BoolVar(&debug, "debug", false, "Enable debug mode")
	cmd.Flags().BoolVar(&prepareEnvironment, "prepare-environment", false, "Prepares the environment for running the app. Requires 'uv' to be installed.")
	cmd.Flags().StringSliceVar(&customEnv, "env", nil, "Set environment variables")
	cmd.Flags().StringVar(&entryPoint, "entry-point", "", "Specify the custom entry point with configuration (.yml file) for the app")

	cmd.PreRunE = root.MustWorkspaceClient
	cmd.RunE = func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		w := cmdctx.WorkspaceClient(ctx)
		workspaceId, err := w.CurrentWorkspaceID(ctx)
		if err != nil {
			return err
		}

		cwd, err := os.Getwd()
		if err != nil {
			return err
		}

		config := apps.NewConfig(w.Config.Host, workspaceId, cwd)
		if entryPoint != "" {
			config.AppSpecFiles = []string{entryPoint}
		}
		spec, err := apps.ReadAppSpecFile(config)
		if err != nil {
			return err
		}

		env := auth.ProcessEnv(cmdctx.ConfigUsed(cmd.Context()))
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

		proxy, err := appproxy.New(ctx, config.AppURL)
		if err != nil {
			return err
		}

		me, err := w.CurrentUser.Me(ctx)
		if err != nil {
			return err
		}

		for key, value := range apps.GetXHeaders(me) {
			proxy.InjectHeader(key, value)
		}

		proxyAddr := fmt.Sprintf("localhost:%d", port)
		go func() {
			cmdio.LogString(ctx, "To access your app go to http://"+proxyAddr)
			err := proxy.ListenAndServe(proxyAddr)
			if err != nil {
				cmd.PrintErrln(err)
			}
		}()

		if debug {
			cmdio.LogString(ctx, "To debug your app, attach a debugger to port "+apps.DEBUG_PORT)
		}

		// Create a channel to handle graceful shutdown
		done := make(chan error, 1)

		// Handle interrupt signal
		sigChan := make(chan os.Signal, 1)
		signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)

		// Wait for either command completion or interrupt
		go func() {
			done <- appCmd.Wait()
		}()

		select {
		case err := <-done:
			if err != nil {
				return err
			}
		case <-sigChan:
			// Give the process a chance to cleanup
			if err := appCmd.Process.Signal(os.Interrupt); err != nil {
				return fmt.Errorf("failed to send interrupt signal: %w", err)
			}

			// Wait for process to finish with timeout
			select {
			case err := <-done:
				return err
			case <-time.After(10 * time.Second):
				// Force kill if timeout
				if err := appCmd.Process.Kill(); err != nil {
					return fmt.Errorf("failed to kill process: %w", err)
				}
				return errors.New("process killed after timeout")
			}
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
