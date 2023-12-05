package handler

import (
	"fmt"
	"testing"
)

func showColors(t *testing.T, colors ttyColors) {
	t.Log(colors[ttyColorInvalid].Sprint("invalid"))
	t.Log(colors[ttyColorTime].Sprint("time"))
	t.Log(
		fmt.Sprint(
			colors[ttyColorAttrKey].Sprint("key"),
			colors[ttyColorAttrSeparator].Sprint("="),
			colors[ttyColorAttrValue].Sprint("value"),
		),
	)
	t.Log(colors[ttyColorLevelTrace].Sprint("trace"))
	t.Log(colors[ttyColorLevelDebug].Sprint("debug"))
	t.Log(colors[ttyColorLevelInfo].Sprint("info"))
	t.Log(colors[ttyColorLevelWarn].Sprint("warn"))
	t.Log(colors[ttyColorLevelError].Sprint("error"))
}

func TestTTYColorsEnabled(t *testing.T) {
	showColors(t, newColors(true))
}

func TestTTYColorsDisabled(t *testing.T) {
	showColors(t, newColors(false))
}
