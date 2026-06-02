package apps

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"maps"
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"strings"
	"text/template"

	"github.com/charmbracelet/huh"
	"github.com/databricks/cli/cmd/root"
	"github.com/databricks/cli/libs/aitools/agents"
	"github.com/databricks/cli/libs/aitools/installer"
	"github.com/databricks/cli/libs/apps/generator"
	"github.com/databricks/cli/libs/apps/initializer"
	"github.com/databricks/cli/libs/apps/manifest"
	"github.com/databricks/cli/libs/apps/prompt"
	"github.com/databricks/cli/libs/clicompat"
	"github.com/databricks/cli/libs/cmdctx"
	"github.com/databricks/cli/libs/cmdio"
	"github.com/databricks/cli/libs/env"
	"github.com/databricks/cli/libs/git"
	"github.com/databricks/cli/libs/log"
	ignore "github.com/sabhiram/go-gitignore"
	"github.com/spf13/cobra"
	"go.yaml.in/yaml/v3"
)

const (
	templatePathEnvVar   = "DATABRICKS_APPKIT_TEMPLATE_PATH"
	appkitRepoURL        = "https://github.com/databricks/appkit"
	appkitTemplateDir    = "template"
	appkitDefaultBranch  = "main"
	appkitTemplateTagPfx = "template-v"
	defaultProfile       = "DEFAULT"

	// projectNamePlaceholder is the Go template variable used in template
	// directory names to stand in for the user-provided project name.
	projectNamePlaceholder = "{{.project_name}}"

	// bundleConfigFile is the standard bundle configuration filename.
	bundleConfigFile = "databricks.yml"

	// agenticModeEnvVar controls the agentic app creation flow where resources
	// are not provided upfront. When set to "true", --set requirement and
	// resource validation are skipped.
	agenticModeEnvVar = "DATABRICKS_APPS_AGENTIC_MODE"
)

// normalizeVersion converts a version string to the template tag format "template-vX.X.X".
// Examples: "0.3.0" -> "template-v0.3.0", "v0.3.0" -> "template-v0.3.0",
// "template-v0.3.0" -> "template-v0.3.0", "latest" -> "main"
func normalizeVersion(version string) string {
	if version == "" {
		return version
	}
	if version == "latest" {
		return appkitDefaultBranch
	}
	if strings.HasPrefix(version, appkitTemplateTagPfx) {
		return version
	}
	if strings.HasPrefix(version, "v") {
		return appkitTemplateTagPfx + version[1:]
	}
	if version[0] >= '0' && version[0] <= '9' {
		return appkitTemplateTagPfx + version
	}
	return version
}

func newInitCmd() *cobra.Command {
	var (
		templatePath string
		branch       string
		version      string
		name         string
		warehouseID  string
		description  string
		outputDir    string
		pluginsFlag  []string
		deploy       bool
		run          string
		setValues    []string
		autoApprove  bool
		skipInstall  bool
	)

	cmd := &cobra.Command{
		Use:   "init",
		Short: "Initialize a new AppKit application from a template",
		Long: `Initialize a new AppKit application from a template.

When run without arguments, uses the default AppKit template and an interactive prompt
guides you through the setup. When run with --name, runs in non-interactive mode
(all required flags must be provided).

By default, the command uses the latest released version of AppKit. Use --version
to specify a different version, or --version latest to use the main branch.

Examples:
  # Interactive mode with default template (recommended)
  databricks apps init

  # Use a specific AppKit version
  databricks apps init --version v0.2.0

  # Use the latest development version (main branch)
  databricks apps init --version latest

  # Non-interactive with flags
  databricks apps init --name my-app

  # With analytics feature and SQL Warehouse
  databricks apps init --name my-app --features=analytics \
    --set analytics.sql-warehouse.id=abc123

  # With database resource (all fields required together)
  databricks apps init --name my-app --features=analytics \
    --set analytics.database.instance_name=myinst \
    --set analytics.database.database_name=mydb

  # Multiple plugins with different warehouses
  databricks apps init --name my-app --features=analytics,reporting \
    --set analytics.sql-warehouse.id=wh1 \
    --set reporting.sql-warehouse.id=wh2

  # Create, deploy, and run with dev-remote
  databricks apps init --name my-app --deploy --run=dev-remote

  # With a custom template from a local path
  databricks apps init --template /path/to/template --name my-app

  # With a GitHub URL
  databricks apps init --template https://github.com/user/repo --name my-app

Resource configuration (--set):
  Set resource values using --set plugin.resourceKey.field=value
  Keys are defined in the template's appkit.plugins.json manifest.
  Multi-field resources (e.g., database, secret) require all fields to be set together.

Environment variables:
  DATABRICKS_APPKIT_TEMPLATE_PATH  Override the default template source`,
		Args:    cobra.NoArgs,
		PreRunE: root.MustWorkspaceClient,
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := cmd.Context()

			// Validate mutual exclusivity of --branch and --version
			if cmd.Flags().Changed("branch") && cmd.Flags().Changed("version") {
				return errors.New("--branch and --version are mutually exclusive")
			}

			return runCreate(ctx, createOptions{
				templatePath:   templatePath,
				branch:         branch,
				version:        version,
				name:           name,
				nameProvided:   cmd.Flags().Changed("name"),
				warehouseID:    warehouseID,
				description:    description,
				outputDir:      outputDir,
				plugins:        pluginsFlag,
				deploy:         deploy,
				deployChanged:  cmd.Flags().Changed("deploy"),
				run:            run,
				runChanged:     cmd.Flags().Changed("run"),
				pluginsChanged: cmd.Flags().Changed("features") || cmd.Flags().Changed("plugins"),
				setValues:      setValues,
				autoApprove:    autoApprove,
				skipInstall:    skipInstall,
			})
		},
	}

	cmd.Flags().StringVar(&templatePath, "template", "", "Template path (local directory or GitHub URL)")
	cmd.Flags().StringVar(&branch, "branch", "", "Git branch or tag (for GitHub templates, mutually exclusive with --version)")
	cmd.Flags().StringVar(&version, "version", "", "AppKit version to use (default: auto-detected, use 'latest' for main branch)")
	cmd.Flags().StringVar(&name, "name", "", "Project name (prompts if not provided)")
	cmd.Flags().StringVar(&warehouseID, "warehouse-id", "", "SQL warehouse ID")
	_ = cmd.Flags().MarkDeprecated("warehouse-id", "use --set <plugin>.sql-warehouse.id=<value> instead")
	cmd.Flags().StringArrayVar(&setValues, "set", nil, "Set resource values (format: plugin.resourceKey.field=value, can specify multiple)")
	cmd.Flags().StringVar(&description, "description", "", "App description")
	cmd.Flags().StringVar(&outputDir, "output-dir", "", "Directory to write the project to")
	cmd.Flags().StringSliceVar(&pluginsFlag, "features", nil, "Features/plugins to enable (comma-separated, as defined in template manifest)")
	cmd.Flags().StringSliceVar(&pluginsFlag, "plugins", nil, "Alias for --features")
	_ = cmd.Flags().MarkHidden("plugins")
	cmd.Flags().BoolVar(&deploy, "deploy", false, "Deploy the app after creation")
	cmd.Flags().StringVar(&run, "run", "", "Run the app after creation (none, dev, dev-remote)")
	cmd.Flags().BoolVar(&autoApprove, "auto-approve", false, "Skip confirmation prompts for optional resources. Optional resources are only configured when their values are provided via --set.")
	cmd.Flags().BoolVar(&skipInstall, "skip-install", false, "Skip installing project dependencies (e.g. npm install / uv sync). Cannot be combined with --run.")

	return cmd
}

type createOptions struct {
	templatePath   string
	branch         string
	version        string
	name           string
	nameProvided   bool // true if --name flag was explicitly set (enables "flags mode")
	warehouseID    string
	description    string
	outputDir      string
	plugins        []string
	deploy         bool
	deployChanged  bool // true if --deploy flag was explicitly set
	run            string
	runChanged     bool     // true if --run flag was explicitly set
	pluginsChanged bool     // true if --plugins flag was explicitly set
	setValues      []string // --set plugin.resourceKey.field=value pairs
	autoApprove    bool
	skipInstall    bool
}

