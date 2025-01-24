package template

import (
	"context"
	"errors"
	"fmt"
	"io/fs"

	"github.com/databricks/cli/libs/cmdio"
	"github.com/databricks/cli/libs/jsonschema"
	"github.com/databricks/cli/libs/log"
	"golang.org/x/exp/maps"
)

// The latest template schema version supported by the CLI
const latestSchemaVersion = 1

type retriableError struct {
	err error
}

func (e retriableError) Error() string {
	return e.err.Error()
}

type config struct {
	ctx    context.Context
	values map[string]any
	schema *jsonschema.Schema
}

func newConfig(ctx context.Context, templateFS fs.FS, schemaPath string) (*config, error) {
	schema, err := jsonschema.LoadFS(templateFS, schemaPath)
	if err != nil {
		return nil, err
	}
	if err := validateSchema(schema); err != nil {
		return nil, err
	}

	// Validate that all properties have a description
	for name, p := range schema.Properties {
		if p.Description == "" {
			return nil, fmt.Errorf("template property %s is missing a description", name)
		}
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
	if schema.Version != nil && *schema.Version > latestSchemaVersion {
		return fmt.Errorf("template schema version %d is not supported by this version of the CLI. Please upgrade your CLI to the latest version", *schema.Version)
	}
	return nil
}

// Reads json file at path and assigns values from the file
func (c *config) assignValuesFromFile(path string) error {
	// It's valid to set additional properties in the config file that are not
	// defined in the schema. They will be filtered below. Thus for the duration of
	// the LoadInstance call, we disable the additional properties check,
	// to allow those properties to be loaded.
	c.schema.AdditionalProperties = true
	configFromFile, err := c.schema.LoadInstance(path)
	c.schema.AdditionalProperties = false

	if err != nil {
		return fmt.Errorf("failed to load config from file %s: %w", path, err)
	}

	// Write configs from the file to the input map, not overwriting any existing
	// configurations.
	for name, val := range configFromFile {
		// If a property is not defined in the schema, skip it.
		if _, ok := c.schema.Properties[name]; !ok {
			continue
		}
		// If a value is already assigned, keep the original value.
		if _, ok := c.values[name]; ok {
			continue
		}
		c.values[name] = val
	}
	return nil
}

// Assigns default values from schema to input config map
func (c *config) assignDefaultValues(r *renderer) error {
	for _, p := range c.schema.OrderedProperties() {
		name := p.Name
		property := p.Schema

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

func (c *config) skipPrompt(p jsonschema.Property, r *renderer) (bool, error) {
	// Config already has a value assigned. We don't have to prompt for a user input.
	if _, ok := c.values[p.Name]; ok {
		return true, nil
	}

	if p.Schema.SkipPromptIf == nil {
		return false, nil
	}

	// Validate the partial config against skip_prompt_if schema
	validationErr := p.Schema.SkipPromptIf.ValidateInstance(c.values)
	if validationErr != nil {
		return false, nil
	}

	if p.Schema.Default == nil {
		return false, fmt.Errorf("property %s has skip_prompt_if set but no default value", p.Name)
	}

	// Assign default value to property if we are skipping it.
	if p.Schema.Type != jsonschema.StringType {
		c.values[p.Name] = p.Schema.Default
		return true, nil
	}

	// Execute the default value as a template and assign it to the property.
	var err error
	c.values[p.Name], err = r.executeTemplate(p.Schema.Default.(string))
	if err != nil {
		return false, err
	}
	return true, nil
}

func (c *config) promptOnce(property *jsonschema.Schema, name, defaultVal, description string) error {
	var userInput string
	if property.Enum != nil {
		// List options for the user to select from
		options, err := property.EnumStringSlice()
		if err != nil {
			return err
		}
		userInput, err = cmdio.AskSelect(c.ctx, description, options)
		if err != nil {
			return err
		}
	} else {
		var err error
		userInput, err = cmdio.Ask(c.ctx, description, defaultVal)
		if err != nil {
			return err
		}
	}

	// Convert user input string back to a Go value
	var err error
	c.values[name], err = property.ParseString(userInput)
	if err != nil {
		// Show error and retry if validation fails
		cmdio.LogString(c.ctx, "Validation failed: "+err.Error())
		return retriableError{err: err}
	}

	// Validate the partial config which includes the new value
	err = c.schema.ValidateInstance(c.values)
	if err != nil {
		// Show error and retry if validation fails
		cmdio.LogString(c.ctx, "Validation failed: "+err.Error())
		return retriableError{err: err}
	}
	return nil
}

// Prompts user for values for properties that do not have a value set yet
func (c *config) promptForValues(r *renderer) error {
	for _, p := range c.schema.OrderedProperties() {
		name := p.Name
		property := p.Schema

		// Skip prompting if we can.
		skip, err := c.skipPrompt(p, r)
		if err != nil {
			return err
		}
		if skip {
			continue
		}

		// Compute default value to display by converting it to a string
		var defaultVal string
		if property.Default != nil {
			defaultValRaw, err := property.DefaultString()
			if err != nil {
				return err
			}
			defaultVal, err = r.executeTemplate(defaultValRaw)
			if err != nil {
				return err
			}
		}

		// Compute description for the prompt
		description, err := r.executeTemplate(property.Description)
		if err != nil {
			return err
		}

		// We wrap this function in a retry loop to allow retries when the user
		// entered value is invalid.
		for {
			err = c.promptOnce(property, name, defaultVal, description)
			if err == nil {
				break
			}
			if !errors.As(err, &retriableError{}) {
				return err
			}
		}
	}
	return nil
}

// Prompt user for any missing config values. Assign default values if
// terminal is not TTY
func (c *config) promptOrAssignDefaultValues(r *renderer) error {
	// TODO: replace with IsPromptSupported call (requires fixing TestAccBundleInitErrorOnUnknownFields test)
	if cmdio.IsOutTTY(c.ctx) && cmdio.IsInTTY(c.ctx) && !cmdio.IsGitBash(c.ctx) {
		return c.promptForValues(r)
	}
	log.Debugf(c.ctx, "Terminal is not TTY. Assigning default values to template input parameters")
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
