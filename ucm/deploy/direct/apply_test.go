package direct_test

import (
	"context"
	"errors"
	"testing"

	"github.com/databricks/cli/ucm"
	"github.com/databricks/cli/ucm/config/resources"
	"github.com/databricks/cli/ucm/deploy/direct"
	"github.com/databricks/databricks-sdk-go/service/catalog"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// recordingClient captures every SDK call in order so tests can assert both
// the set of calls and their sequencing without mocking the wire protocol.
type recordingClient struct {
	Calls []string

	CreatedCatalogs []catalog.CreateCatalog
	UpdatedCatalogs []catalog.UpdateCatalog
	DeletedCatalogs []string

	CreatedSchemas []catalog.CreateSchema
	UpdatedSchemas []catalog.UpdateSchema
	DeletedSchemas []string

	CreatedStorageCredentials []catalog.CreateStorageCredential
	UpdatedStorageCredentials []catalog.UpdateStorageCredential
	DeletedStorageCredentials []string

	CreatedExternalLocations []catalog.CreateExternalLocation
	UpdatedExternalLocations []catalog.UpdateExternalLocation
	DeletedExternalLocations []string

	Permissions []catalog.UpdatePermissions

	FailOn string
}

func (r *recordingClient) trip(op string) error {
	r.Calls = append(r.Calls, op)
	if r.FailOn == op {
		return errors.New("forced")
	}
	return nil
}

func (r *recordingClient) GetCatalog(_ context.Context, _ string) (*catalog.CatalogInfo, error) {
	return nil, nil
}

func (r *recordingClient) CreateCatalog(_ context.Context, in catalog.CreateCatalog) (*catalog.CatalogInfo, error) {
	if err := r.trip("CreateCatalog:" + in.Name); err != nil {
		return nil, err
	}
	r.CreatedCatalogs = append(r.CreatedCatalogs, in)
	return &catalog.CatalogInfo{Name: in.Name}, nil
}

func (r *recordingClient) UpdateCatalog(_ context.Context, in catalog.UpdateCatalog) (*catalog.CatalogInfo, error) {
	if err := r.trip("UpdateCatalog:" + in.Name); err != nil {
		return nil, err
	}
	r.UpdatedCatalogs = append(r.UpdatedCatalogs, in)
	return &catalog.CatalogInfo{Name: in.Name}, nil
}

func (r *recordingClient) DeleteCatalog(_ context.Context, name string) error {
	if err := r.trip("DeleteCatalog:" + name); err != nil {
		return err
	}
	r.DeletedCatalogs = append(r.DeletedCatalogs, name)
	return nil
}

func (r *recordingClient) GetSchema(_ context.Context, _ string) (*catalog.SchemaInfo, error) {
	return nil, nil
}

func (r *recordingClient) CreateSchema(_ context.Context, in catalog.CreateSchema) (*catalog.SchemaInfo, error) {
	if err := r.trip("CreateSchema:" + in.CatalogName + "." + in.Name); err != nil {
		return nil, err
	}
	r.CreatedSchemas = append(r.CreatedSchemas, in)
	return &catalog.SchemaInfo{Name: in.Name, CatalogName: in.CatalogName}, nil
}

func (r *recordingClient) UpdateSchema(_ context.Context, in catalog.UpdateSchema) (*catalog.SchemaInfo, error) {
	if err := r.trip("UpdateSchema:" + in.FullName); err != nil {
		return nil, err
	}
	r.UpdatedSchemas = append(r.UpdatedSchemas, in)
	return &catalog.SchemaInfo{FullName: in.FullName}, nil
}

func (r *recordingClient) DeleteSchema(_ context.Context, fullName string) error {
	if err := r.trip("DeleteSchema:" + fullName); err != nil {
		return err
	}
	r.DeletedSchemas = append(r.DeletedSchemas, fullName)
	return nil
}

func (r *recordingClient) GetStorageCredential(_ context.Context, _ string) (*catalog.StorageCredentialInfo, error) {
	return nil, nil
}

func (r *recordingClient) CreateStorageCredential(_ context.Context, in catalog.CreateStorageCredential) (*catalog.StorageCredentialInfo, error) {
	if err := r.trip("CreateStorageCredential:" + in.Name); err != nil {
		return nil, err
	}
	r.CreatedStorageCredentials = append(r.CreatedStorageCredentials, in)
	return &catalog.StorageCredentialInfo{Name: in.Name}, nil
}

func (r *recordingClient) UpdateStorageCredential(_ context.Context, in catalog.UpdateStorageCredential) (*catalog.StorageCredentialInfo, error) {
	if err := r.trip("UpdateStorageCredential:" + in.Name); err != nil {
		return nil, err
	}
	r.UpdatedStorageCredentials = append(r.UpdatedStorageCredentials, in)
	return &catalog.StorageCredentialInfo{Name: in.Name}, nil
}

func (r *recordingClient) DeleteStorageCredential(_ context.Context, name string) error {
	if err := r.trip("DeleteStorageCredential:" + name); err != nil {
		return err
	}
	r.DeletedStorageCredentials = append(r.DeletedStorageCredentials, name)
	return nil
}

func (r *recordingClient) GetExternalLocation(_ context.Context, _ string) (*catalog.ExternalLocationInfo, error) {
	return nil, nil
}

func (r *recordingClient) CreateExternalLocation(_ context.Context, in catalog.CreateExternalLocation) (*catalog.ExternalLocationInfo, error) {
	if err := r.trip("CreateExternalLocation:" + in.Name); err != nil {
		return nil, err
	}
	r.CreatedExternalLocations = append(r.CreatedExternalLocations, in)
	return &catalog.ExternalLocationInfo{Name: in.Name}, nil
}

func (r *recordingClient) UpdateExternalLocation(_ context.Context, in catalog.UpdateExternalLocation) (*catalog.ExternalLocationInfo, error) {
	if err := r.trip("UpdateExternalLocation:" + in.Name); err != nil {
		return nil, err
	}
	r.UpdatedExternalLocations = append(r.UpdatedExternalLocations, in)
	return &catalog.ExternalLocationInfo{Name: in.Name}, nil
}

func (r *recordingClient) DeleteExternalLocation(_ context.Context, name string) error {
	if err := r.trip("DeleteExternalLocation:" + name); err != nil {
		return err
	}
	r.DeletedExternalLocations = append(r.DeletedExternalLocations, name)
	return nil
}

func (r *recordingClient) UpdatePermissions(_ context.Context, in catalog.UpdatePermissions) error {
	if err := r.trip("UpdatePermissions:" + string(in.SecurableType) + ":" + in.FullName); err != nil {
		return err
	}
	r.Permissions = append(r.Permissions, in)
	return nil
}

func TestApply_CreateHappyPath(t *testing.T) {
	u := ucmWith(
		map[string]*resources.Catalog{"main": {Name: "main", Comment: "prod"}},
		map[string]*resources.Schema{"raw": {Name: "raw", Catalog: "main"}},
		map[string]*resources.Grant{
			"analysts": {
				Securable:  resources.Securable{Type: "schema", Name: "main.raw"},
				Principal:  "analysts",
				Privileges: []string{"SELECT", "USE_SCHEMA"},
			},
		},
	)
	state := direct.NewState()
	plan := direct.CalculatePlan(u, state)

	client := &recordingClient{}
	require.NoError(t, direct.Apply(t.Context(), u, client, plan, state))

	assert.Equal(t, []string{
		"CreateCatalog:main",
		"CreateSchema:main.raw",
		"UpdatePermissions:schema:main.raw",
	}, client.Calls)

	assert.Equal(t, "prod", state.Catalogs["main"].Comment)
	assert.Equal(t, "main", state.Schemas["raw"].Catalog)
	require.NotNil(t, state.Grants["analysts"])
	assert.ElementsMatch(t, []string{"SELECT", "USE_SCHEMA"}, state.Grants["analysts"].Privileges)
}

func TestApply_GrantCoalescingPerSecurable(t *testing.T) {
	u := ucmWith(nil, nil, map[string]*resources.Grant{
		"a_select": {
			Securable:  resources.Securable{Type: "schema", Name: "main.raw"},
			Principal:  "analysts",
			Privileges: []string{"SELECT"},
		},
		"b_use": {
			Securable:  resources.Securable{Type: "schema", Name: "main.raw"},
			Principal:  "developers",
			Privileges: []string{"USE_SCHEMA"},
		},
	})

	state := direct.NewState()
	plan := direct.CalculatePlan(u, state)

	client := &recordingClient{}
	require.NoError(t, direct.Apply(t.Context(), u, client, plan, state))

	require.Len(t, client.Permissions, 1, "two grants on the same securable must collapse to one call")
	call := client.Permissions[0]
	assert.Equal(t, "schema", call.SecurableType)
	assert.Equal(t, "main.raw", call.FullName)
	assert.Len(t, call.Changes, 2)
}

func TestApply_DeleteReversesOrder(t *testing.T) {
	u := &ucm.Ucm{}
	state := direct.NewState()
	state.Catalogs["main"] = &direct.CatalogState{Name: "main"}
	state.Schemas["raw"] = &direct.SchemaState{Name: "raw", Catalog: "main"}
	state.Grants["analysts"] = &direct.GrantState{
		SecurableType: "schema",
		SecurableName: "main.raw",
		Principal:     "analysts",
		Privileges:    []string{"SELECT"},
	}

	client := &recordingClient{}
	plan, err := direct.Destroy(t.Context(), u, client, state)
	require.NoError(t, err)
	require.NotNil(t, plan)

	assert.Equal(t, []string{
		"UpdatePermissions:schema:main.raw",
		"DeleteSchema:main.raw",
		"DeleteCatalog:main",
	}, client.Calls)

	assert.Empty(t, state.Catalogs)
	assert.Empty(t, state.Schemas)
	assert.Empty(t, state.Grants)
}

func TestApply_PreservesStateOnMidApplyError(t *testing.T) {
	u := ucmWith(
		map[string]*resources.Catalog{"main": {Name: "main"}},
		map[string]*resources.Schema{"raw": {Name: "raw", Catalog: "main"}},
		nil,
	)
	state := direct.NewState()
	plan := direct.CalculatePlan(u, state)

	client := &recordingClient{FailOn: "CreateSchema:main.raw"}
	err := direct.Apply(t.Context(), u, client, plan, state)
	require.Error(t, err)

	// Catalog create committed before the schema create failed — its state
	// is kept so the next retry sees the partial progress.
	assert.NotNil(t, state.Catalogs["main"])
	assert.Nil(t, state.Schemas["raw"])
}

func TestApply_StorageCredentialCreateOrdersBeforeCatalog(t *testing.T) {
	u := ucmWith(
		map[string]*resources.Catalog{"main": {Name: "main"}},
		nil,
		nil,
	)
	u.Config.Resources.StorageCredentials = map[string]*resources.StorageCredential{
		"prod": {
			Name:       "prod",
			AwsIamRole: &resources.AwsIamRole{RoleArn: "arn:aws:iam::1:role/uc"},
		},
	}

	state := direct.NewState()
	plan := direct.CalculatePlan(u, state)

	client := &recordingClient{}
	require.NoError(t, direct.Apply(t.Context(), u, client, plan, state))

	assert.Equal(t, []string{
		"CreateStorageCredential:prod",
		"CreateCatalog:main",
	}, client.Calls)

	require.NotNil(t, state.StorageCredentials["prod"])
	assert.Equal(t, "arn:aws:iam::1:role/uc", state.StorageCredentials["prod"].AwsIamRole.RoleArn)
}

func TestApply_StorageCredentialUpdate(t *testing.T) {
	u := &ucm.Ucm{}
	u.Config.Resources.StorageCredentials = map[string]*resources.StorageCredential{
		"prod": {
			Name:       "prod",
			Comment:    "new",
			AwsIamRole: &resources.AwsIamRole{RoleArn: "arn:aws:iam::1:role/new"},
		},
	}
	state := direct.NewState()
	state.StorageCredentials["prod"] = &direct.StorageCredentialState{
		Name:       "prod",
		Comment:    "old",
		AwsIamRole: &direct.AwsIamRoleState{RoleArn: "arn:aws:iam::1:role/old"},
	}
	plan := direct.CalculatePlan(u, state)

	client := &recordingClient{}
	require.NoError(t, direct.Apply(t.Context(), u, client, plan, state))

	assert.Equal(t, []string{"UpdateStorageCredential:prod"}, client.Calls)
	assert.Equal(t, "new", state.StorageCredentials["prod"].Comment)
	assert.Equal(t, "arn:aws:iam::1:role/new", state.StorageCredentials["prod"].AwsIamRole.RoleArn)
}

func TestApply_StorageCredentialDeleteOrdersAfterCatalog(t *testing.T) {
	u := &ucm.Ucm{}
	state := direct.NewState()
	state.Catalogs["main"] = &direct.CatalogState{Name: "main"}
	state.StorageCredentials["prod"] = &direct.StorageCredentialState{
		Name:       "prod",
		AwsIamRole: &direct.AwsIamRoleState{RoleArn: "arn:aws:iam::1:role/uc"},
	}

	client := &recordingClient{}
	plan, err := direct.Destroy(t.Context(), u, client, state)
	require.NoError(t, err)
	require.NotNil(t, plan)

	assert.Equal(t, []string{
		"DeleteCatalog:main",
		"DeleteStorageCredential:prod",
	}, client.Calls)

	assert.Empty(t, state.Catalogs)
	assert.Empty(t, state.StorageCredentials)
}

func TestApply_StorageCredentialRejectsMissingIdentity(t *testing.T) {
	u := &ucm.Ucm{}
	u.Config.Resources.StorageCredentials = map[string]*resources.StorageCredential{
		"bad": {Name: "bad"},
	}
	state := direct.NewState()
	plan := direct.CalculatePlan(u, state)

	client := &recordingClient{}
	err := direct.Apply(t.Context(), u, client, plan, state)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "exactly one identity field")
	assert.Empty(t, client.Calls, "no API call must be made when validation fails")
}

