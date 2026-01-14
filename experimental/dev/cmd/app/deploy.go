package app

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"sync"

	"github.com/databricks/cli/bundle"
	"github.com/databricks/cli/bundle/resources"
	"github.com/databricks/cli/bundle/run"
	"github.com/databricks/cli/cmd/bundle/utils"
	"github.com/databricks/cli/cmd/root"
	"github.com/databricks/cli/libs/cmdio"
	"github.com/databricks/cli/libs/log"
	"github.com/spf13/cobra"
)

func newDeployCmd() *cobra.Command {
	var (
		force     bool
		skipBuild bool
	)

	cmd := &cobra.Command{
		Use:   "deploy",
		Short: "Build, deploy the AppKit application and run it",
		Long: `Build, deploy the AppKit application and run it.

This command runs a deployment pipeline:
1. Builds the frontend (npm run build)
2. Deploys the bundle to the workspace
3. Runs the app

Examples:
  # Deploy to default target
  databricks experimental dev app deploy

  # Deploy to a specific target
  databricks experimental dev app deploy --target prod

  # Skip frontend build (if already built)
  databricks experimental dev app deploy --skip-build

  # Force deploy (override git branch validation)
  databricks experimental dev app deploy --force

  # Set bundle variables
  databricks experimental dev app deploy --var="warehouse_id=abc123"`,
		Args: root.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runDeploy(cmd, force, skipBuild)
		},
	}

	cmd.Flags().StringP("target", "t", "", "Deployment target (e.g., dev, prod)")
	cmd.Flags().BoolVar(&force, "force", false, "Force-override Git branch validation")
	cmd.Flags().BoolVar(&skipBuild, "skip-build", false, "Skip npm build step")
	cmd.Flags().StringSlice("var", []string{}, `Set values for variables defined in bundle config. Example: --var="key=value"`)

	return cmd
}

func runDeploy(cmd *cobra.Command, force, skipBuild bool) error {
	ctx := cmd.Context()

	// Check for bundle configuration
	if _, err := os.Stat("databricks.yml"); os.IsNotExist(err) {
		return errors.New("no databricks.yml found; run this command from a bundle directory")
	}

	// Step 1: Build frontend (unless skipped)
	if !skipBuild {
		if err := runNpmTypegen(ctx); err != nil {
			return err
		}
		if err := runNpmBuild(ctx); err != nil {
			return err
		}
	}

	// Step 2: Deploy bundle
	cmdio.LogString(ctx, "Deploying bundle...")
	b, err := utils.ProcessBundle(cmd, utils.ProcessOptions{
		InitFunc: func(b *bundle.Bundle) {
			b.Config.Bundle.Force = force
		},
		AlwaysPull:   true,
		FastValidate: true,
		Build:        true,
		Deploy:       true,
	})
	if err != nil {
		return fmt.Errorf("deploy failed: %w", err)
	}
	log.Infof(ctx, "Deploy completed")

	// Step 3: Detect and run app
	appKey, err := detectApp(b)
	if err != nil {
		return err
	}

	log.Infof(ctx, "Running app: %s", appKey)
	if err := runApp(ctx, b, appKey); err != nil {
		cmdio.LogString(ctx, "✔ Deployment succeeded, but failed to start app")
		return fmt.Errorf("failed to run app: %w", err)
	}

	cmdio.LogString(ctx, "✔ Deployment complete!")
	return nil
}

// syncBuffer is a thread-safe buffer for capturing command output.
type syncBuffer struct {
	mu  sync.Mutex
	buf bytes.Buffer
}

func (b *syncBuffer) Write(p []byte) (n int, err error) {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.buf.Write(p)
}

func (b *syncBuffer) String() string {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.buf.String()
}

// runNpmTypegen runs npm run typegen in the current directory.
func runNpmTypegen(ctx context.Context) error {
	if _, err := exec.LookPath("npm"); err != nil {
		return errors.New("npm not found: please install Node.js")
	}

	var output syncBuffer

	err := RunWithSpinnerCtx(ctx, "Generating types...", func() error {
		cmd := exec.CommandContext(ctx, "npm", "run", "typegen")
		cmd.Stdout = &output
		cmd.Stderr = &output
		return cmd.Run()
	})
	if err != nil {
		out := output.String()
		if out != "" {
			return fmt.Errorf("typegen failed:\n%s", out)
		}
		return fmt.Errorf("typegen failed: %w", err)
	}
	return nil
}

// runNpmBuild runs npm run build in the current directory.
func runNpmBuild(ctx context.Context) error {
	if _, err := exec.LookPath("npm"); err != nil {
		return errors.New("npm not found: please install Node.js")
	}

	var output syncBuffer

	err := RunWithSpinnerCtx(ctx, "Building frontend...", func() error {
		cmd := exec.CommandContext(ctx, "npm", "run", "build")
		cmd.Stdout = &output
		cmd.Stderr = &output
		return cmd.Run()
	})
	if err != nil {
		out := output.String()
		if out != "" {
			return fmt.Errorf("build failed:\n%s", out)
		}
		return fmt.Errorf("build failed: %w", err)
	}
	return nil
}

// detectApp finds the single app in the bundle configuration.
func detectApp(b *bundle.Bundle) (string, error) {
	apps := b.Config.Resources.Apps

	if len(apps) == 0 {
		return "", errors.New("no apps found in bundle configuration")
	}

	if len(apps) > 1 {
		return "", errors.New("multiple apps found in bundle, cannot auto-detect")
	}

	for key := range apps {
		return key, nil
	}

	return "", errors.New("unexpected error detecting app")
}

// runApp runs the specified app using the runner interface.
func runApp(ctx context.Context, b *bundle.Bundle, appKey string) error {
	ref, err := resources.Lookup(b, appKey, run.IsRunnable)
	if err != nil {
		return fmt.Errorf("failed to lookup app: %w", err)
	}

	runner, err := run.ToRunner(b, ref)
	if err != nil {
		return fmt.Errorf("failed to create runner: %w", err)
	}

	output, err := runner.Run(ctx, &run.Options{})
	if err != nil {
		return fmt.Errorf("failed to run app: %w", err)
	}

	if output != nil {
		resultString, err := output.String()
		if err != nil {
			return err
		}
		log.Infof(ctx, "App output: %s", resultString)
	}

	return nil
}
