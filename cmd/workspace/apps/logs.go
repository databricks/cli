package apps

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
	"github.com/databricks/cli/libs/cmdctx"
	"github.com/databricks/cli/libs/cmdgroup"
	"github.com/databricks/cli/libs/cmdio"
	"github.com/databricks/cli/libs/log"
	"github.com/databricks/cli/libs/logstream"
	"github.com/databricks/databricks-sdk-go/config"
	"github.com/databricks/databricks-sdk-go/service/apps"
	"github.com/gorilla/websocket"
	"github.com/spf13/cobra"
)

const (
	defaultTailLines      = 200
	defaultPrefetchWindow = 2 * time.Second
)

var allowedSources = []string{"APP", "SYSTEM"}

func newLogsCommand() *cobra.Command {
	var (
		tailLines     int
		follow        bool
		outputPath    string
		streamTimeout time.Duration
		searchTerm    string
		sourceFilters []string
	)

	cmd := &cobra.Command{
		Use:   "logs NAME",
		Short: "Show Databricks app logs",
		Long: `Stream stdout/stderr logs for a Databricks app via its log stream.

By default the command fetches the most recent logs (up to --tail-lines, default 200) and exits.
Use --follow to continue streaming logs until cancelled, optionally bounding the duration with --timeout.
Server-side filtering is available through --search (same semantics as the Databricks UI) and client-side filtering
via --source APP|SYSTEM. Use --output-file to mirror the stream to a local file (created with 0600 permissions).`,
		Example: `  # Fetch the last 50 log lines
  databricks apps logs my-app --tail-lines 50

  # Follow logs until interrupted, searching for "ERROR" messages from app sources only
  databricks apps logs my-app --follow --search ERROR --source APP

  # Mirror streamed logs to a local file while following for up to 5 minutes
  databricks apps logs my-app --follow --timeout 5m --output-file /tmp/my-app.log`,
		Args:    root.ExactArgs(1),
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

			name := args[0]
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

			var writer io.Writer = cmd.OutOrStdout()
			var file *os.File
			if outputPath != "" {
				file, err = os.OpenFile(outputPath, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, 0o600)
				if err != nil {
					return err
				}
				defer file.Close()
				writer = io.MultiWriter(writer, file)
			}
			colorizeLogs := outputPath == "" && cmdio.IsTTY(cmd.OutOrStdout())

			sourceMap, err := buildSourceFilter(sourceFilters)
			if err != nil {
				return err
			}

			log.Infof(ctx, "Streaming logs for %s (%s)", name, wsURL)
			return logstream.Run(ctx, logstream.Config{
				Dialer:        newLogStreamDialer(cfg),
				URL:           wsURL,
				Origin:        normalizeOrigin(app.Url),
				Token:         initialToken.AccessToken,
				TokenProvider: tokenProvider,
				Search:        searchTerm,
				Sources:       sourceMap,
				Tail:          tailLines,
				Follow:        follow,
				Prefetch:      defaultPrefetchWindow,
				Writer:        writer,
				UserAgent:     "databricks-cli apps logs",
				Colorize:      colorizeLogs,
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

	cmd.Flags().StringVar(&outputPath, "output-file", "", "Optional file path to write logs in addition to stdout.")

	return cmd
}

func buildLogsURL(appURL string) (string, error) {
	parsed, err := url.Parse(appURL)
	if err != nil {
		return "", err
	}

	switch strings.ToLower(parsed.Scheme) {
	case "https":
		parsed.Scheme = "wss"
	case "http":
		parsed.Scheme = "ws"
	case "wss", "ws":
	default:
		return "", fmt.Errorf("unsupported app URL scheme: %s", parsed.Scheme)
	}

	parsed.Path = path.Join(parsed.Path, "logz/stream")
	if !strings.HasPrefix(parsed.Path, "/") {
		parsed.Path = "/" + parsed.Path
	}

	return parsed.String(), nil
}

func normalizeOrigin(appURL string) string {
	parsed, err := url.Parse(appURL)
	if err != nil {
		return ""
	}
	switch strings.ToLower(parsed.Scheme) {
	case "http", "https":
		return parsed.Scheme + "://" + parsed.Host
	case "ws":
		parsed.Scheme = "http"
	case "wss":
		parsed.Scheme = "https"
	default:
		return ""
	}
	parsed.Path = ""
	parsed.RawQuery = ""
	parsed.Fragment = ""
	return parsed.String()
}

func buildSourceFilter(values []string) (map[string]struct{}, error) {
	if len(values) == 0 {
		return nil, nil
	}
	filter := make(map[string]struct{})
	for _, v := range values {
		trimmed := strings.ToUpper(strings.TrimSpace(v))
		if trimmed == "" {
			continue
		}
		if !slices.Contains(allowedSources, trimmed) {
			return nil, fmt.Errorf("invalid --source value %q (valid: %s)", v, strings.Join(allowedSources, ", "))
		}
		filter[trimmed] = struct{}{}
	}
	if len(filter) == 0 {
		return nil, nil
	}
	return filter, nil
}

func newLogStreamDialer(cfg *config.Config) *websocket.Dialer {
	dialer := &websocket.Dialer{
		Proxy:            http.ProxyFromEnvironment,
		HandshakeTimeout: 30 * time.Second,
	}

	if cfg == nil {
		return dialer
	}

	if transport, ok := cfg.HTTPTransport.(*http.Transport); ok && transport != nil {
		clone := transport.Clone()
		dialer.Proxy = clone.Proxy
		dialer.NetDialContext = clone.DialContext
		if clone.TLSClientConfig != nil {
			dialer.TLSClientConfig = clone.TLSClientConfig.Clone()
		}
		return dialer
	}

	if cfg.InsecureSkipVerify {
		dialer.TLSClientConfig = &tls.Config{InsecureSkipVerify: true}
	}

	return dialer
}
