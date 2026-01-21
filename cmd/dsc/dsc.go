package dsc

import (
	"encoding/json"
	"fmt"
	"io"
	"os"

	"github.com/databricks/cli/cmd/root"
	"github.com/spf13/cobra"
)

func New() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "dsc",
		Short: "Microsoft Desired State Configuration (DSC) operations",
		Long: `Microsoft Desired State Configuration (DSC) operations for Databricks resources.

This command provides a Microsoft DSC-compatible interface for managing Databricks resources,
following the DSC resource manifest pattern used by Microsoft DSC.

Operations:
  get      - Retrieve the current state of a resource
  set      - Create or update a resource to match desired state
  test     - Check if a resource matches desired state (if implemented)
  delete   - Remove a resource
  export   - List/export all resources of a type
  schema   - Get the JSON schema for a resource type
  manifest - Get the full DSC manifest for all resources

All operations accept JSON input via --input flag or stdin and produce JSON output.

Resource categories:
  secret  - Databricks secrets (secrets, scopes, ACLs)`,
		RunE: root.ReportUnknownSubcommand,
	}

	cmd.SetHelpTemplate(`{{.Long}}

Usage:
  {{.UseLine}}
{{if .HasAvailableLocalFlags}}
Flags:
{{.LocalFlags.FlagUsages | trimTrailingWhitespaces}}
{{end}}{{if .HasAvailableInheritedFlags}}
Global Flags:
{{.InheritedFlags.FlagUsages | trimTrailingWhitespaces}}
{{end}}
`)

	cmd.AddCommand(newGetCmd())
	cmd.AddCommand(newSetCmd())
	cmd.AddCommand(newDeleteCmd())
	cmd.AddCommand(newExportCmd())
	cmd.AddCommand(newSchemaCmd())
	cmd.AddCommand(newManifestCmd())

	return cmd
}

// Define the interface for resource operations
type ResourceHandler interface {
	Get(ctx ResourceContext, input json.RawMessage) (any, error)
	Set(ctx ResourceContext, input json.RawMessage) (any, error)
	Delete(ctx ResourceContext, input json.RawMessage) error
	Export(ctx ResourceContext) (any, error)
}

// Define metadata for resources
type ResourceMetadata struct {
	Type        string            `json:"type"`
	Version     string            `json:"version"`
	Description string            `json:"description"`
	Tags        []string          `json:"tags"`
	ExitCodes   map[string]string `json:"exitCodes"`
	Schema      ResourceSchema    `json:"schema"`
}

// Represents the embedded schema
type ResourceSchema struct {
	Embedded any `json:"embedded"`
}

// Represents the resource manifest structure
type ResourceManifest struct {
	Schema      string            `json:"$schema"`
	Type        string            `json:"type"`
	Version     string            `json:"version"`
	Description string            `json:"description"`
	Tags        []string          `json:"tags"`
	ExitCodes   map[string]string `json:"exitCodes"`
	Schema_     ResourceSchema    `json:"schema"`
	Get         *CommandSpec      `json:"get,omitempty"`
	Set         *CommandSpec      `json:"set,omitempty"`
	Delete      *CommandSpec      `json:"delete,omitempty"`
	Export      *CommandSpec      `json:"export,omitempty"`
}

type CommandSpec struct {
	Executable string `json:"executable"`
	Args       []any  `json:"args"`
	Return     string `json:"return,omitempty"`
}

type JSONInputArg struct {
	JSONInputArg string `json:"jsonInputArg"`
	Mandatory    bool   `json:"mandatory"`
}

type ResourceContext struct {
	Cmd *cobra.Command
}

type DSCResult struct {
	ActualState         any      `json:"actualState"`
	InDesiredState      bool     `json:"inDesiredState"`
	DifferingProperties []string `json:"differingProperties,omitempty"`
}

var resourceRegistry = make(map[string]ResourceHandler)
var metadataRegistry = make(map[string]ResourceMetadata)

func RegisterResource(resourceType string, handler ResourceHandler) {
	resourceRegistry[resourceType] = handler
}

func RegisterResourceWithMetadata(resourceType string, handler ResourceHandler, metadata ResourceMetadata) {
	resourceRegistry[resourceType] = handler
	metadataRegistry[resourceType] = metadata
}

func getResourceHandler(resourceType string) (ResourceHandler, error) {
	handler, ok := resourceRegistry[resourceType]
	if !ok {
		return nil, fmt.Errorf("unknown resource type: %s. Available types: %v", resourceType, listResourceTypes())
	}
	return handler, nil
}

func getResourceMetadata(resourceType string) (ResourceMetadata, error) {
	metadata, ok := metadataRegistry[resourceType]
	if !ok {
		return ResourceMetadata{}, fmt.Errorf("no metadata for resource type: %s", resourceType)
	}
	return metadata, nil
}

func listResourceTypes() []string {
	types := make([]string, 0, len(resourceRegistry))
	for t := range resourceRegistry {
		types = append(types, t)
	}
	return types
}

func buildManifest(resourceType string, metadata ResourceMetadata) ResourceManifest {
	inputArg := JSONInputArg{JSONInputArg: "--input", Mandatory: true}

	return ResourceManifest{
		Schema:      "https://aka.ms/dsc/schemas/v3/bundled/resource/manifest.json",
		Type:        resourceType,
		Version:     metadata.Version,
		Description: metadata.Description,
		Tags:        metadata.Tags,
		ExitCodes:   metadata.ExitCodes,
		Schema_:     metadata.Schema,
		Get: &CommandSpec{
			Executable: "databricks",
			Args:       []any{"dsc", "get", "--resource", resourceType, inputArg},
		},
		Set: &CommandSpec{
			Executable: "databricks",
			Args:       []any{"dsc", "set", "--resource", resourceType, inputArg},
		},
		Delete: &CommandSpec{
			Executable: "databricks",
			Args:       []any{"dsc", "delete", "--resource", resourceType, inputArg},
		},
		Export: &CommandSpec{
			Executable: "databricks",
			Args:       []any{"dsc", "export", "--resource", resourceType},
		},
	}
}

func parseInput(inputFlag string) (json.RawMessage, error) {
	var jsonData []byte
	var err error

	if inputFlag != "" {
		jsonData = []byte(inputFlag)
	} else {
		// Check if stdin has data
		stat, _ := os.Stdin.Stat()
		if (stat.Mode() & os.ModeCharDevice) == 0 {
			jsonData, err = io.ReadAll(os.Stdin)
			if err != nil {
				return nil, fmt.Errorf("failed to read from stdin: %w", err)
			}
		}
	}

	if len(jsonData) == 0 {
		return nil, fmt.Errorf("no JSON input provided. Use --input flag or pipe JSON to stdin")
	}

	var raw json.RawMessage
	if err := json.Unmarshal(jsonData, &raw); err != nil {
		return nil, fmt.Errorf("invalid JSON input: %w", err)
	}

	return raw, nil
}
