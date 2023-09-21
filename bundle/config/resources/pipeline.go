package resources

import (
	"strings"

	"github.com/databricks/cli/bundle/config/paths"
	"github.com/databricks/databricks-sdk-go/service/pipelines"
	"github.com/imdario/mergo"
)

type Pipeline struct {
	ID          string       `json:"id,omitempty" bundle:"readonly"`
	Permissions []Permission `json:"permissions,omitempty"`

	paths.Paths

	*pipelines.PipelineSpec
}

// MergeClusters merges cluster definitions with same label.
// The clusters field is a slice, and as such, overrides are appended to it.
// We can identify a cluster by its label, however, so we can use this label
// to figure out which definitions are actually overrides and merge them.
//
// Note: the cluster label is optional and defaults to 'default'.
// We therefore ALSO merge all clusters without a label.
func (p *Pipeline) MergeClusters() error {
	clusters := make(map[string]*pipelines.PipelineCluster)
	output := make([]pipelines.PipelineCluster, 0, len(p.Clusters))

	// Normalize cluster labels.
	// If empty, this defaults to "default".
	// Matching is case insensitive, so labels are lowercased.
	for i := range p.Clusters {
		label := p.Clusters[i].Label
		if label == "" {
			label = "default"
		}
		p.Clusters[i].Label = strings.ToLower(label)
	}

	// Target overrides are always appended, so we can iterate in natural order to
	// first find the base definition, and merge instances we encounter later.
	for i := range p.Clusters {
		label := p.Clusters[i].Label

		// Register pipeline cluster with label if not yet seen before.
		ref, ok := clusters[label]
		if !ok {
			output = append(output, p.Clusters[i])
			clusters[label] = &output[len(output)-1]
			continue
		}

		// Merge this instance into the reference.
		err := mergo.Merge(ref, &p.Clusters[i], mergo.WithOverride, mergo.WithAppendSlice)
		if err != nil {
			return err
		}
	}

	// Overwrite resulting slice.
	p.Clusters = output
	return nil
}
