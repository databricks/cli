package log

import "log/slog"

type ReplaceAttrFunction func(groups []string, a slog.Attr) slog.Attr

// ReplaceAttrFunctions enables grouping functions that replace attributes
// from a [slog.Handler]. Useful when multiple attributes need replacing.
type ReplaceAttrFunctions []ReplaceAttrFunction

// ReplaceAttr can be used as a value to pass to a handler to combine
// multiple functions to replace attributes.
func (fns ReplaceAttrFunctions) ReplaceAttr(groups []string, a slog.Attr) slog.Attr {
	for _, fn := range fns {
		a = fn(groups, a)
	}
	return a
}
