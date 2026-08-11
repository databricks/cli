package debug

import (
	"io"
	"os"

	"github.com/databricks/cli/cmd/root"
	"github.com/databricks/cli/libs/dyn/jsonsaver"
	"github.com/databricks/cli/libs/dyn/yamlloader"
	"github.com/spf13/cobra"
)

// NewYamlToJSONCommand returns a command that prints a YAML file as JSON.
//
// It exists for acceptance test helpers: those are stdlib-only Python and cannot parse
// YAML, and reimplementing the loader there would diverge from how the bundle reads
// the same file (YAML 1.2 scalars, duplicate key handling).
func NewYamlToJSONCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "yaml-to-json FILE",
		Short: "Print a YAML file as JSON, parsed the way the bundle parses it",
		Args:  root.ExactArgs(1),
	}

	cmd.RunE = func(cmd *cobra.Command, args []string) error {
		return yamlToJSON(args[0], cmd.OutOrStdout())
	}

	return cmd
}

func yamlToJSON(path string, out io.Writer) error {
	f, err := os.Open(path)
	if err != nil {
		return err
	}
	defer f.Close()

	v, err := yamlloader.LoadYAML(path, f)
	if err != nil {
		return err
	}

	buf, err := jsonsaver.MarshalIndent(v, "", "  ")
	if err != nil {
		return err
	}

	_, err = out.Write(buf)
	return err
}