// parseSetValues parses --set key=value pairs into the resourceValues map.
// Keys use the format "plugin.resourceKey.field=value".
// Validates that plugin names, resource keys, and field names exist in the manifest.
func parseSetValues(setValues []string, m *manifest.Manifest) (map[string]string, error) {
	rv := make(map[string]string)
	for _, sv := range setValues {
		key, value, ok := strings.Cut(sv, "=")
		if !ok || key == "" {
			return nil, fmt.Errorf("invalid --set format %q, expected plugin.resourceKey.field=value", sv)
		}
		parts := strings.SplitN(key, ".", 3)
		if len(parts) != 3 {
			return nil, fmt.Errorf("invalid --set key %q, expected plugin.resourceKey.field", key)
		}
		pluginName, resourceKey, fieldName := parts[0], parts[1], parts[2]

		plugin := m.GetPluginByName(pluginName)
		if plugin == nil {
			return nil, fmt.Errorf("unknown plugin %q in --set %q; available: %v", pluginName, sv, m.GetPluginNames())
		}

		if !pluginHasResourceField(plugin, resourceKey, fieldName) {
			return nil, fmt.Errorf("plugin %q has no resource with key %q and field %q", pluginName, resourceKey, fieldName)
		}

		rv[resourceKey+"."+fieldName] = value
	}

	// Validate multi-field resources: if any user-provided field is set, all user-provided fields must be set.
	// Fields with BundleIgnore or LocalOnly are auto-populated and exempt from this check.
	for _, p := range m.GetPlugins() {
		for _, r := range append(p.Resources.Required, p.Resources.Optional...) {
			if len(r.Fields) <= 1 {
				continue
			}
			names := r.FieldNames()
			setCount := 0
			totalCheckable := 0
			for _, fn := range names {
				if r.Fields[fn].BundleIgnore || r.Fields[fn].LocalOnly {
					continue
				}
				totalCheckable++
				if rv[r.Key()+"."+fn] != "" {
					setCount++
				}
			}
			if setCount > 0 && setCount < totalCheckable {
				var missing []string
				for _, fn := range names {
					if r.Fields[fn].BundleIgnore || r.Fields[fn].LocalOnly {
						continue
					}
					if rv[r.Key()+"."+fn] == "" {
						missing = append(missing, r.Key()+"."+fn)
					}
				}
				return nil, fmt.Errorf("incomplete resource %q: missing fields %v (all fields must be set together)", r.Key(), missing)
			}
		}
	}

	return rv, nil
}

// pluginHasResourceField checks whether a plugin declares a resource with the given key and field name.
func pluginHasResourceField(p *manifest.Plugin, resourceKey, fieldName string) bool {
	for _, r := range append(p.Resources.Required, p.Resources.Optional...) {
		if r.Key() == resourceKey {
			if _, ok := r.Fields[fieldName]; ok {
				return true
			}
		}
	}
	return false
}

// validateRequiredResources checks that all required resources have at least one
// value in resourceValues. Returns an error with a --set hint if any are missing.
func validateRequiredResources(resources []manifest.Resource, resourceValues map[string]string) error {
	for _, r := range resources {
		found := false
		for k := range resourceValues {
			if strings.HasPrefix(k, r.Key()+".") {
				found = true
				break
			}
		}
		if !found {
			fieldHint := "id"
			if names := r.FieldNames(); len(names) > 0 {
				fieldHint = names[0]
			}
			return fmt.Errorf("missing required resource %q for selected plugins (use --set %s.%s.%s=value)", r.Alias, r.PluginName, r.Key(), fieldHint)
		}
	}
	return nil
}

// tmplBundle holds the generated bundle configuration strings.
type tmplBundle struct {
	Variables       string
	Resources       string
	TargetVariables string
}

// dotEnvVars holds the generated .env file content.
type dotEnvVars struct {
	Content string
	Example string
}

// pluginVar represents a selected plugin in template substitution.
// Fields here are part of the AppKit template contract — the template
// reads them via {{$p.Field}} on map values in templateVars.Plugins.
type pluginVar struct {
	// Stability mirrors manifest.Plugin.Stability ("" for GA, "beta"
	// for beta, future tiers preserved). The AppKit template branches
	// imports on this — see databricks/appkit#264 commit d826a532, which
	// routes beta plugins through the `@databricks/appkit/beta` subpath.
	Stability string
}

// templateVars holds the variables for template substitution.
type templateVars struct {
	ProjectName    string
	AppDescription string
	Profile        string
	WorkspaceHost  string
	Bundle         tmplBundle
	DotEnv         dotEnvVars
	AppEnv         string
	// Plugins maps plugin name to its metadata
	// Missing keys return nil, enabling {{if .plugins.analytics}} conditionals.
	Plugins map[string]*pluginVar
}

// parseDeployAndRunFlags parses the deploy and run flag values into typed values.
func parseDeployAndRunFlags(deploy bool, run string) (bool, prompt.RunMode, error) {
	var runMode prompt.RunMode
	switch run {
	case "dev":
		runMode = prompt.RunModeDev
	case "dev-remote":
		runMode = prompt.RunModeDevRemote
	case "", "none":
		runMode = prompt.RunModeNone
	default:
		return false, prompt.RunModeNone, fmt.Errorf("invalid --run value: %q (must be none, dev, or dev-remote)", run)
	}

	// dev-remote requires --deploy because it needs a deployed app to connect to
	if runMode == prompt.RunModeDevRemote && !deploy {
		return false, prompt.RunModeNone, errors.New("--run=dev-remote requires --deploy (dev-remote needs a deployed app to connect to)")
	}

	return deploy, runMode, nil
}

// promptForPluginsAndDeps prompts for plugins and their resource dependencies using the manifest.
// skipDeployRunPrompt indicates whether to skip prompting for deploy/run (because flags were provided).
func promptForPluginsAndDeps(ctx context.Context, m *manifest.Manifest, preSelectedPlugins []string, skipDeployRunPrompt, autoApprove bool) (*prompt.CreateProjectConfig, error) {
	config := &prompt.CreateProjectConfig{
		Dependencies: make(map[string]string),
		Features:     preSelectedPlugins, // Reuse Features field for plugin names
	}
	theme := prompt.AppkitTheme()

	// Eagerly start fetching resources for ALL plugins in the background.
	// This runs while the user is selecting plugins, so by the time resource
	// pickers appear the data is likely already cached.
	allPluginNames := m.GetPluginNames()
	allPossibleResources := m.CollectResources(allPluginNames)
	allPossibleResources = append(allPossibleResources, m.CollectOptionalResources(allPluginNames)...)
	ctx = prompt.PrefetchResources(ctx, allPossibleResources)

	// Step 1: Plugin selection (skip if plugins already provided via flag)
	selectablePlugins := m.GetSelectablePlugins()
	if len(config.Features) == 0 && len(selectablePlugins) > 0 {
		options := make([]huh.Option[string], 0, len(selectablePlugins))
		for _, p := range selectablePlugins {
			label := p.DisplayName + prompt.RenderStabilityTier(p.StabilityLabel()) + " - " + p.Description
			options = append(options, huh.NewOption(label, p.Name))
		}

		var selected []string
		err := huh.NewMultiSelect[string]().
			Title("Select features").
			Description("space to toggle, enter to confirm").
			Options(options...).
			Value(&selected).
			Height(8).
			WithTheme(theme).
			Run()
		if err != nil {
			return nil, err
		}
		if len(selected) == 0 {
			prompt.PrintAnswered(ctx, "Plugins", "None")
		} else {
			prompt.PrintAnswered(ctx, "Plugins", fmt.Sprintf("%d selected", len(selected)))
		}
		config.Features = selected
	}

	// Always include mandatory plugins.
	config.Features = appendUnique(config.Features, m.GetMandatoryPluginNames()...)

	// Collect resources for the user's actual selection.
	resources := m.CollectResources(config.Features)
	optionalResources := m.CollectOptionalResources(config.Features)

	// Step 2: Prompt for required plugin resource dependencies
	for _, r := range resources {
		values, err := promptForResource(ctx, r, theme, true)
		if err != nil {
			return nil, err
		}
		maps.Copy(config.Dependencies, values)
	}

	// Step 3: Prompt for optional plugin resource dependencies.
	// With --auto-approve, optional resources are skipped here; they're only
	// configured when their values are supplied via --set (merged later).
	if !autoApprove {
		for _, r := range optionalResources {
			values, err := promptForResource(ctx, r, theme, false)
			if err != nil {
				return nil, err
			}
			maps.Copy(config.Dependencies, values)
		}
	}

	// Step 4: Description
	config.Description = prompt.DefaultAppDescription
	err := huh.NewInput().
		Title("Description").
		Placeholder(prompt.DefaultAppDescription).
		Value(&config.Description).
		WithTheme(theme).
		Run()
	if err != nil {
		return nil, err
	}
	if config.Description == "" {
		config.Description = prompt.DefaultAppDescription
	}
	prompt.PrintAnswered(ctx, "Description", config.Description)

	// Step 5: Deploy and run options (skip if any deploy/run flag was provided)
	if !skipDeployRunPrompt {
		config.Deploy, config.RunMode, err = prompt.PromptForDeployAndRun(ctx)
		if err != nil {
			return nil, err
		}
	}

	return config, nil
}

