package direct

import (
	"context"
	"fmt"
	"sort"
	"strings"

	"github.com/databricks/cli/libs/log"
	"github.com/databricks/cli/ucm"
	"github.com/databricks/cli/ucm/config/resources"
	"github.com/databricks/cli/ucm/deployplan"
	"github.com/databricks/databricks-sdk-go/service/catalog"
)

// Apply executes the given plan against the provided client, updating state
// in place as each action succeeds. On a mid-plan error the already-applied
// state is preserved — the caller is expected to persist state afterwards so
// a retry sees the partial progress.
//
// Execution order is the natural UC dependency order:
//
//	storage_credential creates+updates → external_location creates+updates
//	→ catalog creates+updates → schema creates+updates → grants reconcile
//	→ schema deletes → catalog deletes → external_location deletes
//	→ storage_credential deletes
//
// Grants are reconciled per securable in a single pass (Create, Update, and
// Delete share the code path) because the UC API treats grants as a full
// replacement of the (principal, privileges) set on a given securable. A
// per-grant-key plan shape still makes individual additions/removals
// observable to users in the plan output.
func Apply(ctx context.Context, u *ucm.Ucm, client Client, plan *deployplan.Plan, state *State) error {
	if err := applyStorageCredentialCreates(ctx, u, client, plan, state); err != nil {
		return err
	}
	if err := applyExternalLocationCreates(ctx, u, client, plan, state); err != nil {
		return err
	}
	if err := applyCatalogCreates(ctx, u, client, plan, state); err != nil {
		return err
	}
	if err := applySchemaCreates(ctx, u, client, plan, state); err != nil {
		return err
	}
	if err := applyVolumeCreates(ctx, u, client, plan, state); err != nil {
		return err
	}
	if err := applyConnectionCreates(ctx, u, client, plan, state); err != nil {
		return err
	}
	if err := applyGrantChanges(ctx, u, client, plan, state); err != nil {
		return err
	}
	if err := applyConnectionDeletes(ctx, client, plan, state); err != nil {
		return err
	}
	if err := applyVolumeDeletes(ctx, client, plan, state); err != nil {
		return err
	}
	if err := applySchemaDeletes(ctx, client, plan, state); err != nil {
		return err
	}
	if err := applyCatalogDeletes(ctx, client, plan, state); err != nil {
		return err
	}
	if err := applyExternalLocationDeletes(ctx, client, plan, state); err != nil {
		return err
	}
	if err := applyStorageCredentialDeletes(ctx, client, plan, state); err != nil {
		return err
	}
	return nil
}

// Destroy builds a plan where every recorded resource gets a Delete action
// and runs it through Apply. The resulting plan is returned for rendering by
// the caller.
func Destroy(ctx context.Context, u *ucm.Ucm, client Client, state *State) (*deployplan.Plan, error) {
	plan := deployplan.NewPlanTerraform()
	for key := range state.Grants {
		plan.Plan["resources.grants."+key] = &deployplan.PlanEntry{Action: deployplan.Delete}
	}
	for key := range state.Connections {
		plan.Plan["resources.connections."+key] = &deployplan.PlanEntry{Action: deployplan.Delete}
	}
	for key := range state.Volumes {
		plan.Plan["resources.volumes."+key] = &deployplan.PlanEntry{Action: deployplan.Delete}
	}
	for key := range state.Schemas {
		plan.Plan["resources.schemas."+key] = &deployplan.PlanEntry{Action: deployplan.Delete}
	}
	for key := range state.Catalogs {
		plan.Plan["resources.catalogs."+key] = &deployplan.PlanEntry{Action: deployplan.Delete}
	}
	for key := range state.ExternalLocations {
		plan.Plan["resources.external_locations."+key] = &deployplan.PlanEntry{Action: deployplan.Delete}
	}
	for key := range state.StorageCredentials {
		plan.Plan["resources.storage_credentials."+key] = &deployplan.PlanEntry{Action: deployplan.Delete}
	}
	if err := Apply(ctx, u, client, plan, state); err != nil {
		return plan, err
	}
	return plan, nil
}

