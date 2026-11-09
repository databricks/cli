package bitmap

import (
	"encoding/json"
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
	cmd.AddCommand(newBitmapCommand())
	cmd.AddCommand(newDecodeCommand())
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

// Values of the -o flag.
const (
	outputCompressed = "compressed"
	outputText       = "text"
	outputJSON       = "json"
)

// outputFormat is the -o flag of the bitmap and decode commands. It shadows the
// global --output flag, which only accepts text and json.
type outputFormat string

func (f *outputFormat) String() string {
	return string(*f)
}

func (f *outputFormat) Set(s string) error {
	switch s {
	case outputCompressed, outputText, outputJSON:
		*f = outputFormat(s)
		return nil
	}
	return fmt.Errorf("accepted values are %s, %s and %s", outputCompressed, outputText, outputJSON)
}

func (f *outputFormat) Type() string {
	return "format"
}

const outputHelp = `output format:
  compressed: base64 of the DEFLATE-compressed wire format
  text:       a "size=N context=C" header, then one "<value> <field>" line per
              set field: "T" or "F" for a boolean field holding that value, "1"
              for any other field; unset fields are omitted
  json:       size, context, and trees of the fields present ("presence") and of
              the boolean fields seen holding "true" and "false"`

func addOutputFlag(cmd *cobra.Command, output *outputFormat, def string) {
	*output = outputFormat(def)
	cmd.Flags().VarP(output, "output", "o", outputHelp)
}

func newBitmapCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "bitmap",
		Short: "Print the bitmap of the current bundle",
		Args:  root.NoArgs,
	}
	var output outputFormat
	addOutputFlag(cmd, &output, outputCompressed)
	cmd.RunE = func(cmd *cobra.Command, args []string) error {
		bits, schema, err := loadBits(cmd)
		if err != nil {
			return err
		}
		return render(cmd.OutOrStdout(), output, schema, bits, bundlebitmap.ContextFullBundle)
	}
	return cmd
}

func newDecodeCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "decode",
		Short: "Decode a compressed bitmap read from stdin",
		Long:  "The input is the output of 'bundle bitmap bitmap'.",
		Args:  root.NoArgs,
	}
	var output outputFormat
	addOutputFlag(cmd, &output, outputText)
	cmd.RunE = func(cmd *cobra.Command, args []string) error {
		encoded, err := io.ReadAll(cmd.InOrStdin())
		if err != nil {
			return err
		}
		bits, context, err := bundlebitmap.Decode(strings.TrimSpace(string(encoded)))
		if err != nil {
			return err
		}
		if context != bundlebitmap.ContextFullBundle {
			return fmt.Errorf("unsupported bitmap context %d", context)
		}
		// The schema is append-only, so an older bitmap decodes against a prefix of
		// it; a newer one has bits this binary cannot name.
		schema := bundlebitmap.EmbeddedSchema()
		if len(bits) > len(schema) {
			return fmt.Errorf("bitmap has %d bits but the schema has %d fields; decode it with a newer CLI", len(bits), len(schema))
		}
		return render(cmd.OutOrStdout(), output, schema, bits, context)
	}
	return cmd
}

func render(out io.Writer, output outputFormat, schema []string, bits []bool, context uint16) error {
	switch output {
	case outputCompressed:
		encoded, err := bundlebitmap.Encode(bits, context)
		if err != nil {
			return err
		}
		return printLines(out, []string{encoded})
	case outputText:
		return printLines(out, bundlebitmap.Text(schema, bits, context))
	case outputJSON:
		data, err := json.MarshalIndent(bundlebitmap.ToJSON(schema, bits, context), "", "  ")
		if err != nil {
			return err
		}
		return printLines(out, []string{string(data)})
	}
	return fmt.Errorf("unknown output format %q", output)
}

// loadBits loads the bundle and returns its bitmap against the embedded schema.
func loadBits(cmd *cobra.Command) ([]bool, []string, error) {
	b, err := utils.ProcessBundle(cmd, utils.ProcessOptions{Validate: true})
	if err != nil {
		return nil, nil, err
	}
	bits, err := b.Bitmap.Bits(b.Telemetry)
	if err != nil {
		return nil, nil, err
	}
	return bits, bundlebitmap.EmbeddedSchema(), nil
}

func printLines(out io.Writer, lines []string) error {
	for _, l := range lines {
		if _, err := fmt.Fprintln(out, l); err != nil {
			return err
		}
	}
	return nil
}
