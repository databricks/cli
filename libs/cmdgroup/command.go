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

func (c *CommandWithGroupFlag) AddFlagGroup(name string) *FlagGroup {
	fg := &FlagGroup{name: name, flagSet: pflag.NewFlagSet(name, pflag.ContinueOnError)}
	c.flagGroups = append(c.flagGroups, fg)
	return fg
}

type FlagGroup struct {
	name    string
	flagSet *pflag.FlagSet
}

func (c *FlagGroup) Name() string {
	return c.name
}

func (c *FlagGroup) FlagSet() *pflag.FlagSet {
	return c.flagSet
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
func tmpl(w io.Writer, text string, data interface{}) error {
	t := template.New("top")
	t.Funcs(templateFuncs)
	template.Must(t.Parse(text))
	return t.Execute(w, data)
}