func applyStorageCredentialCreates(ctx context.Context, u *ucm.Ucm, client Client, plan *deployplan.Plan, state *State) error {
	for _, key := range sortedPlanKeysByGroup(plan, "storage_credentials") {
		entry := plan.Plan[key]
		name := strings.TrimPrefix(key, "resources.storage_credentials.")
		cfg := u.Config.Resources.StorageCredentials[name]
		switch entry.Action {
		case deployplan.Create:
			log.Infof(ctx, "direct: creating storage_credential %s", name)
			in, err := storageCredentialCreateInput(cfg)
			if err != nil {
				return fmt.Errorf("create storage_credential %s: %w", name, err)
			}
			if _, err := client.CreateStorageCredential(ctx, in); err != nil {
				return fmt.Errorf("create storage_credential %s: %w", name, err)
			}
			state.StorageCredentials[name] = ptrStorageCredential(storageCredentialStateFromConfig(cfg))
		case deployplan.Update:
			log.Infof(ctx, "direct: updating storage_credential %s", name)
			in, err := storageCredentialUpdateInput(cfg)
			if err != nil {
				return fmt.Errorf("update storage_credential %s: %w", name, err)
			}
			if _, err := client.UpdateStorageCredential(ctx, in); err != nil {
				return fmt.Errorf("update storage_credential %s: %w", name, err)
			}
			state.StorageCredentials[name] = ptrStorageCredential(storageCredentialStateFromConfig(cfg))
		}
	}
	return nil
}

func applyStorageCredentialDeletes(ctx context.Context, client Client, plan *deployplan.Plan, state *State) error {
	for _, key := range reverseSortedPlanKeysByGroup(plan, "storage_credentials") {
		entry := plan.Plan[key]
		if entry.Action != deployplan.Delete {
			continue
		}
		name := strings.TrimPrefix(key, "resources.storage_credentials.")
		rec, ok := state.StorageCredentials[name]
		if !ok {
			continue
		}
		log.Infof(ctx, "direct: deleting storage_credential %s", rec.Name)
		if err := client.DeleteStorageCredential(ctx, rec.Name); err != nil {
			return fmt.Errorf("delete storage_credential %s: %w", rec.Name, err)
		}
		delete(state.StorageCredentials, name)
	}
	return nil
}

func applyExternalLocationCreates(ctx context.Context, u *ucm.Ucm, client Client, plan *deployplan.Plan, state *State) error {
	for _, key := range sortedPlanKeysByGroup(plan, "external_locations") {
		entry := plan.Plan[key]
		name := strings.TrimPrefix(key, "resources.external_locations.")
		cfg := u.Config.Resources.ExternalLocations[name]
		switch entry.Action {
		case deployplan.Create:
			log.Infof(ctx, "direct: creating external_location %s", name)
			if _, err := client.CreateExternalLocation(ctx, externalLocationCreateInput(cfg)); err != nil {
				return fmt.Errorf("create external_location %s: %w", name, err)
			}
			state.ExternalLocations[name] = ptrExternalLocation(externalLocationStateFromConfig(cfg))
		case deployplan.Update:
			log.Infof(ctx, "direct: updating external_location %s", name)
			if _, err := client.UpdateExternalLocation(ctx, externalLocationUpdateInput(cfg)); err != nil {
				return fmt.Errorf("update external_location %s: %w", name, err)
			}
			state.ExternalLocations[name] = ptrExternalLocation(externalLocationStateFromConfig(cfg))
		}
	}
	return nil
}

func applyExternalLocationDeletes(ctx context.Context, client Client, plan *deployplan.Plan, state *State) error {
	for _, key := range reverseSortedPlanKeysByGroup(plan, "external_locations") {
		entry := plan.Plan[key]
		if entry.Action != deployplan.Delete {
			continue
		}
		name := strings.TrimPrefix(key, "resources.external_locations.")
		rec, ok := state.ExternalLocations[name]
		if !ok {
			continue
		}
		log.Infof(ctx, "direct: deleting external_location %s", rec.Name)
		if err := client.DeleteExternalLocation(ctx, rec.Name); err != nil {
			return fmt.Errorf("delete external_location %s: %w", rec.Name, err)
		}
		delete(state.ExternalLocations, name)
	}
	return nil
}

