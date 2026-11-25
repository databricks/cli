package resources

import (
	"context"
	"net/url"

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

// Destroy messages for pipelines, picked per pipeline from the cascade_on_destroy value in
// the deployment state.
const (
	PipelineDestroyWithCascadeMessage = `This action will result in the deletion of the following Lakeflow Spark Declarative Pipelines along with the
Streaming Tables (STs) and Materialized Views (MVs) managed by them. Set 'cascade_on_destroy: false' on a pipeline to retain datasets on pipeline deletion:`

	PipelineDestroyNoCascadeMessage = `This action will result in the deletion of the following Lakeflow Spark Declarative Pipelines.
The Streaming Tables (STs) and Materialized Views (MVs) managed by them are retained, as 'cascade_on_destroy' is set to false:`
)

func (*Pipeline) Approval() Approval {
	return Approval{
		Order: 20,
		Deploy: `
This action will result in the deletion or recreation of the following Lakeflow Spark Declarative Pipelines along with the
Streaming Tables (STs) and Materialized Views (MVs) managed by them. Recreating the pipelines will
restore the defined STs and MVs through full refresh. Note that recreation is necessary when pipeline
properties such as the 'catalog' or 'storage' are changed:`,
		// Destroy reads cascade_on_destroy from the state rather than the configuration,
		// because the state holds the value that the delete uses.
		DestroyFromState: true,
	}
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
