package phases

import (
	"context"
	"fmt"
	"sort"
	"strings"

	"github.com/databricks/cli/libs/log"
	"github.com/databricks/cli/ucm/config"
	"github.com/databricks/cli/ucm/config/resources"
	"github.com/databricks/cli/ucm/deploy/direct"
	"github.com/databricks/databricks-sdk-go/service/catalog"
)

// Generate kinds. Strings match the ucm resource group keys so callers can
// pass a CSV like "catalog,schema,volume" through verbatim.
const (
	KindCatalog           = "catalog"
	KindSchema            = "schema"
	KindStorageCredential = "storage_credential"
	KindExternalLocation  = "external_location"
	KindVolume            = "volume"
	KindConnection        = "connection"
)

// DefaultGenerateKinds is the scan set used when --kinds is omitted. Grants
// are intentionally absent — they reconcile per-securable on deploy, so
// seeding them from a live scan adds noise without improving idempotency.
var DefaultGenerateKinds = []string{
	KindCatalog,
	KindSchema,
	KindStorageCredential,
	KindExternalLocation,
	KindVolume,
	KindConnection,
}

// Skipped system-owned catalogs. `main` is technically user-visible but is a
// workspace default the user never declared, so emitting it would force
// spurious diffs on every subsequent ucm deploy.
var skipCatalogNames = map[string]struct{}{
	"system":                {},
	"hive_metastore":        {},
	"__databricks_internal": {},
	"main":                  {},
}

// skipSchemaNames is the set of system-provided schema names that exist
// inside every catalog and cannot be declared by the user.
var skipSchemaNames = map[string]struct{}{
	"information_schema": {},
}

// GenerateResult is the typed return of Generate. Callers are expected to
// persist Root as ucm.yml and State under .databricks/ucm/<target>/resources.json.
// Warnings surface non-fatal issues like credential fields the SDK cannot
// round-trip — they are informational and should be printed verbatim.
type GenerateResult struct {
	Root     *config.Root
	State    *direct.State
	Warnings []string
}

// GenerateOptions controls the scan. Zero values are valid: Kinds defaults
// to DefaultGenerateKinds, Name to "scanned".
type GenerateOptions struct {
	// Name becomes ucm.name in the emitted config. Required in practice —
	// every ucm.yml must have a name — but the phase defaults it to
	// "scanned" so callers can't panic on an empty Name.
	Name string

	// Host is the workspace URL the scan ran against. Recorded as
	// workspace.host so subsequent `ucm deploy` lands against the same
	// workspace.
	Host string

	// Kinds is the list of resource kinds to scan. Nil means "scan all kinds
	// in DefaultGenerateKinds". Unknown kinds return an error up front.
	Kinds []string
}

