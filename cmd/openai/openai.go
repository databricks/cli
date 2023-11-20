package openai

import (
	"encoding/json"
	"fmt"
	"regexp"
	"strings"

	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
)

type OpenAIFunction struct {
	Name        string `json:"name"`
	Description string `json:"description"`
	Parameters  struct {
		Type       string                            `json:"type"`
		Properties map[string]OpenAIFunctionProperty `json:"properties"`
	} `json:"parameters"`
	Required []string `json:"required,omitempty"`
}

type OpenAIFunctionProperty struct {
	Type        string                    `json:"type"`
	Items       *OpenAIFunctionArrayItems `json:"items,omitempty"`
	Description string                    `json:"description,omitempty"`
	Enum        []string                  `json:"enum,omitempty"`
}

type OpenAIFunctionArrayItems struct {
	Type string `json:"type"`
}

// Adapted from the cobra package. Returns true if the default value for this flag represents a zero value.
// func hasZeroDefaultValue(f *pflag.Flag) bool {
// 	switch f.Value.Type() {
// 	case "bool":
// 		// return f.DefValue == "false"
// 		return false // technically incorrect, but usually intended
// 	case "duration":
// 		// Beginning in Go 1.7, duration zero values are "0s"
// 		return f.DefValue == "0" || f.DefValue == "0s"
// 	case "int", "int8", "int32", "int64", "uint", "uint8", "uint32", "uint64", "count", "float32", "float64":
// 		return f.DefValue == "0"
// 	case "string":
// 		return f.DefValue == ""
// 	case "ip", "ipMask", "ipNet":
// 		return f.DefValue == "<nil>"
// 	case "intSlice", "stringSlice", "stringArray":
// 		return f.DefValue == "[]"
// 	default:
// 		switch f.Value.String() {
// 		case "false":
// 			return true
// 		case "<nil>":
// 			return true
// 		case "":
// 			return true
// 		case "0":
// 			return true
// 		}
// 		return false
// 	}
// }

func toJsonSchemaType(flagType string) string {
	switch flagType {
	case "bool":
		return "boolean"
	case "duration", "float32", "float64":
		return "number"
	case "int", "int8", "int32", "int64", "uint", "uint8", "uint32", "uint64", "count":
		return "integer"
	case "string", "ip", "ipMask", "ipNet":
		return "string"
	case "intSlice", "stringSlice", "stringArray":
		return "array"
	case "JSON":
		return "object"
	default:
		return "string"
	}
}

func toArrayItems(flagType string) *OpenAIFunctionArrayItems {
	switch flagType {
	case "intSlice":
		return &OpenAIFunctionArrayItems{Type: "integer"}
	case "stringSlice", "stringArray":
		return &OpenAIFunctionArrayItems{Type: "string"}
	default:
		return nil
	}
}

func generateFunctionDefinitions(definitions *[]OpenAIFunction, c *cobra.Command, root bool) {
	if c.Runnable() {
		fun := OpenAIFunction{}
		nameReplacer := strings.NewReplacer("databricks ", "", " ", "_")
		fun.Name = nameReplacer.Replace(c.CommandPath())
		// if len(fun.Name) > 64 {
		// }
		// if c.Long != "" && len(c.Long) < 512 {
		// 	fun.Description = c.Long
		// } else {
		fun.Description = c.Short
		// }
		descriptionReplacer := strings.NewReplacer("\n", "", "   ", " ")
		fun.Description = descriptionReplacer.Replace(fun.Description)

		fun.Parameters.Type = "object"
		fun.Parameters.Properties = map[string]OpenAIFunctionProperty{}

		// Search for uppercase words with possible "_" symbols in them
		argsRe := regexp.MustCompile(`[\p{Lu}_]+`)
		args := argsRe.FindAllString(c.Use, -1)
		for _, prop := range args {
			fun.Parameters.Properties[prop] = OpenAIFunctionProperty{
				Type:        "string",
				Description: prop,
			}
			fun.Required = append(fun.Required, prop)
		}

		c.Flags().VisitAll(func(flag *pflag.Flag) {
			if flag.Hidden {
				return
			}
			property := OpenAIFunctionProperty{
				Type:        toJsonSchemaType(flag.Value.Type()),
				Items:       toArrayItems(flag.Value.Type()),
				Description: flag.Usage,
			}
			fun.Parameters.Properties[flag.Name] = property
			// Treat all flags as optional, there doesn't seem to be a way to check
			// whether something is required by the command logic or not..
			// if hasZeroDefaultValue(flag) {
			// 	fun.Required = append(fun.Required, flagName)
			// }
		})

		*definitions = append(*definitions, fun)
	}
	for _, child := range c.Commands() {
		if c.IsAvailableCommand() {
			generateFunctionDefinitions(definitions, child, false)
		}
	}
}

func New() *cobra.Command {
	return &cobra.Command{
		Use:   "openai-functions",
		Short: `Databricks function definitions for OpenAI API`,
		RunE: func(cmd *cobra.Command, args []string) error {
			definitions := []OpenAIFunction{}
			generateFunctionDefinitions(&definitions, cmd.Root(), true)
			json, err := json.MarshalIndent(definitions, "", "  ")
			if err != nil {
				return err
			}
			fmt.Println(string(json))
			return nil
		},
	}
}