func applyCatalogCreates(ctx context.Context, u *ucm.Ucm, client Client, plan *deployplan.Plan, state *State) error {
	for _, key := range sortedPlanKeysByGroup(plan, "catalogs") {
		entry := plan.Plan[key]
		name := strings.TrimPrefix(key, "resources.catalogs.")
		cfg := u.Config.Resources.Catalogs[name]
		switch entry.Action {
		case deployplan.Create:
			log.Infof(ctx, "direct: creating catalog %s", name)
			if _, err := client.CreateCatalog(ctx, catalogCreateInput(cfg)); err != nil {
				return fmt.Errorf("create catalog %s: %w", name, err)
			}
			state.Catalogs[name] = ptrCatalog(catalogStateFromConfig(cfg))
		case deployplan.Update:
			log.Infof(ctx, "direct: updating catalog %s", name)
			if _, err := client.UpdateCatalog(ctx, catalogUpdateInput(cfg)); err != nil {
				return fmt.Errorf("update catalog %s: %w", name, err)
			}
			state.Catalogs[name] = ptrCatalog(catalogStateFromConfig(cfg))
		}
	}
	return nil
}

func applySchemaCreates(ctx context.Context, u *ucm.Ucm, client Client, plan *deployplan.Plan, state *State) error {
	for _, key := range sortedPlanKeysByGroup(plan, "schemas") {
		entry := plan.Plan[key]
		name := strings.TrimPrefix(key, "resources.schemas.")
		cfg := u.Config.Resources.Schemas[name]
		switch entry.Action {
		case deployplan.Create:
			log.Infof(ctx, "direct: creating schema %s.%s", cfg.CatalogName, cfg.Name)
			if _, err := client.CreateSchema(ctx, schemaCreateInput(cfg)); err != nil {
				return fmt.Errorf("create schema %s.%s: %w", cfg.CatalogName, cfg.Name, err)
			}
			state.Schemas[name] = ptrSchema(schemaStateFromConfig(cfg))
		case deployplan.Update:
			log.Infof(ctx, "direct: updating schema %s.%s", cfg.CatalogName, cfg.Name)
			if _, err := client.UpdateSchema(ctx, schemaUpdateInput(cfg)); err != nil {
				return fmt.Errorf("update schema %s.%s: %w", cfg.CatalogName, cfg.Name, err)
			}
			state.Schemas[name] = ptrSchema(schemaStateFromConfig(cfg))
		}
	}
	return nil
}

func applyVolumeCreates(ctx context.Context, u *ucm.Ucm, client Client, plan *deployplan.Plan, state *State) error {
	for _, key := range sortedPlanKeysByGroup(plan, "volumes") {
		entry := plan.Plan[key]
		name := strings.TrimPrefix(key, "resources.volumes.")
		cfg := u.Config.Resources.Volumes[name]
		switch entry.Action {
		case deployplan.Create:
			log.Infof(ctx, "direct: creating volume %s.%s.%s", cfg.CatalogName, cfg.SchemaName, cfg.Name)
			in, err := volumeCreateInput(cfg)
			if err != nil {
				return fmt.Errorf("create volume %s.%s.%s: %w", cfg.CatalogName, cfg.SchemaName, cfg.Name, err)
			}
			if _, err := client.CreateVolume(ctx, in); err != nil {
				return fmt.Errorf("create volume %s.%s.%s: %w", cfg.CatalogName, cfg.SchemaName, cfg.Name, err)
			}
			state.Volumes[name] = ptrVolume(volumeStateFromConfig(cfg))
		case deployplan.Update:
			log.Infof(ctx, "direct: updating volume %s.%s.%s", cfg.CatalogName, cfg.SchemaName, cfg.Name)
			if _, err := client.UpdateVolume(ctx, volumeUpdateInput(cfg)); err != nil {
				return fmt.Errorf("update volume %s.%s.%s: %w", cfg.CatalogName, cfg.SchemaName, cfg.Name, err)
			}
			state.Volumes[name] = ptrVolume(volumeStateFromConfig(cfg))
		}
	}
	return nil
}

