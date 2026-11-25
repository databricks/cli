package resourcemutator

import (
	"cmp"
	"context"
	"iter"
	"reflect"
	"slices"

	"github.com/databricks/cli/bundle"
	"github.com/databricks/cli/bundle/config"
	"github.com/databricks/cli/bundle/config/resources"
	"github.com/databricks/cli/bundle/metrics"
	"github.com/databricks/cli/libs/diag"
)

type applyPresets struct{}

// Apply all presets, e.g. the prefix presets that
// adds a prefix to all names of all resources.
func ApplyPresets() bundle.Mutator {
	return &applyPresets{}
}

func (m *applyPresets) Name() string {
	return "ApplyPresets"
}

func (m *applyPresets) Apply(ctx context.Context, b *bundle.Bundle) diag.Diagnostics {
	var diags diag.Diagnostics

	if d := validatePauseStatus(b); d != nil {
		diags = diags.Extend(d)
	}

	t := b.Config.Presets
	prefix := t.NamePrefix

	b.Metrics.AddBoolValue(metrics.PresetsNamePrefixIsSet, prefix != "")

	p := &resources.PresetSettings{
		NamePrefix:              prefix,
		Tags:                    toTagArray(t.Tags),
		TriggerPauseStatus:      t.TriggerPauseStatus,
		JobsMaxConcurrentRuns:   t.JobsMaxConcurrentRuns,
		PipelinesDevelopment:    config.IsExplicitlyEnabled(t.PipelinesDevelopment),
		SkipNamePrefixForSchema: b.Config.Experimental != nil && config.IsExplicitlyEnabled(b.Config.Experimental.SkipNamePrefixForSchema),
		Tagging:                 b.Tagging,
	}

	for r := range resourcesImplementing[resources.PresetApplier](&b.Config.Resources) {
		r.ApplyPresets(p)
	}

	return diags
}

// resourcesImplementing yields every resource in r that implements I, grouped by
// resource type in canonical order.
//
// If resource itself is nil, ignore it (It is currently allowed, but should have generic check with location-enriched diagnostics that disallows nils anywhere).
func resourcesImplementing[I any](r *config.Resources) iter.Seq[I] {
	return func(yield func(I) bool) {
		for _, group := range r.AllResources() {
			for _, res := range group.Resources {
				if reflect.ValueOf(res).IsNil() {
					continue
				}
				v, ok := res.(I)
				if ok && !yield(v) {
					return
				}
			}
		}
	}
}

func validatePauseStatus(b *bundle.Bundle) diag.Diagnostics {
	p := b.Config.Presets.TriggerPauseStatus
	if p == "" || p == config.Paused || p == config.Unpaused {
		return nil
	}
	return diag.Diagnostics{{
		Summary:   "Invalid value for trigger_pause_status, should be PAUSED or UNPAUSED",
		Severity:  diag.Error,
		Locations: []diag.Location{b.Config.GetLocation("presets.trigger_pause_status")},
	}}
}

// toTagArray converts a map of tags to an array of tags.
// We sort tags so ensure stable ordering.
func toTagArray(tags map[string]string) []resources.Tag {
	var tagArray []resources.Tag
	if tags == nil {
		return tagArray
	}
	for key, value := range tags {
		tagArray = append(tagArray, resources.Tag{Key: key, Value: value})
	}
	slices.SortFunc(tagArray, func(a, b resources.Tag) int {
		return cmp.Compare(a.Key, b.Key)
	})
	return tagArray
}
