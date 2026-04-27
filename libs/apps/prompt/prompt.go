package prompt

import (
	"context"
	"errors"
	"fmt"
	"maps"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"

	"github.com/charmbracelet/huh"
	"github.com/charmbracelet/lipgloss"
	"github.com/databricks/cli/libs/apps/manifest"
	"github.com/databricks/cli/libs/cmdctx"
	"github.com/databricks/cli/libs/cmdio"
	"github.com/databricks/cli/libs/log"
	"github.com/databricks/databricks-sdk-go/listing"
	"github.com/databricks/databricks-sdk-go/service/apps"
	"github.com/databricks/databricks-sdk-go/service/postgres"
)

// DefaultAppDescription is the default description for new apps.
const DefaultAppDescription = "A Databricks App powered by AppKit"

// Brand palette — tuned for legibility on both light and dark terminals.
var (
	colorRed    = lipgloss.Color("#E84040") // Bright Databricks red
	colorGray   = lipgloss.Color("#A1A1AA") // Light gray, legible on dark backgrounds
	colorYellow = lipgloss.Color("#FFAB00") // Databricks yellow / amber
	colorOrange = lipgloss.Color("#FF5F40") // Databricks orange (code blocks)
	colorDimmed = lipgloss.AdaptiveColor{Light: "#A1A1AA", Dark: "#52525B"}
)

// AppkitTheme returns a custom theme for appkit prompts.
func AppkitTheme() *huh.Theme {
	t := huh.ThemeBase()

	t.Focused.Title = t.Focused.Title.Foreground(colorRed).Bold(true)
	t.Focused.Description = t.Focused.Description.Foreground(colorGray)
	t.Focused.SelectedOption = t.Focused.SelectedOption.Foreground(colorYellow)
	t.Focused.TextInput.Placeholder = t.Focused.TextInput.Placeholder.Foreground(colorGray)
	bracketBorder := lipgloss.Border{Left: "[", Right: "]"}
	t.Focused.FocusedButton = t.Focused.FocusedButton.
		Foreground(colorYellow).
		Background(lipgloss.Color("")).
		Bold(true).
		BorderStyle(bracketBorder).
		BorderLeft(true).BorderRight(true).
		BorderTop(false).BorderBottom(false).
		BorderForeground(colorYellow)
	t.Focused.BlurredButton = t.Focused.BlurredButton.
		Foreground(colorDimmed).
		Background(lipgloss.Color("")).
		Bold(false)

	return t
}

// Styles for printing answered prompts.
var (
	answeredTitleStyle = lipgloss.NewStyle().
				Foreground(colorGray)
	answeredValueStyle = lipgloss.NewStyle().
				Foreground(colorYellow).
				Bold(true)
)

// Stability tier styles, applied to the parenthetical suffix in plugin labels.
var (
	stabilityExperimentalStyle = lipgloss.NewStyle().Foreground(colorOrange)
	stabilityPreviewStyle      = lipgloss.NewStyle().Foreground(colorYellow)
	stabilityUnknownStyle      = lipgloss.NewStyle().Foreground(colorGray)
)

// RenderStabilityTier renders a stability tier as a colored " (tier)" suffix,
// or returns "" for stable/unset. Unknown tiers are rendered in gray so we
// remain forward-compatible with future tier names.
func RenderStabilityTier(tier string) string {
	if tier == "" {
		return ""
	}
	var style lipgloss.Style
	switch tier {
	case "experimental":
		style = stabilityExperimentalStyle
	case "preview":
		style = stabilityPreviewStyle
	default:
		style = stabilityUnknownStyle
	}
	return " " + style.Render("("+tier+")")
}

// PrintAnswered prints a completed prompt answer to keep history visible.
func PrintAnswered(ctx context.Context, title, value string) {
	cmdio.LogString(ctx, fmt.Sprintf("%s %s", answeredTitleStyle.Render(title+":"), answeredValueStyle.Render(value)))
}

// printAnswered is an alias for internal use.
func printAnswered(ctx context.Context, title, value string) {
	PrintAnswered(ctx, title, value)
}

// RunMode specifies how to run the app after creation.
type RunMode string

const (
	RunModeNone      RunMode = "none"
	RunModeDev       RunMode = "dev"
	RunModeDevRemote RunMode = "dev-remote"
)

// CreateProjectConfig holds the configuration gathered from the interactive prompt.
type CreateProjectConfig struct {
	ProjectName  string
	Description  string
	Features     []string
	Dependencies map[string]string // e.g., {"sql_warehouse_id": "abc123"}
	Deploy       bool              // Whether to deploy the app after creation
	RunMode      RunMode           // How to run the app after creation
}

// App name constraints.
const (
	MaxAppNameLength = 30
	DevTargetPrefix  = "dev-"
)

// projectNamePattern is the compiled regex for validating project names.
// Pre-compiled for efficiency since validation is called on every keystroke.
var projectNamePattern = regexp.MustCompile(`^[a-z][a-z0-9-]*$`)