func applyVolumeDeletes(ctx context.Context, client Client, plan *deployplan.Plan, state *State) error {
	for _, key := range reverseSortedPlanKeysByGroup(plan, "volumes") {
		entry := plan.Plan[key]
		if entry.Action != deployplan.Delete {
			continue
		}
		name := strings.TrimPrefix(key, "resources.volumes.")
		rec, ok := state.Volumes[name]
		if !ok {
			continue
		}
		fullName := rec.CatalogName + "." + rec.SchemaName + "." + rec.Name
		log.Infof(ctx, "direct: deleting volume %s", fullName)
		if err := client.DeleteVolume(ctx, fullName); err != nil {
			return fmt.Errorf("delete volume %s: %w", fullName, err)
		}
		delete(state.Volumes, name)
	}
	return nil
}

func applyConnectionCreates(ctx context.Context, u *ucm.Ucm, client Client, plan *deployplan.Plan, state *State) error {
	for _, key := range sortedPlanKeysByGroup(plan, "connections") {
		entry := plan.Plan[key]
		name := strings.TrimPrefix(key, "resources.connections.")
		cfg := u.Config.Resources.Connections[name]
		switch entry.Action {
		case deployplan.Create:
			log.Infof(ctx, "direct: creating connection %s", cfg.Name)
			in, err := connectionCreateInput(cfg)
			if err != nil {
				return fmt.Errorf("create connection %s: %w", cfg.Name, err)
			}
			if _, err := client.CreateConnection(ctx, in); err != nil {
				return fmt.Errorf("create connection %s: %w", cfg.Name, err)
			}
			state.Connections[name] = ptrConnection(connectionStateFromConfig(cfg))
		case deployplan.Update:
			log.Infof(ctx, "direct: updating connection %s", cfg.Name)
			if _, err := client.UpdateConnection(ctx, connectionUpdateInput(cfg)); err != nil {
				return fmt.Errorf("update connection %s: %w", cfg.Name, err)
			}
			state.Connections[name] = ptrConnection(connectionStateFromConfig(cfg))
		}
	}
	return nil
}

func applyConnectionDeletes(ctx context.Context, client Client, plan *deployplan.Plan, state *State) error {
	for _, key := range reverseSortedPlanKeysByGroup(plan, "connections") {
		entry := plan.Plan[key]
		if entry.Action != deployplan.Delete {
			continue
		}
		name := strings.TrimPrefix(key, "resources.connections.")
		rec, ok := state.Connections[name]
		if !ok {
			continue
		}
		log.Infof(ctx, "direct: deleting connection %s", rec.Name)
		if err := client.DeleteConnection(ctx, rec.Name); err != nil {
			return fmt.Errorf("delete connection %s: %w", rec.Name, err)
		}
		delete(state.Connections, name)
	}
	return nil
}

// applyGrantChanges coalesces every grant Create/Update/Delete by securable
// and issues one UpdatePermissions call per affected securable carrying the
// full desired assignment set. The per-key plan entries are preserved in
// state so the next run can diff individual grants; the batching only
// affects the apply wire call.
//
// All three action kinds go through the same code path: any plan entry on a
// grant means "reconcile that securable". For a Delete we look up the
// previously-recorded state to learn which securable the removed key was
// attached to; for a Create/Update we read it from config.
func applyGrantChanges(ctx context.Context, u *ucm.Ucm, client Client, plan *deployplan.Plan, state *State) error {
	touched := touchedGrantSecurables(plan, u.Config.Resources.Grants, state)
	if len(touched) == 0 {
		return nil
	}
	desired := grantsBySecurable(u.Config.Resources.Grants)
	for _, sec := range sortSecurables(touched) {
		log.Infof(ctx, "direct: reconciling grants on %s %s", sec.Type, sec.Name)
		in := buildUpdatePermissions(sec, desired[sec])
		in.Changes = append(in.Changes, revocationsForRemovedPrincipals(state, sec, desired[sec])...)
		if err := client.UpdatePermissions(ctx, in); err != nil {
			return fmt.Errorf("update grants on %s %s: %w", sec.Type, sec.Name, err)
		}
	}
	for _, key := range sortedPlanKeysByGroup(plan, "grants") {
		entry := plan.Plan[key]
		name := strings.TrimPrefix(key, "resources.grants.")
		switch entry.Action {
		case deployplan.Create, deployplan.Update:
			cfg := u.Config.Resources.Grants[name]
			state.Grants[name] = ptrGrant(grantStateFromConfig(cfg))
		case deployplan.Delete:
			delete(state.Grants, name)
		}
	}
	return nil
}

