package main

import (
	"context"
	"fmt"

	"github.com/databricks/cli/cmd"
	"github.com/spf13/cobra"
)

type Group struct {
	Name        string
	Package     string
	Command     *cobra.Command
	Subcommands []*cobra.Command
}

func Find(name string) *Group {
	root := cmd.New(context.Background())
	for _, c := range root.Commands() {
		if c.Use != name {
			continue
		}
		return &Group{
			Name:        name,
			Package:     c.Annotations["package"],
			Command:     c,
			Subcommands: c.Commands(),
		}
	}

	return nil
}

func (g *Group) Prompt() string {
	msg := fmt.Sprintf(`
We're authoring documentation and examples for the "%s" command group.

All output must be valid Markdown.

Do not include expected command output; you don't know.

Every command has its own Markdown header.

The documentation should be written in Markdown, with code blocks for each command invocation.
By concatenating the code blocks, you should be able to run the script and see the output.

Below is the help output of each one of the commands.
`, Invocation(g.Command))

	sep := "SEPARATOR BETWEEN INSTRUCTION AND HELP OUTPUT"
	all := append([]*cobra.Command{g.Command}, g.Subcommands...)
	for _, c := range all {
		inv := Invocation(c)
		msg += "\n\n" + sep + "\n\n$ " + inv + " --help\n\n" + CaptureHelp(c)
	}

	msg += "\n\n" + sep + "\n\n"
	return msg
}