// ValidateProjectName validates the project name for length and pattern constraints.
// It checks that the name plus the "dev-" prefix doesn't exceed 30 characters,
// and that the name follows the pattern: starts with a letter, contains only
// lowercase letters, numbers, or hyphens.
func ValidateProjectName(s string) error {
	if s == "" {
		return errors.New("project name is required")
	}

	// Check length constraint (dev- prefix + name <= 30)
	totalLength := len(DevTargetPrefix) + len(s)
	if totalLength > MaxAppNameLength {
		maxAllowed := MaxAppNameLength - len(DevTargetPrefix)
		return fmt.Errorf("name too long (max %d chars)", maxAllowed)
	}

	// Check pattern
	if !projectNamePattern.MatchString(s) {
		return errors.New("must start with a letter, use only lowercase letters, numbers, or hyphens")
	}

	return nil
}

// PrintHeader prints the AppKit header banner.
func PrintHeader(ctx context.Context) {
	headerStyle := lipgloss.NewStyle().
		Foreground(colorRed).
		Bold(true)

	subtitleStyle := lipgloss.NewStyle().
		Foreground(colorGray)

	cmdio.LogString(ctx, "")
	cmdio.LogString(ctx, headerStyle.Render("◆ Create a new Databricks AppKit project"))
	cmdio.LogString(ctx, subtitleStyle.Render("  Full-stack TypeScript • React • Tailwind CSS"))
	cmdio.LogString(ctx, "")
}

// PromptForProjectName prompts only for project name.
// Used as the first step before resolving templates.
// outputDir is used to check if the destination directory already exists.
func PromptForProjectName(ctx context.Context, outputDir string) (string, error) {
	PrintHeader(ctx)
	theme := AppkitTheme()

	var name string
	err := huh.NewInput().
		Title("App name").
		Description("lowercase letters, numbers, hyphens (max 26 chars)").
		Placeholder("my-app").
		Value(&name).
		Validate(func(s string) error {
			if err := ValidateProjectName(s); err != nil {
				return err
			}
			destDir := s
			if outputDir != "" {
				destDir = filepath.Join(outputDir, s)
			}
			if _, err := os.Stat(destDir); err == nil {
				return fmt.Errorf("directory %s already exists", destDir)
			}
			return nil
		}).
		WithTheme(theme).
		Run()
	if err != nil {
		return "", err
	}

	printAnswered(ctx, "Project name", name)
	return name, nil
}

// PromptForDeployAndRun prompts for post-creation deploy and run options.
func PromptForDeployAndRun(ctx context.Context) (deploy bool, runMode RunMode, err error) {
	theme := AppkitTheme()

	// Deploy after creation?
	err = huh.NewConfirm().
		Title("Deploy after creation?").
		Description("Run 'databricks apps deploy' after setup").
		Value(&deploy).
		WithTheme(theme).
		Run()
	if err != nil {
		return false, RunModeNone, err
	}
	if deploy {
		printAnswered(ctx, "Deploy after creation", "Yes")
	} else {
		printAnswered(ctx, "Deploy after creation", "No")
	}

	// Build run options - dev-remote requires deploy (needs a deployed app to connect to)
	runOptions := []huh.Option[string]{
		huh.NewOption("No, I'll run it later", string(RunModeNone)),
		huh.NewOption("Yes, run locally (npm run dev)", string(RunModeDev)),
	}
	if deploy {
		runOptions = append(runOptions, huh.NewOption("Yes, run with remote bridge (dev-remote)", string(RunModeDevRemote)))
	}

	// Run the app?
	runModeStr := string(RunModeNone)
	err = huh.NewSelect[string]().
		Title("Run the app after creation?").
		Description("Choose how to start the development server").
		Options(runOptions...).
		Value(&runModeStr).
		WithTheme(theme).
		Run()
	if err != nil {
		return false, RunModeNone, err
	}

	runModeLabels := map[string]string{
		string(RunModeNone):      "No",
		string(RunModeDev):       "Yes (local)",
		string(RunModeDevRemote): "Yes (remote)",
	}
	printAnswered(ctx, "Run after creation", runModeLabels[runModeStr])

	return deploy, RunMode(runModeStr), nil
}

// PromptFromList shows a picker for items and returns the selected ID.
// If required is false and items are empty, returns ("", nil). If required is true and items are empty, returns an error.
func PromptFromList(ctx context.Context, title, emptyMessage string, items []ListItem, required bool) (string, error) {
	id, _, err := promptFromListWithLabel(ctx, title, emptyMessage, items, required)
	return id, err
}

