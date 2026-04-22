package tfdyn

import (
	"testing"

	"github.com/databricks/cli/libs/dyn"
	"github.com/databricks/cli/libs/dyn/convert"
	"github.com/databricks/cli/ucm/config/resources"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestConvertStorageCredential(t *testing.T) {
	tests := []struct {
		name string
		key  string
		src  resources.StorageCredential
		want map[string]any
	}{
		{
			name: "aws iam role",
			key:  "sales_cred",
			src: resources.StorageCredential{
				Name:       "sales_cred",
				AwsIamRole: &resources.AwsIamRole{RoleArn: "arn:aws:iam::1:role/uc"},
			},
			want: map[string]any{
				"name": "sales_cred",
				"aws_iam_role": map[string]any{
					"role_arn": "arn:aws:iam::1:role/uc",
				},
			},
		},
		{
			name: "azure managed identity",
			key:  "azure_cred",
			src: resources.StorageCredential{
				Name: "azure_cred",
				AzureManagedIdentity: &resources.AzureManagedIdentity{
					AccessConnectorId: "/subscriptions/x/rg/acme/providers/Microsoft.Databricks/accessConnectors/uc",
				},
			},
			want: map[string]any{
				"name": "azure_cred",
				"azure_managed_identity": map[string]any{
					"access_connector_id": "/subscriptions/x/rg/acme/providers/Microsoft.Databricks/accessConnectors/uc",
				},
			},
		},
		{
			name: "databricks gcp sa",
			key:  "gcp_cred",
			src: resources.StorageCredential{
				Name:                        "gcp_cred",
				DatabricksGcpServiceAccount: &resources.DatabricksGcpServiceAccount{},
			},
			want: map[string]any{
				"name":                           "gcp_cred",
				"databricks_gcp_service_account": map[string]any{},
			},
		},
		{
			name: "with comment and read_only",
			key:  "ro_cred",
			src: resources.StorageCredential{
				Name:       "ro_cred",
				Comment:    "read-only",
				ReadOnly:   true,
				AwsIamRole: &resources.AwsIamRole{RoleArn: "arn:aws:iam::1:role/ro"},
			},
			want: map[string]any{
				"name":      "ro_cred",
				"comment":   "read-only",
				"read_only": true,
				"aws_iam_role": map[string]any{
					"role_arn": "arn:aws:iam::1:role/ro",
				},
			},
		},
		{
			name: "defaults name from key",
			key:  "inferred",
			src:  resources.StorageCredential{AwsIamRole: &resources.AwsIamRole{RoleArn: "arn:aws:iam::1:role/x"}},
			want: map[string]any{
				"name": "inferred",
				"aws_iam_role": map[string]any{
					"role_arn": "arn:aws:iam::1:role/x",
				},
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			vin, err := convert.FromTyped(tc.src, dyn.NilValue)
			require.NoError(t, err)
			out := NewResources()
			err = storageCredentialConverter{}.Convert(t.Context(), tc.key, vin, out)
			require.NoError(t, err)
			got, ok := out.StorageCredential[tc.key]
			require.True(t, ok)
			assert.Equal(t, tc.want, got.AsAny())
		})
	}
}

func TestConvertStorageCredential_ErrorsOnMissingIdentity(t *testing.T) {
	vin, err := convert.FromTyped(resources.StorageCredential{Name: "bad"}, dyn.NilValue)
	require.NoError(t, err)
	out := NewResources()
	err = storageCredentialConverter{}.Convert(t.Context(), "bad", vin, out)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "exactly one identity")
}
