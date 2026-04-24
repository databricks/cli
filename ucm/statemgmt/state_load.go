// Package statemgmt wires the local terraform.tfstate into the ucm config tree
// for summary-time rendering. Parallels bundle/statemgmt without importing
// from it so the fork boundary stays clean.
package statemgmt

import (
	"context"
	"encoding/json"
	"errors"
	"os"

	"github.com/databricks/cli/libs/diag"
	"github.com/databricks/cli/libs/log"
	"github.com/databricks/cli/ucm"
	"github.com/databricks/cli/ucm/deploy"
	tfjson "github.com/hashicorp/terraform-json"
)

// SupportedStateVersion pins the tfstate schema we understand. Matches the
// version terraform itself emits today (also matches bundle/deploy/terraform's
// SupportedStateVersion for ease of cross-check).
const SupportedStateVersion = 4

// resourcesState is the partial shape of terraform.tfstate we care about.
// Matches bundle/deploy/terraform/util.go's decoder so any divergence in the
// future is immediately visible.
type resourcesState struct {
	Version   int             `json:"version"`
	Resources []stateResource `json:"resources"`
}

type stateResource struct {
	Type      string                  `json:"type"`
	Name      string                  `json:"name"`
	Mode      tfjson.ResourceMode     `json:"mode"`
	Instances []stateResourceInstance `json:"instances"`
}

type stateResourceInstance struct {
	Attributes stateInstanceAttributes `json:"attributes"`
}

type stateInstanceAttributes struct {
	ID string `json:"id"`
}

// terraformToGroup maps databricks_* terraform resource types onto the
// corresponding key in Config.Resources. Grants and anything not URL-bearing
// are deliberately absent — those groups don't carry an ID on their struct.
var terraformToGroup = map[string]string{
	"databricks_catalog":            "catalogs",
	"databricks_schema":             "schemas",
	"databricks_volume":             "volumes",
	"databricks_storage_credential": "storage_credentials",
	"databricks_external_location":  "external_locations",
	"databricks_connection":         "connections",
}

// Load reads the local terraform.tfstate at deploy.LocalTfStatePath(u) and
// copies attributes.id onto u.Config.Resources.<kind>[key].ID for each
// managed instance whose type is mapped in terraformToGroup.
//
// Behaviours:
//   - Missing tfstate file → no-op (first-run case before any deploy).
//   - Unreadable / malformed file → diag warning; config left untouched.
//   - Unknown terraform resource type → logged at debug, skipped.
//   - Config key absent (resource was removed from ucm.yml after deploy) →
//     logged at debug, skipped. Stale state is not a summary-time error.
func Load(ctx context.Context, u *ucm.Ucm) diag.Diagnostics {
	if u == nil {
		return nil
	}

	path := deploy.LocalTfStatePath(u)
	raw, err := os.ReadFile(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			log.Debugf(ctx, "statemgmt: no local tfstate at %s; skipping ID hydration", path)
			return nil
		}
		return diag.Warningf("statemgmt: read %s: %v", path, err)
	}

	var state resourcesState
	if err := json.Unmarshal(raw, &state); err != nil {
		return diag.Warningf("statemgmt: parse %s: %v", path, err)
	}

	if state.Version != SupportedStateVersion {
		return diag.Warningf("statemgmt: unsupported tfstate version %d in %s (want %d)", state.Version, path, SupportedStateVersion)
	}

	applyState(ctx, u, &state)
	return nil
}

// applyState walks the parsed tfstate and hydrates the .ID field on every
// managed instance whose (type, name) address matches a still-declared
// resource in u.Config.
func applyState(ctx context.Context, u *ucm.Ucm, state *resourcesState) {
	for _, r := range state.Resources {
		if r.Mode != tfjson.ManagedResourceMode {
			continue
		}
		group, ok := terraformToGroup[r.Type]
		if !ok {
			log.Debugf(ctx, "statemgmt: skip state address %s.%s (type not URL-bearing)", r.Type, r.Name)
			continue
		}
		for _, inst := range r.Instances {
			setResourceID(ctx, u, group, r.Name, inst.Attributes.ID)
		}
	}
}

// setResourceID writes id onto u.Config.Resources.<group>[key] when the key
// still exists; logs and skips otherwise so stale state doesn't fail summary.
func setResourceID(ctx context.Context, u *ucm.Ucm, group, key, id string) {
	switch group {
	case "catalogs":
		if r, ok := u.Config.Resources.Catalogs[key]; ok {
			r.ID = id
			return
		}
	case "schemas":
		if r, ok := u.Config.Resources.Schemas[key]; ok {
			r.ID = id
			return
		}
	case "volumes":
		if r, ok := u.Config.Resources.Volumes[key]; ok {
			r.ID = id
			return
		}
	case "storage_credentials":
		if r, ok := u.Config.Resources.StorageCredentials[key]; ok {
			r.ID = id
			return
		}
	case "external_locations":
		if r, ok := u.Config.Resources.ExternalLocations[key]; ok {
			r.ID = id
			return
		}
	case "connections":
		if r, ok := u.Config.Resources.Connections[key]; ok {
			r.ID = id
			return
		}
	}
	log.Debugf(ctx, "statemgmt: state address %s.%s has no matching declared resource; skipping", group, key)
}
