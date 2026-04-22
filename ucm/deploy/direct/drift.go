package direct

import (
	"context"
	"fmt"
	"reflect"
	"sort"
	"strings"
)

// FieldDrift records a single drift finding for one field of one resource.
// State is what ucm's recorded state has; Live is what the SDK read back.
// Values are rendered via fmt.Sprintf("%v", ...) at display time so nested
// maps/slices survive the JSON/text output paths without a custom marshaller.
type FieldDrift struct {
	Field string `json:"field"`
	State any    `json:"state"`
	Live  any    `json:"live"`
}

// ResourceDrift bundles all field-level drift for a single state entry.
// Key is the ucm plan key (e.g. "resources.catalogs.sales").
type ResourceDrift struct {
	Key    string       `json:"key"`
	Fields []FieldDrift `json:"fields"`
}

// Report is the full drift result returned by ComputeDrift.
type Report struct {
	Drift []ResourceDrift `json:"drift"`
}

// HasDrift reports whether the report contains any drift findings.
func (r *Report) HasDrift() bool { return len(r.Drift) > 0 }

// ComputeDrift walks every resource recorded in state, fetches the live UC
// object via client, and returns a Report of field-by-field mismatches.
//
// Missing-live (404) is itself treated as drift: the field "_exists" flips
// from true (state) to false (live) so the user sees that the object was
// deleted out-of-band. Non-404 SDK errors propagate — drift cannot be
// meaningfully computed without a definitive live view.
//
// The grants surface is intentionally skipped: the UC Grants API returns an
// authoritative set per-securable which doesn't map cleanly onto ucm's
// per-key grant state; comparing principals across the whole securable
// belongs in a follow-up.
func ComputeDrift(ctx context.Context, client Client, state *State) (*Report, error) {
	r := &Report{}

	for _, key := range sortedKeys(state.Catalogs) {
		rec := state.Catalogs[key]
		diff, err := driftCatalog(ctx, client, rec)
		if err != nil {
			return nil, fmt.Errorf("read catalog %s: %w", rec.Name, err)
		}
		if len(diff) > 0 {
			r.Drift = append(r.Drift, ResourceDrift{Key: "resources.catalogs." + key, Fields: diff})
		}
	}

	for _, key := range sortedKeys(state.Schemas) {
		rec := state.Schemas[key]
		diff, err := driftSchema(ctx, client, rec)
		if err != nil {
			return nil, fmt.Errorf("read schema %s.%s: %w", rec.Catalog, rec.Name, err)
		}
		if len(diff) > 0 {
			r.Drift = append(r.Drift, ResourceDrift{Key: "resources.schemas." + key, Fields: diff})
		}
	}

	for _, key := range sortedKeys(state.StorageCredentials) {
		rec := state.StorageCredentials[key]
		diff, err := driftStorageCredential(ctx, client, rec)
		if err != nil {
			return nil, fmt.Errorf("read storage_credential %s: %w", rec.Name, err)
		}
		if len(diff) > 0 {
			r.Drift = append(r.Drift, ResourceDrift{Key: "resources.storage_credentials." + key, Fields: diff})
		}
	}

	for _, key := range sortedKeys(state.ExternalLocations) {
		rec := state.ExternalLocations[key]
		diff, err := driftExternalLocation(ctx, client, rec)
		if err != nil {
			return nil, fmt.Errorf("read external_location %s: %w", rec.Name, err)
		}
		if len(diff) > 0 {
			r.Drift = append(r.Drift, ResourceDrift{Key: "resources.external_locations." + key, Fields: diff})
		}
	}

	for _, key := range sortedKeys(state.Volumes) {
		rec := state.Volumes[key]
		diff, err := driftVolume(ctx, client, rec)
		if err != nil {
			return nil, fmt.Errorf("read volume %s.%s.%s: %w", rec.CatalogName, rec.SchemaName, rec.Name, err)
		}
		if len(diff) > 0 {
			r.Drift = append(r.Drift, ResourceDrift{Key: "resources.volumes." + key, Fields: diff})
		}
	}

	for _, key := range sortedKeys(state.Connections) {
		rec := state.Connections[key]
		diff, err := driftConnection(ctx, client, rec)
		if err != nil {
			return nil, fmt.Errorf("read connection %s: %w", rec.Name, err)
		}
		if len(diff) > 0 {
			r.Drift = append(r.Drift, ResourceDrift{Key: "resources.connections." + key, Fields: diff})
		}
	}

	return r, nil
}

