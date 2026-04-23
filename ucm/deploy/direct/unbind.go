package direct

import (
	"context"
	"fmt"

	"github.com/databricks/cli/libs/log"
)

// UnbindResource drops the recorded state entry for the given kind+key.
// Returns a descriptive error when the key is not currently bound so the
// caller can surface the intent mismatch to the operator. kind follows the
// same singular-name vocabulary as ImportResource: "catalog", "schema",
// "storage_credential", "external_location", "volume", "connection".
func UnbindResource(ctx context.Context, state *State, kind, key string) error {
	present := false
	switch kind {
	case "catalog":
		_, present = state.Catalogs[key]
		delete(state.Catalogs, key)
	case "schema":
		_, present = state.Schemas[key]
		delete(state.Schemas, key)
	case "storage_credential":
		_, present = state.StorageCredentials[key]
		delete(state.StorageCredentials, key)
	case "external_location":
		_, present = state.ExternalLocations[key]
		delete(state.ExternalLocations, key)
	case "volume":
		_, present = state.Volumes[key]
		delete(state.Volumes, key)
	case "connection":
		_, present = state.Connections[key]
		delete(state.Connections, key)
	default:
		return fmt.Errorf("unsupported unbind kind %q", kind)
	}
	if !present {
		return fmt.Errorf("no bound %s with key %q in direct state", kind, key)
	}
	log.Infof(ctx, "direct: unbound resources.%s.%s", pluralDirectKind(kind), key)
	return nil
}

// pluralDirectKind maps the singular kind string to the plural form used in
// resources.<group>.<key> log messages. Kept local so the CLI layer never
// has to spell the mapping out.
func pluralDirectKind(kind string) string {
	switch kind {
	case "catalog":
		return "catalogs"
	case "schema":
		return "schemas"
	case "storage_credential":
		return "storage_credentials"
	case "external_location":
		return "external_locations"
	case "volume":
		return "volumes"
	case "connection":
		return "connections"
	}
	return kind + "s"
}
