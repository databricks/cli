package root

import (
	"context"
	"fmt"
	"reflect"
	"slices"

	"github.com/spf13/cobra"
)

// CommandSpec declares a command. Top-level commands are added to a [Tree] as
// specs, so that each is only built when an invocation needs it. Until then,
// the tree holds a placeholder with the spec's fields, which are what the root
// shows about the command in help, completion, and suggestions. Build
// completes a command that already has these fields, for example with its
// flags, run function, and subcommands. [NewCommand] panics if Build changes
// them.
type CommandSpec struct {
	Use        string
	Short      string
	GroupID    string
	Aliases    []string
	SuggestFor []string
	Hidden     bool
	Deprecated string

	Build func(ctx context.Context, cmd *cobra.Command)
}

// command returns a command with the spec's fields.
func (s CommandSpec) command() *cobra.Command {
	return &cobra.Command{
		Use:        s.Use,
		Short:      s.Short,
		GroupID:    s.GroupID,
		Aliases:    slices.Clone(s.Aliases),
		SuggestFor: slices.Clone(s.SuggestFor),
		Hidden:     s.Hidden,
		Deprecated: s.Deprecated,
	}
}

// specFields returns the spec fields of cmd, without Build.
func specFields(cmd *cobra.Command) CommandSpec {
	return CommandSpec{
		Use:        cmd.Use,
		Short:      cmd.Short,
		GroupID:    cmd.GroupID,
		Aliases:    cmd.Aliases,
		SuggestFor: cmd.SuggestFor,
		Hidden:     cmd.Hidden,
		Deprecated: cmd.Deprecated,
	}
}

// NewCommand builds the command that spec declares. Use it to add commands
// below the top level, which are built together with their parent. Their
// parent's help shows the spec's fields too, so Build must not change them.
func NewCommand(ctx context.Context, spec CommandSpec) *cobra.Command {
	cmd := spec.command()
	spec.Build(ctx, cmd)
	// Placeholders show the spec's fields, so the command must show the same.
	if !reflect.DeepEqual(specFields(cmd), specFields(spec.command())) {
		panic(fmt.Sprintf("root: building %q changed fields that its spec declares; set them in the spec instead", spec.Use))
	}
	return cmd
}
