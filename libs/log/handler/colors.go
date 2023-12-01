package handler

import "github.com/fatih/color"

// ttyColors is a slice of colors that can be enabled or disabled.
// This adds one level of indirection to the colors such that they
// can be easily be enabled or disabled together, regardless of
// global settings in the color package.
type ttyColors []*color.Color

// ttyColor is an enum for the colors in ttyColors.
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
	ttyColors := make(ttyColors, ttyColorLevelLast)
	ttyColors[ttyColorInvalid] = color.New(color.FgWhite)
	ttyColors[ttyColorTime] = color.New(color.FgBlack, color.Bold)
	ttyColors[ttyColorMessage] = color.New(color.Reset)
	ttyColors[ttyColorAttrKey] = color.New(color.Faint)
	ttyColors[ttyColorAttrSeparator] = color.New(color.Faint)
	ttyColors[ttyColorAttrValue] = color.New(color.Reset)
	ttyColors[ttyColorLevelTrace] = color.New(color.FgMagenta)
	ttyColors[ttyColorLevelDebug] = color.New(color.FgCyan)
	ttyColors[ttyColorLevelInfo] = color.New(color.FgBlue)
	ttyColors[ttyColorLevelWarn] = color.New(color.FgYellow)
	ttyColors[ttyColorLevelError] = color.New(color.FgRed)

	if enable {
		for _, color := range ttyColors {
			color.EnableColor()
		}
	} else {
		for _, color := range ttyColors {
			color.DisableColor()
		}
	}

	return ttyColors
}
