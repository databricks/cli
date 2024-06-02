package bundle

import (
	"encoding/json"
	"reflect"

	"github.com/databricks/cli/bundle/config"
	"github.com/databricks/cli/bundle/schema"
	"github.com/databricks/cli/cmd/root"
	"github.com/databricks/cli/libs/jsonschema"
	"github.com/spf13/cobra"
)

func overrideVariables(s *jsonschema.Schema) error {
	// Override schema for default values to allow for multiple primitive types.
	// These are normalized to strings when converted to the typed representation.
	err := s.SetByPath("variables.*.default", jsonschema.Schema{
		AnyOf: []*jsonschema.Schema{
			{
				Type: jsonschema.StringType,
			},
			{
				Type: jsonschema.BooleanType,
			},
			{
				Type: jsonschema.NumberType,
			},
			{
				Type: jsonschema.IntegerType,
			},
		},
	})
	if err != nil {
		return err
	}

	// Override schema for variables in targets to allow just specifying the value
	// along side overriding the variable definition if needed.
	ns, err := s.GetByPath("variables.*")
	if err != nil {
		return err
	}
	return s.SetByPath("targets.*.variables.*", jsonschema.Schema{
		AnyOf: []*jsonschema.Schema{
			{
				Type: jsonschema.StringType,
			},
			{
				Type: jsonschema.BooleanType,
			},
			{
				Type: jsonschema.NumberType,
			},
			{
				Type: jsonschema.IntegerType,
			},
			&ns,
		},
	})
}

func newSchemaCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "schema",
		Short: "Generate JSON Schema for bundle configuration",
		Args:  root.NoArgs,
	}

	cmd.RunE = func(cmd *cobra.Command, args []string) error {
		// Load embedded schema descriptions.
		docs, err := schema.LoadBundleDescriptions()
		if err != nil {
			return err
		}

		// Generate the JSON schema from the bundle configuration struct in Go.
		schema, err := schema.New(reflect.TypeOf(config.Root{}), docs)
		if err != nil {
			return err
		}

		// Override schema for variables to take into account normalization of default
		// variable values and variable overrides in a target.
		err = overrideVariables(schema)
		if err != nil {
			return err
		}

		// Print the JSON schema to stdout.
		result, err := json.MarshalIndent(schema, "", "  ")
		if err != nil {
			return err
		}
		cmd.OutOrStdout().Write(result)
		return nil
	}

	return cmd
}