// promptForResource prompts the user for a resource value.
// Returns a map of value keys to values. For single-field resources the key is "resource_key.field".
// For multi-field resources, keys use "resource_key.field_name".
func promptForResource(ctx context.Context, r manifest.Resource, theme *huh.Theme, required bool) (map[string]string, error) {
	if fn, ok := prompt.GetPromptFunc(r.Type); ok {
		if !required {
			var configure bool
			err := huh.NewConfirm().
				Title(fmt.Sprintf("Configure %s?", r.Alias)).
				Description(r.Description + " (optional)").
				Value(&configure).
				WithTheme(theme).
				Run()
			if err != nil {
				return nil, err
			}
			if !configure {
				prompt.PrintAnswered(ctx, r.Alias, "skipped")
				return nil, nil
			}
		}
		return fn(ctx, r, required)
	}

	// Generic text input for unregistered resource types
	var value string
	description := r.Description
	if !required {
		description += " (optional, press enter to skip)"
	}

	input := huh.NewInput().
		Title(r.Alias).
		Description(description).
		Value(&value)

	if required {
		input = input.Validate(func(s string) error {
			if s == "" {
				return errors.New("this field is required")
			}
			return nil
		})
	}

	if err := input.WithTheme(theme).Run(); err != nil {
		return nil, err
	}

	if value == "" && !required {
		prompt.PrintAnswered(ctx, r.Alias, "skipped")
		return nil, nil
	}
	prompt.PrintAnswered(ctx, r.Alias, value)

	// Use composite key from Fields when available.
	names := r.FieldNames()
	if len(names) >= 1 {
		return map[string]string{r.Key() + "." + names[0]: value}, nil
	}
	return map[string]string{r.Key(): value}, nil
}

// cloneRepo clones a git repository to a temporary directory.
func cloneRepo(ctx context.Context, repoURL, branch string) (string, error) {
	tempDir, err := os.MkdirTemp("", "appkit-template-*")
	if err != nil {
		return "", fmt.Errorf("create temp dir: %w", err)
	}

	if err := git.Clone(ctx, repoURL, branch, tempDir); err != nil {
		os.RemoveAll(tempDir)
		return "", err
	}

	return tempDir, nil
}

// resolveTemplate resolves a template synchronously with a spinner.
// Used by commands that don't benefit from background cloning (e.g., manifest).
func resolveTemplate(ctx context.Context, templatePath, branch, subdir string) (string, func(), error) {
	ch := resolveTemplateAsync(ctx, templatePath, branch, subdir)
	return awaitTemplate(ctx, ch, "")
}

// templateResult holds the outcome of a background template resolution.
type templateResult struct {
	path    string
	cleanup func()
	err     error
}

// resolveTemplateAsync starts resolving the template in a background goroutine.
// For local paths this completes immediately; for GitHub URLs it clones the repo.
// The caller reads the result from the returned channel, optionally showing a
// spinner if the clone hasn't finished by the time it's needed.
func resolveTemplateAsync(ctx context.Context, templatePath, branch, subdir string) <-chan templateResult {
	ch := make(chan templateResult, 1)

	// Local path — instant.
	if !strings.HasPrefix(templatePath, "https://") {
		ch <- templateResult{path: templatePath}
		return ch
	}

	repoURL, urlSubdir, urlBranch := git.ParseGitHubURL(templatePath)
	if branch == "" {
		branch = urlBranch
	}
	if subdir == "" {
		subdir = urlSubdir
	}

	go func() {
		tempDir, err := cloneRepo(ctx, repoURL, branch)
		if err != nil {
			ch <- templateResult{err: err}
			return
		}
		cleanup := func() { os.RemoveAll(tempDir) }
		localPath := tempDir
		if subdir != "" {
			localPath = filepath.Join(tempDir, subdir)
		}
		ch <- templateResult{path: localPath, cleanup: cleanup}
	}()

	return ch
}

// awaitTemplate waits for the background clone to finish.
// If the result is already available it returns immediately with a
// checkmark; otherwise it shows a spinner while waiting.
// refLabel, if non-empty (e.g. "version 0.24.0" or "branch feature-x"),
// is appended to spinner and done messages.
func awaitTemplate(ctx context.Context, ch <-chan templateResult, refLabel string) (string, func(), error) {
	suffix := ""
	if refLabel != "" {
		suffix = " (" + refLabel + ")"
	}
	select {
	case res := <-ch:
		// Clone finished while the user was typing — print completion.
		if res.err == nil && res.cleanup != nil {
			prompt.PrintDone(ctx, "Template cloned"+suffix)
		}
		return res.path, res.cleanup, res.err
	default:
		// Still cloning — show a spinner for the remaining wait.
		var res templateResult
		err := prompt.RunWithSpinnerCtx(ctx, "Cloning template"+suffix+"...", func() error {
			res = <-ch
			return res.err
		})
		return res.path, res.cleanup, err
	}
}

// commitInPlace derives the app name from the cwd basename and verifies that
// the cwd is suitable for in-place scaffolding (empty modulo .git).
// Returns the derived app name on success.
func commitInPlace() (string, error) {
	appName, err := prompt.DeriveInPlaceAppName(".")
	if err != nil {
		return "", err
	}
	if err := prompt.CheckInPlaceDirectory("."); err != nil {
		return "", err
	}
	return appName, nil
}

// shouldSkipPluginSelection returns true when the template has a plugin
// manifest but no {{.project_name}} subdirectory — meaning plugins are
// pre-baked in the code files (e.g. app-templates/appkit-* repos) and
// the user should not be prompted to select plugins.
func shouldSkipPluginSelection(templateDir string) bool {
	if !manifest.HasManifest(templateDir) {
		return false
	}
	entries, err := os.ReadDir(templateDir)
	if err != nil {
		return false
	}
	for _, e := range entries {
		if e.IsDir() && strings.Contains(e.Name(), projectNamePlaceholder) {
			return false
		}
	}
	return true
}

// replaceProjectName updates the project name in key files after copying a
// pre-rendered template.  It sets bundle.name and the first
// resources.apps.*.name in databricks.yml, and the name field in
// package.json.
func replaceProjectName(destDir, newName string) error {
	// Update package.json name field via JSON round-trip.
	pkgPath := filepath.Join(destDir, "package.json")
	if data, err := os.ReadFile(pkgPath); err == nil {
		var pkg map[string]any
		if err := json.Unmarshal(data, &pkg); err == nil {
			pkg["name"] = newName
			out, err := json.MarshalIndent(pkg, "", "  ")
			if err == nil {
				// Preserve trailing newline convention.
				out = append(out, '\n')
				if err := os.WriteFile(pkgPath, out, 0o644); err != nil {
					return fmt.Errorf("write package.json: %w", err)
				}
			}
		}
	}

	// Update databricks.yml using yaml.Node to preserve comments and formatting.
	ymlPath := filepath.Join(destDir, bundleConfigFile)
	data, err := os.ReadFile(ymlPath)
	if err != nil {
		return err
	}

	var doc yaml.Node
	if err := yaml.Unmarshal(data, &doc); err != nil {
		return fmt.Errorf("parse %s: %w", bundleConfigFile, err)
	}

	setYAMLValue(&doc, []string{"bundle", "name"}, newName)
	setFirstAppName(&doc, newName)

	var buf bytes.Buffer
	enc := yaml.NewEncoder(&buf)
	enc.SetIndent(2)
	if err := enc.Encode(&doc); err != nil {
		return fmt.Errorf("encode %s: %w", bundleConfigFile, err)
	}
	if err := enc.Close(); err != nil {
		return fmt.Errorf("encode %s: %w", bundleConfigFile, err)
	}

	return os.WriteFile(ymlPath, buf.Bytes(), 0o644)
}

