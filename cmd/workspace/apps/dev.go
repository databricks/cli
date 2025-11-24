package apps

import (
	"bytes"
	"context"
	_ "embed"
	"errors"
	"fmt"
	"net"
	"os"
	"os/exec"
	"os/signal"
	"strconv"
	"syscall"
	"time"

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

func newRunDevCommand() *cobra.Command {
	var (
		appName    string
		clientPath string
		port       int
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
	cmd.Flags().IntVar(&port, "port", vitePort, "Port to run the Vite server on")

	cmd.RunE = func(cmd *cobra.Command, args []string) error {
		ctx := cmd.Context()
		w := cmdctx.WorkspaceClient(ctx)

		if appName == "" {
			return errors.New("app name is required (use --app-name)")
		}

		if _, err := os.Stat(clientPath); os.IsNotExist(err) {
			return fmt.Errorf("client directory not found: %s", clientPath)
		}

		bridge := NewViteBridge(ctx, w, appName, port)

		appDomain, err := bridge.GetAppDomain()
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
	}

	cmd.ValidArgsFunction = cobra.NoFileCompletions

	return cmd
}

func init() {
	cmdOverrides = append(cmdOverrides, func(cmd *cobra.Command) {
		cmd.AddCommand(newRunDevCommand())
	})
}
