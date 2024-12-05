package cmdgroup

import (
	"io"
	"strings"
	"text/template"
	"unicode"

	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
)

type CommandWithGroupFlag struct {
	cmd        *cobra.Command
	flagGroups []*FlagGroup
}

func (c *CommandWithGroupFlag) Command() *cobra.Command {
	return c.cmd
}

func (c *CommandWithGroupFlag) FlagGroups() []*FlagGroup {
	return c.flagGroups
}

func (c *CommandWithGroupFlag) NonGroupedFlags() *pflag.FlagSet {
	nonGrouped := pflag.NewFlagSet("non-grouped", pflag.ContinueOnError)
	c.cmd.LocalFlags().VisitAll(func(f *pflag.Flag) {
		for _, fg := range c.flagGroups {
			if fg.Has(f) {
				return
			}
		}
		nonGrouped.AddFlag(f)
	})

	return nonGrouped
}

func (c *CommandWithGroupFlag) HasNonGroupedFlags() bool {
	return c.NonGroupedFlags().HasFlags()
}

func NewCommandWithGroupFlag(cmd *cobra.Command) *CommandWithGroupFlag {
	cmdWithFlagGroups := &CommandWithGroupFlag{cmd: cmd, flagGroups: make([]*FlagGroup, 0)}
	cmd.SetUsageFunc(func(c *cobra.Command) error {
		err := tmpl(c.OutOrStderr(), c.UsageTemplate(), cmdWithFlagGroups)
		if err != nil {
			c.PrintErrln(err)
		}
		return nil
	})
	cmd.SetUsageTemplate(usageTemplate)
	return cmdWithFlagGroups
}

func (c *CommandWithGroupFlag) AddFlagGroup(fg *FlagGroup) {
	c.flagGroups = append(c.flagGroups, fg)
	c.cmd.Flags().AddFlagSet(fg.FlagSet())
}

type FlagGroup struct {
	name        string
	description string
	flagSet     *pflag.FlagSet
}

func NewFlagGroup(name string) *FlagGroup {
	return &FlagGroup{name: name, flagSet: pflag.NewFlagSet(name, pflag.ContinueOnError)}
}

func (c *FlagGroup) Name() string {
	return c.name
}

func (c *FlagGroup) Description() string {
	return c.description
}

func (c *FlagGroup) SetDescription(description string) {
	c.description = description
}

func (c *FlagGroup) FlagSet() *pflag.FlagSet {
	return c.flagSet
}

func (c *FlagGroup) Has(f *pflag.Flag) bool {
	return c.flagSet.Lookup(f.Name) != nil
}

var templateFuncs = template.FuncMap{
	"trim":                    strings.TrimSpace,
	"trimRightSpace":          trimRightSpace,
	"trimTrailingWhitespaces": trimRightSpace,
}

func trimRightSpace(s string) string {
	return strings.TrimRightFunc(s, unicode.IsSpace)
}

// tmpl executes the given template text on data, writing the result to w.
func tmpl(w io.Writer, text string, data any) error {
	t := template.New("top")
	t.Funcs(templateFuncs)
	template.Must(t.Parse(text))
	return t.Execute(w, data)
}