// promptFromListWithLabel shows a picker and returns both the selected ID and its display label.
func promptFromListWithLabel(ctx context.Context, title, emptyMessage string, items []ListItem, required bool) (string, string, error) {
	if len(items) == 0 {
		if required {
			return "", "", errors.New(emptyMessage)
		}
		return "", "", nil
	}
	theme := AppkitTheme()
	options := make([]huh.Option[string], 0, len(items))
	labels := make(map[string]string)
	for _, it := range items {
		options = append(options, huh.NewOption(it.Label, it.ID))
		labels[it.ID] = it.Label
	}
	var selected string
	err := huh.NewSelect[string]().
		Title(title).
		Description(fmt.Sprintf("%d available — / to filter", len(items))).
		Options(options...).
		Value(&selected).
		Height(8).
		WithTheme(theme).
		Run()
	if err != nil {
		return "", "", err
	}
	printAnswered(ctx, title, labels[selected])
	return selected, labels[selected], nil
}

// awaitFetcher waits for a background PagedFetcher's first page. If the data
// is already available it returns immediately; otherwise a spinner is shown.
func awaitFetcher(ctx context.Context, f *PagedFetcher, spinnerMsg string) error {
	if f.IsDone() {
		return f.Err
	}
	return RunWithSpinnerCtx(ctx, spinnerMsg, func() error {
		return f.WaitForFirstPage(ctx)
	})
}

// getFetcher returns a PagedFetcher from the cache, waiting for its first page.
// If the cache has no entry, it creates one synchronously using the paged
// constructor registered in pagedConstructors.
func getFetcher(ctx context.Context, resourceType, spinnerMsg string) (*PagedFetcher, error) {
	ctor := pagedConstructors[resourceType]
	return getFetcherByKey(ctx, resourceType, spinnerMsg, ctor)
}

// getFetcherByKey returns a PagedFetcher from the cache under the given key,
// waiting for its first page. If the cache has no entry, it falls back to
// creating one synchronously using the provided constructor.
func getFetcherByKey(ctx context.Context, cacheKey, spinnerMsg string, fallbackCtor pagedConstructor) (*PagedFetcher, error) {
	if cache := CacheFromContext(ctx); cache != nil {
		if f := cache.GetFetcher(cacheKey); f != nil {
			if err := awaitFetcher(ctx, f, spinnerMsg); err != nil {
				return nil, err
			}
			return f, nil
		}
	}
	if fallbackCtor == nil {
		return nil, fmt.Errorf("no lister registered for cache key %q", cacheKey)
	}
	var f *PagedFetcher
	err := RunWithSpinnerCtx(ctx, spinnerMsg, func() error {
		var fetchErr error
		f, fetchErr = fallbackCtor(ctx)
		return fetchErr
	})
	if err != nil {
		return nil, err
	}
	return f, nil
}

// promptManualInput shows a text input for the user to type a resource name/ID
// manually. prefetchedLabels provides tab-complete suggestions.
func promptManualInput(ctx context.Context, title string, prefetchedLabels []string) (string, error) {
	theme := AppkitTheme()
	var value string
	err := huh.NewInput().
		Title(title).
		Placeholder("Type a name or ID").
		Suggestions(prefetchedLabels).
		Value(&value).
		WithTheme(theme).
		Run()
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(value), nil
}

// promptFromPagedFetcher shows a picker backed by a PagedFetcher. When more
// pages are available and the total is under maxTotalResults, a "Load more..."
// option is appended. Once capped (>= maxTotalResults), an "Enter name/ID
// manually..." option replaces it.
// SearchFunc performs a server-side search by name/query. When non-nil, the
// manual input fallback triggers a search instead of accepting raw input.
// This is currently supported by Jobs (name filter). Other resource types can
// pass nil until their APIs add server-side filtering support.
type SearchFunc func(ctx context.Context, query string) ([]ListItem, error)

