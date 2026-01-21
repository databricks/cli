package dsc

import (
	"github.com/databricks/cli/cmd/root"
	"github.com/databricks/cli/libs/cmdio"
	"github.com/spf13/cobra"
)

func newGetCmd() *cobra.Command {
	var resourceType string
	var input string

	cmd := &cobra.Command{
		Use:   "get",
		Short: "Get the current state of a resource",
		Long: `Get the current state of a resource.

Retrieves and returns the current state of a specified resource.
Input can be provided via the --input flag or piped through stdin.

Examples:
  databricks dsc get --resource Databricks/Secret --input '{"scope": "my-scope", "key": "my-key"}'
  echo '{"scope": "my-scope", "key": "my-key"}' | databricks dsc get --resource Databricks/Secret`,
		PreRunE: root.MustWorkspaceClient,
		RunE: func(cmd *cobra.Command, args []string) error {
			handler, err := getResourceHandler(resourceType)
			if err != nil {
				return err
			}

			jsonInput, err := parseInput(input)
			if err != nil {
				return err
			}

			ctx := ResourceContext{Cmd: cmd}
			result, err := handler.Get(ctx, jsonInput)
			if err != nil {
				return err
			}

			return cmdio.Render(cmd.Context(), result)
		},
	}

	cmd.Flags().StringVar(&resourceType, "resource", "", "Resource type (e.g., Databricks/Secret)")
	cmd.Flags().StringVar(&input, "input", "", "JSON input for the resource")
	_ = cmd.MarkFlagRequired("resource")

	return cmd
}

func newSetCmd() *cobra.Command {
	var resourceType string
	var input string

	cmd := &cobra.Command{
		Use:   "set",
		Short: "Create or update a resource to match desired state",
		Long: `Create or update a resource to match desired state.

Creates a new resource or updates an existing one to match the specified
desired state. Returns the actual state after the operation.

Examples:
  databricks dsc set --resource Databricks/Secret --input '{"scope": "my-scope", "key": "my-key", "stringValue": "secret-value"}'`,
		PreRunE: root.MustWorkspaceClient,
		RunE: func(cmd *cobra.Command, args []string) error {
			handler, err := getResourceHandler(resourceType)
			if err != nil {
				return err
			}

			jsonInput, err := parseInput(input)
			if err != nil {
				return err
			}

			ctx := ResourceContext{Cmd: cmd}
			result, err := handler.Set(ctx, jsonInput)
			if err != nil {
				return err
			}

			return cmdio.Render(cmd.Context(), result)
		},
	}

	cmd.Flags().StringVar(&resourceType, "resource", "", "Resource type (e.g., Databricks/Secret)")
	cmd.Flags().StringVar(&input, "input", "", "JSON input for the resource")
	_ = cmd.MarkFlagRequired("resource")

	return cmd
}

func newDeleteCmd() *cobra.Command {
	var resourceType string
	var input string

	cmd := &cobra.Command{
		Use:   "delete",
		Short: "Delete a resource",
		Long: `Delete a resource.

Removes the specified resource. Returns success if the resource was deleted
or did not exist.

Examples:
  databricks dsc delete --resource Databricks/Secret --input '{"scope": "my-scope", "key": "my-key"}'`,
		PreRunE: root.MustWorkspaceClient,
		RunE: func(cmd *cobra.Command, args []string) error {
			handler, err := getResourceHandler(resourceType)
			if err != nil {
				return err
			}

			jsonInput, err := parseInput(input)
			if err != nil {
				return err
			}

			ctx := ResourceContext{Cmd: cmd}
			return handler.Delete(ctx, jsonInput)
		},
	}

	cmd.Flags().StringVar(&resourceType, "resource", "", "Resource type (e.g., Databricks/Secret)")
	cmd.Flags().StringVar(&input, "input", "", "JSON input for the resource")
	_ = cmd.MarkFlagRequired("resource")

	return cmd
}

func newExportCmd() *cobra.Command {
	var resourceType string

	cmd := &cobra.Command{
		Use:   "export",
		Short: "Export all resources of a type",
		Long: `Export all resources of a type.

Lists and exports all resources of the specified type. This is useful for
discovering existing resources or for backup purposes.

Examples:
  databricks dsc export --resource Databricks/SecretScope`,
		PreRunE: root.MustWorkspaceClient,
		RunE: func(cmd *cobra.Command, args []string) error {
			handler, err := getResourceHandler(resourceType)
			if err != nil {
				return err
			}

			ctx := ResourceContext{Cmd: cmd}
			result, err := handler.Export(ctx)
			if err != nil {
				return err
			}

			return cmdio.Render(cmd.Context(), result)
		},
	}

	cmd.Flags().StringVar(&resourceType, "resource", "", "Resource type (e.g., Databricks/SecretScope)")
	_ = cmd.MarkFlagRequired("resource")

	return cmd
}

func newSchemaCmd() *cobra.Command {
	var resourceType string

	cmd := &cobra.Command{
		Use:   "schema",
		Short: "Get the JSON schema for a resource type",
		Long: `Get the JSON schema for a resource type.

Returns the embedded JSON schema that describes the input format for a
specific resource type. This is useful for validation and tooling integration.

Examples:
  databricks dsc schema --resource Databricks/Secret
  databricks dsc schema --resource Databricks/SecretScope`,
		RunE: func(cmd *cobra.Command, args []string) error {
			metadata, err := getResourceMetadata(resourceType)
			if err != nil {
				return err
			}

			return cmdio.Render(cmd.Context(), metadata.Schema.Embedded)
		},
	}

	cmd.Flags().StringVar(&resourceType, "resource", "", "Resource type (e.g., Databricks/Secret)")
	_ = cmd.MarkFlagRequired("resource")

	return cmd
}

// ManifestOutput represents the full manifest output.
type ManifestOutput struct {
	Resources []ResourceManifest `json:"resources"`
}

func newManifestCmd() *cobra.Command {
	var resourceType string

	cmd := &cobra.Command{
		Use:   "manifest",
		Short: "Get Microsoft DSC resource manifests",
		Long: `Get Microsoft DSC resource manifests.

Returns the full Microsoft DSC resource manifest(s) in JSON format. This can be used
during build processes to generate manifest files, or for Microsoft DSC tooling integration.

When --resource is specified, returns the manifest for that specific resource.
Without --resource, returns manifests for all registered resources.

Examples:
  # Get manifest for all resources
  databricks dsc manifest

  # Get manifest for a specific resource
  databricks dsc manifest --resource Databricks/Secret`,
		RunE: func(cmd *cobra.Command, args []string) error {
			if resourceType != "" {
				// Return single resource manifest
				metadata, err := getResourceMetadata(resourceType)
				if err != nil {
					return err
				}
				manifest := buildManifest(resourceType, metadata)
				return cmdio.Render(cmd.Context(), manifest)
			}

			// Return all manifests
			var manifests []ResourceManifest
			for _, rt := range listResourceTypes() {
				metadata, err := getResourceMetadata(rt)
				if err != nil {
					continue
				}
				manifests = append(manifests, buildManifest(rt, metadata))
			}

			return cmdio.Render(cmd.Context(), ManifestOutput{Resources: manifests})
		},
	}

	cmd.Flags().StringVar(&resourceType, "resource", "", "Resource type (optional, e.g., Databricks/Secret)")

	return cmd
}
