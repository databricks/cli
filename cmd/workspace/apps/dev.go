package apps

import (
	"context"
	_ "embed"
	"errors"
	"fmt"
	"net"
	"os"
	"os/exec"
	"os/signal"
	"syscall"
	"time"

	"github.com/databricks/cli/cmd/root"
	"github.com/databricks/cli/libs/cmdctx"
	"github.com/databricks/cli/libs/cmdio"
	"github.com/spf13/cobra"
)

//go:embed vite-server.js
var viteServerScript []byte

// TODO: Handle multiple ports
const vitePort = "5173"

func isViteReady() bool {
	conn, err := net.DialTimeout("tcp", "localhost:"+vitePort, 100*time.Millisecond)
	if err != nil {
		return false
	}
	conn.Close()
	return true
}

func createViteServerScript() (string, error) {
	tmpFile, err := os.CreateTemp("", "vite-server-*.js")
	if err != nil {
		return "", fmt.Errorf("failed to create temporary file for vite-server.js: %w", err)
	}
	defer tmpFile.Close()

	if _, err := tmpFile.Write(viteServerScript); err != nil {
		os.Remove(tmpFile.Name())
		return "", fmt.Errorf("failed to write vite-server.js: %w", err)
	}

	return tmpFile.Name(), nil
}

func startViteDevServer(ctx context.Context, appURL string) (*exec.Cmd, string, chan error, error) {
	scriptPath, err := createViteServerScript()
	if err != nil {
		return nil, "", nil, err
	}

	// Arguments: node vite-server.js <appURL>
	viteCmd := exec.Command("node", scriptPath, appURL)
	viteCmd.Stdout = os.Stdout
	viteCmd.Stderr = os.Stderr

	err = viteCmd.Start()
	if err != nil {
		os.Remove(scriptPath)
		return nil, "", nil, fmt.Errorf("failed to start Vite server: %w", err)
	}

	cmdio.LogString(ctx, "🚀 Starting Vite development server...")

	viteErr := make(chan error, 1)
	go func() {
		if err := viteCmd.Wait(); err != nil {
			viteErr <- fmt.Errorf("Vite server exited with error: %w", err)
		} else {
			viteErr <- errors.New("Vite server exited unexpectedly")
		}
	}()

	maxAttempts := 50

	for range maxAttempts {
		select {
		case err := <-viteErr:
			os.Remove(scriptPath)
			return nil, "", nil, err
		default:
			if isViteReady() {
				return viteCmd, scriptPath, viteErr, nil
			}
			time.Sleep(100 * time.Millisecond)
		}
	}

	viteCmd.Process.Kill()
	os.Remove(scriptPath)
	return nil, "", nil, errors.New("timeout waiting for Vite server to be ready")
}

func newRunDevCommand() *cobra.Command {
	var (
		appName    string
		clientPath string
	)

	cmd := &cobra.Command{}

	cmd.Use = "dev-remote"
	cmd.Hidden = true
	cmd.Short = `Run Databricks app locally with WebSocket bridge to remote server.`
	cmd.Long = `Run Databricks app locally with WebSocket bridge to remote server.

  Starts a local development server and establishes a WebSocket bridge
  to the remote Databricks app for development.
    `

	cmd.PreRunE = root.MustWorkspaceClient

	cmd.Flags().StringVar(&appName, "app-name", "", "Name of the app to connect to (required)")
	cmd.Flags().StringVar(&clientPath, "client-path", "./client", "Path to the Vite client directory")

	cmd.RunE = func(cmd *cobra.Command, args []string) error {
		ctx := cmd.Context()
		w := cmdctx.WorkspaceClient(ctx)

		if appName == "" {
			return errors.New("app name is required (use --app-name)")
		}

		if _, err := os.Stat(clientPath); os.IsNotExist(err) {
			return fmt.Errorf("client directory not found: %s", clientPath)
		}

		bridge := NewViteBridge(ctx, w, appName)

		appDomain, err := bridge.GetAppDomain()
		if err != nil {
			return fmt.Errorf("failed to get app domain: %w", err)
		}

		viteCmd, scriptPath, viteErr, err := startViteDevServer(ctx, appDomain.String())
		if err != nil {
			return err
		}

		defer func() {
			time.Sleep(100 * time.Millisecond)
			os.Remove(scriptPath)
		}()

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
				viteCmd.Process.Signal(os.Interrupt)
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
					viteCmd.Process.Kill()
				}
				<-viteErr
			}
			return nil
		}
	}

	cmd.ValidArgsFunction = cobra.NoFileCompletions

	return cmd
}

func init() {
	cmdOverrides = append(cmdOverrides, func(cmd *cobra.Command) {
		cmd.AddCommand(newRunDevCommand())
	})
}