func promptFromPagedFetcher(ctx context.Context, title, emptyMessage string, fetcher *PagedFetcher, required bool, searchFn SearchFunc) (string, string, error) {
	if len(fetcher.Items) == 0 && !fetcher.HasMore {
		if required {
			return "", "", errors.New(emptyMessage)
		}
		return "", "", nil
	}
	theme := AppkitTheme()

	for {
		options := make([]huh.Option[string], 0, len(fetcher.Items)+2)
		labels := make(map[string]string, len(fetcher.Items))

		var desc string
		if fetcher.HasMore && !fetcher.Capped {
			desc = fmt.Sprintf("%d results loaded — / to search", len(fetcher.Items))
			options = append(options, huh.NewOption("+ Load more results", moreID))
		} else if fetcher.Capped {
			desc = fmt.Sprintf("%d results loaded — / to search", len(fetcher.Items))
			manualLabel := "Not listed? Enter ID manually..."
			if searchFn != nil {
				manualLabel = "Not listed? Search by name..."
			}
			options = append(options, huh.NewOption(manualLabel, manualID))
		} else {
			desc = fmt.Sprintf("%d available — / to search", len(fetcher.Items))
		}

		for _, it := range fetcher.Items {
			options = append(options, huh.NewOption(it.Label, it.ID))
			labels[it.ID] = it.Label
		}

		var selected string
		err := huh.NewSelect[string]().
			Title(title).
			Description(desc).
			Options(options...).
			Value(&selected).
			Height(8).
			WithTheme(theme).
			Run()
		if err != nil {
			return "", "", err
		}

		switch selected {
		case moreID:
			if err := RunWithSpinnerCtx(ctx, "Fetching more results...", func() error {
				return fetcher.LoadMore(ctx)
			}); err != nil {
				return "", "", err
			}
			continue

		case manualID:
			suggestions := make([]string, 0, len(fetcher.Items))
			for _, it := range fetcher.Items {
				suggestions = append(suggestions, it.Label)
			}
			query, inputErr := promptManualInput(ctx, title, suggestions)
			if inputErr != nil {
				return "", "", inputErr
			}
			if query == "" {
				if required {
					continue
				}
				return "", "", nil
			}

			if searchFn == nil {
				printAnswered(ctx, title, query)
				return query, query, nil
			}

			var results []ListItem
			if searchErr := RunWithSpinnerCtx(ctx, fmt.Sprintf("Searching for %q...", query), func() error {
				var fetchErr error
				results, fetchErr = searchFn(ctx, query)
				return fetchErr
			}); searchErr != nil {
				return "", "", searchErr
			}
			if len(results) == 0 {
				printAnswered(ctx, title, query)
				return query, query, nil
			}
			if len(results) == 1 {
				printAnswered(ctx, title, results[0].Label)
				return results[0].ID, results[0].Label, nil
			}
			id, label, pickErr := promptFromListWithLabel(ctx, title+" — search results", "no matches", results, required)
			if pickErr != nil {
				return "", "", pickErr
			}
			return id, label, nil

		default:
			printAnswered(ctx, title, labels[selected])
			return selected, labels[selected], nil
		}
	}
}

// promptForPagedResource gets a PagedFetcher (from cache or on-demand), then
// shows the paged picker with Load more / Enter manually support.
// Pass a non-nil searchFn to enable server-side search in the manual input
// fallback (currently only Jobs supports this).
func promptForPagedResource(ctx context.Context, r manifest.Resource, required bool, title, emptyMsg, spinnerMsg string, searchFn SearchFunc) (map[string]string, error) {
	f, err := getFetcher(ctx, r.Type, spinnerMsg)
	if err != nil {
		return nil, err
	}
	value, _, promptErr := promptFromPagedFetcher(ctx, title, emptyMsg, f, required, searchFn)
	if promptErr != nil {
		return nil, promptErr
	}
	return singleValueResult(r, value), nil
}

// resourceTitle returns a prompt title for a resource, including the plugin name
// for context when available (e.g. "Select SQL Warehouse for Analytics").
func resourceTitle(fallback string, r manifest.Resource) string {
	title := r.Alias
	if title == "" {
		title = fallback
	}
	if r.PluginDisplayName != "" {
		title = fmt.Sprintf("%s for %s", title, r.PluginDisplayName)
	}
	return title
}

// singleValueResult wraps a single value into the resource values map.
// Uses the first field name from Fields for the composite key (resource_key.field),
// or falls back to the resource key if no Fields are defined.
func singleValueResult(r manifest.Resource, value string) map[string]string {
	if value == "" {
		return nil
	}
	names := r.FieldNames()
	if len(names) >= 1 {
		return map[string]string{r.Key() + "." + names[0]: value}
	}
	return map[string]string{r.Key(): value}
}

// PromptForSecret shows a two-step picker for secret scope and key.
func PromptForSecret(ctx context.Context, r manifest.Resource, required bool) (map[string]string, error) {
	// Step 1: pick scope
	var scopes []ListItem
	err := RunWithSpinnerCtx(ctx, "Fetching secret scopes...", func() error {
		var fetchErr error
		scopes, fetchErr = ListSecretScopes(ctx)
		return fetchErr
	})
	if err != nil {
		return nil, err
	}
	scope, err := PromptFromList(ctx, "Select Secret Scope", "no secret scopes found", scopes, required)
	if err != nil {
		return nil, err
	}
	if scope == "" {
		return nil, nil
	}

	// Step 2: pick key within scope
	var keys []ListItem
	err = RunWithSpinnerCtx(ctx, "Fetching secret keys...", func() error {
		var fetchErr error
		keys, fetchErr = ListSecretKeys(ctx, scope)
		return fetchErr
	})
	if err != nil {
		return nil, err
	}
	key, err := PromptFromList(ctx, "Select Secret Key", "no keys found in scope "+scope, keys, required)
	if err != nil {
		return nil, err
	}
	if key == "" {
		return nil, nil
	}

	return map[string]string{
		r.Key() + ".scope": scope,
		r.Key() + ".key":   key,
	}, nil
}

// PromptForJob shows a picker for jobs. When the user selects "Enter manually"
// (after the 500-item cap), the input triggers a server-side name search via
// the Jobs API's Name filter before accepting the value.
func PromptForJob(ctx context.Context, r manifest.Resource, required bool) (map[string]string, error) {
	title := resourceTitle("Select Job", r)
	return promptForPagedResource(ctx, r, required, title, "no jobs found", "Fetching jobs...", SearchJobs)
}

