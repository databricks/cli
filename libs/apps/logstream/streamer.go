package logstream

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/databricks/cli/libs/flags"
	"github.com/gorilla/websocket"
)

const (
	handshakeErrorBodyLimit = 4 * 1024
	defaultUserAgent        = "databricks-cli logstream"
	closeCodeUnauthorized   = 4401
	closeCodeForbidden      = 4403
)

// Dialer defines the subset of websocket.Dialer used by the streamer.
type Dialer interface {
	DialContext(ctx context.Context, urlStr string, requestHeader http.Header) (*websocket.Conn, *http.Response, error)
}

// TokenProvider refreshes tokens when the streamer needs a new bearer token.
type TokenProvider func(context.Context) (string, error)

// AppStatusChecker checks if the app is still running.
// Returns nil if app is running, or an error if the app is stopped/unavailable.
type AppStatusChecker func(context.Context) error

// Config holds the options for running a log stream.
type Config struct {
	Dialer           Dialer
	URL              string
	Origin           string
	Token            string
	TokenProvider    TokenProvider
	AppStatusChecker AppStatusChecker
	Search           string
	Sources          map[string]struct{}
	Tail             int
	Follow           bool
	Prefetch         time.Duration
	Writer           io.Writer
	UserAgent        string
	Colorize         bool
	OutputFormat     flags.Output
}

// Run connects to the log stream described by cfg and copies frames to the writer.
func Run(ctx context.Context, cfg Config) error {
	if cfg.Writer == nil {
		return errors.New("logstream: writer is required")
	}

	streamer := &logStreamer{
		dialer:           cfg.Dialer,
		url:              cfg.URL,
		origin:           cfg.Origin,
		token:            cfg.Token,
		tokenProvider:    cfg.TokenProvider,
		appStatusChecker: cfg.AppStatusChecker,
		search:           cfg.Search,
		sources:          cfg.Sources,
		tail:             cfg.Tail,
		follow:           cfg.Follow,
		prefetch:         cfg.Prefetch,
		writer:           cfg.Writer,
		userAgent:        cfg.UserAgent,
		formatter:        newLogFormatter(cfg.Colorize, cfg.OutputFormat),
	}
	if streamer.userAgent == "" {
		streamer.userAgent = defaultUserAgent
	}
	return streamer.Run(ctx)
}

// logStreamer manages the websocket connection and log consumption.
type logStreamer struct {
	dialer           Dialer
	url              string
	origin           string
	token            string
	tokenProvider    TokenProvider
	appStatusChecker AppStatusChecker
	search           string
	sources          map[string]struct{}
	tail             int
	follow           bool
	prefetch         time.Duration
	writer           io.Writer
	tailFlushed      bool
	userAgent        string
	formatter        *logFormatter
}

// Run establishes the websocket connection and manages reconnections.
// It is not safe to call Run concurrently on the same logStreamer instance.
func (s *logStreamer) Run(ctx context.Context) error {
	if s.dialer == nil {
		s.dialer = &websocket.Dialer{}
	}

	backoff := newBackoffStrategy(initialReconnectBackoff, maxReconnectBackoff)

	for {
		shouldContinue, err := func() (bool, error) {
			respStatusCode, err := s.connectAndConsume(ctx)
			if err != nil {
				if ctx.Err() != nil {
					return false, ctx.Err()
				}

				if s.follow && (s.shouldRefreshForStatus(respStatusCode) || s.shouldRefreshForError(err)) {
					if err := s.refreshToken(ctx); err != nil {
						return false, err
					}
					backoff.Reset()
					return true, nil
				}

				if !s.follow {
					return false, err
				}

				// Before retrying, check if the app is still running (if checker is provided).
				if s.appStatusChecker != nil {
					if statusErr := s.appStatusChecker(ctx); statusErr != nil {
						return false, fmt.Errorf("app is no longer available: %w", statusErr)
					}
				}

				return true, nil
			}

			backoff.Reset()
			if !s.follow {
				return false, nil
			}
			// Connection closed normally while following - check if app is still running.
			if s.appStatusChecker != nil {
				if statusErr := s.appStatusChecker(ctx); statusErr != nil {
					return false, fmt.Errorf("app is no longer available: %w", statusErr)
				}
			}
			return true, nil
		}()
		if err != nil {
			return err
		}

		if shouldContinue {
			if err := backoff.Wait(ctx); err != nil {
				return err
			}
			backoff.Next()
			continue
		}

		return nil
	}
}

func (s *logStreamer) connectAndConsume(ctx context.Context) (*int, error) {
	if err := s.ensureToken(ctx); err != nil {
		return nil, err
	}

	headers := http.Header{}
	headers.Set("Authorization", "Bearer "+s.token)
	headers.Set("User-Agent", s.userAgent)
	if s.origin != "" {
		headers.Set("Origin", s.origin)
	}

	conn, resp, err := s.dialer.DialContext(ctx, s.url, headers)
	defer closeBody(resp)
	if err != nil {
		err = decorateDialError(err, resp)

		var statusCode *int
		if resp != nil {
			statusCode = &resp.StatusCode
		}

		return statusCode, err
	}

	stopWatch := watchContext(ctx, conn)
	defer stopWatch()

	err = s.consume(ctx, conn)
	return nil, err
}

