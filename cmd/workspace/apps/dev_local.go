package apps

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/databricks/cli/cmd/root"
	"github.com/databricks/cli/internal/appenv"
	"github.com/databricks/cli/libs/appproxy"
	"github.com/databricks/cli/libs/apps/runlocal"
	"github.com/databricks/cli/libs/auth"
	"github.com/databricks/cli/libs/cmdctx"
	"github.com/databricks/cli/libs/cmdio"
	"github.com/databricks/databricks-sdk-go"
	"github.com/spf13/cobra"
)

// Databricks Apps send a SIGKILL signal 15 seconds after a SIGTERM
// https://docs.databricks.com/aws/en/dev-tools/databricks-apps/app-development#important-guidelines-for-implementing-databricks-apps
const SHUTDOWN_TIMEOUT = 15 * time.Second

func setupWorkspaceAndConfig(cmd *cobra.Command, entryPoint string, appPort int) (*runlocal.Config, *runlocal.AppSpec, error) {
	ctx := cmd.Context()
	w := cmdctx.WorkspaceClient(ctx)
	workspaceId, err := w.CurrentWorkspaceID(ctx)
	if err != nil {
		return nil, nil, err
	}

	cwd, err := os.Getwd()
	if err != nil {
		return nil, nil, err
	}

	config := runlocal.NewConfig(w.Config.Host, workspaceId, cwd, runlocal.DEFAULT_HOST, appPort)
	if entryPoint != "" {
		config.AppSpecFiles = []string{entryPoint}
	}
	spec, err := runlocal.ReadAppSpecFile(config)
	if err != nil {
		return nil, nil, err
	}

	return config, spec, nil
}

func setupApp(cmd *cobra.Command, config *runlocal.Config, spec *runlocal.AppSpec, customEnv []string, prepareEnvironment bool) (runlocal.App, []string, error) {
	ctx := cmd.Context()
	cfg := cmdctx.ConfigUsed(ctx)
	app, err := runlocal.NewApp(ctx, config, spec)
	if err != nil {
		return nil, nil, err
	}

	env := auth.ProcessEnv(cfg)
	if cfg.Profile != "" {
		env = append(env, "DATABRICKS_CONFIG_PROFILE="+cfg.Profile)
	}

	appEnv, err := spec.LoadEnvVars(ctx, customEnv)
	if err != nil {
		return app, nil, err
	}
	env = append(env, appEnv...)

	if prepareEnvironment {
		err := app.PrepareEnvironment()
		if err != nil {
			return app, nil, err
		}
	}

	return app, env, nil
}

func startAppProcess(cmd *cobra.Command, config *runlocal.Config, app runlocal.App, env []string, debug bool) (*exec.Cmd, error) {
	ctx := cmd.Context()
	specCommand, err := app.GetCommand(debug)
	if err != nil {
		return nil, err
	}

	cmdio.LogString(ctx, "Running command: "+strings.Join(specCommand, " "))
	appCmd := exec.Command(specCommand[0], specCommand[1:]...)
	appCmd.Stdin = cmd.InOrStdin()
	appCmd.Stdout = cmd.OutOrStdout()
	appCmd.Stderr = cmd.ErrOrStderr()

	var appCmdEnv []string
	appEnvs := runlocal.GetBaseEnvVars(config)
	for _, envVar := range appEnvs {
		appCmdEnv = append(appCmdEnv, envVar.String())
	}
	appCmdEnv = append(appCmdEnv, env...)
	appCmd.Env = appCmdEnv
	appCmd.Dir = config.AppPath

	err = appCmd.Start()
	if err != nil {
		return nil, err
	}

	return appCmd, nil
}

func setupProxy(ctx context.Context, cmd *cobra.Command, config *runlocal.Config, w *databricks.WorkspaceClient, port int, debug bool) error {
	proxy, err := appproxy.New(ctx, config.AppURL)
	if err != nil {
		return err
	}

	me, err := w.CurrentUser.Me(ctx)
	if err != nil {
		return err
	}

	for key, value := range runlocal.GetXHeaders(me) {
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
		cmdio.LogString(ctx, "To debug your app, attach a debugger to port "+config.DebugPort)
	}

	return nil
}

