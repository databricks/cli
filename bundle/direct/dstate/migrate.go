package dstate

import (
	"encoding/json"
	"fmt"

	"github.com/databricks/cli/bundle/direct/dresources"
	"github.com/databricks/cli/libs/structs/structpath"
	"github.com/databricks/databricks-sdk-go/service/catalog"
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

// migrateV1ToV2 migrates permissions and grants entries from the old format
// to the new format (__embed__ keys, permission_level -> level).
func migrateV1ToV2(db *Database) error {
	for key, entry := range db.State {
		if len(entry.State) == 0 {
			continue
		}

		path, pathErr := structpath.ParsePath(key)
		if pathErr != nil || path == nil {
			continue
		}
		// path points to the last node; read its key directly.
		lastKey, ok := path.StringKey()
		if !ok {
			continue
		}

		var (
			migrated json.RawMessage
			err      error
		)

		switch lastKey {
		case "permissions":
			migrated, err = migratePermissionsEntry(entry.State)
		case "grants":
			migrated, err = migrateGrantsEntry(entry.State)
		default:
			continue
		}

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

// oldGrantsStateV1 is the grants state format before v2.
type oldGrantsStateV1 struct {
	SecurableType string                        `json:"securable_type"`
	FullName      string                        `json:"full_name"`
	Grants        []catalog.PrivilegeAssignment `json:"grants,omitempty"`
}

func migrateGrantsEntry(raw json.RawMessage) (json.RawMessage, error) {
	var old oldGrantsStateV1
	if err := json.Unmarshal(raw, &old); err != nil {
		return nil, err
	}

	newState := dresources.GrantsState{
		SecurableType: old.SecurableType,
		FullName:      old.FullName,
		EmbeddedSlice: make([]catalog.PrivilegeAssignment, len(old.Grants)),
	}
	copy(newState.EmbeddedSlice, old.Grants)

	return json.MarshalIndent(newState, "  ", " ")
}
