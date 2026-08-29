package testdiff

import (
	"flag"
	"strings"
)

var OverwriteMode = false

func init() {
	flag.BoolVar(&OverwriteMode, "update", false, "Overwrite golden files")
}

func NormalizeNewlines(input string) string {
	output := strings.ReplaceAll(input, "\r\n", "\n")
	return strings.ReplaceAll(output, "\r", "\n")
}