func applySchemaDeletes(ctx context.Context, client Client, plan *deployplan.Plan, state *State) error {
	for _, key := range reverseSortedPlanKeysByGroup(plan, "schemas") {
		entry := plan.Plan[key]
		if entry.Action != deployplan.Delete {
			continue
		}
		name := strings.TrimPrefix(key, "resources.schemas.")
		rec, ok := state.Schemas[name]
		if !ok {
			continue
		}
		fullName := rec.Catalog + "." + rec.Name
		log.Infof(ctx, "direct: deleting schema %s", fullName)
		if err := client.DeleteSchema(ctx, fullName); err != nil {
			return fmt.Errorf("delete schema %s: %w", fullName, err)
		}
		delete(state.Schemas, name)
	}
	return nil
}

func applyCatalogDeletes(ctx context.Context, client Client, plan *deployplan.Plan, state *State) error {
	for _, key := range reverseSortedPlanKeysByGroup(plan, "catalogs") {
		entry := plan.Plan[key]
		if entry.Action != deployplan.Delete {
			continue
		}
		name := strings.TrimPrefix(key, "resources.catalogs.")
		rec, ok := state.Catalogs[name]
		if !ok {
			continue
		}
		log.Infof(ctx, "direct: deleting catalog %s", rec.Name)
		if err := client.DeleteCatalog(ctx, rec.Name); err != nil {
			return fmt.Errorf("delete catalog %s: %w", rec.Name, err)
		}
		delete(state.Catalogs, name)
	}
	return nil
}

// ---- SDK input builders ----

func catalogCreateInput(c *resources.Catalog) catalog.CreateCatalog {
	in := c.CreateCatalog
	if in.Properties == nil {
		in.Properties = copyTags(c.Tags)
	}
	return in
}

func catalogUpdateInput(c *resources.Catalog) catalog.UpdateCatalog {
	return catalog.UpdateCatalog{
		Name:       c.Name,
		Comment:    c.Comment,
		Properties: copyTags(c.Tags),
	}
}

func schemaCreateInput(s *resources.Schema) catalog.CreateSchema {
	in := s.CreateSchema
	if in.Properties == nil {
		in.Properties = copyTags(s.Tags)
	}
	return in
}

func schemaUpdateInput(s *resources.Schema) catalog.UpdateSchema {
	return catalog.UpdateSchema{
		FullName:   s.CatalogName + "." + s.Name,
		Comment:    s.Comment,
		Properties: copyTags(s.Tags),
	}
}

// storageCredentialIdentityCount counts which of the one-of identity fields
// are set on the config struct. Matches the tfdyn converter's validation.
func storageCredentialIdentityCount(c *resources.StorageCredential) int {
	n := 0
	if c.AwsIamRole != nil {
		n++
	}
	if c.AzureManagedIdentity != nil {
		n++
	}
	if c.AzureServicePrincipal != nil {
		n++
	}
	if c.DatabricksGcpServiceAccount != nil {
		n++
	}
	return n
}