// setYAMLValue walks a yaml.Node tree following the given key path and
// replaces the leaf scalar value. It handles both document and mapping nodes.
func setYAMLValue(node *yaml.Node, keys []string, value string) {
	if node.Kind == yaml.DocumentNode && len(node.Content) > 0 {
		setYAMLValue(node.Content[0], keys, value)
		return
	}
	if node.Kind != yaml.MappingNode || len(keys) == 0 {
		return
	}
	for i := 0; i < len(node.Content)-1; i += 2 {
		if node.Content[i].Value == keys[0] {
			if len(keys) == 1 {
				node.Content[i+1].Value = value
				// Clear any explicit style so the encoder picks the
				// simplest representation for the new value.
				node.Content[i+1].Style = 0
				return
			}
			setYAMLValue(node.Content[i+1], keys[1:], value)
			return
		}
	}
}

// setFirstAppName sets the name field of the first app entry under
// resources.apps in a yaml.Node tree.
func setFirstAppName(node *yaml.Node, name string) {
	if node.Kind == yaml.DocumentNode && len(node.Content) > 0 {
		setFirstAppName(node.Content[0], name)
		return
	}
	if node.Kind != yaml.MappingNode {
		return
	}
	// Find resources → apps → first entry → name.
	appsNode := yamlMapLookup(yamlMapLookup(node, "resources"), "apps")
	if appsNode == nil || appsNode.Kind != yaml.MappingNode || len(appsNode.Content) < 2 {
		return
	}
	// First app entry is at Content[1] (Content[0] is the key).
	setYAMLValue(appsNode.Content[1], []string{"name"}, name)
}

// yamlMapLookup returns the value node for a key in a mapping node, or nil.
func yamlMapLookup(node *yaml.Node, key string) *yaml.Node {
	if node == nil || node.Kind != yaml.MappingNode {
		return nil
	}
	for i := 0; i < len(node.Content)-1; i += 2 {
		if node.Content[i].Value == key {
			return node.Content[i+1]
		}
	}
	return nil
}

// findProjectSrcDir locates the actual source directory inside a template.
// Templates may nest their content inside a {{.project_name}} directory.
func findProjectSrcDir(templateDir string) string {
	entries, err := os.ReadDir(templateDir)
	if err != nil {
		return templateDir
	}
	for _, e := range entries {
		if e.IsDir() && strings.Contains(e.Name(), projectNamePlaceholder) {
			return filepath.Join(templateDir, e.Name())
		}
	}
	return templateDir
}

// startBackgroundNpmInstall copies the package files from the template into
// destDir and launches `npm ci` in the background. The caller should await
// the returned channel BEFORE writing other files to destDir to prevent
// concurrent writes. Returns nil if the template is not a Node.js project
// or npm is not available.
//
// IMPORTANT: All reads from srcProjectDir happen synchronously before the
// goroutine launches. The template directory may be cleaned up after this
// function returns, so file reads must not be deferred to the goroutine.
func startBackgroundNpmInstall(ctx context.Context, srcProjectDir, destDir, projectName string) <-chan error {
	lockFile := filepath.Join(srcProjectDir, "package-lock.json")
	if _, err := os.Stat(lockFile); err != nil {
		return nil
	}

	if _, err := exec.LookPath("npm"); err != nil {
		return nil
	}

	if err := os.MkdirAll(destDir, 0o755); err != nil {
		log.Warnf(ctx, "Failed to create %s: %v, skipping background npm install", destDir, err)
		return nil
	}

	// Copy package.json (apply template substitution so the file is valid JSON)
	// and package-lock.json (no template vars — copy raw).
	var pkgWritten bool
	for _, name := range []string{"package.json", "package.json.tmpl"} {
		src := filepath.Join(srcProjectDir, name)
		content, err := os.ReadFile(src)
		if err != nil {
			continue
		}
		minVars := templateData(templateVars{
			ProjectName:    projectName,
			AppDescription: prompt.DefaultAppDescription,
			Plugins:        make(map[string]*pluginVar),
		})
		tmpl, err := template.New(name).Option("missingkey=zero").Parse(string(content))
		if err != nil {
			pkgWritten = os.WriteFile(filepath.Join(destDir, "package.json"), content, 0o644) == nil
			break
		}
		var buf bytes.Buffer
		if err := tmpl.Execute(&buf, minVars); err != nil {
			pkgWritten = os.WriteFile(filepath.Join(destDir, "package.json"), content, 0o644) == nil
			break
		}
		pkgWritten = os.WriteFile(filepath.Join(destDir, "package.json"), buf.Bytes(), 0o644) == nil
		break
	}

	if !pkgWritten {
		log.Warnf(ctx, "Failed to write package.json to %s, skipping background npm install", destDir)
		return nil
	}

	// Copy any file: protocol dependencies (e.g., local .tgz tarballs) so npm ci can resolve them.
	pkgData, err := os.ReadFile(filepath.Join(destDir, "package.json"))
	if err != nil {
		log.Warnf(ctx, "Failed to read package.json for file dep copy: %v", err)
	} else {
		copyFileDeps(ctx, pkgData, srcProjectDir, destDir)
	}

	// Copy package-lock.json raw (never has template vars).
	lockData, err := os.ReadFile(lockFile)
	if err != nil {
		log.Warnf(ctx, "Failed to read package-lock.json: %v, skipping background npm install", err)
		return nil
	}
	if err := os.WriteFile(filepath.Join(destDir, "package-lock.json"), lockData, 0o644); err != nil {
		log.Warnf(ctx, "Failed to write package-lock.json: %v, skipping background npm install", err)
		return nil
	}

	ch := make(chan error, 1)
	go func() {
		cmd := exec.CommandContext(ctx, "npm", "ci", "--no-audit", "--no-fund", "--prefer-offline")
		cmd.Dir = destDir
		cmd.Stdout = nil
		cmd.Stderr = nil
		ch <- cmd.Run()
	}()

	log.Debugf(ctx, "Started background npm install in %s", destDir)
	return ch
}

// copyFileDeps copies local file: protocol dependencies (e.g., .tgz tarballs)
// from srcDir to destDir so that npm ci can resolve them.
func copyFileDeps(ctx context.Context, pkgJSON []byte, srcDir, destDir string) {
	var pkg struct {
		Dependencies    map[string]string `json:"dependencies"`
		DevDependencies map[string]string `json:"devDependencies"`
	}
	if err := json.Unmarshal(pkgJSON, &pkg); err != nil {
		log.Debugf(ctx, "Failed to parse package.json for file dep copy: %v", err)
		return
	}
	for _, deps := range []map[string]string{pkg.Dependencies, pkg.DevDependencies} {
		for _, v := range deps {
			if !strings.HasPrefix(v, "file:") {
				continue
			}
			relPath := filepath.Clean(strings.TrimPrefix(v, "file:"))
			src := filepath.Join(srcDir, relPath)
			data, err := os.ReadFile(src)
			if err != nil {
				log.Debugf(ctx, "Skipping file dep %s: %v", relPath, err)
				continue
			}
			dst := filepath.Join(destDir, relPath)
			if err := os.MkdirAll(filepath.Dir(dst), 0o755); err != nil {
				log.Debugf(ctx, "Failed to create dir for file dep %s: %v", relPath, err)
				continue
			}
			if err := os.WriteFile(dst, data, 0o644); err != nil {
				log.Debugf(ctx, "Failed to copy file dep %s: %v", relPath, err)
			}
		}
	}
}

// awaitBackgroundNpmInstall waits for the background npm install to complete.
// Shows an instant checkmark if already done, or a spinner for the remainder.
func awaitBackgroundNpmInstall(ctx context.Context, ch <-chan error) error {
	select {
	case err := <-ch:
		if err == nil {
			prompt.PrintDone(ctx, "Dependencies installed")
		}
		return err
	default:
		var installErr error
		err := prompt.RunWithSpinnerCtx(ctx, "Installing dependencies...", func() error {
			installErr = <-ch
			return installErr
		})
		return err
	}
}

