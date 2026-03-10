package resources

import (
	"context"
	"net/url"

	"github.com/databricks/cli/libs/log"
	"github.com/databricks/databricks-sdk-go"
	"github.com/databricks/databricks-sdk-go/marshal"
	"github.com/databricks/databricks-sdk-go/service/pipelines"
)

type Pipeline struct {
	BaseResource
	pipelines.CreatePipeline //nolint CreatePipeline also defines Id field with the same json tag "id"

	Permissions []PipelinePermission `json:"permissions,omitempty"`
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
	baseURL.Path = "pipelines/" + p.ID
	p.URL = baseURL.String()
}

func (p *Pipeline) GetName() string {
	return p.Name
}

func (p *Pipeline) GetURL() string {
	return p.URL
}