func TestApply_ExternalLocationCreateOrdersAfterStorageCredential(t *testing.T) {
	u := &ucm.Ucm{}
	u.Config.Resources.StorageCredentials = map[string]*resources.StorageCredential{
		"prod": {
			Name:       "prod",
			AwsIamRole: &resources.AwsIamRole{RoleArn: "arn:aws:iam::1:role/uc"},
		},
	}
	u.Config.Resources.ExternalLocations = map[string]*resources.ExternalLocation{
		"data": {
			Name:           "data",
			Url:            "s3://bucket/prefix",
			CredentialName: "prod",
		},
	}

	state := direct.NewState()
	plan := direct.CalculatePlan(u, state)

	client := &recordingClient{}
	require.NoError(t, direct.Apply(t.Context(), u, client, plan, state))

	assert.Equal(t, []string{
		"CreateStorageCredential:prod",
		"CreateExternalLocation:data",
	}, client.Calls)

	require.NotNil(t, state.ExternalLocations["data"])
	assert.Equal(t, "s3://bucket/prefix", state.ExternalLocations["data"].Url)
	assert.Equal(t, "prod", state.ExternalLocations["data"].CredentialName)
}

func TestApply_ExternalLocationUpdate(t *testing.T) {
	u := &ucm.Ucm{}
	u.Config.Resources.ExternalLocations = map[string]*resources.ExternalLocation{
		"data": {
			Name:           "data",
			Url:            "s3://bucket/new",
			CredentialName: "prod",
			Comment:        "new",
		},
	}
	state := direct.NewState()
	state.ExternalLocations["data"] = &direct.ExternalLocationState{
		Name:           "data",
		Url:            "s3://bucket/old",
		CredentialName: "prod",
		Comment:        "old",
	}
	plan := direct.CalculatePlan(u, state)

	client := &recordingClient{}
	require.NoError(t, direct.Apply(t.Context(), u, client, plan, state))

	assert.Equal(t, []string{"UpdateExternalLocation:data"}, client.Calls)
	assert.Equal(t, "s3://bucket/new", state.ExternalLocations["data"].Url)
	assert.Equal(t, "new", state.ExternalLocations["data"].Comment)
}