// Generate scans the live UC estate behind client and returns a config.Root +
// direct.State pair seeded with everything it found. The scan order matches
// the UC dependency graph (credentials → locations → catalogs → schemas →
// volumes → connections) so we never have to re-walk anything.
func Generate(ctx context.Context, client direct.Client, opts GenerateOptions) (*GenerateResult, error) {
	log.Info(ctx, "Phase: generate")
	kinds, err := normalizeKinds(opts.Kinds)
	if err != nil {
		return nil, err
	}
	name := opts.Name
	if name == "" {
		name = "scanned"
	}

	root := &config.Root{
		Ucm:       config.Ucm{Name: name},
		Workspace: config.Workspace{Host: opts.Host},
		Resources: config.Resources{},
	}
	state := direct.NewState()
	var warnings []string

	wantCatalogs := kinds[KindCatalog]
	wantSchemas := kinds[KindSchema]
	wantVolumes := kinds[KindVolume]

	// Catalogs (and, transitively, schemas + volumes) require a catalog
	// listing up front. Cache it so we don't re-call ListCatalogs when the
	// user asked for schemas or volumes without asking for catalogs
	// themselves.
	var catalogs []catalog.CatalogInfo
	if wantCatalogs || wantSchemas || wantVolumes {
		catalogs, err = client.ListCatalogs(ctx)
		if err != nil {
			return nil, fmt.Errorf("list catalogs: %w", err)
		}
	}

	if kinds[KindStorageCredential] {
		w, err := scanStorageCredentials(ctx, client, root, state)
		if err != nil {
			return nil, err
		}
		warnings = append(warnings, w...)
	}

	if kinds[KindExternalLocation] {
		if err := scanExternalLocations(ctx, client, root, state); err != nil {
			return nil, err
		}
	}

	if wantCatalogs {
		scanCatalogs(catalogs, root, state)
	}

	// Schemas and volumes iterate over the catalog list even when catalogs
	// themselves are not in the scan set — the only way to enumerate them is
	// per-catalog, and skipping the parent catalog emit is a separate choice.
	if wantSchemas || wantVolumes {
		var schemasByCatalog map[string][]catalog.SchemaInfo
		if wantVolumes {
			schemasByCatalog = make(map[string][]catalog.SchemaInfo)
		}
		for _, c := range catalogs {
			if skipCatalog(c) {
				continue
			}
			log.Debugf(ctx, "generate: listing schemas in %s", c.Name)
			schemas, err := client.ListSchemas(ctx, c.Name)
			if err != nil {
				return nil, fmt.Errorf("list schemas in %s: %w", c.Name, err)
			}
			if wantSchemas {
				addSchemas(schemas, root, state)
			}
			if wantVolumes {
				schemasByCatalog[c.Name] = schemas
			}
		}
		if wantVolumes {
			for _, c := range catalogs {
				if skipCatalog(c) {
					continue
				}
				for _, s := range schemasByCatalog[c.Name] {
					if skipSchema(s) {
						continue
					}
					log.Debugf(ctx, "generate: listing volumes in %s.%s", c.Name, s.Name)
					volumes, err := client.ListVolumes(ctx, c.Name, s.Name)
					if err != nil {
						return nil, fmt.Errorf("list volumes in %s.%s: %w", c.Name, s.Name, err)
					}
					addVolumes(volumes, root, state)
				}
			}
		}
	}

	if kinds[KindConnection] {
		if err := scanConnections(ctx, client, root, state); err != nil {
			return nil, err
		}
	}

	return &GenerateResult{Root: root, State: state, Warnings: warnings}, nil
}

// normalizeKinds expands the caller's kind list (or the default set) into a
// lookup map. Unknown kinds return an error so typos don't silently narrow
// the scan.
func normalizeKinds(in []string) (map[string]bool, error) {
	list := in
	if len(list) == 0 {
		list = DefaultGenerateKinds
	}
	out := make(map[string]bool, len(list))
	for _, k := range list {
		k = strings.TrimSpace(k)
		if k == "" {
			continue
		}
		switch k {
		case KindCatalog, KindSchema, KindStorageCredential,
			KindExternalLocation, KindVolume, KindConnection:
			out[k] = true
		default:
			return nil, fmt.Errorf("unknown kind %q (want one of: %s)", k, strings.Join(DefaultGenerateKinds, ", "))
		}
	}
	return out, nil
}

func skipCatalog(c catalog.CatalogInfo) bool {
	if _, skip := skipCatalogNames[c.Name]; skip {
		return true
	}
	switch c.CatalogType {
	case catalog.CatalogTypeSystemCatalog, catalog.CatalogTypeInternalCatalog:
		return true
	}
	return false
}

func skipSchema(s catalog.SchemaInfo) bool {
	_, skip := skipSchemaNames[s.Name]
	return skip
}

func scanCatalogs(catalogs []catalog.CatalogInfo, root *config.Root, state *direct.State) {
	for _, c := range catalogs {
		if skipCatalog(c) {
			continue
		}
		res := &resources.Catalog{
			Name:        c.Name,
			Comment:     c.Comment,
			StorageRoot: c.StorageRoot,
			Tags:        copyTags(c.Properties),
		}
		key := c.Name
		ensureCatalogMap(root)[key] = res
		state.Catalogs[key] = &direct.CatalogState{
			Name:        c.Name,
			Comment:     c.Comment,
			StorageRoot: c.StorageRoot,
			Tags:        copyTags(c.Properties),
		}
	}
}

func addSchemas(schemas []catalog.SchemaInfo, root *config.Root, state *direct.State) {
	for _, s := range schemas {
		if skipSchema(s) {
			continue
		}
		res := &resources.Schema{
			Name:    s.Name,
			Catalog: s.CatalogName,
			Comment: s.Comment,
			Tags:    copyTags(s.Properties),
		}
		key := s.CatalogName + "_" + s.Name
		ensureSchemaMap(root)[key] = res
		state.Schemas[key] = &direct.SchemaState{
			Name:    s.Name,
			Catalog: s.CatalogName,
			Comment: s.Comment,
			Tags:    copyTags(s.Properties),
		}
	}
}