// PromptForSQLWarehouseResource shows a picker for SQL warehouses (manifest.Resource version).
func PromptForSQLWarehouseResource(ctx context.Context, r manifest.Resource, required bool) (map[string]string, error) {
	title := resourceTitle("Select SQL Warehouse", r)
	return promptForPagedResource(ctx, r, required, title, "no SQL warehouses found. Create one in your workspace first", "Fetching SQL warehouses...", nil)
}

const backID = "__back__"

// promptUCCatalog shows a picker for UC catalogs (shared first step for volume/function pickers).
func promptUCCatalog(ctx context.Context, required bool) (string, error) {
	f, err := getFetcherByKey(ctx, cacheKeyCatalogs, "Fetching catalogs...", ListCatalogs)
	if err != nil {
		return "", err
	}
	id, _, promptErr := promptFromPagedFetcher(ctx, "Select Catalog", "no catalogs found", f, required, nil)
	return id, promptErr
}

// promptFromListWithBack shows a picker with a "← Go back" option prepended.
// Returns ("", nil) if the user selects "Go back".
func promptFromListWithBack(ctx context.Context, title string, items []ListItem) (string, error) {
	withBack := make([]ListItem, 0, len(items)+1)
	withBack = append(withBack, ListItem{ID: backID, Label: "← Go back"})
	withBack = append(withBack, items...)
	value, err := PromptFromList(ctx, title, "", withBack, true)
	if err != nil {
		return "", err
	}
	if value == backID {
		return "", nil
	}
	return value, nil
}

// promptUCResource runs a three-step catalog → schema → resource picker with back navigation.
// Empty results at any level show a message and navigate back automatically.
func promptUCResource(ctx context.Context, r manifest.Resource, required bool, resourceLabel, spinnerMsg string, listFn func(context.Context, string, string) ([]ListItem, error)) (map[string]string, error) {
	for {
		catalogName, err := promptUCCatalog(ctx, required)
		if err != nil || catalogName == "" {
			return nil, err
		}

		for {
			var schemas []ListItem
			err = RunWithSpinnerCtx(ctx, "Fetching schemas...", func() error {
				var fetchErr error
				schemas, fetchErr = ListSchemas(ctx, catalogName)
				return fetchErr
			})
			if err != nil {
				return nil, err
			}
			if len(schemas) == 0 {
				cmdio.LogString(ctx, fmt.Sprintf("No schemas found in %s, try another catalog.", catalogName))
				break // back to catalog picker
			}

			schemaName, err := promptFromListWithBack(ctx, "Select Schema", schemas)
			if err != nil {
				return nil, err
			}
			if schemaName == "" {
				break // back to catalog picker
			}

			var items []ListItem
			err = RunWithSpinnerCtx(ctx, spinnerMsg, func() error {
				var fetchErr error
				items, fetchErr = listFn(ctx, catalogName, schemaName)
				return fetchErr
			})
			if err != nil {
				return nil, err
			}
			if len(items) == 0 {
				cmdio.LogString(ctx, fmt.Sprintf("No %ss found in %s.%s, try another schema.", resourceLabel, catalogName, schemaName))
				continue // back to schema picker
			}

			value, err := promptFromListWithBack(ctx, "Select "+resourceLabel, items)
			if err != nil {
				return nil, err
			}
			if value == "" {
				continue // back to schema picker
			}
			return singleValueResult(r, value), nil
		}
	}
}

// PromptForServingEndpoint shows a picker for serving endpoints.
func PromptForServingEndpoint(ctx context.Context, r manifest.Resource, required bool) (map[string]string, error) {
	title := resourceTitle("Select Serving Endpoint", r)
	return promptForPagedResource(ctx, r, required, title, "no serving endpoints found", "Fetching serving endpoints...", nil)
}

// volumePathToSecurableName converts a volume path (/Volumes/catalog/schema/vol)
// to the securable_full_name format (catalog.schema.vol) used by DABs.
func volumePathToSecurableName(path string) string {
	trimmed := strings.TrimPrefix(path, "/Volumes/")
	if trimmed == path {
		return path
	}
	return strings.ReplaceAll(trimmed, "/", ".")
}

// PromptForVolume shows a three-step picker for UC volumes: catalog -> schema -> volume.
// Stores two values: the volume path for .env and the dot-separated securable
// name for the DABs YAML securable_full_name field.
func PromptForVolume(ctx context.Context, r manifest.Resource, required bool) (map[string]string, error) {
	result, err := promptUCResource(ctx, r, required, "Volume", "Fetching volumes...", ListVolumesInSchema)
	if err != nil || result == nil {
		return result, err
	}
	// promptUCResource stores the volume path (/Volumes/cat/schema/vol) under
	// the first manifest field (e.g., "path"). The DABs spec also needs the
	// securable_full_name (cat.schema.vol) under "id".
	idKey := r.Key() + ".id"
	if _, exists := result[idKey]; !exists {
		for _, v := range result {
			result[idKey] = volumePathToSecurableName(v)
			break
		}
	}
	return result, nil
}

