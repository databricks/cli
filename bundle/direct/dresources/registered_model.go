package dresources

import (
	"context"
	"fmt"
	"slices"

	"github.com/databricks/cli/bundle/config/resources"
	"github.com/databricks/cli/bundle/deployplan"
	"github.com/databricks/cli/libs/structs/structpath"
	"github.com/databricks/cli/libs/utils"
	"github.com/databricks/databricks-sdk-go"
	"github.com/databricks/databricks-sdk-go/service/catalog"
)

// registeredModelAliasesPath is the state path of the aliases slice, used to detect alias changes.
var registeredModelAliasesPath = structpath.MustParsePath("aliases")

type ResourceRegisteredModel struct {
	client *databricks.WorkspaceClient
}

func (*ResourceRegisteredModel) New(client *databricks.WorkspaceClient) *ResourceRegisteredModel {
	return &ResourceRegisteredModel{
		client: client,
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
		// ignore_remote_changes in registered_models.yml rather than zeroed here.
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

func registeredModelAliasKey(alias catalog.RegisteredModelAlias) (string, string) {
	return "alias_name", alias.AliasName
}

// KeyedSlices compares aliases by name, so the order the backend lists them in does not produce phantom diffs.
func (*ResourceRegisteredModel) KeyedSlices() map[string]any {
	return map[string]any{"aliases": registeredModelAliasKey}
}

func (r *ResourceRegisteredModel) DoCreate(ctx context.Context, config *catalog.CreateRegisteredModelRequest) (string, *catalog.RegisteredModelInfo, error) {
	response, err := r.client.RegisteredModels.Create(ctx, *config)
	if err != nil {
		return "", nil, err
	}

	return response.FullName, response, nil
}

// WaitAfterCreate sets the configured aliases through the alias API. It is not a wait, but unlike
// DoCreate it runs after the model is saved to state: an alias can only point at a registered version,
// so a failure here leaves the model tracked and the next deploy plans the alias again.
func (r *ResourceRegisteredModel) WaitAfterCreate(ctx context.Context, id string, config *catalog.CreateRegisteredModelRequest) (*catalog.RegisteredModelInfo, error) {
	return nil, r.syncAliases(ctx, id, config.Aliases, nil)
}

// See schemaForceSend. Verified against a real workspace: {"comment": ""} clears it.
var registeredModelForceSend = []string{"Comment"}

func makeUpdateRegisteredModel(fullName, newName string, config *catalog.CreateRegisteredModelRequest) catalog.UpdateRegisteredModelRequest {
	// UC rejects an update that sets both name and new_name.
	name := config.Name
	if newName != "" {
		name = ""
	}
	return catalog.UpdateRegisteredModelRequest{
		FullName:        fullName,
		Comment:         config.Comment,
		ForceSendFields: utils.FilterFields[catalog.UpdateRegisteredModelRequest](append(slices.Clone(registeredModelForceSend), config.ForceSendFields...), "Owner", "NewName"),

		// Owner is settable in the config (it comes from the embedded
		// CreateRegisteredModelRequest) and create sends it, but update never has: a
		// change to it is silently dropped rather than applied.
		Owner: "",

		// Set by DoUpdateWithID when the name changes.
		NewName: newName,

		// The update API ignores aliases; syncAliases applies them through the alias API.
		Aliases: nil,

		BrowseOnly:      config.BrowseOnly,
		CreatedAt:       config.CreatedAt,
		CreatedBy:       config.CreatedBy,
		MetastoreId:     config.MetastoreId,
		UpdatedAt:       config.UpdatedAt,
		UpdatedBy:       config.UpdatedBy,
		SchemaName:      config.SchemaName,
		StorageLocation: config.StorageLocation,
		Name:            name,
		CatalogName:     config.CatalogName,
	}
}

func (r *ResourceRegisteredModel) DoUpdate(ctx context.Context, id string, config *catalog.CreateRegisteredModelRequest, entry *PlanEntry) (*catalog.RegisteredModelInfo, error) {
	response, err := r.client.RegisteredModels.Update(ctx, makeUpdateRegisteredModel(id, "", config))
	if err != nil {
		return nil, err
	}
	return response, r.syncChangedAliases(ctx, id, config, entry)
}

// DoUpdateWithID renames the model in place when the name changes, keeping its versions; the ID is
// the full name, so it changes too.
func (r *ResourceRegisteredModel) DoUpdateWithID(ctx context.Context, id string, config *catalog.CreateRegisteredModelRequest, entry *PlanEntry) (string, *catalog.RegisteredModelInfo, error) {
	nameFromID, err := getNameFromID(id)
	if err != nil {
		return "", nil, err
	}

	// The catalog and schema come from config: a renamed parent moves the model with it, leaving id
	// stale (see followParentRenames).
	parent := config.CatalogName + "." + config.SchemaName + "."
	newID := parent + config.Name

	if entry.Changes.OnlyParentRenamed() {
		// UC already moved the model with its parent; only the ID changes.
		return newID, nil, nil
	}

	newName := ""
	if config.Name != nameFromID {
		newName = config.Name
	}

	response, err := r.client.RegisteredModels.Update(ctx, makeUpdateRegisteredModel(parent+nameFromID, newName, config))
	if err != nil {
		return "", nil, err
	}
	return newID, response, r.syncChangedAliases(ctx, newID, config, entry)
}

// syncChangedAliases applies the configured aliases when the plan changes them.
func (r *ResourceRegisteredModel) syncChangedAliases(ctx context.Context, fullName string, config *catalog.CreateRegisteredModelRequest, entry *PlanEntry) error {
	if !entry.Changes.HasChange(registeredModelAliasesPath) {
		return nil
	}
	if entry.RemoteState == nil {
		// The plan read the model at a stale ID: an interrupted deploy renamed its parent.
		remote, err := r.DoRead(ctx, fullName)
		if err != nil {
			return err
		}
		return r.syncAliases(ctx, fullName, config.Aliases, remote.Aliases)
	}
	remote, ok := entry.RemoteState.(*catalog.RegisteredModelInfo)
	if !ok {
		return fmt.Errorf("internal error: unexpected remote state type %T", entry.RemoteState)
	}
	return r.syncAliases(ctx, fullName, config.Aliases, remote.Aliases)
}

// syncAliases makes the model's aliases match desired, given the ones it has remotely. The alias API
// sets or deletes one alias per call. Sets go first, so no alias is deleted unless every set succeeded.
func (r *ResourceRegisteredModel) syncAliases(ctx context.Context, fullName string, desired, remote []catalog.RegisteredModelAlias) error {
	remoteVersions := make(map[string]int, len(remote))
	for _, alias := range remote {
		remoteVersions[alias.AliasName] = alias.VersionNum
	}

	desiredNames := make(map[string]bool, len(desired))
	for _, alias := range desired {
		desiredNames[alias.AliasName] = true
		if version, ok := remoteVersions[alias.AliasName]; ok && version == alias.VersionNum {
			continue
		}
		_, err := r.client.RegisteredModels.SetAlias(ctx, catalog.SetRegisteredModelAliasRequest{
			FullName:   fullName,
			Alias:      alias.AliasName,
			VersionNum: alias.VersionNum,
		})
		if err != nil {
			return err
		}
	}

	for _, alias := range remote {
		if desiredNames[alias.AliasName] {
			continue
		}
		err := r.client.RegisteredModels.DeleteAlias(ctx, catalog.DeleteAliasRequest{
			FullName: fullName,
			Alias:    alias.AliasName,
		})
		if err != nil {
			return err
		}
	}

	return nil
}

// OverrideChangeDesc leaves aliases unmanaged when the config has no aliases section. MLflow clients
// commonly assign aliases to the versions they register (e.g. a "champion" picked by an evaluation
// job), so without a section what the model reports is not drift. The difference then arrives as a
// single whole-field change with New nil; `aliases: []` keeps New non-nil, so it stays managed.
func (*ResourceRegisteredModel) OverrideChangeDesc(_ context.Context, path *structpath.PathNode, change *ChangeDesc, _ *catalog.RegisteredModelInfo) error {
	if change.Action == deployplan.Update && change.New == nil && path.String() == "aliases" {
		change.Action = deployplan.Skip
		change.Reason = deployplan.ReasonUnmanaged
	}
	return nil
}

func (r *ResourceRegisteredModel) DoDelete(ctx context.Context, id string, _ *catalog.CreateRegisteredModelRequest) error {
	return r.client.RegisteredModels.Delete(ctx, catalog.DeleteRegisteredModelRequest{
		FullName: id,
	})
}
