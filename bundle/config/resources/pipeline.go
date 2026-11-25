package resources

import (
	"context"
	"net/url"
	"reflect"

	"github.com/databricks/cli/libs/jsonschema"
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

func (*Pipeline) SchemaTransforms() map[reflect.Type]func(jsonschema.Schema) jsonschema.Schema {
	return map[reflect.Type]func(jsonschema.Schema) jsonschema.Schema{
		reflect.TypeFor[Pipeline](): func(s jsonschema.Schema) jsonschema.Schema {
			// Even though DABs supports this field, TF provider does not. Thus, we
			// should not expose it to the user.
			delete(s.Properties, "dry_run")

			// These fields are only meant to be set by the DABs client (ie the CLI)
			// and thus should not be exposed to the user. These are used to annotate
			// pipelines that were created by DABs.
			delete(s.Properties, "deployment")
			return s
		},
		// The CLI sets these to track the bundle deployment in the Deployment
		// Metadata Service; they are not user-configurable, so they must not
		// appear in the JSON schema or the generated annotation files. The
		// parent "deployment" block is already removed from the Pipeline schema
		// above; this removes the fields from the type definition itself.
		reflect.TypeFor[pipelines.PipelineDeployment](): func(s jsonschema.Schema) jsonschema.Schema {
			delete(s.Properties, "deployment_id")
			delete(s.Properties, "version_id")
			return s
		},
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
