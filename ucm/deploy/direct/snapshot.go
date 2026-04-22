package direct

import (
	"github.com/databricks/cli/ucm/config/resources"
)

// Snapshot-helpers live in their own file so plan.go (the diff logic) stays
// focused on the Create/Update/Skip/Delete decision tree. Each helper takes
// a pointer into the loaded ucm config and returns the comparable state
// representation.

func catalogStateFromConfig(c *resources.Catalog) CatalogState {
	if c == nil {
		return CatalogState{}
	}
	return CatalogState{
		Name:        c.Name,
		Comment:     c.Comment,
		StorageRoot: c.StorageRoot,
		Tags:        copyTags(c.Tags),
	}
}

func schemaStateFromConfig(s *resources.Schema) SchemaState {
	if s == nil {
		return SchemaState{}
	}
	return SchemaState{
		Name:    s.Name,
		Catalog: s.Catalog,
		Comment: s.Comment,
		Tags:    copyTags(s.Tags),
	}
}

func grantStateFromConfig(g *resources.Grant) GrantState {
	if g == nil {
		return GrantState{}
	}
	privs := make([]string, len(g.Privileges))
	copy(privs, g.Privileges)
	return GrantState{
		SecurableType: g.Securable.Type,
		SecurableName: g.Securable.Name,
		Principal:     g.Principal,
		Privileges:    privs,
	}
}

func storageCredentialStateFromConfig(c *resources.StorageCredential) StorageCredentialState {
	if c == nil {
		return StorageCredentialState{}
	}
	s := StorageCredentialState{
		Name:           c.Name,
		Comment:        c.Comment,
		ReadOnly:       c.ReadOnly,
		SkipValidation: c.SkipValidation,
	}
	if c.AwsIamRole != nil {
		s.AwsIamRole = &AwsIamRoleState{RoleArn: c.AwsIamRole.RoleArn}
	}
	if c.AzureManagedIdentity != nil {
		s.AzureManagedIdentity = &AzureManagedIdentityState{
			AccessConnectorId: c.AzureManagedIdentity.AccessConnectorId,
			ManagedIdentityId: c.AzureManagedIdentity.ManagedIdentityId,
		}
	}
	if c.AzureServicePrincipal != nil {
		s.AzureServicePrincipal = &AzureServicePrincipalState{
			DirectoryId:   c.AzureServicePrincipal.DirectoryId,
			ApplicationId: c.AzureServicePrincipal.ApplicationId,
			ClientSecret:  c.AzureServicePrincipal.ClientSecret,
		}
	}
	if c.DatabricksGcpServiceAccount != nil {
		s.DatabricksGcpServiceAccount = &DatabricksGcpServiceAccountState{}
	}
	return s
}

func externalLocationStateFromConfig(e *resources.ExternalLocation) ExternalLocationState {
	if e == nil {
		return ExternalLocationState{}
	}
	return ExternalLocationState{
		Name:           e.Name,
		Url:            e.Url,
		CredentialName: e.CredentialName,
		Comment:        e.Comment,
		ReadOnly:       e.ReadOnly,
		SkipValidation: e.SkipValidation,
		Fallback:       e.Fallback,
	}
}

func volumeStateFromConfig(v *resources.Volume) VolumeState {
	if v == nil {
		return VolumeState{}
	}
	return VolumeState{
		Name:            v.Name,
		CatalogName:     v.CatalogName,
		SchemaName:      v.SchemaName,
		VolumeType:      v.VolumeType,
		StorageLocation: v.StorageLocation,
		Comment:         v.Comment,
	}
}

func copyTags(tags map[string]string) map[string]string {
	if len(tags) == 0 {
		return nil
	}
	out := make(map[string]string, len(tags))
	for k, v := range tags {
		out[k] = v
	}
	return out
}
