package template

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"slices"

	"github.com/databricks/cli/libs/cmdio"
	"github.com/databricks/cli/libs/jsonschema"
)

type config struct {
	ctx    context.Context
	values map[string]any
	schema *jsonschema.Schema
}

func newConfig(ctx context.Context, schemaPath string) (*config, error) {
	// Read config schema
	schema, err := jsonschema.Load(schemaPath)
	if err != nil {
		return nil, err
	}
	if err := validateSchema(schema); err != nil {
		return nil, err
	}

	// Return config
	return &config{
		ctx:    ctx,
		schema: schema,
		values: make(map[string]any, 0),
	}, nil
}

func validateSchema(schema *jsonschema.Schema) error {
	for _, v := range schema.Properties {
		if v.Type == jsonschema.ArrayType || v.Type == jsonschema.ObjectType {
			return fmt.Errorf("property type %s is not supported by bundle templates", v.Type)
		}
	}
	return nil
}

// Reads json file at path and assigns values from the file
func (c *config) assignValuesFromFile(path string) error {
	// Read the config file
	configFromFile := make(map[string]any, 0)
	b, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	err = json.Unmarshal(b, &configFromFile)
	if err != nil {
		return err
	}

	// Cast any integer properties, from float to integer. Required because
	// the json unmarshaller treats all json numbers as floating point
	for name, floatVal := range configFromFile {
		property, ok := c.schema.Properties[name]
		if !ok {
			return fmt.Errorf("%s is not defined as an input parameter for the template", name)
		}
		if property.Type != jsonschema.IntegerType {
			continue
		}
		v, err := toInteger(floatVal)
		if err != nil {
			return fmt.Errorf("failed to cast value %v of property %s from file %s to an integer: %w", floatVal, name, path, err)
		}
		configFromFile[name] = v
	}

	// Write configs from the file to the input map, not overwriting any existing
	// configurations.
	for name, val := range configFromFile {
		if _, ok := c.values[name]; ok {
			continue
		}
		c.values[name] = val
	}
	return nil
}

// Assigns default values from schema to input config map
func (c *config) assignDefaultValues() error {
	for name, property := range c.schema.Properties {
		// Config already has a value assigned
		if _, ok := c.values[name]; ok {
			continue
		}

		// No default value defined for the property
		if property.Default == nil {
			continue
		}

		// Assign default value if property is not an integer
		if property.Type != jsonschema.IntegerType {
			c.values[name] = property.Default
			continue
		}

		// Cast default value to int before assigning to an integer configuration.
		// Required because untyped field Default will read all numbers as floats
		// during unmarshalling
		v, err := toInteger(property.Default)
		if err != nil {
			return fmt.Errorf("failed to cast default value %v of property %s to an integer: %w", property.Default, name, err)
		}
		c.values[name] = v
	}
	return nil
}

// Prompts user for values for properties that do not have a value set yet
func (c *config) promptForValues() error {
	for name, property := range c.schema.Properties {
		// Config already has a value assigned
		if _, ok := c.values[name]; ok {
			continue
		}

		// Compute default value to display by converting it to a string
		var defaultVal string
		var err error
		if property.Default != nil {
			defaultVal, err = toString(property.Default, property.Type)
			if err != nil {
				return err
			}
		}

		// Get user input by running the prompt
		var userInput string
		if property.Enum != nil {
			// convert list of enums to string slice
			enums, err := toStringSlice(property.Enum, property.Type)
			if err != nil {
				return err
			}
			userInput, err = cmdio.AskSelect(c.ctx, property.Description, enums)
			if err != nil {
				return err
			}
		} else {
			userInput, err = cmdio.Ask(c.ctx, property.Description, defaultVal)
			if err != nil {
				return err
			}

		}

		// Convert user input string back to a value
		c.values[name], err = fromString(userInput, property.Type)
		if err != nil {
			return err
		}
	}
	return nil
}

// Prompt user for any missing config values. Assign default values if
// terminal is not TTY
func (c *config) promptOrAssignDefaultValues() error {
	if cmdio.IsOutTTY(c.ctx) && cmdio.IsInTTY(c.ctx) {
		return c.promptForValues()
	}
	return c.assignDefaultValues()
}

// Validates the configuration. If passes, the configuration is ready to be used
// to initialize the template.
func (c *config) validate() error {
	validateFns := []func() error{
		c.validateValuesDefined,
		c.validateValuesType,
		c.validateEnumValues,
	}

	for _, fn := range validateFns {
		err := fn()
		if err != nil {
			return err
		}
	}
	return nil
}

// Validates all input properties have a user defined value assigned to them
func (c *config) validateValuesDefined() error {
	for k := range c.schema.Properties {
		if _, ok := c.values[k]; ok {
			continue
		}
		return fmt.Errorf("no value has been assigned to input parameter %s", k)
	}
	return nil
}

// Validates the types of all input properties values match their types defined in the schema
func (c *config) validateValuesType() error {
	for k, v := range c.values {
		fieldInfo, ok := c.schema.Properties[k]
		if !ok {
			return fmt.Errorf("%s is not defined as an input parameter for the template", k)
		}
		err := validateType(v, fieldInfo.Type)
		if err != nil {
			return fmt.Errorf("incorrect type for %s. %w", k, err)
		}
	}
	return nil
}

func (c *config) validateEnumValues() error {
	for k, schema := range c.schema.Properties {
		if schema.Enum == nil {
			// skip if property is not enum
			continue
		}
		if !slices.Contains(schema.Enum, c.values[k]) {
			enums, err := toStringSlice(schema.Enum, schema.Type)
			if err != nil {
				return err
			}
			valueString, err := toString(c.values[k], schema.Type)
			if err != nil {
				return err
			}
			return fmt.Errorf("expect value of property %q to be one of %s. Found: %s", k, strings.Join(enums, ", "), valueString)
		}
	}
	return nil
}