func runCreate(ctx context.Context, opts createOptions) error {
	// --skip-install leaves the project without installed dependencies, so
	// downstream `--run dev` / `--run dev-remote` would immediately fail.
	// Reject the combination up front rather than after the scaffold runs.
	if opts.skipInstall && opts.run != "" && opts.run != "none" {
		return errors.New("--skip-install cannot be combined with --run (dev/dev-remote require dependencies to be installed)")
	}

	var selectedPlugins []string
	var resourceValues map[string]string
	var shouldDeploy bool
	var runMode prompt.RunMode
	isInteractive := cmdio.IsPromptSupported(ctx)

	// Use plugins from flags if provided
	if len(opts.plugins) > 0 {
		selectedPlugins = opts.plugins
	}

	// Resolve template path (supports local paths and GitHub URLs)
	templateSrc := opts.templatePath
	if templateSrc == "" {
		templateSrc = env.Get(ctx, templatePathEnvVar)
	}

	// Resolve the git reference (branch/tag) to use for default appkit template.
	// refLabel is a human-readable description of the ref we're cloning
	// (e.g. "version 0.24.0", "branch feature-x"). It's surfaced in the
	// interactive header and the clone spinner so the user can cancel before
	// naming the project. Empty when there's nothing meaningful to show
	// (e.g. a custom --template URL with no explicit branch).
	gitRef := opts.branch
	var refLabel string
	usingDefaultTemplate := templateSrc == ""
	if usingDefaultTemplate {
		// Using default appkit template - resolve version
		switch {
		case opts.branch != "":
			// --branch takes precedence (already set in gitRef)
			refLabel = "branch " + opts.branch
		case opts.version != "":
			gitRef = normalizeVersion(opts.version)
			refLabel = "version " + opts.version
		default:
			resolved, err := clicompat.ResolveAppKitVersion(ctx)
			if err != nil {
				return fmt.Errorf("could not resolve AppKit template version: %w; use --version to specify a version manually", err)
			}
			gitRef = normalizeVersion(resolved)
			refLabel = "version " + resolved
		}
		templateSrc = appkitRepoURL
	} else if opts.branch != "" {
		// Custom template with an explicit branch — show it for traceability.
		refLabel = "branch " + opts.branch
	}

	// Start cloning in the background so it runs while the user types the name.
	branchForClone := opts.branch
	subdirForClone := ""
	if usingDefaultTemplate {
		branchForClone = gitRef
		subdirForClone = appkitTemplateDir
	}
	templateCh := resolveTemplateAsync(ctx, templateSrc, branchForClone, subdirForClone)
	defer func() {
		select {
		case res := <-templateCh:
			if res.cleanup != nil {
				res.cleanup()
			}
		default:
		}
	}()

	// Step 1: Get project name (clone runs in parallel for remote templates)
	if opts.name == prompt.InPlaceName && opts.outputDir != "" {
		return prompt.ErrNameDotWithOutputDir
	}

	var (
		destDir string
		inPlace bool
	)
	switch {
	case opts.name == prompt.InPlaceName:
		appName, err := commitInPlace()
		if err != nil {
			return err
		}
		opts.name = appName
		destDir = "."
		inPlace = true
	case opts.name != "":
		if err := prompt.ValidateProjectName(opts.name); err != nil {
			return err
		}
		destDir = opts.name
		if opts.outputDir != "" {
			destDir = filepath.Join(opts.outputDir, opts.name)
		}
		if _, err := os.Stat(destDir); err == nil {
			return fmt.Errorf("directory %s already exists", destDir)
		}
	default:
		if !isInteractive {
			return errors.New("--name is required in non-interactive mode")
		}
		// Print the AppKit header once so it covers both the in-place
		// scaffold-location prompt below and the project-name prompt that
		// may follow, and so the resolved template ref is visible before
		// the user commits to either path.
		prompt.PrintHeader(ctx, refLabel)
		// Offer in-place scaffolding when the current directory is empty
		// (modulo .git) and its basename is a valid app name. Skipped when
		// --output-dir was set, since in-place targets cwd and would silently
		// drop the flag — same reasoning as the --name . / --output-dir mutex
		// above.
		if opts.outputDir == "" {
			if basename, ok := prompt.ShouldOfferInPlace("."); ok {
				useCurrent, err := prompt.PromptScaffoldLocation(ctx, basename)
				if err != nil {
					return err
				}
				if useCurrent {
					// Re-check immediately before committing — the directory may
					// have changed between offer and answer.
					if err := prompt.CheckInPlaceDirectory("."); err != nil {
						return err
					}
					opts.name = basename
					destDir = "."
					inPlace = true
				}
			}
		}
		if !inPlace {
			name, err := prompt.PromptForProjectName(ctx, opts.outputDir)
			if err != nil {
				return err
			}
			if name == prompt.InPlaceName {
				appName, err := commitInPlace()
				if err != nil {
					return err
				}
				opts.name = appName
				destDir = "."
				inPlace = true
			} else {
				opts.name = name
				destDir = name
				if opts.outputDir != "" {
					destDir = filepath.Join(opts.outputDir, name)
				}
			}
		}
	}

	// Step 2: Wait for template (may already be done if the user took time typing the name)
	resolvedPath, cleanup, err := awaitTemplate(ctx, templateCh, refLabel)
	// Only fall back to the embedded version when the version was auto-resolved
	// from the manifest, not when the user explicitly passed --version or --branch.
	versionAutoResolved := opts.version == "" && opts.branch == ""
	if err != nil && usingDefaultTemplate && versionAutoResolved && clicompat.IsNotFoundError(err) {
		fallbackVersion, fbErr := clicompat.ResolveEmbeddedAppKitVersion()
		if fbErr == nil && fallbackVersion != "" && normalizeVersion(fallbackVersion) != branchForClone {
			log.Warnf(ctx, "Template version not found, falling back to embedded version %s", fallbackVersion)
			fallbackRef := normalizeVersion(fallbackVersion)
			templateCh = resolveTemplateAsync(ctx, templateSrc, fallbackRef, appkitTemplateDir)
			refLabel = "version " + fallbackVersion
			resolvedPath, cleanup, err = awaitTemplate(ctx, templateCh, refLabel)
		} else if fbErr != nil {
			log.Warnf(ctx, "Could not resolve embedded AppKit version: %v", fbErr)
		}
	}
	if err != nil {
		return err
	}
	if cleanup != nil {
		defer cleanup()
	}

	// Check for generic subdirectory first (default for multi-template repos)
	templateDir := filepath.Join(resolvedPath, "generic")
	if _, err := os.Stat(templateDir); errors.Is(err, fs.ErrNotExist) {
		// Fall back to the provided path directly
		templateDir = resolvedPath
		if _, err := os.Stat(templateDir); errors.Is(err, fs.ErrNotExist) {
			return fmt.Errorf("template not found at %s (also checked %s/generic)", resolvedPath, resolvedPath)
		}
	}

	// Check whether plugin selection should be skipped (pre-baked plugins
	// in a pre-rendered template with a manifest but no {{.project_name}} dir).
	skipPluginSelection := shouldSkipPluginSelection(templateDir)
	if skipPluginSelection {
		log.Debugf(ctx, "Skipping plugin selection for pre-rendered template at %s", templateDir)
	}

	// Agentic mode: resources are not provided upfront, skip --set
	// requirement and validation.
	agenticMode := env.Get(ctx, agenticModeEnvVar) == "true"

	// Start npm install in the background so it runs while the user answers prompts.
	// This is a Node.js-only optimisation — non-Node templates skip this.
	// Honour --skip-install by not kicking off the background install at all.
	srcProjectDir := findProjectSrcDir(templateDir)
	var npmInstallCh <-chan error
	if !opts.skipInstall {
		npmInstallCh = startBackgroundNpmInstall(ctx, srcProjectDir, destDir, opts.name)
	}

	// Step 3: Load manifest from template (optional — templates without it skip plugin/resource logic)
	var m *manifest.Manifest
	if manifest.HasManifest(templateDir) {
		var err error
		m, err = manifest.Load(templateDir)
		if err != nil {
			return fmt.Errorf("load manifest: %w", err)
		}
		log.Debugf(ctx, "Loaded manifest with %d plugins", len(m.Plugins))
		for name, p := range m.Plugins {
			log.Debugf(ctx, "  Plugin %q: %d required resources, %d optional resources, requiredByTemplate=%v",
				name, len(p.Resources.Required), len(p.Resources.Optional), p.RequiredByTemplate)
		}
	} else {
		log.Debugf(ctx, "No manifest found in template, skipping plugin/resource configuration")
		m = &manifest.Manifest{Plugins: map[string]manifest.Plugin{}}
	}

	// When --name is provided, user is in "flags mode" - use defaults instead of prompting
	flagsMode := opts.nameProvided

	// Skip deploy/run prompts if in flags mode or if deploy/run flags were explicitly set
	skipDeployRunPrompt := flagsMode || opts.deployChanged || opts.runChanged

	if skipPluginSelection {
		// Pre-rendered templates already have their plugins configured in code.
		// Skip plugin/resource prompting entirely — just use mandatory plugins.
		if isInteractive && !skipDeployRunPrompt {
			var err error
			shouldDeploy, runMode, err = prompt.PromptForDeployAndRun(ctx)
			if err != nil {
				return err
			}
		}
	} else if isInteractive && !opts.pluginsChanged && !flagsMode {
		// Interactive mode without --plugins flag: prompt for plugins, dependencies, description
		config, err := promptForPluginsAndDeps(ctx, m, selectedPlugins, skipDeployRunPrompt, opts.autoApprove)
		if err != nil {
			return err
		}
		selectedPlugins = config.Features // Features field holds plugin names
		resourceValues = config.Dependencies
		if config.Description != "" {
			opts.description = config.Description
		}
		if !skipDeployRunPrompt {
			shouldDeploy = config.Deploy
			runMode = config.RunMode
		}
	} else {
		// --plugins flag or flags/non-interactive mode: validate plugin names
		if len(selectedPlugins) > 0 {
			if err := m.ValidatePluginNames(selectedPlugins); err != nil {
				return err
			}
		}
		// Prompt for deploy/run in interactive mode when no flags were set
		if isInteractive && !skipDeployRunPrompt {
			var err error
			shouldDeploy, runMode, err = prompt.PromptForDeployAndRun(ctx)
			if err != nil {
				return err
			}
		}
	}

	// Expand deprecated --warehouse-id into --set values for each plugin that has a sql-warehouse resource.
	if opts.warehouseID != "" {
		for _, p := range m.GetPlugins() {
			for _, r := range append(p.Resources.Required, p.Resources.Optional...) {
				if r.Type == "sql_warehouse" {
					opts.setValues = append(opts.setValues, fmt.Sprintf("%s.%s.id=%s", p.Name, r.Key(), opts.warehouseID))
				}
			}
		}
	}

	// Parse --set values (override any prompted values)
	setVals, err := parseSetValues(opts.setValues, m)
	if err != nil {
		return err
	}
	if len(setVals) > 0 {
		if resourceValues == nil {
			resourceValues = make(map[string]string, len(setVals))
		}
		maps.Copy(resourceValues, setVals)
	}

	// Always include mandatory plugins regardless of user selection or flags.
	selectedPlugins = appendUnique(selectedPlugins, m.GetMandatoryPluginNames()...)

	// Warn when --features adds plugins that the pre-rendered template
	// cannot inject (its server.ts and app.yaml are already finalised).
	if skipPluginSelection && opts.pluginsChanged {
		mandatoryNames := m.GetMandatoryPluginNames()
		mandatory := make(map[string]bool, len(mandatoryNames))
		for _, n := range mandatoryNames {
			mandatory[n] = true
		}
		for _, p := range opts.plugins {
			if !mandatory[p] {
				log.Warnf(ctx, "Adding feature %q to a pre-rendered template is not currently supported.\n"+
					"To add it manually, register the plugin in server/server.ts and run `npx @databricks/appkit plugin sync --write`.\n"+
					"To use all features dynamically, run `databricks apps init` without --template.", p)
			}
		}
	}

	// In flags/non-interactive mode, resolve derived values and validate resources.
	// Agentic mode skips validation — resources are filled in later.
	if !agenticMode && (flagsMode || !isInteractive) {
		resources := m.CollectResources(selectedPlugins)

		// Resolve derived values for resources that support it.
		if resourceValues == nil {
			resourceValues = make(map[string]string)
		}
		for _, r := range resources {
			resolveFn, ok := prompt.GetResolveFunc(r.Type)
			if !ok {
				continue
			}
			resolved, err := resolveFn(ctx, r, resourceValues)
			if err != nil {
				log.Warnf(ctx, "Could not resolve derived values for %s: %v", r.Alias, err)
				continue
			}
			for k, v := range resolved {
				if resourceValues[k] == "" {
					resourceValues[k] = v
				}
			}
		}

		// Validate that all required resources are provided.
		if err := validateRequiredResources(resources, resourceValues); err != nil {
			return err
		}
	}

	// Apply flag values for deploy/run when in flags mode, flags were explicitly set, or non-interactive
	if skipDeployRunPrompt || !isInteractive {
		var err error
		shouldDeploy, runMode, err = parseDeployAndRunFlags(opts.deploy, opts.run)
		if err != nil {
			return err
		}
	}

	// Track whether we started creating the project for cleanup on failure.
	// The background npm install may have created destDir early.
	var projectCreated bool
	var runErr error
	defer func() {
		if runErr == nil || (!projectCreated && npmInstallCh == nil) {
			return
		}
		if inPlace {
			// destDir is "." here; a wholesale RemoveAll would wipe the
			// user's current directory (including any pre-existing .git).
			// Leave the partial scaffold and tell the user to clean up.
			log.Warnf(ctx, "scaffold failed in current directory; review and clean up generated files manually (e.g. with git status / git clean -fd)")
			return
		}
		os.RemoveAll(destDir)
	}()

	// Set description default
	if opts.description == "" {
		opts.description = prompt.DefaultAppDescription
	}

	// Get workspace host and profile from context
	workspaceHost := ""
	profile := ""
	if w := cmdctx.WorkspaceClient(ctx); w != nil && w.Config != nil {
		workspaceHost = w.Config.Host
		profile = w.Config.Profile
	}

	// Get selected plugins for generation
	selectedPluginList := generator.GetSelectedPlugins(m, selectedPlugins)

	log.Debugf(ctx, "Selected plugins: %v", selectedPlugins)
	log.Debugf(ctx, "Selected plugin list count: %d", len(selectedPluginList))
	log.Debugf(ctx, "Resource values: %d entries", len(resourceValues))

	// Build generator config
	genConfig := generator.Config{
		ProjectName:    opts.name,
		WorkspaceHost:  workspaceHost,
		Profile:        profile,
		ResourceValues: resourceValues,
	}

	// Generate configurations from selected plugins
	bundleVars := generator.GenerateBundleVariables(selectedPluginList, genConfig)
	bundleRes := generator.GenerateBundleResources(selectedPluginList, genConfig)
	targetVars := generator.GenerateTargetVariables(selectedPluginList, genConfig)

	log.Debugf(ctx, "Generated bundle variables:\n%s", bundleVars)
	log.Debugf(ctx, "Generated bundle resources:\n%s", bundleRes)
	log.Debugf(ctx, "Generated target variables:\n%s", targetVars)

	plugins := make(map[string]*pluginVar, len(selectedPlugins))
	for _, name := range selectedPlugins {
		pv := &pluginVar{}
		if mp, ok := m.Plugins[name]; ok {
			pv.Stability = mp.Stability
		}
		plugins[name] = pv
	}

	// Template variables with generated content
	vars := templateVars{
		ProjectName:    opts.name,
		AppDescription: opts.description,
		Profile:        profile,
		WorkspaceHost:  workspaceHost,
		Bundle: tmplBundle{
			Variables:       bundleVars,
			Resources:       bundleRes,
			TargetVariables: targetVars,
		},
		DotEnv: dotEnvVars{
			Content: generator.GenerateDotEnv(selectedPluginList, genConfig),
			Example: generator.GenerateDotEnvExample(selectedPluginList),
		},
		AppEnv:  generator.GenerateAppEnv(selectedPluginList, genConfig),
		Plugins: plugins,
	}

	// Await background npm install BEFORE copying the template so there are
	// no concurrent writes to destDir. npm ci ran with the raw lock file; the
	// dependency tree is determined entirely by package-lock.json which has no
	// template variables, so the installed node_modules is valid.
	if npmInstallCh != nil {
		if err := awaitBackgroundNpmInstall(ctx, npmInstallCh); err != nil {
			log.Warnf(ctx, "Background npm install failed: %v, will retry during project initialization", err)
			os.RemoveAll(filepath.Join(destDir, "node_modules"))
		}
	}

	// Copy template with variable substitution
	var fileCount int
	runErr = prompt.RunWithSpinnerCtx(ctx, "Creating project...", func() error {
		var copyErr error
		fileCount, copyErr = copyTemplate(ctx, templateDir, destDir, vars)
		return copyErr
	})
	if runErr != nil {
		return runErr
	}
	projectCreated = true // From here on, cleanup on failure

	// For pre-rendered templates, update package.json name (not a .tmpl file)
	// and serve as a safety net for the agentic flow.
	if skipPluginSelection {
		if err := replaceProjectName(destDir, opts.name); err != nil {
			log.Warnf(ctx, "Could not update project name in output files: %v", err)
		}
	}

	// Get absolute path
	absOutputDir, err := filepath.Abs(destDir)
	if err != nil {
		absOutputDir = destDir
	}

	// Initialize project based on type (Node.js, Python, etc.).
	// For Node.js, if the background install succeeded node_modules exists
	// and the initializer skips the redundant install step.
	// With --skip-install we bypass Initialize entirely and instead prepend
	// the install command to NextSteps so the user knows to install first.
	var nextStepsCmd string
	projectInitializer := initializer.GetProjectInitializer(absOutputDir)
	if projectInitializer != nil {
		if opts.skipInstall {
			nextStepsCmd = prependInstall(projectInitializer.InstallCommand(), projectInitializer.NextSteps())
		} else {
			result := projectInitializer.Initialize(ctx, absOutputDir)
			if !result.Success {
				if result.Error != nil {
					return fmt.Errorf("%s: %w", result.Message, result.Error)
				}
				return errors.New(result.Message)
			}
			nextStepsCmd = projectInitializer.NextSteps()
		}
	}

	// Validate dev-remote is only supported for appkit projects
	if runMode == prompt.RunModeDevRemote {
		if projectInitializer == nil || !projectInitializer.SupportsDevRemote() {
			return errors.New("--run=dev-remote is only supported for Node.js projects with @databricks/appkit")
		}
	}

	// Show next steps only if user didn't choose to deploy or run
	showNextSteps := !shouldDeploy && runMode == prompt.RunModeNone
	if showNextSteps {
		prompt.PrintSuccess(ctx, opts.name, absOutputDir, fileCount, nextStepsCmd, inPlace)
	} else {
		prompt.PrintSuccess(ctx, opts.name, absOutputDir, fileCount, "", inPlace)
	}

	// Print any onSetupMessage declared by selected plugins in the template manifest.
	var notes []prompt.SetupNote
	for _, name := range selectedPlugins {
		p, ok := m.Plugins[name]
		if !ok || p.OnSetupMessage == "" {
			continue
		}
		notes = append(notes, prompt.SetupNote{Name: p.DisplayName, Message: p.OnSetupMessage})
	}
	if len(notes) > 0 {
		prompt.PrintSetupNotes(ctx, notes)
	}

	// Recommend skills installation if coding agents are detected without skills.
	// In flags mode, only print a hint — never prompt interactively.
	if flagsMode {
		if !agents.HasDatabricksSkillsInstalled(ctx) {
			cmdio.LogString(ctx, "Tip: coding agents detected without Databricks skills. Run 'databricks aitools install' to install them.")
		}
	} else if err := agents.RecommendSkillsInstall(ctx, installer.InstallAllSkills); err != nil {
		log.Warnf(ctx, "Skills recommendation failed: %v", err)
	}

	// Execute post-creation actions (deploy and/or run)
	if shouldDeploy || runMode != prompt.RunModeNone {
		// Change to project directory for subsequent commands
		if err := os.Chdir(absOutputDir); err != nil {
			return fmt.Errorf("failed to change to project directory: %w", err)
		}
		if profile == "" {
			// If the profile is not set, it means the DEFAULT profile was used to infer the workspace host, we set it so that it's used for the deploy and dev-remote commands
			profile = defaultProfile
		}
	}

	if shouldDeploy {
		cmdio.LogString(ctx, "")
		if err := runPostCreateDeploy(ctx, profile); err != nil {
			cmdio.LogString(ctx, fmt.Sprintf("⚠ Deploy failed: %v", err))
			cmdio.LogString(ctx, "  You can deploy manually with: databricks apps deploy")
		} else {
			prompt.PrintDone(ctx, "Deploy complete")
		}
	}

	if runMode != prompt.RunModeNone {
		cmdio.LogString(ctx, "")
		if err := runPostCreateDev(ctx, runMode, projectInitializer, absOutputDir, profile); err != nil {
			return err
		}
	}

	return nil
}

