package template

import (
	"context"
	"fmt"

	"github.com/databricks/cli/libs/cmdio"
	"github.com/databricks/cli/libs/jsonschema"
	"golang.org/x/exp/maps"
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

	// Do not allow template input variables that are not defined in the schema.
	schema.AdditionalProperties = false

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
	// Load the config file.
	configFromFile, err := c.schema.LoadInstance(path)
	if err != nil {
		return fmt.Errorf("failed to load config from file %s: %w", path, err)
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
func (c *config) assignDefaultValues(r *renderer) error {
	for name, property := range c.schema.Properties {
		// Config already has a value assigned
		if _, ok := c.values[name]; ok {
			continue
		}
		// No default value defined for the property
		if property.Default == nil {
			continue
		}
		defaultValRaw, err := jsonschema.ToString(property.Default, property.Type)
		if err != nil {
			return err
		}
		defaultVal, err := r.executeTemplate(defaultValRaw)
		if err != nil {
			return err
		}
		c.values[name] = defaultVal
	}
	return nil
}

// Prompts user for values for properties that do not have a value set yet
func (c *config) promptForValues(r *renderer) error {
	for _, p := range c.schema.OrderedProperties() {
		name := p.Name
		property := p.Schema

		// Config already has a value assigned
		if _, ok := c.values[name]; ok {
			continue
		}

		// Compute default value to display by converting it to a string
		var defaultVal string
		var err error
		if property.Default != nil {
			defaultValRaw, err := jsonschema.ToString(property.Default, property.Type)
			if err != nil {
				return err
			}
			defaultVal, err = r.executeTemplate(defaultValRaw)
			if err != nil {
				return err
			}
		}

		description, err := r.executeTemplate(property.Description)
		if err != nil {
			return err
		}

		// Get user input by running the prompt
		var userInput string
		if property.Enum != nil {
			// convert list of enums to string slice
			enums, err := jsonschema.ToStringSlice(property.Enum, property.Type)
			if err != nil {
				return err
			}
			userInput, err = cmdio.AskSelect(c.ctx, description, enums)
			if err != nil {
				return err
			}
		} else {
			userInput, err = cmdio.Ask(c.ctx, description, defaultVal)
			if err != nil {
				return err
			}

		}

		// Convert user input string back to a value
		c.values[name], err = jsonschema.FromString(userInput, property.Type)
		if err != nil {
			return err
		}
	}
	return nil
}

// Prompt user for any missing config values. Assign default values if
// terminal is not TTY
func (c *config) promptOrAssignDefaultValues(r *renderer) error {
	if cmdio.IsOutTTY(c.ctx) && cmdio.IsInTTY(c.ctx) {
		return c.promptForValues(r)
	}
	return c.assignDefaultValues(r)
}

// Validates the configuration. If passes, the configuration is ready to be used
// to initialize the template.
func (c *config) validate() error {
	// All properties in the JSON schema should have a value defined.
	c.schema.Required = maps.Keys(c.schema.Properties)
	if err := c.schema.ValidateInstance(c.values); err != nil {
		return fmt.Errorf("validation for template input parameters failed. %w", err)
	}
	return nil
}