// PromptForVectorSearchIndex shows a picker for vector search indexes.
func PromptForVectorSearchIndex(ctx context.Context, r manifest.Resource, required bool) (map[string]string, error) {
	title := resourceTitle("Select Vector Search Index", r)
	return promptForPagedResource(ctx, r, required, title, "no vector search indexes found", "Fetching vector search indexes...", nil)
}

// PromptForUCFunction shows a three-step picker for UC functions: catalog -> schema -> function.
func PromptForUCFunction(ctx context.Context, r manifest.Resource, required bool) (map[string]string, error) {
	return promptUCResource(ctx, r, required, "UC Function", "Fetching functions...", ListFunctionsInSchema)
}

// PromptForUCConnection shows a picker for UC connections.
func PromptForUCConnection(ctx context.Context, r manifest.Resource, required bool) (map[string]string, error) {
	title := resourceTitle("Select UC Connection", r)
	return promptForPagedResource(ctx, r, required, title, "no connections found", "Fetching connections...", nil)
}

// PromptForDatabase shows a two-step picker for database instance and database name.
func PromptForDatabase(ctx context.Context, r manifest.Resource, required bool) (map[string]string, error) {
	// Step 1: pick a Lakebase instance
	f, err := getFetcherByKey(ctx, cacheKeyDatabaseInstances, "Fetching database instances...", ListDatabaseInstances)
	if err != nil {
		return nil, err
	}
	instanceName, _, err := promptFromPagedFetcher(ctx, "Select Database Instance", "no database instances found", f, required, nil)
	if err != nil {
		return nil, err
	}
	if instanceName == "" {
		return nil, nil
	}

	// Step 2: pick a database within the instance
	var databases []ListItem
	err = RunWithSpinnerCtx(ctx, "Fetching databases...", func() error {
		var fetchErr error
		databases, fetchErr = ListDatabases(ctx, instanceName)
		return fetchErr
	})
	if err != nil {
		return nil, err
	}
	dbName, err := PromptFromList(ctx, "Select Database", "no databases found in instance "+instanceName, databases, required)
	if err != nil {
		return nil, err
	}
	if dbName == "" {
		return nil, nil
	}

	return map[string]string{
		r.Key() + ".instance_name": instanceName,
		r.Key() + ".database_name": dbName,
	}, nil
}

// PromptForPostgres shows a three-step picker for Lakebase Autoscaling (V2): project, branch, then database.
func PromptForPostgres(ctx context.Context, r manifest.Resource, required bool) (map[string]string, error) {
	// Step 1: pick a project
	f, err := getFetcherByKey(ctx, cacheKeyPostgresProjects, "Fetching Postgres projects...", ListPostgresProjects)
	if err != nil {
		return nil, err
	}
	projectName, _, err := promptFromPagedFetcher(ctx, "Select Postgres Project", "no Postgres projects found", f, required, nil)
	if err != nil {
		return nil, err
	}
	if projectName == "" {
		return nil, nil
	}

	// Step 2: pick a branch within the project
	var branches []ListItem
	err = RunWithSpinnerCtx(ctx, "Fetching branches...", func() error {
		var fetchErr error
		branches, fetchErr = ListPostgresBranches(ctx, projectName)
		return fetchErr
	})
	if err != nil {
		return nil, err
	}
	branchName, err := PromptFromList(ctx, "Select Branch", "no branches found in project "+projectName, branches, required)
	if err != nil {
		return nil, err
	}
	if branchName == "" {
		return nil, nil
	}

	// Step 3: pick a database within the branch
	var databases []ListItem
	err = RunWithSpinnerCtx(ctx, "Fetching databases...", func() error {
		var fetchErr error
		databases, fetchErr = ListPostgresDatabases(ctx, branchName)
		return fetchErr
	})
	if err != nil {
		return nil, err
	}
	dbName, pgDatabaseName, err := promptFromListWithLabel(ctx, "Select Database", "no databases found in branch "+branchName, databases, required)
	if err != nil {
		return nil, err
	}
	if dbName == "" {
		return nil, nil
	}

	// Start with prompted values (fields without resolve).
	result := map[string]string{
		r.Key() + ".branch":   branchName,
		r.Key() + ".database": dbName,
	}

	// Resolve derived values (host, databaseName, endpointPath) — non-fatal.
	var resolved map[string]string
	resolveErr := RunWithSpinnerCtx(ctx, "Resolving connection details...", func() error {
		var err error
		resolved, err = ResolvePostgresValues(ctx, r, branchName, dbName, pgDatabaseName)
		return err
	})
	if resolveErr != nil {
		log.Warnf(ctx, "Could not resolve connection details: %v", resolveErr)
	}
	maps.Copy(result, resolved)

	return result, nil
}

