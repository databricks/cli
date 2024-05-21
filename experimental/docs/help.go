package main

import (
	"bytes"
	"slices"
	"strings"

	"github.com/spf13/cobra"
)

func CaptureHelp(cmd *cobra.Command) string {
	var buf bytes.Buffer
	cmd.SetOut(&buf)
	if err := cmd.Help(); err != nil {
		panic(err)
	}
	return buf.String()
}

func Invocation(cmd *cobra.Command) string {
	var args []string

	for cmd != nil {
		args = append(args, cmd.Use)
		cmd = cmd.Parent()
	}

	slices.Reverse(args)
	return strings.Join(args, " ")
}
