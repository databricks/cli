package resources

import (
	"context"
	"net/url"

	"github.com/databricks/cli/libs/log"
	"github.com/databricks/databricks-sdk-go"
	"github.com/databricks/databricks-sdk-go/marshal"
	"github.com/databricks/databricks-sdk-go/service/pipelines"
)

type PipelinePermissionLevel string

// PipelinePermission holds the permission level setting for a single principal.
// Multiple of these can be defined on any pipeline.
type PipelinePermission struct {
	Level PipelinePermissionLevel `json:"level"`

	UserName             string `json:"user_name,omitempty"`
	ServicePrincipalName string `json:"service_principal_name,omitempty"`
	GroupName            string `json:"group_name,omitempty"`
}

type Pipeline struct {
	ID             string               `json:"id,omitempty" bundle:"readonly"`
	Permissions    []PipelinePermission `json:"permissions,omitempty"`
	ModifiedStatus ModifiedStatus       `json:"modified_status,omitempty" bundle:"internal"`
	URL            string               `json:"url,omitempty" bundle:"internal"`

	pipelines.CreatePipeline
}

func (s *Pipeline) UnmarshalJSON(b []byte) error {
	return marshal.Unmarshal(b, s)
}

func (s Pipeline) MarshalJSON() ([]byte, error) {
	return marshal.Marshal(s)
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

func (j *Pipeline) ResourceDescription() ResourceDescription {
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

func (s *Pipeline) GetURL() string {
	return s.URL
}
