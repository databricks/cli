package resources

import (
	"context"
	"fmt"
	"sort"
	"testing"

	"github.com/databricks/databricks-sdk-go/apierr"
	"github.com/databricks/databricks-sdk-go/experimental/mocks"
	"github.com/databricks/databricks-sdk-go/service/catalog"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func TestSchemaNotFound(t *testing.T) {
	ctx := context.Background()

	m := mocks.NewMockWorkspaceClient(t)
	m.GetMockSchemasAPI().On("GetByFullName", mock.Anything, "non-existent-schema").Return(nil, &apierr.APIError{
		StatusCode: 404,
	})

	s := &Schema{}
	exists, err := s.Exists(ctx, m.WorkspaceClient, "non-existent-schema")

	require.Falsef(t, exists, "Exists should return false when getting a 404 response from Workspace")
	require.NoErrorf(t, err, "Exists should not return an error when getting a 404 response from Workspace")
}

func TestSchemaGrantPrivilegesExhaustive(t *testing.T) {
	// Privileges that are NOT valid for schemas and should be skipped.
	// These are valid for other securable types (catalogs, connections, etc.)
	// Source: https://docs.databricks.com/en/data-governance/unity-catalog/manage-privileges/privileges.html
	skippedPrivileges := map[catalog.Privilege]string{
		// Catalog-level privileges
		catalog.PrivilegeCreateCatalog: "catalog-level",
		catalog.PrivilegeCreateSchema:  "catalog-level",
		catalog.PrivilegeUseCatalog:    "catalog-level",

		// Connection-level privileges
		catalog.PrivilegeCreateConnection: "connection-level",
		catalog.PrivilegeUseConnection:    "connection-level",

		// Storage-level privileges
		catalog.PrivilegeCreateExternalLocation:  "storage-level",
		catalog.PrivilegeCreateExternalTable:     "storage-level",
		catalog.PrivilegeCreateExternalVolume:    "storage-level",
		catalog.PrivilegeCreateManagedStorage:    "storage-level",
		catalog.PrivilegeCreateStorageCredential: "storage-level",
		catalog.PrivilegeReadFiles:               "storage-level",
		catalog.PrivilegeReadPrivateFiles:        "storage-level",
		catalog.PrivilegeWriteFiles:              "storage-level",
		catalog.PrivilegeWritePrivateFiles:       "storage-level",

		// Metastore-level privileges
		catalog.PrivilegeCreateProvider:          "metastore-level",
		catalog.PrivilegeCreateRecipient:         "metastore-level",
		catalog.PrivilegeCreateShare:             "metastore-level",
		catalog.PrivilegeManageAllowlist:         "metastore-level",
		catalog.PrivilegeUseProvider:             "metastore-level",
		catalog.PrivilegeUseRecipient:            "metastore-level",
		catalog.PrivilegeUseMarketplaceAssets:    "metastore-level",
		catalog.PrivilegeCreateServiceCredential: "metastore-level",

		// Share-level privileges
		catalog.PrivilegeSetSharePermission: "share-level",
		catalog.PrivilegeUseShare:           "share-level",

		// Clean room-level privileges
		catalog.PrivilegeCreateCleanRoom:      "clean-room-level",
		catalog.PrivilegeExecuteCleanRoomTask: "clean-room-level",
		catalog.PrivilegeModifyCleanRoom:      "clean-room-level",

		// Foreign securable privileges
		catalog.PrivilegeCreateForeignCatalog:   "foreign-securable",
		catalog.PrivilegeCreateForeignSecurable: "foreign-securable",

		// Table/view-level privileges (not directly grantable on schema)
		catalog.PrivilegeCreateView: "table-level",

		// Generic privileges that don't apply to schemas
		catalog.PrivilegeAccess: "generic",
		catalog.PrivilegeBrowse: "generic",
		catalog.PrivilegeCreate: "generic",
		catalog.PrivilegeUsage:  "generic",
	}

	// Get all SDK privileges dynamically
	var p catalog.Privilege
	sdkPrivileges := p.Values()

	// Get all privileges defined in our SchemaGrantPrivilege enum
	definedPrivileges := SchemaGrantPrivilege("").Values()
	definedPrivilegeMap := make(map[catalog.Privilege]bool)
	for _, priv := range definedPrivileges {
		definedPrivilegeMap[catalog.Privilege(priv)] = true
	}

	// Build list of missing and unexpected privileges
	var missingPrivileges []catalog.Privilege
	var unexpectedPrivileges []catalog.Privilege

	// Check each SDK privilege
	for _, sdkPriv := range sdkPrivileges {
		isInDefined := definedPrivilegeMap[sdkPriv]
		isInSkipList := skippedPrivileges[sdkPriv] != ""

		if !isInDefined && !isInSkipList {
			// SDK has a privilege that we neither defined nor explicitly skipped
			missingPrivileges = append(missingPrivileges, sdkPriv)
		}

		if isInDefined && isInSkipList {
			// We defined a privilege that's in the skip list (contradiction)
			unexpectedPrivileges = append(unexpectedPrivileges, sdkPriv)
		}
	}

	// Check for privileges we defined that don't exist in SDK
	sdkPrivilegeMap := make(map[catalog.Privilege]bool)
	for _, priv := range sdkPrivileges {
		sdkPrivilegeMap[priv] = true
	}

	var invalidPrivileges []SchemaGrantPrivilege
	for _, definedPriv := range definedPrivileges {
		if !sdkPrivilegeMap[catalog.Privilege(definedPriv)] {
			invalidPrivileges = append(invalidPrivileges, definedPriv)
		}
	}

	// Report errors
	if len(missingPrivileges) > 0 {
		sort.Slice(missingPrivileges, func(i, j int) bool {
			return missingPrivileges[i] < missingPrivileges[j]
		})
		assert.Fail(t, fmt.Sprintf(
			"Found %d SDK privilege(s) that are not in SchemaGrantPrivilege and not in skip list.\n"+
				"If these privileges are valid for schemas, add them to SchemaGrantPrivilege in schema.go.\n"+
				"If they are NOT valid for schemas, add them to skippedPrivileges in schema_test.go.\n"+
				"Missing privileges: %v",
			len(missingPrivileges), missingPrivileges))
	}

	if len(unexpectedPrivileges) > 0 {
		sort.Slice(unexpectedPrivileges, func(i, j int) bool {
			return unexpectedPrivileges[i] < unexpectedPrivileges[j]
		})
		assert.Fail(t, fmt.Sprintf(
			"Found %d privilege(s) that are both defined in SchemaGrantPrivilege AND in skip list.\n"+
				"This is a contradiction - remove them from either SchemaGrantPrivilege or the skip list.\n"+
				"Conflicting privileges: %v",
			len(unexpectedPrivileges), unexpectedPrivileges))
	}

	if len(invalidPrivileges) > 0 {
		sort.Slice(invalidPrivileges, func(i, j int) bool {
			return invalidPrivileges[i] < invalidPrivileges[j]
		})
		assert.Fail(t, fmt.Sprintf(
			"Found %d privilege(s) in SchemaGrantPrivilege that don't exist in the SDK.\n"+
				"Remove these from SchemaGrantPrivilege in schema.go.\n"+
				"Invalid privileges: %v",
			len(invalidPrivileges), invalidPrivileges))
	}
}
