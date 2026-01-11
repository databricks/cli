package appkit

import (
	"context"
	"crypto/tls"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path"
	"slices"
	"strings"
	"time"

	"github.com/databricks/cli/cmd/root"
	"github.com/databricks/cli/libs/apps/logstream"
	"github.com/databricks/cli/libs/cmdctx"
	"github.com/databricks/cli/libs/cmdgroup"
	"github.com/databricks/cli/libs/cmdio"
	"github.com/databricks/cli/libs/flags"
	"github.com/databricks/cli/libs/log"
	"github.com/databricks/databricks-sdk-go/config"
	"github.com/databricks/databricks-sdk-go/service/apps"
	"github.com/gorilla/websocket"
	"github.com/spf13/cobra"
)

const (
	defaultTailLines        = 200
	defaultPrefetchWindow   = 2 * time.Second
	defaultHandshakeTimeout = 30 * time.Second
)

var allowedSources = []string{"APP", "SYSTEM"}

func newLogsCmd() *cobra.Command {
	var (
		name          string
		tailLines     int
		follow        bool
		outputPath    string
		streamTimeout time.Duration
		searchTerm    string
		sourceFilters []string
	)

	cmd := &cobra.Command{
		Use:   "logs",
		Short: "Stream logs for an AppKit application",
		Long: `Stream stdout/stderr logs for an AppKit application.

By default the command fetches the most recent logs (up to --tail-lines, default 200) and exits.
Use --follow to continue streaming logs until cancelled, optionally bounding the duration with --timeout.
Server-side filtering is available through --search and client-side filtering via --source APP|SYSTEM.
Use --output-file to mirror the stream to a local file.

Examples:
  # Interactive mode - select app from picker
  databricks experimental appkit logs

  # Fetch the last 50 log lines
  databricks experimental appkit logs --name my-app --tail-lines 50

  # Follow logs until interrupted, searching for "ERROR" messages from app sources only
  databricks experimental appkit logs --name my-app --follow --search ERROR --source APP

  # Mirror streamed logs to a local file while following for up to 5 minutes
  databricks experimental appkit logs --name my-app --follow --timeout 5m --output-file /tmp/my-app.log`,
		Args:    root.NoArgs,
		PreRunE: root.MustWorkspaceClient,
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := cmd.Context()

			if tailLines < 0 {
				return errors.New("--tail-lines cannot be negative")
			}

			if follow && streamTimeout > 0 {
				var cancel context.CancelFunc
				ctx, cancel = context.WithTimeout(ctx, streamTimeout)
				defer cancel()
			}

			// Prompt for app name if not provided
			if name == "" {
				selected, err := PromptForAppSelection(ctx, "Select an app to view logs")
				if err != nil {
					return err
				}
				name = selected
			}

			w := cmdctx.WorkspaceClient(ctx)
			app, err := w.Apps.Get(ctx, apps.GetAppRequest{Name: name})
			if err != nil {
				return err
			}
			if app.Url == "" {
				return fmt.Errorf("app %s does not have a public URL; deploy and start it before streaming logs", name)
			}

			wsURL, err := buildLogsURL(app.Url)
			if err != nil {
				return err
			}

			cfg := cmdctx.ConfigUsed(ctx)
			if cfg == nil {
				return errors.New("missing workspace configuration")
			}

			tokenSource := cfg.GetTokenSource()
			if tokenSource == nil {
				return errors.New("configuration does not support OAuth tokens")
			}

			initialToken, err := tokenSource.Token(ctx)
			if err != nil {
				return err
			}

			tokenProvider := func(ctx context.Context) (string, error) {
				tok, err := tokenSource.Token(ctx)
				if err != nil {
					return "", err
				}
				return tok.AccessToken, nil
			}

			appStatusChecker := func(ctx context.Context) error {
				app, err := w.Apps.Get(ctx, apps.GetAppRequest{Name: name})
				if err != nil {
					return err
				}
				if app.ComputeStatus == nil {
					return errors.New("app status unavailable")
				}
				switch app.ComputeStatus.State {
				case apps.ComputeStateStopped, apps.ComputeStateDeleting, apps.ComputeStateError:
					return fmt.Errorf("app is %s", app.ComputeStatus.State)
				default:
					return nil
				}
			}

			writer := cmd.OutOrStdout()
			var file *os.File
			if outputPath != "" {
				file, err = os.OpenFile(outputPath, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, 0o600)
				if err != nil {
					return err
				}
				defer file.Close()
				writer = io.MultiWriter(writer, file)
			}

			outputFormat := root.OutputType(cmd)
			colorizeLogs := outputPath == "" && outputFormat == flags.OutputText && cmdio.IsTTY(cmd.OutOrStdout())

			sourceMap, err := buildSourceFilter(sourceFilters)
			if err != nil {
				return err
			}

			log.Infof(ctx, "Streaming logs for %s (%s)", name, wsURL)
			return logstream.Run(ctx, logstream.Config{
				Dialer:           newLogStreamDialer(cfg),
				URL:              wsURL,
				Origin:           normalizeOrigin(app.Url),
				Token:            initialToken.AccessToken,
				TokenProvider:    tokenProvider,
				AppStatusChecker: appStatusChecker,
				Search:           searchTerm,
				Sources:          sourceMap,
				Tail:             tailLines,
				Follow:           follow,
				Prefetch:         defaultPrefetchWindow,
				Writer:           writer,
				UserAgent:        "databricks-cli appkit logs",
				Colorize:         colorizeLogs,
				OutputFormat:     outputFormat,
			})
		},
	}

	streamGroup := cmdgroup.NewFlagGroup("Streaming")
	streamGroup.FlagSet().IntVar(&tailLines, "tail-lines", defaultTailLines, "Number of recent log lines to show before streaming. Set to 0 to show everything.")
	streamGroup.FlagSet().BoolVarP(&follow, "follow", "f", false, "Continue streaming logs until interrupted.")
	streamGroup.FlagSet().DurationVar(&streamTimeout, "timeout", 0, "Maximum time to stream when --follow is set. 0 disables the timeout.")

	filterGroup := cmdgroup.NewFlagGroup("Filtering")
	filterGroup.FlagSet().StringVar(&searchTerm, "search", "", "Send a search term to the log service before streaming.")
	filterGroup.FlagSet().StringSliceVar(&sourceFilters, "source", nil, "Restrict logs to APP and/or SYSTEM sources.")

	wrappedCmd := cmdgroup.NewCommandWithGroupFlag(cmd)
	wrappedCmd.AddFlagGroup(streamGroup)
	wrappedCmd.AddFlagGroup(filterGroup)

	cmd.Flags().StringVar(&name, "name", "", "Name of the app to view logs (prompts if not provided)")
	cmd.Flags().StringVar(&outputPath, "output-file", "", "Mirror log output to a local file.")

	return cmd
}

