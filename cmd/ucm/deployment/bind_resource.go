package deployment

import (
	"fmt"

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
