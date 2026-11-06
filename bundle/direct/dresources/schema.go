package dresources

import (
	"context"
	"fmt"
	"slices"

	"github.com/databricks/cli/bundle/config/resources"
	"github.com/databricks/cli/libs/log"
	"github.com/databricks/cli/libs/utils"
	"github.com/databricks/databricks-sdk-go"
	"github.com/databricks/databricks-sdk-go/marshal"
	"github.com/databricks/databricks-sdk-go/service/catalog"
)

// SchemaState is the schema's config plus its schema_id. The backend assigns schema_id and a
// rename keeps it, so recording it lets the plan recognize the schema under a new name (see
// InternalID).
type SchemaState struct {
	catalog.CreateSchema
	SchemaId string `json:"schema_id,omitempty"`
}

func (s *SchemaState) UnmarshalJSON(b []byte) error {
	return marshal.Unmarshal(b, s)
}

func (s SchemaState) MarshalJSON() ([]byte, error) {
	return marshal.Marshal(s)
}

type ResourceSchema struct {
	client *databricks.WorkspaceClient
}

func (*ResourceSchema) New(client *databricks.WorkspaceClient) *ResourceSchema {
	return &ResourceSchema{client: client}
}

func (*ResourceSchema) PrepareState(input *resources.Schema) *SchemaState {
	return &SchemaState{
		CreateSchema: input.CreateSchema,
		SchemaId:     "", // Recorded from the backend when the schema is created or updated.
	}
}

func (*ResourceSchema) RemapState(info *catalog.SchemaInfo) *SchemaState {
	return &SchemaState{
		CatalogName:             info.CatalogName,
		Comment:                 info.Comment,
		CustomMaxRetentionHours: info.CustomMaxRetentionHours,
		Name:                    info.Name,
		Properties:              info.Properties,
		StorageRoot:             info.StorageRoot,
		ForceSendFields:         utils.FilterFields[catalog.CreateSchema](info.ForceSendFields),
		SchemaId:                info.SchemaId,
	}
}

func (r *ResourceSchema) DoRead(ctx context.Context, id string) (*catalog.SchemaInfo, error) {
	return r.client.Schemas.GetByFullName(ctx, id)
}

func (r *ResourceSchema) DoCreate(ctx context.Context, config *SchemaState) (string, *catalog.SchemaInfo, error) {
	response, err := r.client.Schemas.Create(ctx, config.CreateSchema)
	if err != nil || response == nil {
		return "", nil, err
	}
	config.SchemaId = response.SchemaId
	return response.FullName, response, nil
}

// UC drops a field the payload omits, so a value the config stops declaring never
// reaches the backend and every later plan reports the same pending change. Worse, when
// nothing else is in the payload UC rejects it outright with "Nothing to update" (400),
// which made a schema undeployable once someone set a description on it outside the
// bundle. Sending these explicitly is the only way to express "the bundle owns this".
//
// Only fields verified against a real workspace belong here: {"comment": ""} and
// {"custom_max_retention_hours": 0} both clear. Do not add a field without probing it -- UC rejects
// an empty value on some (owner: "" is "Could not find principal with name .", new_name: "" is "not
// a valid name"), and ForceSendFields is inert for maps, so listing properties would be a no-op.
//
// custom_max_retention_hours is force-sent so the direct engine can clear it. The terraform
// databricks_schema resource cannot set this field at all, so a schema config that uses it deploys
// differently under terraform (which drops it) than under direct; such configs are kept out of the
// migrate invariant (schema.yml is excluded there, and schema_tf_compat.yml covers migrate instead).
var schemaForceSend = []string{"Comment", "CustomMaxRetentionHours"}

func makeUpdateSchema(id, newName string, config *catalog.CreateSchema) catalog.UpdateSchema {
	return catalog.UpdateSchema{
		Comment:                      config.Comment,
		CustomMaxRetentionHours:      config.CustomMaxRetentionHours,
		EnablePredictiveOptimization: "", // Not supported by DABs
		FullName:                     id,
		NewName:                      newName,
		Owner:                        "", // Not supported by DABs
		Properties:                   config.Properties,
		ForceSendFields:              utils.FilterFields[catalog.UpdateSchema](append(slices.Clone(schemaForceSend), config.ForceSendFields...), "EnablePredictiveOptimization", "NewName", "Owner"),
	}
}

// DoUpdate updates the schema in place and returns remote state.
func (r *ResourceSchema) DoUpdate(ctx context.Context, id string, config *SchemaState, _ *PlanEntry) (*catalog.SchemaInfo, error) {
	response, err := r.client.Schemas.Update(ctx, makeUpdateSchema(id, "", &config.CreateSchema))
	if err != nil || response == nil {
		return nil, err
	}

	if response.FullName != id {
		log.Warnf(ctx, "schemas: response contains unexpected full_name=%#v (expected %#v)", response.FullName, id)
	}

	config.SchemaId = response.SchemaId
	return response, nil
}

// DoUpdateWithID renames the schema in place when the name changes; the ID is the full name, so it changes too.
func (r *ResourceSchema) DoUpdateWithID(ctx context.Context, id string, config *SchemaState, entry *PlanEntry) (string, *catalog.SchemaInfo, error) {
	nameFromID, err := getNameFromID(id)
	if err != nil {
		return "", nil, err
	}

	newName := ""
	if config.Name != nameFromID {
		newName = config.Name
	}

	// The catalog comes from config: a renamed catalog moves the schema with it, leaving id
	// stale (see followParentRenames).
	fullName := config.CatalogName + "." + nameFromID

	if entry.Changes.OnlyParentRenamed() {
		// UC already moved the schema with its catalog; only the ID changes.
		remote, ok := entry.RemoteState.(*catalog.SchemaInfo)
		if !ok {
			return "", nil, fmt.Errorf("internal error: unexpected remote state type %T", entry.RemoteState)
		}
		config.SchemaId = remote.SchemaId
		return fullName, nil, nil
	}

	response, err := r.client.Schemas.Update(ctx, makeUpdateSchema(fullName, newName, &config.CreateSchema))
	if err != nil || response == nil {
		return "", nil, err
	}

	config.SchemaId = response.SchemaId
	return response.FullName, response, nil
}

// IDFromState returns the full name, which is the ID.
func (*ResourceSchema) IDFromState(state *SchemaState) string {
	return state.CatalogName + "." + state.Name
}

// InternalID returns the schema_id.
func (*ResourceSchema) InternalID(state *SchemaState) string {
	return state.SchemaId
}

func (r *ResourceSchema) DoDelete(ctx context.Context, id string, _ *SchemaState) error {
	return r.client.Schemas.Delete(ctx, catalog.DeleteSchemaRequest{
		FullName:        id,
		Force:           true,
		ForceSendFields: nil,
	})
}