// resolvePostgresResource adapts ResolvePostgresValues for the generic ResolveResourceFunc signature.
func resolvePostgresResource(ctx context.Context, r manifest.Resource, provided map[string]string) (map[string]string, error) {
	branchName := provided[r.Key()+".branch"]
	dbName := provided[r.Key()+".database"]
	if branchName == "" || dbName == "" {
		return nil, nil
	}
	return ResolvePostgresValues(ctx, r, branchName, dbName, "")
}

// ResolvePostgresValues resolves derived field values (host, databaseName, endpointPath)
// from a branch and database resource name. If pgDatabaseName is already known
// (e.g. from a prior prompt), pass it to skip the ListDatabases API call.
func ResolvePostgresValues(ctx context.Context, r manifest.Resource, branchName, dbName, pgDatabaseName string) (map[string]string, error) {
	var host, endpointPath string
	endpoints, err := ListPostgresEndpoints(ctx, branchName)
	if err != nil {
		return nil, fmt.Errorf("resolving endpoint details: %w", err)
	}
	for _, ep := range endpoints {
		if ep.Status != nil && ep.Status.EndpointType == postgres.EndpointTypeEndpointTypeReadWrite {
			endpointPath = ep.Name
			if ep.Status.Hosts != nil && ep.Status.Hosts.Host != "" {
				host = ep.Status.Hosts.Host
			}
			break
		}
	}

	if pgDatabaseName == "" {
		databases, err := ListPostgresDatabases(ctx, branchName)
		if err != nil {
			return nil, fmt.Errorf("resolving database name: %w", err)
		}
		for _, db := range databases {
			if db.ID == dbName {
				pgDatabaseName = db.Label
				break
			}
		}
	}

	resolvedValues := map[string]string{
		"postgres:host":         host,
		"postgres:databaseName": pgDatabaseName,
		"postgres:endpointPath": endpointPath,
	}

	result := make(map[string]string)
	applyResolvedValues(r, resolvedValues, result)
	return result, nil
}

// applyResolvedValues populates result with values from resolvedValues,
// using the manifest's resolve property to map resolver names to field names.
func applyResolvedValues(r manifest.Resource, resolvedValues, result map[string]string) {
	for _, fieldName := range r.FieldNames() {
		field := r.Fields[fieldName]
		if field.Resolve == "" {
			continue
		}
		if val, ok := resolvedValues[field.Resolve]; ok {
			result[r.Key()+"."+fieldName] = val
		}
	}
}

// PromptForGenieSpace shows a picker for Genie spaces.
// Captures both the space ID and name since the DABs schema requires both fields.
func PromptForGenieSpace(ctx context.Context, r manifest.Resource, required bool) (map[string]string, error) {
	f, err := getFetcher(ctx, r.Type, "Fetching Genie spaces...")
	if err != nil {
		return nil, err
	}

	title := resourceTitle("Select Genie Space", r)
	id, name, err := promptFromPagedFetcher(ctx, title, "no Genie spaces found", f, required, nil)
	if err != nil {
		return nil, err
	}
	if id == "" {
		return nil, nil
	}
	return map[string]string{
		r.Key() + ".id":   id,
		r.Key() + ".name": name,
	}, nil
}

// PromptForExperiment shows a picker for MLflow experiments.
func PromptForExperiment(ctx context.Context, r manifest.Resource, required bool) (map[string]string, error) {
	title := resourceTitle("Select Experiment", r)
	return promptForPagedResource(ctx, r, required, title, "no experiments found", "Fetching experiments...", nil)
}

// TODO: uncomment when bundles support app as an app resource type.
// // PromptForAppResource shows a picker for apps (manifest.Resource version).
// func PromptForAppResource(ctx context.Context, r manifest.Resource, required bool) (map[string]string, error) {
// 	return promptForResourceFromLister(ctx, r, required, "Select App", "no apps found. Create one first with 'databricks apps create <name>'", "Fetching apps...", ListAppsItems)
// }

// Styles for consistent status output.
var (
	doneStyle = lipgloss.NewStyle().
			Foreground(colorYellow).
			Bold(true)
	doneTextStyle = lipgloss.NewStyle().
			Foreground(colorGray)
)

// PrintDone prints a styled "✔ message" completion line.
func PrintDone(ctx context.Context, msg string) {
	cmdio.LogString(ctx, fmt.Sprintf("%s %s", doneStyle.Render("✔"), doneTextStyle.Render(msg)))
}

// stripEllipsis removes a trailing "..." from a string for use in completion messages.
func stripEllipsis(s string) string {
	return strings.TrimSuffix(s, "...")
}

