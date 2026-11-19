package sandbox

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"net"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/spf13/cobra"
)

// NewServerCommand returns the internal sandbox SSH server command.
func NewServerCommand() *cobra.Command {
	var config, sshd string
	var port int
	var connectTimeout time.Duration
	cmd := &cobra.Command{
		Use: "server", Short: "Run the sandbox WebSocket SSH relay", Hidden: true,
		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			if port != 0 && !validForwardPort(port) {
				return errors.New("port must be zero or between 10000 and 65535, excluding 10101, 18767, and 19090")
			}
			if connectTimeout <= 0 {
				return errors.New("connection timeout must be positive")
			}
			listener, err := net.Listen("tcp4", fmt.Sprintf("0.0.0.0:%d", port))
			if err != nil {
				return fmt.Errorf("listen for SSH WebSockets: %w", err)
			}
			if !validForwardPort(listener.Addr().(*net.TCPAddr).Port) {
				listener.Close()
				return errors.New("selected listener port is not supported by sandbox forwarding; retry the command")
			}
			ctx, stop := signal.NotifyContext(cmd.Context(), syscall.SIGTERM, os.Interrupt)
			defer stop()
			output, err := openServerOutput(cmd.OutOrStdout())
			if err != nil {
				listener.Close()
				return fmt.Errorf("open command output: %w", err)
			}
			defer output.Close()
			return Serve(ctx, listener, ServerOptions{
				SSHD: sshd, Config: config, ConnectTimeout: connectTimeout, OutputHeartbeat: 30 * time.Second,
			}, output)
		},
	}
	cmd.Flags().StringVar(&config, "sshd-config", "", "Prepared sshd configuration file")
	cmd.MarkFlagRequired("sshd-config")
	cmd.Flags().StringVar(&sshd, "sshd", "/usr/sbin/sshd", "OpenSSH server executable")
	cmd.Flags().IntVar(&port, "port", 40000, "HTTP/WebSocket listener port (0 selects a free port)")
	cmd.Flags().DurationVar(&connectTimeout, "connect-timeout", 2*time.Minute, "Stop if no SSH connection arrives within this duration after readiness")
	return cmd
}

func validForwardPort(port int) bool {
	return port >= 10000 && port <= 65535 && port != 10101 && port != 18767 && port != 19090
}

func openServerOutput(output io.Writer) (io.WriteCloser, error) {
	if _, ok := output.(*bytes.Buffer); ok || output == io.Discard {
		return serverOutputWriter{output}, nil
	}
	file, ok := output.(*os.File)
	if !ok {
		return nil, errors.New("server output must be a Linux command-execution pipe")
	}
	info, err := file.Stat()
	if err != nil {
		return nil, err
	}
	if info.Mode()&os.ModeNamedPipe == 0 {
		return nil, errors.New("server output must be a Linux command-execution pipe")
	}
	return duplicateServerOutput(file)
}

type serverOutputWriter struct{ io.Writer }

func (serverOutputWriter) Close() error { return nil }