func storageCredentialCreateInput(c *resources.StorageCredential) (catalog.CreateStorageCredential, error) {
	if n := storageCredentialIdentityCount(c); n != 1 {
		return catalog.CreateStorageCredential{}, fmt.Errorf("storage_credential %q: exactly one identity field required, got %d", c.Name, n)
	}
	in := catalog.CreateStorageCredential{
		Name:           c.Name,
		Comment:        c.Comment,
		ReadOnly:       c.ReadOnly,
		SkipValidation: c.SkipValidation,
	}
	if c.AwsIamRole != nil {
		in.AwsIamRole = &catalog.AwsIamRoleRequest{RoleArn: c.AwsIamRole.RoleArn}
	}
	if c.AzureManagedIdentity != nil {
		in.AzureManagedIdentity = &catalog.AzureManagedIdentityRequest{
			AccessConnectorId: c.AzureManagedIdentity.AccessConnectorId,
			ManagedIdentityId: c.AzureManagedIdentity.ManagedIdentityId,
		}
	}
	if c.AzureServicePrincipal != nil {
		in.AzureServicePrincipal = &catalog.AzureServicePrincipal{
			DirectoryId:   c.AzureServicePrincipal.DirectoryId,
			ApplicationId: c.AzureServicePrincipal.ApplicationId,
			ClientSecret:  c.AzureServicePrincipal.ClientSecret,
		}
	}
	if c.DatabricksGcpServiceAccount != nil {
		in.DatabricksGcpServiceAccount = &catalog.DatabricksGcpServiceAccountRequest{}
	}
	return in, nil
}

// storageCredentialUpdateInput mirrors storageCredentialCreateInput except
// Azure managed identity uses the SDK's *Response* type for updates — an
// SDK quirk, not a bug on our side.
func storageCredentialUpdateInput(c *resources.StorageCredential) (catalog.UpdateStorageCredential, error) {
	if n := storageCredentialIdentityCount(c); n != 1 {
		return catalog.UpdateStorageCredential{}, fmt.Errorf("storage_credential %q: exactly one identity field required, got %d", c.Name, n)
	}
	in := catalog.UpdateStorageCredential{
		Name:           c.Name,
		Comment:        c.Comment,
		ReadOnly:       c.ReadOnly,
		SkipValidation: c.SkipValidation,
	}
	if c.AwsIamRole != nil {
		in.AwsIamRole = &catalog.AwsIamRoleRequest{RoleArn: c.AwsIamRole.RoleArn}
	}
	if c.AzureManagedIdentity != nil {
		in.AzureManagedIdentity = &catalog.AzureManagedIdentityResponse{
			AccessConnectorId: c.AzureManagedIdentity.AccessConnectorId,
			ManagedIdentityId: c.AzureManagedIdentity.ManagedIdentityId,
		}
	}
	if c.AzureServicePrincipal != nil {
		in.AzureServicePrincipal = &catalog.AzureServicePrincipal{
			DirectoryId:   c.AzureServicePrincipal.DirectoryId,
			ApplicationId: c.AzureServicePrincipal.ApplicationId,
			ClientSecret:  c.AzureServicePrincipal.ClientSecret,
		}
	}
	if c.DatabricksGcpServiceAccount != nil {
		in.DatabricksGcpServiceAccount = &catalog.DatabricksGcpServiceAccountRequest{}
	}
	return in, nil
}

// volumeCreateInput validates the MANAGED/EXTERNAL invariant and builds the
// SDK Create payload. EXTERNAL volumes require storage_location; MANAGED ones
// must not carry one.
func volumeCreateInput(v *resources.Volume) (catalog.CreateVolumeRequestContent, error) {
	vType := catalog.VolumeType(strings.ToUpper(string(v.VolumeType)))
	if vType != catalog.VolumeTypeManaged && vType != catalog.VolumeTypeExternal {
		return catalog.CreateVolumeRequestContent{}, fmt.Errorf("volume %q: volume_type must be MANAGED or EXTERNAL, got %q", v.Name, v.VolumeType)
	}
	if vType == catalog.VolumeTypeExternal && v.StorageLocation == "" {
		return catalog.CreateVolumeRequestContent{}, fmt.Errorf("volume %q: storage_location is required for EXTERNAL volumes", v.Name)
	}
	if vType == catalog.VolumeTypeManaged && v.StorageLocation != "" {
		return catalog.CreateVolumeRequestContent{}, fmt.Errorf("volume %q: storage_location must not be set for MANAGED volumes", v.Name)
	}
	in := v.CreateVolumeRequestContent
	in.VolumeType = vType
	return in, nil
}

