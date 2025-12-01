package logstream

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"slices"
	"strings"
	"sync"
	"time"

	"github.com/fatih/color"
	"github.com/gorilla/websocket"
)

const (
	handshakeErrorBodyLimit = 4 * 1024
	defaultUserAgent        = "databricks-cli logstream"
	initialReconnectBackoff = 200 * time.Millisecond
	maxReconnectBackoff     = 5 * time.Second
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
		colorize:         cfg.Colorize,
	}
	if streamer.userAgent == "" {
		streamer.userAgent = defaultUserAgent
	}
	return streamer.Run(ctx)
}

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
	colorize         bool
}

// Run establishes the websocket connection and manages reconnections.
// It is not safe to call Run concurrently on the same logStreamer instance.
func (s *logStreamer) Run(ctx context.Context) error {
	if s.dialer == nil {
		s.dialer = &websocket.Dialer{}
	}

	backoff := initialReconnectBackoff
	// Backoff timer starts as a zero-value timer; stopTimer handles the first initialization safely.
	timer := time.NewTimer(0)
	stopTimer(timer, 0)

	for {
		shouldContinue, err := func() (bool, error) {
			resp, err := s.connectAndConsume(ctx)
			if err != nil {
				if ctx.Err() != nil {
					return false, ctx.Err()
				}

				if s.follow && (s.shouldRefreshForStatus(resp) || s.shouldRefreshForError(err)) {
					if err := s.refreshToken(ctx); err != nil {
						return false, err
					}
					backoff = initialReconnectBackoff
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
			if resp != nil && resp.Body != nil {
				defer resp.Body.Close()
			}

			backoff = initialReconnectBackoff
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
			if err := waitForBackoff(ctx, timer, backoff); err != nil {
				return err
			}
			backoff = min(backoff*2, maxReconnectBackoff)
			continue
		}

		return nil
	}
}

func (s *logStreamer) connectAndConsume(ctx context.Context) (*http.Response, error) {
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
	if err != nil {
		err = decorateDialError(err, resp)
		if resp != nil && resp.Body != nil {
			_ = resp.Body.Close()
		}
		return resp, err
	}
	if resp != nil && resp.Body != nil {
		_ = resp.Body.Close()
	}

	stopWatch := watchContext(ctx, conn)
	defer stopWatch()

	err = s.consume(ctx, conn)
	return nil, err
}

func (s *logStreamer) consume(ctx context.Context, conn *websocket.Conn) (retErr error) {
	initial := []byte(s.search)
	if len(initial) == 0 {
		initial = []byte("")
	}

	if err := conn.WriteMessage(websocket.TextMessage, initial); err != nil {
		return err
	}

	buffer := &tailBuffer{size: s.tail}
	flushed := s.tail == 0 || s.tailFlushed
	var flushDeadline time.Time
	if s.tail > 0 && s.prefetch > 0 && !s.tailFlushed {
		flushDeadline = time.Now().Add(s.prefetch)
	}

	defer func() {
		if s.tail > 0 && !flushed {
			if err := buffer.Flush(s.writer); err != nil {
				if retErr == nil {
					retErr = err
				}
				return
			}
			flushed = true
			s.tailFlushed = true
		}
	}()

	for {
		if ctx.Err() != nil {
			return ctx.Err()
		}

		deadline, hasDeadline := ctx.Deadline()
		if !flushDeadline.IsZero() {
			if !hasDeadline || flushDeadline.Before(deadline) {
				deadline = flushDeadline
			}
			hasDeadline = true
		}
		if hasDeadline {
			_ = conn.SetReadDeadline(deadline)
		} else {
			_ = conn.SetReadDeadline(time.Time{})
		}

		_, message, err := conn.ReadMessage()
		if err != nil {
			if ctx.Err() != nil {
				return ctx.Err()
			}
			var netErr net.Error
			if errors.As(err, &netErr) && netErr.Timeout() {
				if !flushDeadline.IsZero() {
					flushDeadline = time.Time{}
					if s.tail > 0 && !flushed {
						if err := buffer.Flush(s.writer); err != nil {
							return err
						}
						flushed = true
						s.tailFlushed = true
						if !s.follow {
							return nil
						}
					}
					continue
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

		entry, err := parseLogEntry(message)
		if err != nil {
			line := formatPlainMessage(message)
			if line == "" {
				continue
			}
			stop, err := s.flushOrBufferLine(line, buffer, &flushed, &flushDeadline)
			if err != nil {
				return err
			}
			if stop {
				return nil
			}
			continue
		}
		source := strings.ToUpper(entry.Source)
		if len(s.sources) > 0 {
			if _, ok := s.sources[source]; !ok {
				continue
			}
		}
		line := s.formatLogEntry(entry)
		stop, err := s.flushOrBufferLine(line, buffer, &flushed, &flushDeadline)
		if err != nil {
			return err
		}
		if stop {
			return nil
		}
	}
}

func (s *logStreamer) flushOrBufferLine(line string, buffer *tailBuffer, flushed *bool, flushDeadline *time.Time) (bool, error) {
	if s.tail > 0 && !*flushed {
		buffer.Add(line)
		ready := buffer.Len() >= s.tail
		if !flushDeadline.IsZero() {
			ready = false
		}
		if ready {
			if !s.follow {
				return false, nil
			}
			if err := buffer.Flush(s.writer); err != nil {
				return false, err
			}
			*flushed = true
			s.tailFlushed = true
			*flushDeadline = time.Time{}
		}
		return false, nil
	}
	if _, err := fmt.Fprintln(s.writer, line); err != nil {
		return false, err
	}
	return false, nil
}

type wsEntry struct {
	Source    string  `json:"source"`
	Timestamp float64 `json:"timestamp"`
	Message   string  `json:"message"`
}

func parseLogEntry(raw []byte) (*wsEntry, error) {
	var entry wsEntry
	if err := json.Unmarshal(raw, &entry); err != nil {
		return nil, err
	}
	return &entry, nil
}

func (s *logStreamer) formatLogEntry(entry *wsEntry) string {
	timestamp := formatTimestamp(entry.Timestamp)
	source := strings.ToUpper(entry.Source)
	message := strings.TrimRight(entry.Message, "\r\n")

	if s.colorize {
		timestamp = color.HiBlackString(timestamp)
		source = color.HiBlueString(source)
	}

	return fmt.Sprintf("%s [%s] %s", timestamp, source, message)
}

func formatPlainMessage(raw []byte) string {
	line := strings.TrimRight(string(raw), "\r\n")
	return line
}

type tailBuffer struct {
	size  int
	lines []string
}

func (b *tailBuffer) Add(line string) {
	if b.size <= 0 {
		return
	}
	b.lines = append(b.lines, line)
	if len(b.lines) > b.size {
		b.lines = slices.Delete(b.lines, 0, len(b.lines)-b.size)
	}
}

func (b *tailBuffer) Len() int {
	return len(b.lines)
}

func (b *tailBuffer) Flush(w io.Writer) error {
	if b.size == 0 {
		return nil
	}
	for _, line := range b.lines {
		if _, err := fmt.Fprintln(w, line); err != nil {
			return err
		}
	}
	b.lines = slices.Clip(b.lines[:0])
	return nil
}

func formatTimestamp(ts float64) string {
	if ts <= 0 {
		return "----------"
	}
	sec := int64(ts)
	nsec := int64((ts - float64(sec)) * 1e9)
	t := time.Unix(sec, nsec).UTC()
	return t.Format(time.RFC3339)
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

func decorateDialError(err error, resp *http.Response) error {
	if resp == nil {
		return err
	}

	var bodySnippet string
	if resp.Body != nil {
		data, readErr := io.ReadAll(io.LimitReader(resp.Body, handshakeErrorBodyLimit))
		_ = resp.Body.Close()
		if readErr == nil {
			bodySnippet = strings.TrimSpace(string(data))
		}
	}

	status := strings.TrimSpace(resp.Status)
	if status == "" && resp.StatusCode != 0 {
		status = fmt.Sprintf("%d %s", resp.StatusCode, http.StatusText(resp.StatusCode))
	}
	if status == "" {
		status = "unknown status"
	}

	detail := "HTTP " + status
	if bodySnippet != "" {
		return fmt.Errorf("%w (%s: %s)", err, detail, bodySnippet)
	}
	return fmt.Errorf("%w (%s)", err, detail)
}

func (s *logStreamer) shouldRefreshForStatus(resp *http.Response) bool {
	if resp == nil {
		return false
	}
	switch resp.StatusCode {
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

func waitForBackoff(ctx context.Context, timer *time.Timer, d time.Duration) error {
	if d <= 0 {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
			return nil
		}
	}
	stopTimer(timer, d)
	select {
	case <-ctx.Done():
		stopTimer(timer, 0)
		return ctx.Err()
	case <-timer.C:
		return nil
	}
}

func stopTimer(timer *time.Timer, d time.Duration) {
	if timer == nil {
		return
	}
	if d <= 0 {
		// For a zero duration we only need to stop and drain.
		if timer.Stop() {
			return
		}
		drainTimer(timer)
		return
	}
	// For a positive duration, either stop an already-started timer or
	// just initialize it when it is still in the zero state.
	if !timer.Stop() {
		drainTimer(timer)
	}
	timer.Reset(d)
}

func drainTimer(timer *time.Timer) {
	select {
	case <-timer.C:
	default:
	}
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
