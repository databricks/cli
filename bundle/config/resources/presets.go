package resources

import (
	"strings"

	"github.com/databricks/cli/libs/tags"
	"github.com/databricks/cli/libs/textutil"
)

// Values of presets.trigger_pause_status.
const (
	TriggerPauseStatusPaused   = "PAUSED"
	TriggerPauseStatusUnpaused = "UNPAUSED"
)

// Tag is an entry of presets.tags.
type Tag struct {
	Key   string
	Value string
}

// PresetSettings are the bundle's presets, with the defaults of the target mode
// applied, in the form that resource types need to apply them to themselves.
type PresetSettings struct {
	// NamePrefix is prepended to the user-facing name of a resource.
	NamePrefix string

	// Tags are sorted by key, so that resources that keep tags in a list get a stable order.
	Tags []Tag

	// TriggerPauseStatus is TriggerPauseStatusPaused, TriggerPauseStatusUnpaused or empty.
	TriggerPauseStatus string

	// JobsMaxConcurrentRuns is presets.jobs_max_concurrent_runs, used for jobs that do not set
	// max_concurrent_runs themselves. Zero leaves those jobs unchanged.
	JobsMaxConcurrentRuns int

	// PipelinesDevelopment is true if presets.pipelines_development is explicitly enabled.
	PipelinesDevelopment bool

	// SkipNamePrefixForSchema is true if experimental.skip_name_prefix_for_schema is explicitly enabled.
	SkipNamePrefixForSchema bool

	// Tagging normalizes tag keys and values for resources whose tags are cloud tags.
	Tagging tags.Cloud
}

// PresetApplier is implemented by resource types that presets change.
//
// If embedded pointer is nil, initialize it with empty struct. The fact that we embed pointer struct is implementation detail.
// It will remain nil if there are no fields in it that are provided by users, but that maybe ok, users could be relying on presets or defaults.
type PresetApplier interface {
	// ApplyPresets applies p to the resource. Values that the resource sets
	// itself take precedence over the presets.
	ApplyPresets(p *PresetSettings)
}

// NamePrefixExempt is implemented by resource types whose name is deliberately
// not prefixed with presets.name_prefix.
type NamePrefixExempt interface {
	// NamePrefixExemptReason returns why the name is not prefixed.
	NamePrefixExemptReason() string
}

// unityCatalogNamePrefixExemptReason is the NamePrefixExemptReason of Unity Catalog resources.
const unityCatalogNamePrefixExemptReason = "Unity Catalog provides its own namespace hierarchy; " +
	"use ${workspace.current_user.short_name} in the name to make it unique per user"

// normalizePrefix prefixes strings like '[dev lennart] ' to 'dev_lennart_'.
// We leave unicode letters and numbers but remove all "special characters."
func normalizePrefix(prefix string) string {
	prefix = strings.ReplaceAll(prefix, "[", "")
	prefix = strings.Trim(prefix, " ")

	// If the prefix ends with a ']', we add an underscore to the end.
	// This makes sure that we get names like "dev_user_endpoint" instead of "dev_userendpoint"
	suffix := ""
	if strings.HasSuffix(prefix, "]") {
		suffix = "_"
	}

	return textutil.NormalizeString(prefix) + suffix
}