// volumeUpdateInput produces a comment-only update. The UC API only supports
// renaming, changing the owner, or updating the comment on a volume — drift
// on catalog/schema/volume_type/storage_location is effectively immutable.
func volumeUpdateInput(v *resources.Volume) catalog.UpdateVolumeRequestContent {
	return catalog.UpdateVolumeRequestContent{
		Name:    v.CatalogName + "." + v.SchemaName + "." + v.Name,
		Comment: v.Comment,
	}
}

// connectionCreateInput validates options is non-empty and builds the SDK
// Create payload. Per-connection-type key validation lives server-side.
func connectionCreateInput(c *resources.Connection) (catalog.CreateConnection, error) {
	if c.ConnectionType == "" {
		return catalog.CreateConnection{}, fmt.Errorf("connection %q: connection_type is required", c.Name)
	}
	if len(c.Options) == 0 {
		return catalog.CreateConnection{}, fmt.Errorf("connection %q: options is required and must be non-empty", c.Name)
	}
	return catalog.CreateConnection{
		Name:           c.Name,
		ConnectionType: catalog.ConnectionType(c.ConnectionType),
		Options:        copyTags(c.Options),
		Comment:        c.Comment,
		Properties:     copyTags(c.Properties),
		ReadOnly:       c.ReadOnly,
	}, nil
}

// connectionUpdateInput produces an options-only update. The UC API allows
// changing only name/owner/options on a connection — connection_type,
// comment, properties, and read_only drift is effectively immutable.
func connectionUpdateInput(c *resources.Connection) catalog.UpdateConnection {
	return catalog.UpdateConnection{
		Name:    c.Name,
		Options: copyTags(c.Options),
	}
}

func externalLocationCreateInput(e *resources.ExternalLocation) catalog.CreateExternalLocation {
	return catalog.CreateExternalLocation{
		Name:           e.Name,
		Url:            e.Url,
		CredentialName: e.CredentialName,
		Comment:        e.Comment,
		ReadOnly:       e.ReadOnly,
		SkipValidation: e.SkipValidation,
		Fallback:       e.Fallback,
	}
}

func externalLocationUpdateInput(e *resources.ExternalLocation) catalog.UpdateExternalLocation {
	return catalog.UpdateExternalLocation{
		Name:           e.Name,
		Url:            e.Url,
		CredentialName: e.CredentialName,
		Comment:        e.Comment,
		ReadOnly:       e.ReadOnly,
		SkipValidation: e.SkipValidation,
		Fallback:       e.Fallback,
	}
}

func buildUpdatePermissions(sec securable, grants []*resources.Grant) catalog.UpdatePermissions {
	changes := make([]catalog.PermissionsChange, 0, len(grants))
	for _, g := range grants {
		privs := make([]catalog.Privilege, 0, len(g.Privileges))
		for _, p := range g.Privileges {
			privs = append(privs, catalog.Privilege(p))
		}
		change := catalog.PermissionsChange{
			Principal: g.Principal,
			Add:       privs,
		}
		if !containsAllPrivileges(privs) {
			change.Remove = []catalog.Privilege{catalog.PrivilegeAllPrivileges}
		}
		changes = append(changes, change)
	}
	return catalog.UpdatePermissions{
		SecurableType: sec.Type,
		FullName:      sec.Name,
		Changes:       changes,
	}
}

func revocationsForRemovedPrincipals(state *State, sec securable, desiredGrants []*resources.Grant) []catalog.PermissionsChange {
	desiredPrincipals := make(map[string]struct{}, len(desiredGrants))
	for _, g := range desiredGrants {
		desiredPrincipals[g.Principal] = struct{}{}
	}
	seen := make(map[string]struct{})
	var revs []catalog.PermissionsChange
	for _, g := range state.Grants {
		if g.SecurableType != sec.Type || g.SecurableName != sec.Name {
			continue
		}
		if _, keep := desiredPrincipals[g.Principal]; keep {
			continue
		}
		if _, dup := seen[g.Principal]; dup {
			continue
		}
		seen[g.Principal] = struct{}{}
		revs = append(revs, catalog.PermissionsChange{
			Principal: g.Principal,
			Remove:    []catalog.Privilege{catalog.PrivilegeAllPrivileges},
		})
	}
	return revs
}

