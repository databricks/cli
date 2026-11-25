package dresources

import (
	"github.com/databricks/cli/bundle/config/resources"
	"github.com/databricks/cli/libs/snapshot"
	"github.com/databricks/cli/libs/testserver"
)

func init() {
	registerTestConfig("internal_immutable_snapshots", &resources.Snapshot{
		RemoteRoot: "/Workspace/Users/" + testserver.SnapshotRootUUID + "/.snapshots",
		BundleID:   "test-bundle-id",
		ACL:        []snapshot.ACLEntry{{UserName: "user@example.com", PermissionLevel: "CAN_READ"}},
	})

	registerKnownMissingInRemoteType("internal_immutable_snapshots",
		"bundle_id",
		"acl",
		"can_manage",
		"zip_path",
	)

	registerKnownMissingInStateType("internal_immutable_snapshots",
		// RemoteRoot is only the input used to compose FullPath; the state persists
		// the composed FullPath (and RelativePath), so it isn't kept in state.
		"remote_root",
	)

	registerCreateRecoveryOptOut("internal_immutable_snapshots", "the snapshot is an immutable upload, and it is not known whether a second upload to the same path is accepted")
}
