package apps

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"github.com/databricks/cli/libs/apps/manifest"
	"github.com/spf13/cobra"
)

// runManifestOnly resolves the template, loads appkit.plugins.json if present, and prints it to stdout (or a message if not found).
func runManifestOnly(ctx context.Context, templatePath, branch, version string) error {
	templateSrc := templatePath
	if templateSrc == "" {
		templateSrc = os.Getenv(templatePathEnvVar)
	}
	gitRef := branch
	usingDefaultTemplate := templateSrc == ""
	if usingDefaultTemplate {
		switch {
		case branch != "":
		case version != "":
			gitRef = normalizeVersion(version)
		default:
			gitRef = appkitDefaultVersion
		}
		templateSrc = appkitRepoURL
	}

	branchForClone := branch
	subdirForClone := ""
	if usingDefaultTemplate {
		branchForClone = gitRef
		subdirForClone = appkitTemplateDir
	}
	resolvedPath, cleanup, err := resolveTemplate(ctx, templateSrc, branchForClone, subdirForClone)
	if err != nil {
		return err
	}
	if cleanup != nil {
		defer cleanup()
	}

	templateDir := filepath.Join(resolvedPath, "generic")
	if _, err := os.Stat(templateDir); os.IsNotExist(err) {
		templateDir = resolvedPath
		if _, err := os.Stat(templateDir); os.IsNotExist(err) {
			return fmt.Errorf("template not found at %s (also checked %s/generic)", resolvedPath, resolvedPath)
		}
	}

	if manifest.HasManifest(templateDir) {
		m, err := manifest.Load(templateDir)
		if err != nil {
			return fmt.Errorf("load manifest: %w", err)
		}
		enc, err := json.MarshalIndent(m, "", "  ")
		if err != nil {
			return fmt.Errorf("encode manifest: %w", err)
		}
		fmt.Fprintln(os.Stdout, string(enc))
		return nil
	}

	fmt.Fprintln(os.Stdout, "No appkit.plugins.json manifest found in this template.")
	return nil
}

func newManifestCmd() *cobra.Command {
	var (
		templatePath string
		branch       string
		version      string
	)

	cmd := &cobra.Command{
		Use:    "manifest",
		Short:  "Print template manifest with available plugins and required resources",
		Hidden: true,
		Long: `Resolves a template (default AppKit repo or --template URL), locates appkit.plugins.json,
and prints its contents to stdout. No workspace authentication is required.

Use the same --template, --branch, and --version flags as "databricks apps init" to target
a specific template. Without --template, uses the default AppKit template (main branch).

Examples:
  # Default template manifest
  databricks apps manifest

  # Specific version of default template
  databricks apps manifest --version v0.2.0

  # Manifest from a GitHub repo
  databricks apps manifest --template https://github.com/user/repo --branch main`,
		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := cmd.Context()
			if cmd.Flags().Changed("branch") && cmd.Flags().Changed("version") {
				return errors.New("--branch and --version are mutually exclusive")
			}
			return runManifestOnly(ctx, templatePath, branch, version)
		},
	}

	cmd.Flags().StringVar(&templatePath, "template", "", "Template path (local directory or GitHub URL)")
	cmd.Flags().StringVar(&branch, "branch", "", "Git branch or tag (for GitHub templates, mutually exclusive with --version)")
	cmd.Flags().StringVar(&version, "version", "", "AppKit version for default template (default: main, use 'latest' for main branch)")
	return cmd
}