// runPostCreateDeploy runs the deploy command in the current directory.
func runPostCreateDeploy(ctx context.Context, profile string) error {
	executable, err := os.Executable()
	if err != nil {
		return fmt.Errorf("failed to get executable path: %w", err)
	}
	args := []string{"apps", "deploy"}
	if profile != "" {
		// We ensure the same profile is used for the deploy command as the one used for the init command
		args = append(args, "--profile", profile)
	}
	cmd := exec.CommandContext(ctx, executable, args...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Stdin = os.Stdin
	return cmd.Run()
}

// runPostCreateDev runs the dev or dev-remote command in the current directory.
func runPostCreateDev(ctx context.Context, mode prompt.RunMode, projectInit initializer.Initializer, workDir, profile string) error {
	switch mode {
	case prompt.RunModeDev:
		if projectInit != nil {
			return projectInit.RunDev(ctx, workDir)
		}
		// Fallback for unknown project types
		cmdio.LogString(ctx, "⚠ Unknown project type, cannot start development server automatically")
		return nil
	case prompt.RunModeDevRemote:
		cmdio.LogString(ctx, "Starting remote development server...")
		executable, err := os.Executable()
		if err != nil {
			return fmt.Errorf("failed to get executable path: %w", err)
		}
		args := []string{"apps", "dev-remote"}
		if profile != "" {
			// We ensure the same profile is used for the dev-remote command as the one used for the init command
			args = append(args, "--profile", profile)
		}
		cmd := exec.CommandContext(ctx, executable, args...)
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		cmd.Stdin = os.Stdin
		return cmd.Run()
	default:
		return nil
	}
}

// prependInstall composes the install command and the project's NextSteps
// suggestion into a single shell snippet, dropping either side if empty.
func prependInstall(installCmd, nextStepsCmd string) string {
	if installCmd == "" {
		return nextStepsCmd
	}
	if nextStepsCmd == "" {
		return installCmd
	}
	return installCmd + " && " + nextStepsCmd
}

// appendUnique appends values to a slice, skipping duplicates.
func appendUnique(base []string, values ...string) []string {
	seen := make(map[string]bool, len(base))
	for _, v := range base {
		seen[v] = true
	}
	for _, v := range values {
		if !seen[v] {
			seen[v] = true
			base = append(base, v)
		}
	}
	return base
}

// buildPluginStrings builds the plugin import and usage strings from selected plugin names.
func buildPluginStrings(pluginNames []string) (pluginImport, pluginUsage string) {
	if len(pluginNames) == 0 {
		return "", ""
	}

	// Plugin names map directly to imports and usage
	// e.g., "analytics" -> import "analytics", usage "analytics()"
	var imports []string
	var usages []string

	for _, name := range pluginNames {
		imports = append(imports, name)
		usages = append(usages, name+"()")
	}

	pluginImport = strings.Join(imports, ", ")
	pluginUsage = strings.Join(usages, ",\n    ")

	return pluginImport, pluginUsage
}

// renameFiles maps source file names to destination names (for files that can't use special chars).
var renameFiles = map[string]string{
	"_gitignore":  ".gitignore",
	"_env":        ".env",
	"_env.local":  ".env.local",
	"_npmrc":      ".npmrc",
	"_prettierrc": ".prettierrc",
	"_eslintrc":   ".eslintrc",
}

// copyTemplate copies the template directory to dest, substituting variables.
func copyTemplate(ctx context.Context, src, dest string, vars templateVars) (int, error) {
	fileCount := 0

	// Find the project_name placeholder directory
	srcProjectDir := ""
	entries, err := os.ReadDir(src)
	if err != nil {
		return 0, err
	}
	for _, e := range entries {
		if e.IsDir() && strings.Contains(e.Name(), projectNamePlaceholder) {
			srcProjectDir = filepath.Join(src, e.Name())
			break
		}
	}

	// If no {{.project_name}} dir found, copy src directly
	if srcProjectDir == "" {
		srcProjectDir = src
	}

	log.Debugf(ctx, "Copying template from: %s", srcProjectDir)

	// Files and directories to skip
	skipFiles := map[string]bool{
		"CLAUDE.md":                       true,
		"AGENTS.md":                       true,
		"databricks_template_schema.json": true,
	}
	skipDirs := map[string]bool{
		"docs":         true,
		"node_modules": true,
		"dist":         true,
		".git":         true,
	}

	// Load .gitignore patterns from the template to skip ignored paths (e.g., dist, node_modules).
	// Checks both _gitignore (template convention) and .gitignore.
	var gitIgnore *ignore.GitIgnore
	for _, name := range []string{"_gitignore", ".gitignore"} {
		p := filepath.Join(srcProjectDir, name)
		if gi, err := ignore.CompileIgnoreFile(p); err == nil {
			gitIgnore = gi
			break
		}
	}

	err = filepath.Walk(srcProjectDir, func(srcPath string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		baseName := filepath.Base(srcPath)

		// Skip certain files
		if skipFiles[baseName] {
			log.Debugf(ctx, "Skipping file: %s", baseName)
			return nil
		}

		// Skip certain directories
		if info.IsDir() && skipDirs[baseName] {
			log.Debugf(ctx, "Skipping directory: %s", baseName)
			return filepath.SkipDir
		}

		// Skip paths matched by .gitignore patterns.
		// Append "/" for directories so patterns like "node_modules/" match correctly.
		if gitIgnore != nil && srcPath != srcProjectDir {
			rel, relErr := filepath.Rel(srcProjectDir, srcPath)
			if relErr == nil {
				matchPath := rel
				if info.IsDir() {
					matchPath = rel + "/"
				}
				if gitIgnore.MatchesPath(matchPath) {
					if info.IsDir() {
						log.Debugf(ctx, "Skipping gitignored directory: %s", rel)
						return filepath.SkipDir
					}
					log.Debugf(ctx, "Skipping gitignored file: %s", rel)
					return nil
				}
			}
		}

		// Calculate relative path from source project dir
		relPath, err := filepath.Rel(srcProjectDir, srcPath)
		if err != nil {
			return err
		}

		// Handle .tmpl extension - strip it
		relPath = strings.TrimSuffix(relPath, ".tmpl")

		// Apply file renames (e.g., _gitignore -> .gitignore)
		fileName := filepath.Base(relPath)
		if newName, ok := renameFiles[fileName]; ok {
			relPath = filepath.Join(filepath.Dir(relPath), newName)
		}

		destPath := filepath.Join(dest, relPath)

		if info.IsDir() {
			log.Debugf(ctx, "Creating directory: %s", relPath)
			return os.MkdirAll(destPath, info.Mode())
		}

		log.Debugf(ctx, "Copying file: %s", relPath)

		// Read file content
		content, err := os.ReadFile(srcPath)
		if err != nil {
			return err
		}

		// Apply Go template substitution to all text files (including .tmpl).
		if isTextFile(srcPath) || strings.HasSuffix(srcPath, ".tmpl") {
			content, err = executeTemplate(ctx, srcPath, content, vars)
			if err != nil {
				return fmt.Errorf("process template %s: %w", srcPath, err)
			}
		}

		// Skip files whose template rendered to only whitespace.
		// This enables conditional file creation: plugin-specific files wrap
		// their entire content in {{if .plugins.<name>}}...{{end}}, rendering
		// to empty when the plugin is not selected.
		if len(bytes.TrimSpace(content)) == 0 {
			log.Debugf(ctx, "Skipping conditionally empty file: %s", relPath)
			return nil
		}

		// Create parent directory
		if err := os.MkdirAll(filepath.Dir(destPath), 0o755); err != nil {
			return err
		}

		// Write file — use restrictive permissions for .env files (may contain secrets).
		perm := info.Mode()
		destName := filepath.Base(destPath)
		if strings.HasPrefix(destName, ".env") {
			perm = 0o600
		}
		if err := os.WriteFile(destPath, content, perm); err != nil {
			return err
		}

		fileCount++
		return nil
	})
	if err != nil {
		log.Debugf(ctx, "Error during template copy: %v", err)
	}
	log.Debugf(ctx, "Copied %d files", fileCount)

	if err == nil {
		err = removeEmptyDirs(dest)
	}

	return fileCount, err
}

// removeEmptyDirs removes empty directories under root, deepest-first.
// It is used to clean up directories that were created eagerly but ended up
// with no files after conditional template rendering skipped their contents.
//
// .git is skipped so in-place scaffolding (root == ".") never walks into a
// pre-existing repo and deletes its empty subdirectories (refs/heads,
// refs/tags, objects/info, objects/pack are all empty after `git init`).
func removeEmptyDirs(root string) error {
	var dirs []string
	err := filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() && d.Name() == ".git" && path != root {
			return filepath.SkipDir
		}
		if d.IsDir() && path != root {
			dirs = append(dirs, path)
		}
		return nil
	})
	if err != nil {
		return err
	}
	for _, dir := range slices.Backward(dirs) {
		_ = os.Remove(dir)
	}
	return nil
}

