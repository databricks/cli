package handler

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"strings"
	"sync"
	"time"

	"github.com/databricks/cli/libs/log"
)

// friendlyHandler implements a custom [slog.Handler] that writes
// human readable (and colorized) log lines to a terminal.
//
// The implementation is based on the guide at:
// https://github.com/golang/example/blob/master/slog-handler-guide/README.md
type friendlyHandler struct {
	opts Options
	goas []groupOrAttrs
	mu   *sync.Mutex
	out  io.Writer

	// List of colors to use for formatting.
	ttyColors

	// Cache (colorized) level strings.
	levelTrace string
	levelDebug string
	levelInfo  string
	levelWarn  string
	levelError string
}

// groupOrAttrs holds either a group name or a list of slog.Attrs.
type groupOrAttrs struct {
	group string      // group name if non-empty
	attrs []slog.Attr // attrs if non-empty
}

func NewFriendlyHandler(out io.Writer, opts *Options) slog.Handler {
	h := &friendlyHandler{out: out, mu: &sync.Mutex{}}
	if opts != nil {
		h.opts = *opts
	}
	if h.opts.Level == nil {
		h.opts.Level = slog.LevelInfo
	}

	h.ttyColors = newColors(opts.Color)

	// Cache (colorized) level strings.
	// The colors to use for each level are configured in `colors.go`.
	h.levelTrace = h.sprintf(ttyColorLevelTrace, "%s", "Trace:")
	h.levelDebug = h.sprintf(ttyColorLevelDebug, "%s", "Debug:")
	h.levelInfo = h.sprintf(ttyColorLevelInfo, "%s", "Info:")
	h.levelWarn = h.sprintf(ttyColorLevelWarn, "%s", "Warn:")
	h.levelError = h.sprintf(ttyColorLevelError, "%s", "Error:")
	return h
}

func (h *friendlyHandler) sprint(color ttyColor, args ...any) string {
	return h.ttyColors[color].Sprint(args...)
}

func (h *friendlyHandler) sprintf(color ttyColor, format string, args ...any) string {
	return h.ttyColors[color].Sprintf(format, args...)
}

func (h *friendlyHandler) coloredLevel(r slog.Record) string {
	switch r.Level {
	case log.LevelTrace:
		return h.levelTrace
	case log.LevelDebug:
		return h.levelDebug
	case log.LevelInfo:
		return h.levelInfo
	case log.LevelWarn:
		return h.levelWarn
	case log.LevelError:
		return h.levelError
	}
	return ""
}

// Enabled implements slog.Handler.
func (h *friendlyHandler) Enabled(ctx context.Context, level slog.Level) bool {
	return level >= h.opts.Level.Level()
}

type handleState struct {
	h *friendlyHandler

	buf    []byte
	prefix string

	// Keep stack of groups to pass to [slog.ReplaceAttr] function.
	groups []string
}

func (h *friendlyHandler) handleState() *handleState {
	return &handleState{
		h: h,

		buf:    make([]byte, 0, 1024),
		prefix: "",
	}
}

func (s *handleState) openGroup(name string) {
	s.groups = append(s.groups, name)
	s.prefix += name + "."
}

func (s *handleState) closeGroup(name string) {
	s.prefix = s.prefix[:len(s.prefix)-len(name)-1]
	s.groups = s.groups[:len(s.groups)-1]
}

func (s *handleState) append(args ...any) {
	s.buf = fmt.Append(s.buf, args...)
}

func (s *handleState) appendf(format string, args ...any) {
	s.buf = fmt.Appendf(s.buf, format, args...)
}

func (s *handleState) appendAttr(a slog.Attr) {
	if rep := s.h.opts.ReplaceAttr; rep != nil && a.Value.Kind() != slog.KindGroup {
		// Resolve before calling ReplaceAttr, so the user doesn't have to.
		a.Value = a.Value.Resolve()
		a = rep(s.groups, a)
	}

	// Resolve the Attr's value before doing anything else.
	a.Value = a.Value.Resolve()

	// Ignore empty Attrs.
	if a.Equal(slog.Attr{}) {
		return
	}

	switch a.Value.Kind() {
	case slog.KindGroup:
		attrs := a.Value.Group()
		// Output only non-empty groups.
		if len(attrs) > 0 {
			if a.Key != "" {
				s.openGroup(a.Key)
			}
			for _, aa := range attrs {
				s.appendAttr(aa)
			}
			if a.Key != "" {
				s.closeGroup(a.Key)
			}
		}
	case slog.KindTime:
		s.append(
			" ",
			s.h.sprint(ttyColorAttrKey, s.prefix, a.Key),
			s.h.sprint(ttyColorAttrSeparator, "="),
			s.h.sprint(ttyColorAttrValue, a.Value.Time().Format(time.RFC3339Nano)),
		)
	default:
		str := a.Value.String()
		format := "%s"

		// Quote values wih spaces, to make them easy to parse.
		if strings.ContainsAny(str, " \t\n") {
			format = "%q"
		}

		s.append(
			" ",
			s.h.sprint(ttyColorAttrKey, s.prefix, a.Key),
			s.h.sprint(ttyColorAttrSeparator, "="),
			s.h.sprint(ttyColorAttrValue, fmt.Sprintf(format, str)),
		)
	}
}

// Handle implements slog.Handler.
func (h *friendlyHandler) Handle(ctx context.Context, r slog.Record) error {
	state := h.handleState()

	if h.opts.Level.Level() <= slog.LevelDebug {
		state.append(h.sprintf(ttyColorTime, "%02d:%02d:%02d ", r.Time.Hour(), r.Time.Minute(), r.Time.Second()))
	}

	state.appendf("%s ", h.coloredLevel(r))
	state.append(h.sprint(ttyColorMessage, r.Message))

	if h.opts.Level.Level() <= slog.LevelDebug {

		// Handle state from WithGroup and WithAttrs.
		goas := h.goas
		if r.NumAttrs() == 0 {
			// If the record has no Attrs, remove groups at the end of the list; they are empty.
			for len(goas) > 0 && goas[len(goas)-1].group != "" {
				goas = goas[:len(goas)-1]
			}
		}
		for _, goa := range goas {
			if goa.group != "" {
				state.openGroup(goa.group)
			} else {
				for _, a := range goa.attrs {
					state.appendAttr(a)
				}
			}
		}

		// Add attributes from the record.
		r.Attrs(func(a slog.Attr) bool {
			state.appendAttr(a)
			return true
		})

	}

	// Add newline.
	state.append("\n")

	// Write the log line.
	h.mu.Lock()
	defer h.mu.Unlock()
	_, err := h.out.Write(state.buf)
	return err
}

func (h *friendlyHandler) withGroupOrAttrs(goa groupOrAttrs) *friendlyHandler {
	h2 := *h
	h2.goas = make([]groupOrAttrs, len(h.goas)+1)
	copy(h2.goas, h.goas)
	h2.goas[len(h2.goas)-1] = goa
	return &h2
}

// WithGroup implements slog.Handler.
func (h *friendlyHandler) WithGroup(name string) slog.Handler {
	if name == "" {
		return h
	}
	return h.withGroupOrAttrs(groupOrAttrs{group: name})
}

// WithAttrs implements slog.Handler.
func (h *friendlyHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
	if len(attrs) == 0 {
		return h
	}
	return h.withGroupOrAttrs(groupOrAttrs{attrs: attrs})
}
