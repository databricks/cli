package root

import (
	"context"
	"fmt"
	"io"
	"maps"
	"slices"

	"github.com/databricks/cli/libs/cmdgroup"
	"github.com/spf13/cobra"
)

// Tree is the CLI's command tree. It owns the root command, so commands can
// only be added as specs and the tree can only run through [Tree.Execute],
// which builds the commands that the args resolve to before running them.
type Tree struct {
	cmd     *cobra.Command
	pending map[*cobra.Command]CommandSpec
}

// NewTree returns a tree with the CLI's root command and no subcommands.
func NewTree(ctx context.Context) *Tree {
	return &Tree{cmd: newRootCommand(ctx), pending: map[*cobra.Command]CommandSpec{}}
}

// Add adds top-level commands. Each is built when an invocation needs it.
func (t *Tree) Add(specs ...CommandSpec) {
	for _, spec := range specs {
		// Copy the fields so that the caller changing them later can't make
		// the placeholder and the built command differ.
		build := spec.Build
		spec = specFields(spec.command())
		spec.Build = build
		placeholder := spec.command()
		// Cobra only lists a command in help and completion if it is
		// runnable or has subcommands; a placeholder has no subcommands.
		placeholder.RunE = func(cmd *cobra.Command, args []string) error {
			return fmt.Errorf("internal error: %s was not built before running", cmd.CommandPath())
		}
		t.cmd.AddCommand(placeholder)
		t.pending[placeholder] = spec
	}
}

// AddGroups adds the groups that have at least one available top-level
// command, so call it after [Tree.Add]. If at most one group remains, it
// ungroups the top-level commands instead; see [cmdgroup.Configure].
func (t *Tree) AddGroups(groups ...cobra.Group) {
	cmdgroup.Configure(t.cmd, groups)
}

// SetOut sets the destination for the commands' output.
func (t *Tree) SetOut(w io.Writer) { t.cmd.SetOut(w) }

// SetErr sets the destination for the commands' error output.
func (t *Tree) SetErr(w io.Writer) { t.cmd.SetErr(w) }

// SetIn sets the source for the commands' input.
func (t *Tree) SetIn(in io.Reader) { t.cmd.SetIn(in) }

// Execute builds the commands that args resolve to and runs the command.
func (t *Tree) Execute(ctx context.Context, args []string) error {
	t.buildForArgs(ctx, args)
	return execute(ctx, t.cmd, args)
}

// buildAll builds every command and returns the root command.
func (t *Tree) buildAll(ctx context.Context) *cobra.Command {
	placeholders := slices.Collect(maps.Keys(t.pending))
	cmds := make([]*cobra.Command, len(placeholders))
	for i, placeholder := range placeholders {
		cmds[i] = t.buildPending(ctx, placeholder)
	}
	// Cobra recomputes the commands' layout on every removal, so remove and
	// add them all at once.
	t.cmd.RemoveCommand(placeholders...)
	t.cmd.AddCommand(cmds...)
	return t.cmd
}

// replace replaces placeholder with the command its spec builds.
func (t *Tree) replace(ctx context.Context, placeholder *cobra.Command) {
	cmd := t.buildPending(ctx, placeholder)
	t.cmd.RemoveCommand(placeholder)
	t.cmd.AddCommand(cmd)
}

// buildPending builds the command that placeholder stands in for.
func (t *Tree) buildPending(ctx context.Context, placeholder *cobra.Command) *cobra.Command {
	spec := t.pending[placeholder]
	delete(t.pending, placeholder)
	cmd := NewCommand(ctx, spec)
	// The root may have ungrouped the placeholder, for example because its
	// group has no visible commands.
	cmd.GroupID = placeholder.GroupID
	return cmd
}

// buildForArgs builds the command that args resolve to. Cobra picks the
// command to run from args before running any hook, so if it isn't built
// first, cobra runs its placeholder.
func (t *Tree) buildForArgs(ctx context.Context, args []string) {
	// Cobra adds its help and completion commands when executing. Add them now
	// so that args naming them resolve to them.
	t.cmd.InitDefaultHelpCmd()
	complete := &cobra.Command{
		Use:     cobra.ShellCompRequestCmd,
		Aliases: []string{cobra.ShellCompNoDescRequestCmd},
		Hidden:  true,
	}
	t.cmd.AddCommand(complete)
	defer t.cmd.RemoveCommand(complete)

	cmd, _, err := t.cmd.Find(args)
	if err != nil {
		return
	}
	_, pending := t.pending[cmd]
	switch {
	case cmd.Parent() == t.cmd && (cmd.Name() == "help" || isCompletionRequest(cmd)):
		// Help and completion resolve commands from their own arguments, the
		// way cobra parses them. Building everything avoids parsing them here.
		t.buildAll(ctx)
	case pending:
		t.replace(ctx, cmd)
	}
}

func isCompletionRequest(cmd *cobra.Command) bool {
	return cmd.Name() == cobra.ShellCompRequestCmd
}
