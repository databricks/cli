package handler

const (
	ansiReset     = "\x1b[0m"
	ansiBlackBold = "\x1b[30;1m"
	ansiWhite     = "\x1b[37m"
	ansiFaint     = "\x1b[2m"
	ansiRed       = "\x1b[31m"
	ansiYellow    = "\x1b[33m"
	ansiBlue      = "\x1b[34m"
	ansiMagenta   = "\x1b[35m"
	ansiCyan      = "\x1b[36m"
)

// ttyStyle is an SGR escape prefix that wraps a string with a trailing reset.
// An empty value emits the input unchanged so the handler can disable colors
// by zeroing the palette.
type ttyStyle string

func (s ttyStyle) Render(msg string) string {
	if s == "" {
		return msg
	}
	return string(s) + msg + ansiReset
}

type ttyColors []ttyStyle

type ttyColor int

const (
	ttyColorInvalid ttyColor = iota
	ttyColorTime
	ttyColorMessage
	ttyColorAttrKey
	ttyColorAttrSeparator
	ttyColorAttrValue
	ttyColorLevelTrace
	ttyColorLevelDebug
	ttyColorLevelInfo
	ttyColorLevelWarn
	ttyColorLevelError

	// Marker for the last value to know how many colors there are.
	ttyColorLevelLast
)

func newColors(enable bool) ttyColors {
	if !enable {
		return make(ttyColors, ttyColorLevelLast)
	}
	colors := make(ttyColors, ttyColorLevelLast)
	colors[ttyColorInvalid] = ansiWhite
	colors[ttyColorTime] = ansiBlackBold
	colors[ttyColorMessage] = ansiReset
	colors[ttyColorAttrKey] = ansiFaint
	colors[ttyColorAttrSeparator] = ansiFaint
	colors[ttyColorAttrValue] = ansiReset
	colors[ttyColorLevelTrace] = ansiMagenta
	colors[ttyColorLevelDebug] = ansiCyan
	colors[ttyColorLevelInfo] = ansiBlue
	colors[ttyColorLevelWarn] = ansiYellow
	colors[ttyColorLevelError] = ansiRed
	return colors
}