func addVolumes(volumes []catalog.VolumeInfo, root *config.Root, state *direct.State) {
	for _, v := range volumes {
		vt := string(v.VolumeType)
		storage := v.StorageLocation
		// Managed volumes echo back a cloud path; the ucm resource model
		// (matching the SDK) treats that as derived and refuses to set
		// storage_location on MANAGED.
		if strings.EqualFold(vt, "MANAGED") {
			storage = ""
		}
		res := &resources.Volume{
			Name:            v.Name,
			CatalogName:     v.CatalogName,
			SchemaName:      v.SchemaName,
			VolumeType:      vt,
			StorageLocation: storage,
			Comment:         v.Comment,
		}
		key := v.CatalogName + "_" + v.SchemaName + "_" + v.Name
		ensureVolumeMap(root)[key] = res
		state.Volumes[key] = &direct.VolumeState{
			Name:            v.Name,
			CatalogName:     v.CatalogName,
			SchemaName:      v.SchemaName,
			VolumeType:      vt,
			StorageLocation: storage,
			Comment:         v.Comment,
		}
	}
}

// scanStorageCredentials emits a warning per credential whose identity
// carries secret material the SDK refuses to echo back. The generated YAML
// still contains a usable scaffold — the user must fill in the secret before
// the next `ucm deploy`, or the credential will be recreated with an empty
// secret.
func scanStorageCredentials(ctx context.Context, client direct.Client, root *config.Root, state *direct.State) ([]string, error) {
	creds, err := client.ListStorageCredentials(ctx)
	if err != nil {
		return nil, fmt.Errorf("list storage_credentials: %w", err)
	}
	sort.Slice(creds, func(i, j int) bool { return creds[i].Name < creds[j].Name })
	var warnings []string
	for _, c := range creds {
		res, cfgState, warn := convertStorageCredential(c)
		if res == nil {
			warnings = append(warnings, fmt.Sprintf("storage_credential %q: skipped (unsupported identity type)", c.Name))
			continue
		}
		if warn != "" {
			warnings = append(warnings, warn)
		}
		ensureStorageCredentialMap(root)[c.Name] = res
		state.StorageCredentials[c.Name] = cfgState
	}
	return warnings, nil
}

func convertStorageCredential(c catalog.StorageCredentialInfo) (*resources.StorageCredential, *direct.StorageCredentialState, string) {
	res := &resources.StorageCredential{
		Name:     c.Name,
		Comment:  c.Comment,
		ReadOnly: c.ReadOnly,
	}
	st := &direct.StorageCredentialState{
		Name:     c.Name,
		Comment:  c.Comment,
		ReadOnly: c.ReadOnly,
	}
	var warn string
	switch {
	case c.AwsIamRole != nil:
		res.AwsIamRole = &resources.AwsIamRole{RoleArn: c.AwsIamRole.RoleArn}
		st.AwsIamRole = &direct.AwsIamRoleState{RoleArn: c.AwsIamRole.RoleArn}
	case c.AzureManagedIdentity != nil:
		res.AzureManagedIdentity = &resources.AzureManagedIdentity{
			AccessConnectorId: c.AzureManagedIdentity.AccessConnectorId,
			ManagedIdentityId: c.AzureManagedIdentity.ManagedIdentityId,
		}
		st.AzureManagedIdentity = &direct.AzureManagedIdentityState{
			AccessConnectorId: c.AzureManagedIdentity.AccessConnectorId,
			ManagedIdentityId: c.AzureManagedIdentity.ManagedIdentityId,
		}
	case c.AzureServicePrincipal != nil:
		res.AzureServicePrincipal = &resources.AzureServicePrincipal{
			DirectoryId:   c.AzureServicePrincipal.DirectoryId,
			ApplicationId: c.AzureServicePrincipal.ApplicationId,
			ClientSecret:  "", // UC does not echo the secret; user must fill it in.
		}
		st.AzureServicePrincipal = &direct.AzureServicePrincipalState{
			DirectoryId:   c.AzureServicePrincipal.DirectoryId,
			ApplicationId: c.AzureServicePrincipal.ApplicationId,
		}
		warn = fmt.Sprintf("storage_credential %q: azure_service_principal.client_secret not available from UC; set it in ucm.yml before the next deploy", c.Name)
	case c.DatabricksGcpServiceAccount != nil:
		res.DatabricksGcpServiceAccount = &resources.DatabricksGcpServiceAccount{}
		st.DatabricksGcpServiceAccount = &direct.DatabricksGcpServiceAccountState{}
	default:
		return nil, nil, ""
	}
	return res, st, warn
}