// driftCatalog compares every state-recorded field of a catalog against the
// live CatalogInfo. The tag set comes back as Properties on CatalogInfo —
// an SDK naming quirk mirrored in the tfdyn converter.
func driftCatalog(ctx context.Context, client Client, rec *CatalogState) ([]FieldDrift, error) {
	live, err := client.GetCatalog(ctx, rec.Name)
	if err != nil {
		if isNotFound(err) {
			return []FieldDrift{{Field: "_exists", State: true, Live: false}}, nil
		}
		return nil, err
	}
	var diffs []FieldDrift
	diffs = appendStringDrift(diffs, "comment", rec.Comment, live.Comment)
	diffs = appendStringDrift(diffs, "storage_root", rec.StorageRoot, live.StorageRoot)
	diffs = appendMapDrift(diffs, "tags", rec.Tags, live.Properties)
	return diffs, nil
}

// driftSchema mirrors driftCatalog for schemas. The live-read key is the
// fully-qualified `catalog.schema` name, matching the update/delete paths.
func driftSchema(ctx context.Context, client Client, rec *SchemaState) ([]FieldDrift, error) {
	live, err := client.GetSchema(ctx, rec.Catalog+"."+rec.Name)
	if err != nil {
		if isNotFound(err) {
			return []FieldDrift{{Field: "_exists", State: true, Live: false}}, nil
		}
		return nil, err
	}
	var diffs []FieldDrift
	diffs = appendStringDrift(diffs, "comment", rec.Comment, live.Comment)
	diffs = appendMapDrift(diffs, "tags", rec.Tags, live.Properties)
	return diffs, nil
}

// driftStorageCredential compares against the live StorageCredentialInfo.
// Skipped fields (documented):
//   - azure_service_principal.client_secret — UC never echoes the secret
//     back (matches direct apply's reason for persisting it in state).
//   - skip_validation — write-only flag, not surfaced on the read side.
func driftStorageCredential(ctx context.Context, client Client, rec *StorageCredentialState) ([]FieldDrift, error) {
	live, err := client.GetStorageCredential(ctx, rec.Name)
	if err != nil {
		if isNotFound(err) {
			return []FieldDrift{{Field: "_exists", State: true, Live: false}}, nil
		}
		return nil, err
	}
	var diffs []FieldDrift
	diffs = appendStringDrift(diffs, "comment", rec.Comment, live.Comment)
	diffs = appendBoolDrift(diffs, "read_only", rec.ReadOnly, live.ReadOnly)

	// Identity fields: compare only the one the user configured. Cross-identity
	// drift (e.g. user swapped AWS for Azure) would have surfaced on plan-side
	// long before a drift check; the live read would show a different identity
	// shape and we'd still report it via the sub-field comparisons below.
	if rec.AwsIamRole != nil {
		liveArn := ""
		if live.AwsIamRole != nil {
			liveArn = live.AwsIamRole.RoleArn
		}
		diffs = appendStringDrift(diffs, "aws_iam_role.role_arn", rec.AwsIamRole.RoleArn, liveArn)
	}
	if rec.AzureManagedIdentity != nil {
		liveAcc, liveMi := "", ""
		if live.AzureManagedIdentity != nil {
			liveAcc = live.AzureManagedIdentity.AccessConnectorId
			liveMi = live.AzureManagedIdentity.ManagedIdentityId
		}
		diffs = appendStringDrift(diffs, "azure_managed_identity.access_connector_id", rec.AzureManagedIdentity.AccessConnectorId, liveAcc)
		diffs = appendStringDrift(diffs, "azure_managed_identity.managed_identity_id", rec.AzureManagedIdentity.ManagedIdentityId, liveMi)
	}
	if rec.AzureServicePrincipal != nil {
		liveDir, liveApp := "", ""
		if live.AzureServicePrincipal != nil {
			liveDir = live.AzureServicePrincipal.DirectoryId
			liveApp = live.AzureServicePrincipal.ApplicationId
		}
		diffs = appendStringDrift(diffs, "azure_service_principal.directory_id", rec.AzureServicePrincipal.DirectoryId, liveDir)
		diffs = appendStringDrift(diffs, "azure_service_principal.application_id", rec.AzureServicePrincipal.ApplicationId, liveApp)
		// client_secret intentionally skipped — UC does not echo secrets back.
	}
	return diffs, nil
}