// templateData builds the data map for Go template execution.
func templateData(vars templateVars) map[string]any {
	// Sort plugin names for deterministic deprecated compat output.
	pluginNames := slices.Sorted(maps.Keys(vars.Plugins))

	// Only computed for deprecated backward compat keys.
	pluginImports, pluginUsages := buildPluginStrings(pluginNames)

	return map[string]any{
		"profile":        vars.Profile,
		"plugins":        vars.Plugins,
		"projectName":    vars.ProjectName,
		"appDescription": vars.AppDescription,
		"workspaceHost":  vars.WorkspaceHost,
		"bundle": map[string]any{
			"variables":       vars.Bundle.Variables,
			"resources":       vars.Bundle.Resources,
			"targetVariables": vars.Bundle.TargetVariables,
		},
		"dotEnv": map[string]any{
			"content": vars.DotEnv.Content,
			"example": vars.DotEnv.Example,
		},
		"appEnv": vars.AppEnv,

		// backward compatibility (deprecated)
		"variables":        vars.Bundle.Variables,
		"resources":        vars.Bundle.Resources,
		"dotenv":           vars.DotEnv.Content,
		"target_variables": vars.Bundle.TargetVariables,
		"project_name":     vars.ProjectName,
		"app_description":  vars.AppDescription,
		"dotenv_example":   vars.DotEnv.Example,
		"workspace_host":   vars.WorkspaceHost,
		"plugin_imports":   pluginImports,
		"plugin_usages":    pluginUsages,
		"app_env":          vars.AppEnv,
	}
}