// RunWithSpinnerCtx runs a function while showing a spinner with the given title.
// On success, prints a styled checkmark completion line.
// The spinner stops and the function returns early if the context is cancelled.
// Panics in the action are recovered and returned as errors.
func RunWithSpinnerCtx(ctx context.Context, title string, action func() error) error {
	spinner := cmdio.NewSpinner(ctx)
	spinner.Update(title)

	done := make(chan error, 1)
	go func() {
		defer func() {
			if r := recover(); r != nil {
				done <- fmt.Errorf("action panicked: %v", r)
			}
		}()
		done <- action()
	}()

	select {
	case err := <-done:
		spinner.Close()
		if err == nil {
			PrintDone(ctx, stripEllipsis(title))
		}
		return err
	case <-ctx.Done():
		spinner.Close()
		// Wait for action goroutine to complete to avoid orphaned goroutines.
		// For exec.CommandContext, the process is killed when context is cancelled.
		<-done
		return ctx.Err()
	}
}

// ListAllApps fetches all apps the user has access to from the workspace.
func ListAllApps(ctx context.Context) ([]apps.App, error) {
	w := cmdctx.WorkspaceClient(ctx)
	if w == nil {
		return nil, errors.New("no workspace client available")
	}

	iter := w.Apps.List(ctx, apps.ListAppsRequest{})
	return listing.ToSlice(ctx, iter)
}

// PromptForAppSelection shows a picker to select an existing app.
// Returns the selected app name or error if cancelled/no apps found.
func PromptForAppSelection(ctx context.Context, title string) (string, error) {
	if !cmdio.IsPromptSupported(ctx) {
		return "", errors.New("--name is required in non-interactive mode")
	}

	// Fetch all apps the user has access to
	var existingApps []apps.App
	err := RunWithSpinnerCtx(ctx, "Fetching apps...", func() error {
		var fetchErr error
		existingApps, fetchErr = ListAllApps(ctx)
		return fetchErr
	})
	if err != nil {
		return "", fmt.Errorf("failed to fetch apps: %w", err)
	}

	if len(existingApps) == 0 {
		return "", errors.New("no apps found. Create one first with 'databricks apps create <name>'")
	}

	theme := AppkitTheme()

	// Build options
	options := make([]huh.Option[string], 0, len(existingApps))
	for _, app := range existingApps {
		label := app.Name
		if app.Description != "" {
			desc := app.Description
			if len(desc) > 40 {
				desc = desc[:37] + "..."
			}
			label += " — " + desc
		}
		options = append(options, huh.NewOption(label, app.Name))
	}

	var selected string
	err = huh.NewSelect[string]().
		Title(title).
		Description(fmt.Sprintf("%d apps found — / to filter", len(existingApps))).
		Options(options...).
		Value(&selected).
		Height(8).
		WithTheme(theme).
		Run()
	if err != nil {
		return "", err
	}

	printAnswered(ctx, "App", selected)
	return selected, nil
}

// PrintSuccess prints a success message after project creation.
// If nextStepsCmd is non-empty, also prints the "Next steps" section with the given command.
func PrintSuccess(ctx context.Context, projectName, outputDir string, fileCount int, nextStepsCmd string) {
	successStyle := lipgloss.NewStyle().
		Foreground(colorYellow).
		Bold(true)

	dimStyle := lipgloss.NewStyle().
		Foreground(colorGray)

	codeStyle := lipgloss.NewStyle().
		Foreground(colorOrange)

	cmdio.LogString(ctx, "")
	cmdio.LogString(ctx, successStyle.Render("✔ Project created successfully!"))
	cmdio.LogString(ctx, "")
	cmdio.LogString(ctx, dimStyle.Render("  Location: "+outputDir))
	cmdio.LogString(ctx, dimStyle.Render("  Files: "+strconv.Itoa(fileCount)))

	if nextStepsCmd != "" {
		cmdio.LogString(ctx, "")
		cmdio.LogString(ctx, dimStyle.Render("  Next steps:"))
		cmdio.LogString(ctx, "")
		cmdio.LogString(ctx, codeStyle.Render("    cd "+projectName))
		cmdio.LogString(ctx, codeStyle.Render("    "+nextStepsCmd))
	}
	cmdio.LogString(ctx, "")
}

// SetupNote holds the display name and message for a single plugin setup note.
type SetupNote struct {
	Name    string
	Message string
}

// PrintSetupNotes renders a styled "Setup Notes" section for selected plugins.
func PrintSetupNotes(ctx context.Context, notes []SetupNote) {
	headerStyle := lipgloss.NewStyle().
		Foreground(colorYellow).
		Bold(true)

	nameStyle := lipgloss.NewStyle().
		Foreground(colorGray).
		Bold(true)

	msgStyle := lipgloss.NewStyle().
		Foreground(colorGray)

	cmdio.LogString(ctx, headerStyle.Render("  Setup Notes"))
	cmdio.LogString(ctx, "")
	for _, n := range notes {
		cmdio.LogString(ctx, nameStyle.Render("  "+n.Name))
		indented := strings.ReplaceAll(n.Message, "\n", "\n  ")
		cmdio.LogString(ctx, msgStyle.Render("  "+indented))
		cmdio.LogString(ctx, "")
	}
}
