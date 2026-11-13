package sandbox

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"net"
	"os"
	"runtime"
	"time"

	sshtunnel "github.com/databricks/cli/experimental/ssh/sandbox"
	"github.com/spf13/cobra"
)

func newServerCommand() *cobra.Command {
	var config, sshd string
	var port int
	var idleTimeout time.Duration
	cmd := &cobra.Command{
		Use: "server", Short: "Run the sandbox WebSocket SSH relay", Hidden: true,
		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			if port != 0 && (port < 1024 || port > 65535) {
				return errors.New("port must be zero or between 1024 and 65535")
			}
			if idleTimeout <= 0 {
				return errors.New("idle timeout must be positive")
			}
			listener, err := net.Listen("tcp4", fmt.Sprintf("0.0.0.0:%d", port))
			if err != nil {
				return fmt.Errorf("listen for SSH WebSockets: %w", err)
			}
			output, err := openServerOutput(cmd.OutOrStdout())
			if err != nil {
				listener.Close()
				return fmt.Errorf("open command output: %w", err)
			}
			defer output.Close()
			return sshtunnel.Serve(cmd.Context(), listener, sshtunnel.ServerOptions{
				SSHD: sshd, Config: config, IdleTimeout: idleTimeout, OutputHeartbeat: 30 * time.Second,
			}, output)
		},
	}
	cmd.Flags().StringVar(&config, "sshd-config", "", "Prepared sshd configuration file")
	cmd.MarkFlagRequired("sshd-config")
	cmd.Flags().StringVar(&sshd, "sshd", "/usr/sbin/sshd", "OpenSSH server executable")
	cmd.Flags().IntVar(&port, "port", 40000, "HTTP/WebSocket listener port (0 selects a free port)")
	cmd.Flags().DurationVar(&idleTimeout, "idle-timeout", time.Minute, "Stop after this duration without an SSH connection")
	return cmd
}

func openServerOutput(output io.Writer) (io.WriteCloser, error) {
	if _, ok := output.(*bytes.Buffer); ok || output == io.Discard {
		return serverOutputWriter{output}, nil
	}
	file, ok := output.(*os.File)
	if !ok || runtime.GOOS != "linux" {
		return nil, errors.New("server output must be a Linux command-execution pipe")
	}
	info, err := file.Stat()
	if err != nil {
		return nil, err
	}
	if info.Mode()&os.ModeNamedPipe == 0 {
		return nil, errors.New("server output must be a Linux command-execution pipe")
	}
	// Reopening inherited stdout registers its pipe with Go's poller, so
	// Close interrupts blocked writes without changing the inherited fd.
	return os.OpenFile(fmt.Sprintf("/proc/self/fd/%d", file.Fd()), os.O_WRONLY, 0)
}

type serverOutputWriter struct{ io.Writer }

func (serverOutputWriter) Close() error { return nil }
