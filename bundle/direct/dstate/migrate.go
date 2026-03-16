package dstate

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/databricks/cli/bundle/direct/dresources"
	"github.com/databricks/databricks-sdk-go/service/iam"
)

// migrateState runs all necessary migrations on the database.
// It is called after loading state from disk.
func migrateState(db *Database) error {
	if db.StateVersion == currentStateVersion {
		return nil
	}
	if db.StateVersion > currentStateVersion {
		return fmt.Errorf("state version %d is newer than supported version %d; upgrade the CLI", db.StateVersion, currentStateVersion)
	}

	for version := db.StateVersion; version < currentStateVersion; version++ {
		fn, ok := migrations[version]
		if !ok {
			return fmt.Errorf("unsupported state version %d (current: %d)", version, currentStateVersion)
		}
		if err := fn(db); err != nil {
			return fmt.Errorf("migrating state from version %d: %w", version, err)
		}
		db.StateVersion = version + 1
	}

	return nil
}

// migrations maps source version to the function that migrates to version+1.
// Version 0 means state_version was absent in old state files; treat same as 1.
var migrations = map[int]func(*Database) error{
	0: migrateV1ToV2,
	1: migrateV1ToV2,
}

// migrateV1ToV2 migrates permissions entries from the old format
// (iam.AccessControlRequest with "permissions" key and "permission_level" field)
// to the new format (StatePermission with "_" key and "level" field).
func migrateV1ToV2(db *Database) error {
	for key, entry := range db.State {
		if !strings.HasSuffix(key, ".permissions") {
			continue
		}
		if len(entry.State) == 0 {
			continue
		}
		migrated, err := migratePermissionsEntry(entry.State)
		if err != nil {
			return fmt.Errorf("migrating %s: %w", key, err)
		}
		entry.State = migrated
		db.State[key] = entry
	}
	return nil
}

// Old types from main branch — copied here for migration purposes.
type oldPermissionsStateV1 struct {
	ObjectID    string            `json:"object_id"`
	Permissions []oldPermissionV1 `json:"permissions,omitempty"`
}

type oldPermissionV1 struct {
	PermissionLevel      string `json:"permission_level,omitempty"`
	UserName             string `json:"user_name,omitempty"`
	ServicePrincipalName string `json:"service_principal_name,omitempty"`
	GroupName            string `json:"group_name,omitempty"`
}

func migratePermissionsEntry(raw json.RawMessage) (json.RawMessage, error) {
	var old oldPermissionsStateV1
	if err := json.Unmarshal(raw, &old); err != nil {
		return nil, err
	}

	// If old format had no permissions, try parsing as new format (might already be migrated).
	if len(old.Permissions) == 0 {
		return raw, nil
	}

	newState := dresources.PermissionsState{
		ObjectID: old.ObjectID,
	}
	for _, p := range old.Permissions {
		newState.EmbeddedSlice = append(newState.EmbeddedSlice, dresources.StatePermission{
			Level:                iam.PermissionLevel(p.PermissionLevel),
			UserName:             p.UserName,
			ServicePrincipalName: p.ServicePrincipalName,
			GroupName:            p.GroupName,
		})
	}

	return json.MarshalIndent(newState, "  ", " ")
}
