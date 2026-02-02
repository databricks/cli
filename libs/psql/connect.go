package psql

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"strings"
	"time"

	"github.com/databricks/cli/libs/cmdio"
	"github.com/databricks/cli/libs/execv"
)

// RetryConfig holds configuration for connection retry behavior.
type RetryConfig struct {
	MaxRetries    int
	InitialDelay  time.Duration
	MaxDelay      time.Duration
	BackoffFactor float64
}

// ConnectOptions contains the parameters needed to connect to a Postgres instance.
type ConnectOptions struct {
	Host            string
	Username        string
	Password        string
	DefaultDatabase string
	ExtraArgs       []string
}

// Connect launches psql with the given options and retry configuration.
func Connect(ctx context.Context, opts ConnectOptions, retryConfig RetryConfig) error {
	args, env := buildCommand(opts)

	// If retries are disabled, go directly to interactive session
	if retryConfig.MaxRetries <= 0 {
		return execv.Execv(execv.Options{
			Args: args,
			Env:  env,
		})
	}

	return connectWithRetry(ctx, args, env, retryConfig)
}

func buildCommand(opts ConnectOptions) (args, env []string) {
	// Check if database name and port are already specified in extra arguments
	hasDbName := false
	hasPort := false
	for _, arg := range opts.ExtraArgs {
		if arg == "-d" || strings.HasPrefix(arg, "--dbname=") {
			hasDbName = true
		}
		if arg == "-p" || strings.HasPrefix(arg, "--port=") {
			hasPort = true
		}
	}

	args = []string{
		"psql",
		"--host=" + opts.Host,
		"--username=" + opts.Username,
	}

	if !hasPort {
		args = append(args, "--port=5432")
	}

	if !hasDbName {
		args = append(args, "--dbname="+opts.DefaultDatabase)
	}

	args = append(args, opts.ExtraArgs...)

	env = append(os.Environ(),
		"PGPASSWORD="+opts.Password,
		"PGSSLMODE=require",
	)

	return args, env
}

func connectWithRetry(ctx context.Context, args, env []string, retryConfig RetryConfig) error {
	maxRetries := retryConfig.MaxRetries
	delay := retryConfig.InitialDelay

	var lastErr error
	for attempt := range maxRetries {
		if attempt > 0 {
			cmdio.LogString(ctx, fmt.Sprintf("Connection attempt %d/%d failed, retrying in %v...", attempt, maxRetries, delay))

			select {
			case <-ctx.Done():
				return ctx.Err()
			case <-time.After(delay):
			}

			delay = time.Duration(float64(delay) * retryConfig.BackoffFactor)
			if delay > retryConfig.MaxDelay {
				delay = retryConfig.MaxDelay
			}
		}

		if attempt > 0 {
			cmdio.LogString(ctx, fmt.Sprintf("Retrying connection (attempt %d/%d)...", attempt+1, maxRetries))
		}

		err := attemptConnection(ctx, args, env)
		if err == nil {
			return nil
		}

		lastErr = err

		if !errors.Is(err, errRetryable) {
			return err
		}

		if attempt < maxRetries {
			cmdio.LogString(ctx, fmt.Sprintf("Connection failed with retryable error: %v", err))
		}
	}

	return fmt.Errorf("failed to connect after %d attempts, last error: %w", maxRetries, lastErr)
}

func attemptConnection(ctx context.Context, args, env []string) error {
	cmd := exec.CommandContext(ctx, args[0], args[1:]...)
	cmd.Env = env
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout

	// Capture stderr to analyze error messages
	stderrPipe, err := cmd.StderrPipe()
	if err != nil {
		return fmt.Errorf("failed to create stderr pipe: %w", err)
	}

	if err := cmd.Start(); err != nil {
		return err
	}

	// Capture stderr for error analysis.
	// Note: this could be problematic if stderr output is very large.
	stderrBytes, _ := io.ReadAll(io.TeeReader(stderrPipe, os.Stderr))
	stderrOutput := string(stderrBytes)

	err = cmd.Wait()
	if err != nil {
		var exitError *exec.ExitError
		if errors.As(err, &exitError) {
			// psql returns exit code 2 for fatal errors
			if exitError.ExitCode() == 2 {
				connErr := fmt.Errorf("connection failed: psql exited with code %d", exitError.ExitCode())
				if isNonRetryableError(stderrOutput) {
					return connErr
				}
				return &retryableError{err: connErr}
			}
		}
	}

	return err
}
