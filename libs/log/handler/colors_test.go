package handler

import (
	"fmt"
	"testing"
)

func showColors(t *testing.T, colors ttyColors) {
	t.Log(colors[ttyColorInvalid].Render("invalid"))
	t.Log(colors[ttyColorTime].Render("time"))
	t.Log(
		fmt.Sprint(
			colors[ttyColorAttrKey].Render("key"),
			colors[ttyColorAttrSeparator].Render("="),
			colors[ttyColorAttrValue].Render("value"),
		),
	)
	t.Log(colors[ttyColorLevelTrace].Render("trace"))
	t.Log(colors[ttyColorLevelDebug].Render("debug"))
	t.Log(colors[ttyColorLevelInfo].Render("info"))
	t.Log(colors[ttyColorLevelWarn].Render("warn"))
	t.Log(colors[ttyColorLevelError].Render("error"))
}

func TestTTYColorsEnabled(t *testing.T) {
	showColors(t, newColors(true))
}

func TestTTYColorsDisabled(t *testing.T) {
	showColors(t, newColors(false))
}
