package template

import (
	"context"
	"encoding/json"
	"fmt"
	"os"

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
	schemaBytes, err := os.ReadFile(schemaPath)
	if err != nil {
		return nil, err
	}
	schema := &jsonschema.Schema{}
	err = json.Unmarshal(schemaBytes, schema)
	if err != nil {
		return nil, err
	}

	// Return config
	return &config{
		ctx:    ctx,
		schema: schema,
		values: make(map[string]any, 0),
	}, nil
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

	// Cast any integer properties, from float to integer.
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

	// Write configs from files to input map, not overwriting any existing
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

// Prompts user for properties that do not have a value defined yet
func (c *config) promptForValues() error {
	for name, property := range c.schema.Properties {
		// Config already has a value assigned
		if _, ok := c.values[name]; ok {
			continue
		}

		// Initialize Prompt dialog
		var err error
		prompt := cmdio.Prompt(c.ctx)
		prompt.Label = property.Description
		prompt.AllowEdit = true

		// Compute default value to display by converting it to a string
		if property.Default != nil {
			prompt.Default, err = toString(property.Default, property.Type)
			if err != nil {
				return err
			}
		}

		// Get user input by running the prompt
		userInput, err := prompt.Run()
		if err != nil {
			return err
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
// TODO: smoke test for this function
func (c *config) validate() error {
	validateFns := []func() error{
		c.validateValuesDefined,
		c.validateValuesType,
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