func scanExternalLocations(ctx context.Context, client direct.Client, root *config.Root, state *direct.State) error {
	locs, err := client.ListExternalLocations(ctx)
	if err != nil {
		return fmt.Errorf("list external_locations: %w", err)
	}
	for _, l := range locs {
		res := &resources.ExternalLocation{
			Name:           l.Name,
			Url:            l.Url,
			CredentialName: l.CredentialName,
			Comment:        l.Comment,
			ReadOnly:       l.ReadOnly,
			Fallback:       l.Fallback,
		}
		ensureExternalLocationMap(root)[l.Name] = res
		state.ExternalLocations[l.Name] = &direct.ExternalLocationState{
			Name:           l.Name,
			Url:            l.Url,
			CredentialName: l.CredentialName,
			Comment:        l.Comment,
			ReadOnly:       l.ReadOnly,
			Fallback:       l.Fallback,
		}
	}
	return nil
}

func scanConnections(ctx context.Context, client direct.Client, root *config.Root, state *direct.State) error {
	conns, err := client.ListConnections(ctx)
	if err != nil {
		return fmt.Errorf("list connections: %w", err)
	}
	for _, c := range conns {
		res := &resources.Connection{
			Name:           c.Name,
			ConnectionType: string(c.ConnectionType),
			Options:        copyTags(c.Options),
			Comment:        c.Comment,
			Properties:     copyTags(c.Properties),
			ReadOnly:       c.ReadOnly,
		}
		ensureConnectionMap(root)[c.Name] = res
		state.Connections[c.Name] = &direct.ConnectionState{
			Name:           c.Name,
			ConnectionType: string(c.ConnectionType),
			Options:        copyTags(c.Options),
			Comment:        c.Comment,
			Properties:     copyTags(c.Properties),
			ReadOnly:       c.ReadOnly,
		}
	}
	return nil
}

// ensure* lazily initializes the map on Root.Resources so the JSON `omitempty`
// tag on Resources.* drops unused kinds from the emitted YAML.

func ensureCatalogMap(r *config.Root) map[string]*resources.Catalog {
	if r.Resources.Catalogs == nil {
		r.Resources.Catalogs = map[string]*resources.Catalog{}
	}
	return r.Resources.Catalogs
}

func ensureSchemaMap(r *config.Root) map[string]*resources.Schema {
	if r.Resources.Schemas == nil {
		r.Resources.Schemas = map[string]*resources.Schema{}
	}
	return r.Resources.Schemas
}

func ensureStorageCredentialMap(r *config.Root) map[string]*resources.StorageCredential {
	if r.Resources.StorageCredentials == nil {
		r.Resources.StorageCredentials = map[string]*resources.StorageCredential{}
	}
	return r.Resources.StorageCredentials
}

func ensureExternalLocationMap(r *config.Root) map[string]*resources.ExternalLocation {
	if r.Resources.ExternalLocations == nil {
		r.Resources.ExternalLocations = map[string]*resources.ExternalLocation{}
	}
	return r.Resources.ExternalLocations
}

func ensureVolumeMap(r *config.Root) map[string]*resources.Volume {
	if r.Resources.Volumes == nil {
		r.Resources.Volumes = map[string]*resources.Volume{}
	}
	return r.Resources.Volumes
}

func ensureConnectionMap(r *config.Root) map[string]*resources.Connection {
	if r.Resources.Connections == nil {
		r.Resources.Connections = map[string]*resources.Connection{}
	}
	return r.Resources.Connections
}

func copyTags(in map[string]string) map[string]string {
	if len(in) == 0 {
		return nil
	}
	out := make(map[string]string, len(in))
	for k, v := range in {
		out[k] = v
	}
	return out
}