// SIGTERM (not supported on Windows) and SIGINT (Ctrl+C, supported cross-platform)
// are caught to enable graceful shutdown of the app process.
func handleGracefulShutdown(appCmd *exec.Cmd) error {
	done := make(chan error, 1)
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)

	go func() {
		done <- appCmd.Wait()
	}()

	select {
	case err := <-done:
		return err
	case <-sigChan:
		if err := appCmd.Process.Signal(os.Interrupt); err != nil {
			return fmt.Errorf("failed to send interrupt signal: %w", err)
		}

		select {
		case err := <-done:
			return err
		case <-time.After(SHUTDOWN_TIMEOUT):
			if err := appCmd.Process.Kill(); err != nil {
				return fmt.Errorf("failed to kill process: %w", err)
			}
			return errors.New("process killed after timeout")
		}
	}
}

func newDevLocal() *cobra.Command {
	var (
		port               int
		debug              bool
		prepareEnvironment bool
		entryPoint         string
		customEnv          []string
		debugPort          string
		appPort            int
		appName            string
		injectRemoteEnv    bool
	)

	cmd := &cobra.Command{}

	cmd.Use = "dev-local"
	cmd.Short = `Run an app locally with development environment`
	cmd.Long = `Run an app locally with development environment.

	  This command starts an app locally and optionally fetches environment variables from a deployed app.`

	cmd.Flags().IntVar(&port, "port", 8001, "Port on which to run the app app proxy")
	cmd.Flags().IntVar(&appPort, "app-port", runlocal.DEFAULT_PORT, "Port on which to run the app")
	cmd.Flags().BoolVar(&debug, "debug", false, "Enable debug mode")
	cmd.Flags().BoolVar(&prepareEnvironment, "prepare-environment", false, "Prepares the environment for running the app. Requires 'uv' to be installed.")
	cmd.Flags().StringSliceVar(&customEnv, "env", nil, "Set environment variables")
	cmd.Flags().StringVar(&entryPoint, "entry-point", "", "Specify the custom entry point with configuration (.yml file) for the app. Defaults to app.yml")
	cmd.Flags().StringVar(&debugPort, "debug-port", "", "Port on which to run the debugger")
	cmd.Flags().StringVar(&appName, "app-name", "", "Name of the deployed app to fetch environment variables from")
	cmd.Flags().BoolVar(&injectRemoteEnv, "inject-remote-env", true, "Inject environment variables from the deployed app (requires --app-name)")
	cmd.PreRunE = root.MustWorkspaceClient

	cmd.RunE = func(cmd *cobra.Command, args []string) error {
		ctx := cmd.Context()
		w := cmdctx.WorkspaceClient(ctx)

		config, spec, err := setupWorkspaceAndConfig(cmd, entryPoint, appPort)
		if err != nil {
			return err
		}

		if debugPort != "" {
			config.DebugPort = debugPort
		}

		var remoteEnv []string
		if appName != "" && injectRemoteEnv {
			fetcher := appenv.NewEnvFetcher(w)
			result, err := fetcher.Fetch(ctx, appName)
			if err != nil {
				return fmt.Errorf("failed to fetch remote environment variables: %w", err)
			}
			cmdio.LogString(ctx, fmt.Sprintf("Fetched %d environment variables from deployed app", len(result.EnvVars)))

			resolver := appenv.NewSecretResolver(w)
			remoteEnv = resolver.Resolve(ctx, result.EnvVars, result.Resources, spec)

			cmdio.LogString(ctx, "Remote environment variables:")
			for _, envVar := range remoteEnv {
				cmdio.LogString(ctx, "  "+envVar)
			}
		}

		mergedEnv := append(remoteEnv, customEnv...)

		app, env, err := setupApp(cmd, config, spec, mergedEnv, prepareEnvironment)
		if err != nil {
			return err
		}

		appCmd, err := startAppProcess(cmd, config, app, env, debug)
		if err != nil {
			return err
		}

		err = setupProxy(ctx, cmd, config, w, port, debug)
		if err != nil {
			return err
		}

		return handleGracefulShutdown(appCmd)
	}

	cmd.ValidArgsFunction = cobra.NoFileCompletions

	return cmd
}

func init() {
	cmdOverrides = append(cmdOverrides, func(cmd *cobra.Command) {
		cmd.AddCommand(newDevLocal())
	})
}
