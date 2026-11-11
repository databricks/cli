package run

import (
	"errors"
	"fmt"
	"slices"
	"strings"

	"github.com/databricks/cli/bundle/config/resources"
	"github.com/databricks/databricks-sdk-go/service/pipelines"
	flag "github.com/spf13/pflag"
)

// PipelineOptions defines options for running a pipeline update.
type PipelineOptions struct {
	// Perform a full graph update.
	RefreshAll bool

	// Refresh holds CLI selectors before resolution and table names afterward.
	Refresh []string

	// Workspace file paths resolved from Refresh.
	RefreshByPath []string

	// Perform a full graph reset and recompute.
	FullRefreshAll bool

	// FullRefresh holds CLI selectors before resolution and table names afterward.
	FullRefresh []string

	// Workspace file paths resolved from FullRefresh.
	FullRefreshByPath []string

	// Perform an update to validate graph correctness.
	ValidateOnly bool
}

func (o *PipelineOptions) Define(fs *flag.FlagSet) {
	fs.BoolVar(&o.RefreshAll, "refresh-all", false, "Perform a full graph update.")
	o.DefineRefreshFlags(fs)
	fs.BoolVar(&o.ValidateOnly, "validate-only", false, "Perform an update to validate graph correctness.")
}

// PipelineRefreshHelp describes table and file refresh selectors.
const PipelineRefreshHelp = `For pipeline runs, --refresh and --full-refresh accept:
  Tables: orders or table:orders
  Source files: file:./src/pipeline.py (file: is required)

Put multiple selectors in quotes, or repeat the flag:
  --refresh "orders payments"
  --refresh "file:./src/a.py file:./src/b.py"
  --refresh table:orders --refresh table:payments

For a path containing spaces:
  --refresh 'file:` + "`./src/my pipeline.py`" + `'

Each refresh mode accepts either tables or files.
Local file paths are relative to the current directory and map to deployed files.
Absolute /Workspace/ paths are also accepted.
Deploy local changes before running.`

func (o *PipelineOptions) DefineRefreshFlags(fs *flag.FlagSet) {
	fs.Var((*refreshSelectionFlag)(&o.Refresh), "refresh", "Space-separated table:<name> or file:<path> selectors to update. Quote lists or repeat the flag.")
	fs.BoolVar(&o.FullRefreshAll, "full-refresh-all", false, "Perform a full graph reset and recompute.")
	fs.Var((*refreshSelectionFlag)(&o.FullRefresh), "full-refresh", "Space-separated table:<name> or file:<path> selectors to reset and recompute. Quote lists or repeat the flag.")
}

// Validate returns if the combination of options is valid.
func (o *PipelineOptions) Validate(pipeline *resources.Pipeline) error {
	var mutuallyExclusiveFlags []string

	if o.RefreshAll {
		mutuallyExclusiveFlags = append(mutuallyExclusiveFlags, "--refresh-all")
	}
	if o.FullRefreshAll {
		mutuallyExclusiveFlags = append(mutuallyExclusiveFlags, "--full-refresh-all")
	}
	if o.ValidateOnly {
		mutuallyExclusiveFlags = append(mutuallyExclusiveFlags, "--validate-only")
	}

	if len(mutuallyExclusiveFlags) > 1 {
		return fmt.Errorf("%s pipeline run flags are mutually exclusive", strings.Join(mutuallyExclusiveFlags, ", "))
	} else if len(mutuallyExclusiveFlags) == 1 && (len(o.Refresh) > 0 || len(o.FullRefresh) > 0 || len(o.RefreshByPath) > 0 || len(o.FullRefreshByPath) > 0) {
		return fmt.Errorf("cannot use --refresh or --full-refresh together with %s, these flags are mutually exclusive", strings.Join(mutuallyExclusiveFlags, ""))
	}

	// The API ignores file selections when table names are supplied for the same refresh mode.
	if len(o.Refresh) > 0 && len(o.RefreshByPath) > 0 {
		return errors.New("cannot mix table names and source files in --refresh; use only table names or only source files")
	}
	if len(o.FullRefresh) > 0 && len(o.FullRefreshByPath) > 0 {
		return errors.New("cannot mix table names and source files in --full-refresh; use only table names or only source files")
	}

	return nil
}

func (o *PipelineOptions) toPayload(pipeline *resources.Pipeline, pipelineID string, development bool) (*pipelines.StartUpdate, error) {
	if err := o.Validate(pipeline); err != nil {
		return nil, err
	}
	payload := &pipelines.StartUpdate{
		PipelineId: pipelineID,

		// Note: `RefreshAll` is implied if the fields below are not set.
		RefreshSelection:           o.Refresh,
		FullRefresh:                o.FullRefreshAll,
		FullRefreshSelection:       o.FullRefresh,
		RefreshSelectionByPath:     o.RefreshByPath,
		FullRefreshSelectionByPath: o.FullRefreshByPath,
		ValidateOnly:               o.ValidateOnly,

		// Only true is sent. When false the field is omitted, so the backend keeps
		// using the pipeline-level development property while it is being deprecated.
		Development: development,
	}
	return payload, nil
}

func unquoteRefreshTables(tables []string) []string {
	tables = slices.Clone(tables)
	for i, table := range tables {
		tables[i] = unquoteRefreshSelector(table)
	}
	return tables
}
