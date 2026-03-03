package dresources

import (
	"context"
	"fmt"

	"github.com/databricks/cli/bundle/config/resources"
	"github.com/databricks/cli/libs/structs/structpath"
	"github.com/databricks/cli/libs/utils"
	"github.com/databricks/databricks-sdk-go"
	"github.com/databricks/databricks-sdk-go/service/catalog"
)

// Precalculated paths for HasChange checks.
var pathAliases = structpath.MustParsePath("aliases")

type ResourceRegisteredModel struct {
	client *databricks.WorkspaceClient
}

func (*ResourceRegisteredModel) New(client *databricks.WorkspaceClient) *ResourceRegisteredModel {
	return &ResourceRegisteredModel{
		client: client,
	}
}

func getAliasKey(a catalog.RegisteredModelAlias) (string, string) {
	return "alias_name", a.AliasName
}

func (*ResourceRegisteredModel) KeyedSlices() map[string]any {
	return map[string]any{
		"aliases": getAliasKey,
	}
}

func (*ResourceRegisteredModel) PrepareState(input *resources.RegisteredModel) *catalog.CreateRegisteredModelRequest {
	return &input.CreateRegisteredModelRequest
}

func (*ResourceRegisteredModel) RemapState(model *catalog.RegisteredModelInfo) *catalog.CreateRegisteredModelRequest {
	return &catalog.CreateRegisteredModelRequest{
		CatalogName:     model.CatalogName,
		Comment:         model.Comment,
		Name:            model.Name,
		SchemaName:      model.SchemaName,
		StorageLocation: model.StorageLocation,
		ForceSendFields: utils.FilterFields[catalog.CreateRegisteredModelRequest](model.ForceSendFields),

		Aliases:     model.Aliases,
		BrowseOnly:  model.BrowseOnly,
		FullName:    model.FullName,
		MetastoreId: model.MetastoreId,
		Owner:       model.Owner,

		// Output only fields. Remote changes to these are ignored via
		// ignore_remote_changes in resources.yml rather than zeroed here.
		CreatedAt: model.CreatedAt,
		CreatedBy: model.CreatedBy,
		UpdatedAt: model.UpdatedAt,
		UpdatedBy: model.UpdatedBy,
	}
}

func (r *ResourceRegisteredModel) DoRead(ctx context.Context, id string) (*catalog.RegisteredModelInfo, error) {
	return r.client.RegisteredModels.Get(ctx, catalog.GetRegisteredModelRequest{
		FullName:        id,
		IncludeAliases:  true,
		IncludeBrowse:   false,
		ForceSendFields: nil,
	})
}

func (r *ResourceRegisteredModel) DoCreate(ctx context.Context, config *catalog.CreateRegisteredModelRequest) (string, *catalog.RegisteredModelInfo, error) {
	response, err := r.client.RegisteredModels.Create(ctx, *config)
	if err != nil {
		return "", nil, err
	}

	return response.FullName, response, nil
}

// WaitAfterCreate syncs aliases after the model is created and state is saved.
// The Create API does not apply aliases, so we sync them separately.
func (r *ResourceRegisteredModel) WaitAfterCreate(ctx context.Context, id string, config *catalog.CreateRegisteredModelRequest) (*catalog.RegisteredModelInfo, error) {
	// The model was just created, so there are no current aliases.
	return nil, r.syncAliases(ctx, id, config.Aliases, nil)
}

func (r *ResourceRegisteredModel) DoUpdate(ctx context.Context, id string, config *catalog.CreateRegisteredModelRequest, entry *PlanEntry) (*catalog.RegisteredModelInfo, error) {
	updateRequest := catalog.UpdateRegisteredModelRequest{
		FullName:        id,
		Comment:         config.Comment,
		ForceSendFields: utils.FilterFields[catalog.UpdateRegisteredModelRequest](config.ForceSendFields, "Owner", "NewName"),

		// Owner is not part of the configuration tree
		Owner: "",

		// Name updates are not supported yet without recreating. Can be added as a follow-up.
		// Note: TF also does not support changing name without a recreate so the current behavior matches TF.
		NewName: "",

		// Aliases are synced separately via SetAlias/DeleteAlias calls because
		// the Update API ignores the Aliases field.
		Aliases:         nil,
		BrowseOnly:      config.BrowseOnly,
		CreatedAt:       config.CreatedAt,
		CreatedBy:       config.CreatedBy,
		MetastoreId:     config.MetastoreId,
		UpdatedAt:       config.UpdatedAt,
		UpdatedBy:       config.UpdatedBy,
		SchemaName:      config.SchemaName,
		StorageLocation: config.StorageLocation,
		Name:            config.Name,
		CatalogName:     config.CatalogName,
	}

	response, err := r.client.RegisteredModels.Update(ctx, updateRequest)
	if err != nil {
		return nil, err
	}

	if entry.Changes.HasChange(pathAliases) {
		// The planner read the remote state with IncludeAliases=true, so the
		// current aliases come from the plan entry instead of a second GET.
		remote, ok := entry.RemoteState.(*catalog.RegisteredModelInfo)
		if !ok {
			return nil, fmt.Errorf("internal error: unexpected remote state type %T", entry.RemoteState)
		}
		if err := r.syncAliases(ctx, id, config.Aliases, remote.Aliases); err != nil {
			return nil, err
		}
	}

	return response, nil
}

func (r *ResourceRegisteredModel) DoDelete(ctx context.Context, id string, _ *catalog.CreateRegisteredModelRequest) error {
	return r.client.RegisteredModels.Delete(ctx, catalog.DeleteRegisteredModelRequest{
		FullName: id,
	})
}

// syncAliases diffs desired against current aliases and calls SetAlias/DeleteAlias
// to reconcile the difference. The Create and Update APIs ignore the Aliases field,
// so separate API calls are required.
func (r *ResourceRegisteredModel) syncAliases(ctx context.Context, fullName string, desired, current []catalog.RegisteredModelAlias) error {
	desiredByName := make(map[string]int, len(desired))
	for _, a := range desired {
		desiredByName[a.AliasName] = a.VersionNum
	}

	currentByName := make(map[string]int, len(current))
	for _, a := range current {
		currentByName[a.AliasName] = a.VersionNum
	}

	// Set new or updated aliases.
	for _, a := range desired {
		if v, ok := currentByName[a.AliasName]; ok && v == a.VersionNum {
			continue
		}
		_, err := r.client.RegisteredModels.SetAlias(ctx, catalog.SetRegisteredModelAliasRequest{
			FullName:   fullName,
			Alias:      a.AliasName,
			VersionNum: a.VersionNum,
		})
		if err != nil {
			return err
		}
	}

	// Delete removed aliases.
	for _, a := range current {
		if _, ok := desiredByName[a.AliasName]; !ok {
			err := r.client.RegisteredModels.DeleteAlias(ctx, catalog.DeleteAliasRequest{
				FullName: fullName,
				Alias:    a.AliasName,
			})
			if err != nil {
				return err
			}
		}
	}

	return nil
}
