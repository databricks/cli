package app

import (
	"bytes"
	"context"
	_ "embed"
	"errors"
	"fmt"
	"net"
	"net/url"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"strconv"
	"syscall"
	"time"

	"github.com/databricks/cli/bundle/config"
	"github.com/databricks/cli/cmd/root"
	"github.com/databricks/cli/libs/cmdctx"
	"github.com/databricks/cli/libs/cmdio"
	"github.com/spf13/cobra"
)

//go:embed vite-server.js
var viteServerScript []byte

const (
	vitePort               = 5173
	viteReadyCheckInterval = 100 * time.Millisecond
	viteReadyMaxAttempts   = 50
)

func isViteReady(port int) bool {
	conn, err := net.DialTimeout("tcp", "localhost:"+strconv.Itoa(port), viteReadyCheckInterval)
	if err != nil {
		return false
	}
	conn.Close()
	return true
}

// detectAppNameFromBundle tries to extract the app name from a databricks.yml bundle config.
// Returns the app name if found, or empty string if no bundle or no apps found.
func detectAppNameFromBundle() string {
	const bundleFile = "databricks.yml"

	// Check if databricks.yml exists
	if _, err := os.Stat(bundleFile); os.IsNotExist(err) {
		return ""
	}

	// Get current working directory
	cwd, err := os.Getwd()
	if err != nil {
		return ""
	}

	// Load the bundle configuration directly
	rootConfig, diags := config.Load(filepath.Join(cwd, bundleFile))
	if diags.HasError() {
		return ""
	}

	// Check for apps in the bundle
	apps := rootConfig.Resources.Apps
	if len(apps) == 0 {
		return ""
	}

	// If there's exactly one app, return its name
	if len(apps) == 1 {
		for _, app := range apps {
			return app.Name
		}
	}

	// Multiple apps - can't auto-detect
	return ""
}

func startViteDevServer(ctx context.Context, appURL string, port int) (*exec.Cmd, chan error, error) {
	// Pass script through stdin, and pass arguments in order <appURL> <port (optional)>
	viteCmd := exec.Command("node", "-", appURL, strconv.Itoa(port))
	viteCmd.Stdin = bytes.NewReader(viteServerScript)
	viteCmd.Stdout = os.Stdout
	viteCmd.Stderr = os.Stderr

	err := viteCmd.Start()
	if err != nil {
		return nil, nil, fmt.Errorf("failed to start Vite server: %w", err)
	}

	cmdio.LogString(ctx, fmt.Sprintf("🚀 Starting Vite development server on port %d...", port))

	viteErr := make(chan error, 1)
	go func() {
		if err := viteCmd.Wait(); err != nil {
			viteErr <- fmt.Errorf("vite server exited with error: %w", err)
		} else {
			viteErr <- errors.New("vite server exited unexpectedly")
		}
	}()

	for range viteReadyMaxAttempts {
		select {
		case err := <-viteErr:
			return nil, nil, err
		default:
			if isViteReady(port) {
				return viteCmd, viteErr, nil
			}
			time.Sleep(viteReadyCheckInterval)
		}
	}

	_ = viteCmd.Process.Kill()
	return nil, nil, errors.New("timeout waiting for Vite server to be ready")
}

func newDevRemoteCmd() *cobra.Command {
	var (
		appName    string
		clientPath string
		port       int
	)

	cmd := &cobra.Command{
		Use:   "dev-remote",
		Short: "Run AppKit app locally with WebSocket bridge to remote server",
		Long: `Run AppKit app locally with WebSocket bridge to remote server.

Starts a local Vite development server and establishes a WebSocket bridge
to the remote Databricks app for development with hot module replacement.

Examples:
  # Interactive mode - select app from picker
  databricks experimental dev app dev-remote

  # Start development server for a specific app
  databricks experimental dev app dev-remote --name my-app

  # Use a custom client path
  databricks experimental dev app dev-remote --name my-app --client-path ./frontend

  # Use a custom port
  databricks experimental dev app dev-remote --name my-app --port 3000`,
		Args:    root.NoArgs,
		PreRunE: root.MustWorkspaceClient,
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := cmd.Context()

			// Validate client path early (before any network calls)
			if _, err := os.Stat(clientPath); os.IsNotExist(err) {
				return fmt.Errorf("client directory not found: %s", clientPath)
			}

			// Check if port is already in use
			if isViteReady(port) {
				return fmt.Errorf("port %d is already in use; try --port <other>", port)
			}

			w := cmdctx.WorkspaceClient(ctx)

			// Resolve app name with priority: flag > bundle config > prompt
			if appName == "" {
				// Try to detect from bundle config
				appName = detectAppNameFromBundle()
				if appName != "" {
					cmdio.LogString(ctx, fmt.Sprintf("Using app '%s' from bundle configuration", appName))
				}
			}

			if appName == "" {
				// Fall back to interactive prompt
				selected, err := PromptForAppSelection(ctx, "Select an app to connect to")
				if err != nil {
					return err
				}
				appName = selected
			}

			bridge := NewViteBridge(ctx, w, appName, port)

			// Validate app exists and get domain before starting Vite
			var appDomain *url.URL
			err := RunWithSpinnerCtx(ctx, "Connecting to app...", func() error {
				var domainErr error
				appDomain, domainErr = bridge.GetAppDomain()
				return domainErr
			})
			if err != nil {
				return fmt.Errorf("failed to get app domain: %w", err)
			}

			viteCmd, viteErr, err := startViteDevServer(ctx, appDomain.String(), port)
			if err != nil {
				return err
			}

			done := make(chan error, 1)
			sigChan := make(chan os.Signal, 1)
			signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)

			go func() {
				done <- bridge.Start()
			}()

			select {
			case err := <-viteErr:
				bridge.Stop()
				<-done
				return err
			case err := <-done:
				cmdio.LogString(ctx, "Bridge stopped")
				if viteCmd.Process != nil {
					_ = viteCmd.Process.Signal(os.Interrupt)
					<-viteErr
				}
				return err
			case <-sigChan:
				cmdio.LogString(ctx, "\n🛑 Shutting down...")
				bridge.Stop()
				<-done
				if viteCmd.Process != nil {
					if err := viteCmd.Process.Signal(os.Interrupt); err != nil {
						cmdio.LogString(ctx, fmt.Sprintf("Failed to interrupt Vite: %v", err))
						_ = viteCmd.Process.Kill()
					}
					<-viteErr
				}
				return nil
			}
		},
	}

	cmd.Flags().StringVar(&appName, "name", "", "Name of the app to connect to (prompts if not provided)")
	cmd.Flags().StringVar(&clientPath, "client-path", "./client", "Path to the Vite client directory")
	cmd.Flags().IntVar(&port, "port", vitePort, "Port to run the Vite server on")

	return cmd
}