func (s *logStreamer) consume(ctx context.Context, conn *websocket.Conn) (retErr error) {
	if err := conn.WriteMessage(websocket.TextMessage, []byte(s.search)); err != nil {
		return err
	}

	state := newConsumeState(s.tail, s.follow, s.prefetch, s.writer, s.tailFlushed)
	defer func() {
		if err := state.FlushRemaining(); err != nil && retErr == nil {
			retErr = err
		}
		s.tailFlushed = state.IsFlushed()
	}()

	for {
		if ctx.Err() != nil {
			return ctx.Err()
		}

		_ = conn.SetReadDeadline(state.ReadDeadline(ctx))

		_, message, err := conn.ReadMessage()
		if err != nil {
			if ctx.Err() != nil {
				return ctx.Err()
			}
			var netErr net.Error
			if errors.As(err, &netErr) && netErr.Timeout() {
				if state.HasPendingFlushDeadline() {
					shouldContinue, flushErr := state.HandleFlushTimeout()
					if flushErr != nil {
						return flushErr
					}
					if shouldContinue {
						continue
					}
					return nil
				}
			}
			if handled, closeErr := handleCloseError(err); handled {
				return closeErr
			}
			if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
				return ctx.Err()
			}
			return err
		}

		if len(message) == 1 && message[0] == 0 {
			continue
		}

		line := s.formatMessage(message)
		if line == "" {
			continue
		}
		if err := state.ProcessLine(line); err != nil {
			return err
		}
	}
}

func (s *logStreamer) formatMessage(message []byte) string {
	entry, err := parseLogEntry(message)
	if err != nil {
		return s.formatter.FormatPlain(message)
	}
	source := strings.ToUpper(entry.Source)
	if len(s.sources) > 0 {
		if _, ok := s.sources[source]; !ok {
			return ""
		}
	}
	return s.formatter.FormatEntry(entry)
}

func (s *logStreamer) ensureToken(ctx context.Context) error {
	if s.token != "" || s.tokenProvider == nil {
		return nil
	}
	token, err := s.tokenProvider(ctx)
	if err != nil {
		return err
	}
	s.token = token
	return nil
}

func (s *logStreamer) refreshToken(ctx context.Context) error {
	if s.tokenProvider == nil {
		return errors.New("token refresh unavailable")
	}
	s.token = ""
	return s.ensureToken(ctx)
}

func (s *logStreamer) shouldRefreshForStatus(respStatusCode *int) bool {
	if respStatusCode == nil {
		return false
	}
	switch *respStatusCode {
	case http.StatusUnauthorized, http.StatusForbidden:
		return true
	default:
		return false
	}
}

func (s *logStreamer) shouldRefreshForError(err error) bool {
	var closeErr *websocket.CloseError
	if errors.As(err, &closeErr) {
		switch closeErr.Code {
		case closeCodeUnauthorized, closeCodeForbidden:
			return true
		}
	}
	return false
}

func decorateDialError(err error, resp *http.Response) error {
	if resp == nil {
		return err
	}

	var errDetails string
	if resp.Body != nil {
		data, readErr := io.ReadAll(io.LimitReader(resp.Body, handshakeErrorBodyLimit))
		_ = resp.Body.Close()
		if readErr == nil && json.Valid(data) {
			errDetails = fmt.Sprintf(" (details: %s)", string(data))
		}
	}

	return fmt.Errorf("%w (HTTP %s)%s", err, resp.Status, errDetails)
}

func handleCloseError(err error) (bool, error) {
	var closeErr *websocket.CloseError
	if !errors.As(err, &closeErr) {
		return false, err
	}
	if closeErr.Code == websocket.CloseNormalClosure || closeErr.Code == websocket.CloseGoingAway {
		return true, nil
	}
	return true, fmt.Errorf("log stream closed with code %d (%s): %w", closeErr.Code, strings.TrimSpace(closeErr.Text), err)
}

func watchContext(ctx context.Context, conn *websocket.Conn) func() {
	var once sync.Once
	closeCh := make(chan struct{})

	closeConn := func() {
		once.Do(func() {
			_ = conn.WriteControl(websocket.CloseMessage, websocket.FormatCloseMessage(websocket.CloseNormalClosure, "context canceled"), time.Now().Add(time.Second))
			_ = conn.Close()
		})
	}

	go func() {
		select {
		case <-ctx.Done():
			closeConn()
		case <-closeCh:
		}
	}()

	return func() {
		close(closeCh)
		closeConn()
	}
}

func closeBody(resp *http.Response) {
	if resp == nil || resp.Body == nil {
		return
	}

	_ = resp.Body.Close()
}