func containsAllPrivileges(privs []catalog.Privilege) bool {
	for _, p := range privs {
		if p == catalog.PrivilegeAllPrivileges {
			return true
		}
	}
	return false
}

// ---- helpers ----

// securable is the (type, name) tuple the grants API keys by. The UCM
// config's Securable carries the same fields — re-defined here to keep the
// apply-side types in one place.
type securable struct {
	Type string
	Name string
}

// touchedGrantSecurables returns the set of securables that have at least
// one grant plan entry whose action requires a reconcile (Create/Update
// /Delete). The reader resolves key → securable out of config first, falling
// back to state when the key was only present there (i.e. a Delete).
func touchedGrantSecurables(plan *deployplan.Plan, desired map[string]*resources.Grant, state *State) map[securable]struct{} {
	out := map[securable]struct{}{}
	for _, key := range sortedPlanKeysByGroup(plan, "grants") {
		entry := plan.Plan[key]
		switch entry.Action {
		case deployplan.Create, deployplan.Update, deployplan.Delete:
		default:
			continue
		}
		name := strings.TrimPrefix(key, "resources.grants.")
		if cfg, ok := desired[name]; ok {
			out[securable{Type: cfg.Securable.Type, Name: cfg.Securable.Name}] = struct{}{}
			continue
		}
		if rec, ok := state.Grants[name]; ok {
			out[securable{Type: rec.SecurableType, Name: rec.SecurableName}] = struct{}{}
		}
	}
	return out
}

// sortedPlanKeysByGroup returns the plan keys under "resources.<group>." in
// lexical order. Used to make apply ordering deterministic within a step.
func sortedPlanKeysByGroup(plan *deployplan.Plan, group string) []string {
	prefix := "resources." + group + "."
	var keys []string
	for k := range plan.Plan {
		if strings.HasPrefix(k, prefix) {
			keys = append(keys, k)
		}
	}
	sort.Strings(keys)
	return keys
}

// reverseSortedPlanKeysByGroup returns the same set in reverse order.
// Used by delete passes so nested resources are torn down before parents.
func reverseSortedPlanKeysByGroup(plan *deployplan.Plan, group string) []string {
	keys := sortedPlanKeysByGroup(plan, group)
	sort.Sort(sort.Reverse(sort.StringSlice(keys)))
	return keys
}

// grantsBySecurable indexes config grants by (securable type, name). Used by
// both the change and the delete pass to compute the full desired assignment
// set for any touched securable.
func grantsBySecurable(grants map[string]*resources.Grant) map[securable][]*resources.Grant {
	out := make(map[securable][]*resources.Grant)
	for _, g := range grants {
		sec := securable{Type: g.Securable.Type, Name: g.Securable.Name}
		out[sec] = append(out[sec], g)
	}
	return out
}

func sortSecurables(set map[securable]struct{}) []securable {
	out := make([]securable, 0, len(set))
	for s := range set {
		out = append(out, s)
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].Type != out[j].Type {
			return out[i].Type < out[j].Type
		}
		return out[i].Name < out[j].Name
	})
	return out
}

func ptrCatalog(s CatalogState) *CatalogState { return &s }
func ptrSchema(s SchemaState) *SchemaState    { return &s }
func ptrGrant(s GrantState) *GrantState       { return &s }
func ptrStorageCredential(s StorageCredentialState) *StorageCredentialState {
	return &s
}

func ptrExternalLocation(s ExternalLocationState) *ExternalLocationState { return &s }

func ptrVolume(s VolumeState) *VolumeState { return &s }

func ptrConnection(s ConnectionState) *ConnectionState { return &s }
