package resources

import (
	"context"
	"net/url"

	"github.com/databricks/cli/libs/diag"
	"github.com/databricks/cli/libs/log"
	"github.com/databricks/cli/libs/workspaceurls"
	"github.com/databricks/databricks-sdk-go"
	"github.com/databricks/databricks-sdk-go/marshal"
	"github.com/databricks/databricks-sdk-go/service/pipelines"
)

type Pipeline struct {
	BaseResource
	pipelines.CreatePipeline        //nolint:govet // CreatePipeline.Id and our depth-0 ID field both carry json:"id"; the depth-0 field wins
	ID                       string `json:"id,omitempty" bundle:"readonly"`

	Permissions []PipelinePermission `json:"permissions,omitempty"`

	// CascadeOnDestroy controls whether destroying the pipeline also deletes its datasets
	// Nil (unset) uses the server default, which cascades. Set to false to retain datasets
	// when the pipeline is deleted. Delete-time only: never sent on create/update.
	CascadeOnDestroy *bool `json:"cascade_on_destroy,omitempty"`
}

func (p *Pipeline) UnmarshalJSON(b []byte) error {
	return marshal.Unmarshal(b, p)
}

func (p Pipeline) MarshalJSON() ([]byte, error) {
	return marshal.Marshal(p)
}

func (p *Pipeline) Exists(ctx context.Context, w *databricks.WorkspaceClient, id string) (bool, error) {
	_, err := w.Pipelines.Get(ctx, pipelines.GetPipelineRequest{
		PipelineId: id,
	})
	if err != nil {
		log.Debugf(ctx, "pipeline %s does not exist", id)
		return false, err
	}
	return true, nil
}

func (p *Pipeline) ResourceDescription() ResourceDescription {
	return ResourceDescription{
		SingularName:  "pipeline",
		PluralName:    "pipelines",
		SingularTitle: "Pipeline",
		PluralTitle:   "Pipelines",
	}
}

func (p *Pipeline) InitializeURL(baseURL url.URL) {
	if p.ID == "" {
		return
	}
	p.URL = workspaceurls.ResourceURL(baseURL, "pipelines", p.ID)
}

func (p *Pipeline) GetName() string {
	return p.Name
}

func (p *Pipeline) ApplyPresets(t *PresetSettings) {
	p.Name = t.NamePrefix + p.Name
	if t.PipelinesDevelopment {
		//nolint:staticcheck // SA1019: pipeline development is deprecated in the SDK but remains a supported bundle config field
		p.Development = true
	}
	if t.TriggerPauseStatus == TriggerPauseStatusPaused {
		//nolint:staticcheck // SA1019: pipeline continuous is deprecated in the SDK but remains a supported bundle config field
		p.Continuous = false
	}

	if len(t.Tags) > 0 {
		if p.Tags == nil {
			// Note: only create this map if tags is not empty, to avoid inserting "tags: {}" entry in the config
			p.Tags = make(map[string]string, len(t.Tags))
		}
		for _, tag := range t.Tags {
			if p.Tags[tag.Key] == "" {
				p.Tags[tag.Key] = tag.Value
			}
		}
	}
}

func (p *Pipeline) FieldDefaults(DefaultsContext) []FieldDefault {
	return []FieldDefault{
		// Same as terraform
		// https://github.com/databricks/terraform-provider-databricks/blob/v1.75.0/pipelines/resource_pipeline.go#L253
		{"edition", "ADVANCED"},
		{"channel", "CURRENT"},
	}
}

func (p *Pipeline) ValidateProductionMode() diag.Diagnostics {
	//nolint:staticcheck // SA1019: pipeline development is deprecated in the SDK but remains a supported bundle config field
	if p.Development {
		return diag.Errorf("target with 'mode: production' cannot include a pipeline with 'development: true'")
	}
	return nil
}
