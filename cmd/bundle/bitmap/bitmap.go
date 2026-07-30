package bitmap

import (
	"errors"
	"fmt"
	"io"
	"reflect"
	"strings"

	"github.com/databricks/cli/bundle"
	bundlebitmap "github.com/databricks/cli/bundle/bitmap"
	"github.com/databricks/cli/cmd/bundle/utils"
	"github.com/databricks/cli/cmd/root"
	"github.com/spf13/cobra"
)

// NewBitmapCommand returns the hidden "bitmap" command group. It exposes the
// bundle bitmap: a per-field presence map used for telemetry. See
// bundle/bitmap for the format and schema.
func NewBitmapCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "bitmap",
		Short: "Inspect the bundle field bitmap used for telemetry",
		Long: `The bundle bitmap encodes the presence of every bundle configuration field
as a single bit. Its schema is an ordered list of field paths derived from the
bundle configuration type and embedded into the CLI binary.`,
		Hidden: true,
	}
	cmd.AddCommand(newSchemaCommand())
	cmd.AddCommand(newUpdateSchemaCommand())
	cmd.AddCommand(newBitmapTextCommand())
	cmd.AddCommand(newBitmapCommand())
	return cmd
}

func newSchemaCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "schema",
		Short: "Print the embedded bitmap schema, one field path per line",
		Args:  root.NoArgs,
	}
	cmd.RunE = func(cmd *cobra.Command, args []string) error {
		return printLines(cmd.OutOrStdout(), bundlebitmap.EmbeddedSchema())
	}
	return cmd
}

func newUpdateSchemaCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "update-schema",
		Short: "Append newly added fields to the embedded schema",
		Long: `Walks the bundle configuration type and appends any fields not already in the
embedded schema, printing the merged schema. The schema is append-only: removed
fields are kept so that bit positions stay stable.`,
		Args: root.NoArgs,
	}
	var validate bool
	cmd.Flags().BoolVar(&validate, "validate", false, "Fail if the embedded schema is missing fields instead of printing the merged schema")
	cmd.RunE = func(cmd *cobra.Command, args []string) error {
		fresh, err := bundlebitmap.WalkSchema(reflect.TypeFor[bundle.Telemetry]())
		if err != nil {
			return err
		}
		merged, added := bundlebitmap.Merge(bundlebitmap.EmbeddedSchema(), fresh)
		if validate {
			if len(added) > 0 {
				return fmt.Errorf("embedded schema is out of date, %d field(s) missing:\n%s\nrun 'task generate-bitmap-schema' to update", len(added), strings.Join(added, "\n"))
			}
			return nil
		}
		return printLines(cmd.OutOrStdout(), merged)
	}
	return cmd
}

func newBitmapTextCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "bitmap-text",
		Short: "Print the bitmap as '0/1 field_path' lines for the current bundle",
		Args:  root.NoArgs,
	}
	cmd.RunE = func(cmd *cobra.Command, args []string) error {
		bits, schema, err := loadBits(cmd)
		if err != nil {
			return err
		}
		out := cmd.OutOrStdout()
		for i, p := range schema {
			set := 0
			if bits[i] {
				set = 1
			}
			fmt.Fprintf(out, "%d %s\n", set, p)
		}
		return nil
	}
	return cmd
}

func newBitmapCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "bitmap",
		Short: "Print the compressed, base64-encoded bitmap for the current bundle",
		Args:  root.NoArgs,
	}
	cmd.RunE = func(cmd *cobra.Command, args []string) error {
		bits, _, err := loadBits(cmd)
		if err != nil {
			return err
		}
		encoded, err := bundlebitmap.Encode(bits, bundlebitmap.ContextFullBundle)
		if err != nil {
			return err
		}
		fmt.Fprintln(cmd.OutOrStdout(), encoded)
		return nil
	}
	return cmd
}

// loadBits loads the bundle and computes its bitmap against the embedded schema.
func loadBits(cmd *cobra.Command) ([]bool, []string, error) {
	b, err := utils.ProcessBundle(cmd, utils.ProcessOptions{Validate: true})
	if err != nil && !errors.Is(err, root.ErrAlreadyPrinted) {
		return nil, nil, err
	}
	if b == nil {
		return nil, nil, errors.New("failed to load bundle")
	}
	schema := bundlebitmap.EmbeddedSchema()
	bits, err := bundlebitmap.Bits(b.Config, b.Metrics.Telemetry, schema)
	if err != nil {
		return nil, nil, err
	}
	return bits, schema, nil
}

func printLines(out io.Writer, lines []string) error {
	for _, l := range lines {
		if _, err := fmt.Fprintln(out, l); err != nil {
			return err
		}
	}
	return nil
}
