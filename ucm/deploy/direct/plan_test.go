package direct_test

import (
	"testing"

	"github.com/databricks/cli/ucm"
	"github.com/databricks/cli/ucm/config"
	"github.com/databricks/cli/ucm/config/resources"
	"github.com/databricks/cli/ucm/deploy/direct"
	"github.com/databricks/cli/ucm/deployplan"
	"github.com/stretchr/testify/assert"
)

func TestCalculatePlan_EmptyConfigAndState(t *testing.T) {
	u := &ucm.Ucm{Config: config.Root{}}
	plan := direct.CalculatePlan(u, direct.NewState())
	assert.Empty(t, plan.Plan)
}

func TestCalculatePlan_CreatesWhenStateEmpty(t *testing.T) {
	u := ucmWith(
		map[string]*resources.Catalog{"main": {Name: "main"}},
		map[string]*resources.Schema{"raw": {Name: "raw", Catalog: "main"}},
		map[string]*resources.Grant{
			"analysts": {
				Securable:  resources.Securable{Type: "schema", Name: "main.raw"},
				Principal:  "analysts",
				Privileges: []string{"USE_SCHEMA", "SELECT"},
			},
		},
	)

	plan := direct.CalculatePlan(u, direct.NewState())

	assert.Equal(t, deployplan.Create, plan.Plan["resources.catalogs.main"].Action)
	assert.Equal(t, deployplan.Create, plan.Plan["resources.schemas.raw"].Action)
	assert.Equal(t, deployplan.Create, plan.Plan["resources.grants.analysts"].Action)
}

func TestCalculatePlan_DeletesWhenConfigEmpty(t *testing.T) {
	u := &ucm.Ucm{Config: config.Root{}}
	state := direct.NewState()
	state.Catalogs["main"] = &direct.CatalogState{Name: "main"}
	state.Schemas["raw"] = &direct.SchemaState{Name: "raw", Catalog: "main"}
	state.Grants["analysts"] = &direct.GrantState{
		SecurableType: "schema",
		SecurableName: "main.raw",
		Principal:     "analysts",
		Privileges:    []string{"SELECT"},
	}

	plan := direct.CalculatePlan(u, state)

	assert.Equal(t, deployplan.Delete, plan.Plan["resources.catalogs.main"].Action)
	assert.Equal(t, deployplan.Delete, plan.Plan["resources.schemas.raw"].Action)
	assert.Equal(t, deployplan.Delete, plan.Plan["resources.grants.analysts"].Action)
}

func TestCalculatePlan_SkipsUnchangedEntries(t *testing.T) {
	u := ucmWith(
		map[string]*resources.Catalog{"main": {Name: "main", Comment: "prod"}},
		map[string]*resources.Schema{"raw": {Name: "raw", Catalog: "main"}},
		map[string]*resources.Grant{
			"analysts": {
				Securable:  resources.Securable{Type: "schema", Name: "main.raw"},
				Principal:  "analysts",
				Privileges: []string{"SELECT"},
			},
		},
	)

	state := direct.NewState()
	state.Catalogs["main"] = &direct.CatalogState{Name: "main", Comment: "prod"}
	state.Schemas["raw"] = &direct.SchemaState{Name: "raw", Catalog: "main"}
	state.Grants["analysts"] = &direct.GrantState{
		SecurableType: "schema",
		SecurableName: "main.raw",
		Principal:     "analysts",
		Privileges:    []string{"SELECT"},
	}

	plan := direct.CalculatePlan(u, state)

	assert.Equal(t, deployplan.Skip, plan.Plan["resources.catalogs.main"].Action)
	assert.Equal(t, deployplan.Skip, plan.Plan["resources.schemas.raw"].Action)
	assert.Equal(t, deployplan.Skip, plan.Plan["resources.grants.analysts"].Action)
}

