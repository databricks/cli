package resources

import (
	"context"
	"net/url"

	"github.com/databricks/cli/libs/sdkclients"
	"github.com/databricks/cli/libs/sdktypes/pipelines/v2"
	"github.com/databricks/cli/libs/utils"
	sdkpipelines "github.com/databricks/sdk-go/pipelines/v2"

	"github.com/databricks/cli/libs/log"
	"github.com/databricks/cli/libs/workspaceurls"
)

type Pipeline struct {
	BaseResource
	pipelines.CreatePipelineRequest        //nolint:govet // CreatePipeline.Id and our depth-0 ID field both carry json:"id"; the depth-0 field wins
	ID                              string `json:"id,omitempty" bundle:"readonly"`

	Permissions []PipelinePermission `json:"permissions,omitempty"`

	// CascadeOnDestroy controls whether destroying the pipeline also deletes its datasets
	// Nil (unset) uses the server default, which cascades. Set to false to retain datasets
	// when the pipeline is deleted. Delete-time only: never sent on create/update.
	CascadeOnDestroy *bool `json:"cascade_on_destroy,omitempty"`
}

func (p *Pipeline) Exists(ctx context.Context, w *sdkclients.Clients, id string) (bool, error) {
	_, err := w.Pipelines.Get(ctx, sdkpipelines.GetPipelineRequest{
		PipelineId: new(id),
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
	return utils.Deref(p.Name)
}
