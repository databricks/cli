package dynassert

import (
	"fmt"
	"strings"

	"github.com/databricks/cli/libs/dyn"
)

// Dump returns the Go code to recreate the given value.
func Dump(v dyn.Value) string {
	var sb strings.Builder
	dump(v, &sb)
	return sb.String()
}

func dump(v dyn.Value, sb *strings.Builder) {
	sb.WriteString("dyn.NewValue(\n")

	switch v.Kind() {
	case dyn.KindMap:
		sb.WriteString("map[string]dyn.Value{")
		m := v.MustMap()
		for _, p := range m.Pairs() {
			sb.WriteString(fmt.Sprintf("\n%q: ", p.Key.MustString()))
			dump(p.Value, sb)
			sb.WriteByte(',')
		}
		sb.WriteString("\n},\n")
	case dyn.KindSequence:
		sb.WriteString("[]dyn.Value{\n")
		for _, e := range v.MustSequence() {
			dump(e, sb)
			sb.WriteByte(',')
		}
		sb.WriteString("},\n")
	case dyn.KindString:
		sb.WriteString(fmt.Sprintf("%q,\n", v.MustString()))
	case dyn.KindBool:
		sb.WriteString(fmt.Sprintf("%t,\n", v.MustBool()))
	case dyn.KindInt:
		sb.WriteString(fmt.Sprintf("%d,\n", v.MustInt()))
	case dyn.KindFloat:
		sb.WriteString(fmt.Sprintf("%f,\n", v.MustFloat()))
	case dyn.KindTime:
		sb.WriteString(fmt.Sprintf("dyn.NewTime(%q),\n", v.MustTime().String()))
	case dyn.KindNil:
		sb.WriteString("nil,\n")
	default:
		panic(fmt.Sprintf("unhandled kind: %v", v.Kind()))
	}

	// Add location
	sb.WriteString("[]dyn.Location{")
	for _, l := range v.Locations() {
		sb.WriteString(fmt.Sprintf("{File: %q, Line: %d, Column: %d},", l.File, l.Line, l.Column))
	}
	sb.WriteString("},\n")
	sb.WriteString(")")
}