// driftExternalLocation compares against the live ExternalLocationInfo.
// skip_validation is write-only and therefore skipped.
func driftExternalLocation(ctx context.Context, client Client, rec *ExternalLocationState) ([]FieldDrift, error) {
	live, err := client.GetExternalLocation(ctx, rec.Name)
	if err != nil {
		if isNotFound(err) {
			return []FieldDrift{{Field: "_exists", State: true, Live: false}}, nil
		}
		return nil, err
	}
	var diffs []FieldDrift
	diffs = appendStringDrift(diffs, "url", rec.Url, live.Url)
	diffs = appendStringDrift(diffs, "credential_name", rec.CredentialName, live.CredentialName)
	diffs = appendStringDrift(diffs, "comment", rec.Comment, live.Comment)
	diffs = appendBoolDrift(diffs, "read_only", rec.ReadOnly, live.ReadOnly)
	diffs = appendBoolDrift(diffs, "fallback", rec.Fallback, live.Fallback)
	return diffs, nil
}

// driftVolume compares against the live VolumeInfo using the three-level
// fully-qualified name.
func driftVolume(ctx context.Context, client Client, rec *VolumeState) ([]FieldDrift, error) {
	fullName := rec.CatalogName + "." + rec.SchemaName + "." + rec.Name
	live, err := client.GetVolume(ctx, fullName)
	if err != nil {
		if isNotFound(err) {
			return []FieldDrift{{Field: "_exists", State: true, Live: false}}, nil
		}
		return nil, err
	}
	var diffs []FieldDrift
	diffs = appendStringDrift(diffs, "volume_type", rec.VolumeType, string(live.VolumeType))
	diffs = appendStringDrift(diffs, "storage_location", rec.StorageLocation, live.StorageLocation)
	diffs = appendStringDrift(diffs, "comment", rec.Comment, live.Comment)
	return diffs, nil
}

// driftConnection compares against the live ConnectionInfo. read_only and
// connection_type are immutable server-side but still reported so a manual
// drop-and-recreate with a different shape is visible.
func driftConnection(ctx context.Context, client Client, rec *ConnectionState) ([]FieldDrift, error) {
	live, err := client.GetConnection(ctx, rec.Name)
	if err != nil {
		if isNotFound(err) {
			return []FieldDrift{{Field: "_exists", State: true, Live: false}}, nil
		}
		return nil, err
	}
	var diffs []FieldDrift
	diffs = appendStringDrift(diffs, "connection_type", rec.ConnectionType, string(live.ConnectionType))
	diffs = appendStringDrift(diffs, "comment", rec.Comment, live.Comment)
	diffs = appendBoolDrift(diffs, "read_only", rec.ReadOnly, live.ReadOnly)
	diffs = appendMapDrift(diffs, "options", rec.Options, live.Options)
	diffs = appendMapDrift(diffs, "properties", rec.Properties, live.Properties)
	return diffs, nil
}

// appendStringDrift appends a FieldDrift entry when state and live disagree.
func appendStringDrift(diffs []FieldDrift, field, state, live string) []FieldDrift {
	if state == live {
		return diffs
	}
	return append(diffs, FieldDrift{Field: field, State: state, Live: live})
}

// appendBoolDrift mirrors appendStringDrift for booleans.
func appendBoolDrift(diffs []FieldDrift, field string, state, live bool) []FieldDrift {
	if state == live {
		return diffs
	}
	return append(diffs, FieldDrift{Field: field, State: state, Live: live})
}

// appendMapDrift compares two string maps. Nil and empty are treated as
// equal so "no tags set" never reads as drift against a nil-returning SDK.
func appendMapDrift(diffs []FieldDrift, field string, state, live map[string]string) []FieldDrift {
	if len(state) == 0 && len(live) == 0 {
		return diffs
	}
	if reflect.DeepEqual(state, live) {
		return diffs
	}
	return append(diffs, FieldDrift{Field: field, State: state, Live: live})
}

// isNotFound returns true for the SDK's not-found signalling. Comparing
// error text is the most portable check because apierr is not re-exported
// on the catalog service surface we already depend on and the strings are
// stable across recent SDK minor versions.
func isNotFound(err error) bool {
	if err == nil {
		return false
	}
	msg := strings.ToLower(err.Error())
	for _, needle := range [...]string{"does not exist", "not found", "resource_does_not_exist"} {
		if strings.Contains(msg, needle) {
			return true
		}
	}
	return false
}

// sortedKeys returns the keys of a map in lexical order. Keeps drift output
// stable across runs regardless of Go map iteration order.
func sortedKeys[V any](m map[string]V) []string {
	out := make([]string, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	sort.Strings(out)
	return out
}
