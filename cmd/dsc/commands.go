package dsc

import (
	"github.com/databricks/cli/cmd/root"
	"github.com/spf13/cobra"
)

const dscSubcommandHelpTemplate = `
Description:
  {{.Long}}

Usage:
  {{.UseLine}}

Options:
{{.LocalFlags.FlagUsages | trimTrailingWhitespaces}}
`

func newGetCmd() *cobra.Command {
	var resourceType string
	var input string

	cmd := &cobra.Command{
		Use:   "get",
		Short: "Get the current state of a resource",
		Long:  `Retrieves and returns the current state of a specified resource.`,
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

			return renderJSON(ctx, result)
		},
	}

	cmd.SetHelpTemplate(dscSubcommandHelpTemplate)
	cmd.Flags().StringVarP(&resourceType, "resource", "r", "", "Specify the DSC resource type (REQUIRED)")
	cmd.Flags().StringVarP(&input, "input", "i", "", "JSON input for the resource instance (REQUIRED)")
	_ = cmd.MarkFlagRequired("resource")
	_ = cmd.MarkFlagRequired("input")

	return cmd
}

func newSetCmd() *cobra.Command {
	var resourceType string
	var input string

	cmd := &cobra.Command{
		Use:   "set",
		Short: "Create or update a resource to match desired state",
		Long:  `Creates a new resource or updates an existing one to match the specified desired state.`,
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
			return handler.Set(ctx, jsonInput)
		},
	}

	cmd.SetHelpTemplate(dscSubcommandHelpTemplate)
	cmd.Flags().StringVarP(&resourceType, "resource", "r", "", "Specify the DSC resource type (REQUIRED)")
	cmd.Flags().StringVarP(&input, "input", "i", "", "JSON input for the resource instance (REQUIRED)")
	_ = cmd.MarkFlagRequired("resource")
	_ = cmd.MarkFlagRequired("input")

	return cmd
}

func newDeleteCmd() *cobra.Command {
	var resourceType string
	var input string

	cmd := &cobra.Command{
		Use:   "delete",
		Short: "Delete a resource",
		Long:  `Removes the specified resource.`,
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

	cmd.SetHelpTemplate(dscSubcommandHelpTemplate)
	cmd.Flags().StringVarP(&resourceType, "resource", "r", "", "Specify the DSC resource type (REQUIRED)")
	cmd.Flags().StringVarP(&input, "input", "i", "", "JSON input for the resource instance (REQUIRED)")
	_ = cmd.MarkFlagRequired("resource")
	_ = cmd.MarkFlagRequired("input")

	return cmd
}

func newExportCmd() *cobra.Command {
	var resourceType string

	cmd := &cobra.Command{
		Use:   "export",
		Short: "Export all resources of a type",
		Long:  `Lists and exports all resources of the specified type.`,
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

			return renderJSON(ctx, result)
		},
	}

	cmd.SetHelpTemplate(dscSubcommandHelpTemplate)
	cmd.Flags().StringVarP(&resourceType, "resource", "r", "", "Specify the DSC resource type (REQUIRED)")
	_ = cmd.MarkFlagRequired("resource")

	return cmd
}

func newSchemaCmd() *cobra.Command {
	var resourceType string

	cmd := &cobra.Command{
		Use:   "schema",
		Short: "Get the JSON schema for a resource type",
		Long:  `Returns the JSON schema that describes the input format for a resource type.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			metadata, err := getResourceMetadata(resourceType)
			if err != nil {
				return err
			}

			ctx := ResourceContext{Cmd: cmd}
			return renderJSON(ctx, metadata.Schema.Embedded)
		},
	}

	cmd.SetHelpTemplate(dscSubcommandHelpTemplate)
	cmd.Flags().StringVarP(&resourceType, "resource", "r", "", "Specify the DSC resource type (REQUIRED)")
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
		Long:  `Returns the Microsoft DSC resource manifest(s) in JSON format.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := ResourceContext{Cmd: cmd}
			if resourceType != "" {
				// Return single resource manifest
				metadata, err := getResourceMetadata(resourceType)
				if err != nil {
					return err
				}
				manifest := buildManifest(resourceType, metadata)
				return renderJSON(ctx, manifest)
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

			return renderJSON(ctx, ManifestOutput{Resources: manifests})
		},
	}

	cmd.SetHelpTemplate(dscSubcommandHelpTemplate)
	cmd.Flags().StringVarP(&resourceType, "resource", "r", "", "Specify the DSC resource type (optional)")

	return cmd
}