func TestApply_ExternalLocationDestroyOrder(t *testing.T) {
	u := &ucm.Ucm{}
	state := direct.NewState()
	state.Catalogs["main"] = &direct.CatalogState{Name: "main"}
	state.ExternalLocations["data"] = &direct.ExternalLocationState{
		Name:           "data",
		Url:            "s3://bucket/prefix",
		CredentialName: "prod",
	}
	state.StorageCredentials["prod"] = &direct.StorageCredentialState{
		Name:       "prod",
		AwsIamRole: &direct.AwsIamRoleState{RoleArn: "arn:aws:iam::1:role/uc"},
	}

	client := &recordingClient{}
	plan, err := direct.Destroy(t.Context(), u, client, state)
	require.NoError(t, err)
	require.NotNil(t, plan)

	assert.Equal(t, []string{
		"DeleteCatalog:main",
		"DeleteExternalLocation:data",
		"DeleteStorageCredential:prod",
	}, client.Calls)

	assert.Empty(t, state.Catalogs)
	assert.Empty(t, state.ExternalLocations)
	assert.Empty(t, state.StorageCredentials)
}

func TestApply_RevokesPrincipalsNotInConfig(t *testing.T) {
	u := ucmWith(nil, nil, map[string]*resources.Grant{
		"analysts": {
			Securable:  resources.Securable{Type: "schema", Name: "main.raw"},
			Principal:  "analysts",
			Privileges: []string{"SELECT"},
		},
	})

	state := direct.NewState()
	state.Grants["legacy"] = &direct.GrantState{
		SecurableType: "schema",
		SecurableName: "main.raw",
		Principal:     "contractors",
		Privileges:    []string{"MODIFY"},
	}

	plan := direct.CalculatePlan(u, state)
	client := &recordingClient{}
	require.NoError(t, direct.Apply(t.Context(), u, client, plan, state))

	require.Len(t, client.Permissions, 1)
	changes := client.Permissions[0].Changes

	var sawRevocation bool
	for _, c := range changes {
		if c.Principal == "contractors" && len(c.Remove) > 0 {
			sawRevocation = true
		}
	}
	assert.True(t, sawRevocation, "removed principals must be revoked in the reconcile call")
}