func TestCalculatePlan_UpdatesOnFieldDrift(t *testing.T) {
	u := ucmWith(
		map[string]*resources.Catalog{"main": {Name: "main", Comment: "new"}},
		map[string]*resources.Schema{"raw": {Name: "raw", Catalog: "main", Comment: "new"}},
		map[string]*resources.Grant{
			"analysts": {
				Securable:  resources.Securable{Type: "schema", Name: "main.raw"},
				Principal:  "analysts",
				Privileges: []string{"SELECT", "MODIFY"},
			},
		},
	)

	state := direct.NewState()
	state.Catalogs["main"] = &direct.CatalogState{Name: "main", Comment: "old"}
	state.Schemas["raw"] = &direct.SchemaState{Name: "raw", Catalog: "main", Comment: "old"}
	state.Grants["analysts"] = &direct.GrantState{
		SecurableType: "schema",
		SecurableName: "main.raw",
		Principal:     "analysts",
		Privileges:    []string{"SELECT"},
	}

	plan := direct.CalculatePlan(u, state)

	assert.Equal(t, deployplan.Update, plan.Plan["resources.catalogs.main"].Action)
	assert.Equal(t, deployplan.Update, plan.Plan["resources.schemas.raw"].Action)
	assert.Equal(t, deployplan.Update, plan.Plan["resources.grants.analysts"].Action)
}

func TestCalculatePlan_PrivilegeReorderIsSkip(t *testing.T) {
	u := ucmWith(nil, nil, map[string]*resources.Grant{
		"analysts": {
			Securable:  resources.Securable{Type: "schema", Name: "main.raw"},
			Principal:  "analysts",
			Privileges: []string{"MODIFY", "SELECT"},
		},
	})

	state := direct.NewState()
	state.Grants["analysts"] = &direct.GrantState{
		SecurableType: "schema",
		SecurableName: "main.raw",
		Principal:     "analysts",
		Privileges:    []string{"SELECT", "MODIFY"},
	}

	plan := direct.CalculatePlan(u, state)
	assert.Equal(t, deployplan.Skip, plan.Plan["resources.grants.analysts"].Action)
}

func TestCalculatePlan_StorageCredentialCreate(t *testing.T) {
	u := &ucm.Ucm{Config: config.Root{}}
	u.Config.Resources.StorageCredentials = map[string]*resources.StorageCredential{
		"prod": {
			Name:       "prod",
			AwsIamRole: &resources.AwsIamRole{RoleArn: "arn:aws:iam::1:role/uc"},
		},
	}
	plan := direct.CalculatePlan(u, direct.NewState())
	assert.Equal(t, deployplan.Create, plan.Plan["resources.storage_credentials.prod"].Action)
}

func TestCalculatePlan_StorageCredentialDelete(t *testing.T) {
	u := &ucm.Ucm{Config: config.Root{}}
	state := direct.NewState()
	state.StorageCredentials["prod"] = &direct.StorageCredentialState{
		Name:       "prod",
		AwsIamRole: &direct.AwsIamRoleState{RoleArn: "arn:aws:iam::1:role/uc"},
	}
	plan := direct.CalculatePlan(u, state)
	assert.Equal(t, deployplan.Delete, plan.Plan["resources.storage_credentials.prod"].Action)
}

func TestCalculatePlan_StorageCredentialSkip(t *testing.T) {
	u := &ucm.Ucm{Config: config.Root{}}
	u.Config.Resources.StorageCredentials = map[string]*resources.StorageCredential{
		"prod": {
			Name:       "prod",
			Comment:    "prod",
			AwsIamRole: &resources.AwsIamRole{RoleArn: "arn:aws:iam::1:role/uc"},
		},
	}
	state := direct.NewState()
	state.StorageCredentials["prod"] = &direct.StorageCredentialState{
		Name:       "prod",
		Comment:    "prod",
		AwsIamRole: &direct.AwsIamRoleState{RoleArn: "arn:aws:iam::1:role/uc"},
	}
	plan := direct.CalculatePlan(u, state)
	assert.Equal(t, deployplan.Skip, plan.Plan["resources.storage_credentials.prod"].Action)
}