func buildLogsURL(appURL string) (string, error) {
	parsed, err := url.Parse(appURL)
	if err != nil {
		return "", fmt.Errorf("invalid app URL: %w", err)
	}
	parsed.Scheme = "wss"
	parsed.Path = path.Join(parsed.Path, "logs/stream")
	return parsed.String(), nil
}

func normalizeOrigin(appURL string) string {
	parsed, err := url.Parse(appURL)
	if err != nil {
		return appURL
	}
	parsed.Path = ""
	parsed.RawQuery = ""
	parsed.Fragment = ""
	return parsed.String()
}

func buildSourceFilter(sources []string) (map[string]struct{}, error) {
	if len(sources) == 0 {
		return nil, nil
	}

	sourceMap := make(map[string]struct{})
	for _, s := range sources {
		upper := strings.ToUpper(s)
		if !slices.Contains(allowedSources, upper) {
			return nil, fmt.Errorf("invalid source %q; allowed values: %v", s, allowedSources)
		}
		sourceMap[upper] = struct{}{}
	}
	return sourceMap, nil
}

func newLogStreamDialer(cfg *config.Config) *websocket.Dialer {
	tlsConfig := &tls.Config{
		MinVersion: tls.VersionTLS12,
	}
	if cfg.InsecureSkipVerify {
		tlsConfig.InsecureSkipVerify = true
	}

	return &websocket.Dialer{
		Proxy:            http.ProxyFromEnvironment,
		HandshakeTimeout: defaultHandshakeTimeout,
		TLSClientConfig:  tlsConfig,
	}
}