// executeTemplate processes a file using Go's text/template engine.
// On parse errors (e.g., files containing non-Go {{...}} syntax), the original
// content is returned with a warning logged instead of failing the process.
func executeTemplate(ctx context.Context, path string, content []byte, vars templateVars) ([]byte, error) {
	tmpl, err := template.New(filepath.Base(path)).
		Option("missingkey=zero").
		Parse(string(content))
	if err != nil {
		log.Warnf(ctx, "Skipping template substitution for %s: %v", filepath.Base(path), err)
		return content, nil
	}

	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, templateData(vars)); err != nil {
		log.Warnf(ctx, "Skipping template substitution for %s: %v", filepath.Base(path), err)
		return content, nil
	}

	return buf.Bytes(), nil
}

// textExtensions contains file extensions that should be treated as text files.
var textExtensions = map[string]bool{
	".ts": true, ".tsx": true, ".js": true, ".jsx": true,
	".json": true, ".yaml": true, ".yml": true,
	".md": true, ".txt": true, ".html": true, ".css": true,
	".scss": true, ".less": true, ".sql": true,
	".sh": true, ".bash": true, ".zsh": true,
	".py": true, ".go": true, ".rs": true,
	".toml": true, ".ini": true, ".cfg": true,
	".env": true, ".gitignore": true, ".npmrc": true,
	".prettierrc": true, ".eslintrc": true,
}

// textBaseNames contains file names (without extension) that should be treated as text files.
var textBaseNames = map[string]bool{
	"Makefile": true, "Dockerfile": true, "LICENSE": true,
	"README": true, ".gitignore": true, ".env": true,
	".nvmrc": true, ".node-version": true,
	"_gitignore": true, "_env": true, "_npmrc": true,
}

// isTextFile checks if a file is likely a text file based on extension.
func isTextFile(path string) bool {
	ext := strings.ToLower(filepath.Ext(path))
	if textExtensions[ext] {
		return true
	}
	return textBaseNames[filepath.Base(path)]
}