func TestCalculatePlan_StorageCredentialUpdateOnIdentityDrift(t *testing.T) {
	u := &ucm.Ucm{Config: config.Root{}}
	u.Config.Resources.StorageCredentials = map[string]*resources.StorageCredential{
		"prod": {
			Name:       "prod",
			AwsIamRole: &resources.AwsIamRole{RoleArn: "arn:aws:iam::1:role/new"},
		},
	}
	state := direct.NewState()
	state.StorageCredentials["prod"] = &direct.StorageCredentialState{
		Name:       "prod",
		AwsIamRole: &direct.AwsIamRoleState{RoleArn: "arn:aws:iam::1:role/old"},
	}
	plan := direct.CalculatePlan(u, state)
	assert.Equal(t, deployplan.Update, plan.Plan["resources.storage_credentials.prod"].Action)
}

func TestCalculatePlan_ExternalLocationCreate(t *testing.T) {
	u := &ucm.Ucm{Config: config.Root{}}
	u.Config.Resources.ExternalLocations = map[string]*resources.ExternalLocation{
		"prod": {
			Name:           "prod",
			Url:            "s3://bucket/prefix",
			CredentialName: "prod",
		},
	}
	plan := direct.CalculatePlan(u, direct.NewState())
	assert.Equal(t, deployplan.Create, plan.Plan["resources.external_locations.prod"].Action)
}

func TestCalculatePlan_ExternalLocationDelete(t *testing.T) {
	u := &ucm.Ucm{Config: config.Root{}}
	state := direct.NewState()
	state.ExternalLocations["prod"] = &direct.ExternalLocationState{
		Name:           "prod",
		Url:            "s3://bucket/prefix",
		CredentialName: "prod",
	}
	plan := direct.CalculatePlan(u, state)
	assert.Equal(t, deployplan.Delete, plan.Plan["resources.external_locations.prod"].Action)
}

func TestCalculatePlan_ExternalLocationSkip(t *testing.T) {
	u := &ucm.Ucm{Config: config.Root{}}
	u.Config.Resources.ExternalLocations = map[string]*resources.ExternalLocation{
		"prod": {
			Name:           "prod",
			Url:            "s3://bucket/prefix",
			CredentialName: "prod",
			Comment:        "prod",
			ReadOnly:       true,
		},
	}
	state := direct.NewState()
	state.ExternalLocations["prod"] = &direct.ExternalLocationState{
		Name:           "prod",
		Url:            "s3://bucket/prefix",
		CredentialName: "prod",
		Comment:        "prod",
		ReadOnly:       true,
	}
	plan := direct.CalculatePlan(u, state)
	assert.Equal(t, deployplan.Skip, plan.Plan["resources.external_locations.prod"].Action)
}

func TestCalculatePlan_ExternalLocationUpdateOnUrlDrift(t *testing.T) {
	u := &ucm.Ucm{Config: config.Root{}}
	u.Config.Resources.ExternalLocations = map[string]*resources.ExternalLocation{
		"prod": {
			Name:           "prod",
			Url:            "s3://bucket/new",
			CredentialName: "prod",
		},
	}
	state := direct.NewState()
	state.ExternalLocations["prod"] = &direct.ExternalLocationState{
		Name:           "prod",
		Url:            "s3://bucket/old",
		CredentialName: "prod",
	}
	plan := direct.CalculatePlan(u, state)
	assert.Equal(t, deployplan.Update, plan.Plan["resources.external_locations.prod"].Action)
}

func ucmWith(catalogs map[string]*resources.Catalog, schemas map[string]*resources.Schema, grants map[string]*resources.Grant) *ucm.Ucm {
	u := &ucm.Ucm{Config: config.Root{}}
	u.Config.Resources.Catalogs = catalogs
	u.Config.Resources.Schemas = schemas
	u.Config.Resources.Grants = grants
	return u
}
