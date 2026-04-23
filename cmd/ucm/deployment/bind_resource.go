package deployment

import (
	"fmt"
	"strings"

	"github.com/databricks/cli/ucm"
	"github.com/databricks/cli/ucm/phases"
)

// resolveBindable returns the phases.ImportKind the given ucm.yml key maps to.
// Returns an error when the key matches zero or multiple kinds — bind is
// unambiguous by design.
func resolveBindable(u *ucm.Ucm, key string) (phases.ImportKind, error) {
	var matches []phases.ImportKind
	if _, ok := u.Config.Resources.Catalogs[key]; ok {
		matches = append(matches, phases.ImportCatalog)
	}
	if _, ok := u.Config.Resources.Schemas[key]; ok {
		matches = append(matches, phases.ImportSchema)
	}
	if _, ok := u.Config.Resources.StorageCredentials[key]; ok {
		matches = append(matches, phases.ImportStorageCredential)
	}
	if _, ok := u.Config.Resources.ExternalLocations[key]; ok {
		matches = append(matches, phases.ImportExternalLocation)
	}
	if _, ok := u.Config.Resources.Volumes[key]; ok {
		matches = append(matches, phases.ImportVolume)
	}
	if _, ok := u.Config.Resources.Connections[key]; ok {
		matches = append(matches, phases.ImportConnection)
	}
	if _, ok := u.Config.Resources.Grants[key]; ok {
		return "", fmt.Errorf("grants are not bindable (they reconcile per securable, not by name)")
	}
	switch len(matches) {
	case 0:
		return "", fmt.Errorf("no bindable resource with key %q in ucm.yml", key)
	case 1:
		return matches[0], nil
	default:
		return "", fmt.Errorf("ambiguous key %q: matches %v", key, matches)
	}
}

// validateBindName asserts the UC_NAME argument matches the shape the UC
// surface expects for the given kind. Catching shape mismatches here turns a
// late state-write error into a clear upfront message.
func validateBindName(kind phases.ImportKind, name string) error {
	if name == "" {
		return fmt.Errorf("empty UC_NAME")
	}
	dots := strings.Count(name, ".")
	switch kind {
	case phases.ImportCatalog,
		phases.ImportStorageCredential,
		phases.ImportExternalLocation,
		phases.ImportConnection:
		if dots != 0 {
			return fmt.Errorf("%s name %q must be a single identifier with no dots", kind, name)
		}
	case phases.ImportSchema:
		if dots != 1 {
			return fmt.Errorf("schema name %q must be `<catalog>.<schema>`", name)
		}
	case phases.ImportVolume:
		if dots != 2 {
			return fmt.Errorf("volume name %q must be `<catalog>.<schema>.<volume>`", name)
		}
	}
	return nil
}
