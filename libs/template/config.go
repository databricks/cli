package template

import (
	"context"
	"errors"
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
		defaultVal, err := property.DefaultString()
		if err != nil {
			return err
		}
		defaultVal, err = r.executeTemplate(defaultVal)
		if err != nil {
			return err
		}
		defaultValTyped, err := property.ParseString(defaultVal)
		if err != nil {
			return err
		}
		c.values[name] = defaultValTyped
	}
	return nil
}

func (c *config) promptSelect(property jsonschema.Property, r *renderer) error {
	name := property.Name
	schema := property.Schema

	// Compute description
	description, err := r.executeTemplate(schema.Description)
	if err != nil {
		return err
	}

	// List of options to display to user
	options, err := schema.EnumStringSlice()
	if err != nil {
		return err
	}

	// Get user input
	userInput, err := cmdio.AskSelect(c.ctx, description, options)
	if err != nil {
		return err
	}

	// Convert user input string back to a value
	c.values[name], err = schema.ParseString(userInput)
	if err != nil {
		return err
	}

	// Validate the partial config based on this update
	return c.schema.ValidateInstance(c.values)
}

func (c *config) promptText(property jsonschema.Property, r *renderer) error {
	name := property.Name
	schema := property.Schema

	// Compute description
	description, err := r.executeTemplate(schema.Description)
	if err != nil {
		return err
	}

	// Compute default value
	defaultRaw, err := schema.DefaultString()
	if err != nil {
		return err
	}
	defaultVal, err := r.executeTemplate(defaultRaw)
	if err != nil {
		return err
	}

	for {
		// Get user input. Parse the string back to a value
		userInput, err := cmdio.Ask(c.ctx, description, defaultVal)
		if err != nil {
			return err
		}

		// Convert user input string back to a Go value
		c.values[name], err = schema.ParseString(userInput)
		// Error parsing user input. Retry if parsing fails.
		if err != nil {
			target := &jsonschema.ParseStringError{}
			if errors.As(err, target) {
				cmdio.LogString(c.ctx, fmt.Sprintf("Validation failed: %q is not a %s", target.Value, target.ExpectedType))
			} else {
				cmdio.LogString(c.ctx, fmt.Sprintf("Validation failed: %s", err.Error()))
			}
			continue
		}

		// Validate the partial config based on this new value by the user
		err = c.schema.ValidateInstance(c.values)
		// Error validating user input. Retry if validation fails.
		if err != nil {
			target := &jsonschema.PatternMatchFailedError{}
			if errors.As(err, target) {
				cmdio.LogString(c.ctx, fmt.Sprintf("Validation failed: %q is expected to match regex pattern %s. %s", target.PropertyValue, target.Pattern, target.FailureMessage))
			} else {
				cmdio.LogString(c.ctx, fmt.Sprintf("Validation failed: %s", target.Error()))
			}
			continue
		}
		return err
	}
}

func (c *config) prompt(property jsonschema.Property, r *renderer) error {
	if property.Schema.Enum != nil {
		return c.promptSelect(property, r)
	}
	return c.promptText(property, r)
}

// Prompts user for values for properties that do not have a value set yet
func (c *config) promptForValues(r *renderer) error {
	for _, p := range c.schema.OrderedProperties() {
		err := c.prompt(p, r)
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
	// For final validation, all properties in the JSON schema should have a value defined.
	c.schema.Required = maps.Keys(c.schema.Properties)
	if err := c.schema.ValidateInstance(c.values); err != nil {
		return fmt.Errorf("validation for template input parameters failed. %w", err)
	}
	return nil
}
