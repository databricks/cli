package main

import (
	"log"
	"os"

	"github.com/databricks/bricks/experimental/tfprovider"
	"github.com/databricks/bricks/experimental/tfprovider/golang"
	"github.com/databricks/bricks/experimental/tfprovider/jsonschema"
	"github.com/spf13/cobra"
)

var command = &cobra.Command{
	Use: "tfprovider",
	PreRun: func(cmd *cobra.Command, args []string) {
		log.SetOutput(os.Stderr)
	},
}

var genCommand = &cobra.Command{
	Use: "gen",
}

var jsonschemaCommand = &cobra.Command{
	Use: "jsonschema",
	RunE: func(cmd *cobra.Command, args []string) error {
		schema, err := tfprovider.ProduceProviderSchema()
		if err != nil {
			return err
		}

		return jsonschema.Generate(schema)
	},
}

var gostructCommand = &cobra.Command{
	Use: "golang",
	RunE: func(cmd *cobra.Command, args []string) error {
		schema, err := tfprovider.ProduceProviderSchema()
		if err != nil {
			return err
		}
		return golang.Generate(schema)
	},
}

func main() {
	genCommand.AddCommand(jsonschemaCommand)
	genCommand.AddCommand(gostructCommand)
	command.AddCommand(genCommand)
	command.Execute()
}
