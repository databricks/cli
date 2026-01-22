package dsc

import (
	"context"
	"encoding/json"
	"fmt"
	"reflect"

	"github.com/databricks/cli/libs/cmdctx"
	"github.com/databricks/databricks-sdk-go"
)

// ============================================================================
// Workspace Client Helper
// ============================================================================

func getWorkspaceClient(ctx ResourceContext) (context.Context, *databricks.WorkspaceClient) {
	cmdCtx := ctx.Cmd.Context()
	return cmdCtx, cmdctx.WorkspaceClient(cmdCtx)
}

// ============================================================================
// Input parsing and validation
// ============================================================================

func unmarshalInput[T any](input json.RawMessage) (T, error) {
	var req T
	if err := json.Unmarshal(input, &req); err != nil {
		return req, fmt.Errorf("failed to parse input: %w", err)
	}
	return req, nil
}

type RequiredField struct {
	Name  string
	Value string
}

func validateRequired(fields ...RequiredField) error {
	for _, f := range fields {
		if f.Value == "" {
			return fmt.Errorf("%s is required", f.Name)
		}
	}
	return nil
}

func validateAtLeastOne(description string, values ...string) error {
	for _, v := range values {
		if v != "" {
			return nil
		}
	}
	return fmt.Errorf("either %s is required", description)
}

// ============================================================================
// Metadata Builder
// ============================================================================

type MetadataConfig struct {
	ResourceType      string
	Version           string
	Description       string
	SchemaDescription string
	ResourceName      string
	Tags              []string
	Descriptions      PropertyDescriptions
	SchemaType        reflect.Type
}

func DefaultExitCodes() map[string]string {
	return map[string]string{
		"0": "Success",
		"1": "Error",
	}
}

func BuildMetadata(cfg MetadataConfig) ResourceMetadata {
	schema, _ := GenerateSchemaWithOptions(cfg.SchemaType, SchemaOptions{
		Descriptions:      cfg.Descriptions,
		SchemaDescription: cfg.SchemaDescription,
		ResourceName:      cfg.ResourceName,
	})

	version := cfg.Version
	if version == "" {
		version = "0.1.0"
	}

	return ResourceMetadata{
		Type:        cfg.ResourceType,
		Version:     version,
		Description: cfg.Description,
		Tags:        cfg.Tags,
		ExitCodes:   DefaultExitCodes(),
		Schema: ResourceSchema{
			Embedded: schema,
		},
	}
}

func NotFoundError(resourceType string, identifiers ...string) error {
	return fmt.Errorf("%s not found: %v", resourceType, identifiers)
}

// ============================================================================
// Output helpers
// ============================================================================

func renderJSON(ctx ResourceContext, v any) error {
	data, err := json.Marshal(v)
	if err != nil {
		return fmt.Errorf("failed to marshal output: %w", err)
	}
	ctx.Cmd.Println(string(data))
	return nil
}
