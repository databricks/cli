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
	"time"

	"github.com/gorilla/websocket"
)

const handshakeErrorBodyLimit = 4 * 1024
const defaultUserAgent = "databricks-cli logstream"
const initialReconnectBackoff = 200 * time.Millisecond
const maxReconnectBackoff = 5 * time.Second

// Dialer defines the subset of websocket.Dialer used by the streamer.
type Dialer interface {
	DialContext(ctx context.Context, urlStr string, requestHeader http.Header) (*websocket.Conn, *http.Response, error)
}

// TokenProvider refreshes tokens when the streamer needs a new bearer token.
type TokenProvider func(context.Context) (string, error)

// Config holds the options for running a log stream.
type Config struct {
	Dialer        Dialer
	URL           string
	Origin        string
	Token         string
	TokenProvider TokenProvider
	Search        string
	Sources       map[string]struct{}
	Tail          int
	Follow        bool
	Prefetch      time.Duration
	Writer        io.Writer
	UserAgent     string
}

// Run connects to the log stream described by cfg and copies frames to the writer.
func Run(ctx context.Context, cfg Config) error {
	if cfg.Writer == nil {
		return errors.New("logstream: writer is required")
	}

	streamer := &logStreamer{
		dialer:        cfg.Dialer,
		url:           cfg.URL,
		origin:        cfg.Origin,
		token:         cfg.Token,
		tokenProvider: cfg.TokenProvider,
		search:        cfg.Search,
		sources:       cfg.Sources,
		tail:          cfg.Tail,
		follow:        cfg.Follow,
		prefetch:      cfg.Prefetch,
		writer:        cfg.Writer,
		userAgent:     cfg.UserAgent,
	}
	if streamer.userAgent == "" {
		streamer.userAgent = defaultUserAgent
	}
	return streamer.Run(ctx)
}

type logStreamer struct {
	dialer        Dialer
	url           string
	origin        string
	token         string
	tokenProvider TokenProvider
	search        string
	sources       map[string]struct{}
	tail          int
	follow        bool
	prefetch      time.Duration
	writer        io.Writer
	tailFlushed   bool
	userAgent     string
}

func (s *logStreamer) Run(ctx context.Context) error {
	if s.dialer == nil {
		s.dialer = &websocket.Dialer{}
	}
	backoff := initialReconnectBackoff
	timer := time.NewTimer(time.Hour)
	stopTimer(timer)
	defer timer.Stop()
	for {
		if err := s.ensureToken(ctx); err != nil {
			return err
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
		} else if resp != nil && resp.Body != nil {
			_ = resp.Body.Close()
		}
		if err != nil {
			if ctx.Err() != nil {
				return ctx.Err()
			}
			if s.follow && s.shouldRefreshForStatus(resp) {
				if err := s.refreshToken(ctx); err != nil {
					return err
				}
				backoff = time.Second
				continue
			}
			if !s.follow {
				return err
			}
			if err := waitForBackoff(ctx, timer, backoff); err != nil {
				return err
			}
			backoff = min(backoff*2, maxReconnectBackoff)
			continue
		}

		backoff = time.Second
		err = s.consume(ctx, conn)
		_ = conn.Close()
		if err == nil {
			if !s.follow {
				return nil
			}
			continue
		}

		if ctx.Err() != nil {
			return ctx.Err()
		}
		if s.follow && s.shouldRefreshForError(err) {
			if err := s.refreshToken(ctx); err != nil {
				return err
			}
			continue
		}
		if !s.follow {
			return err
		}

		if err := waitForBackoff(ctx, timer, backoff); err != nil {
			return err
		}
		backoff = min(backoff*2, maxReconnectBackoff)
	}
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

	closeCh := make(chan struct{})
	defer close(closeCh)
	go func() {
		select {
		case <-ctx.Done():
			_ = conn.WriteControl(websocket.CloseMessage, websocket.FormatCloseMessage(websocket.CloseNormalClosure, "context canceled"), time.Now().Add(time.Second))
			_ = conn.Close()
		case <-closeCh:
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
		line := formatLogEntry(entry)
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
		if !s.follow && !flushDeadline.IsZero() {
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

func formatLogEntry(entry *wsEntry) string {
	return fmt.Sprintf("%s [%s] %s", formatTimestamp(entry.Timestamp), entry.Source, strings.TrimRight(entry.Message, "\r\n"))
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

	detail := fmt.Sprintf("HTTP %s", status)
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
		case 4401, 4403:
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
	resetTimer(timer, d)
	select {
	case <-ctx.Done():
		stopTimer(timer)
		return ctx.Err()
	case <-timer.C:
		return nil
	}
}

func stopTimer(timer *time.Timer) {
	if timer == nil {
		return
	}
	if !timer.Stop() {
		drainTimer(timer)
	}
}

func resetTimer(timer *time.Timer, d time.Duration) {
	if timer == nil {
		return
	}
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
