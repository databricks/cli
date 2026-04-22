package tfdyn

import (
	"context"
	"fmt"

	"github.com/databricks/cli/libs/dyn"
)

var storageCredentialIdentityFields = []string{
	"aws_iam_role",
	"azure_managed_identity",
	"azure_service_principal",
	"databricks_gcp_service_account",
}

// convertStorageCredentialResource transforms a ucm storage_credential entry
// into a dyn.Value shaped like a databricks_storage_credential Terraform
// block. Exactly one identity field must be present.
func convertStorageCredentialResource(_ context.Context, key string, vin dyn.Value) (dyn.Value, error) {
	pairs := []dyn.Pair{}
	appendString(&pairs, vin, "name", key)
	appendStringIfSet(&pairs, vin, "comment")

	var seen string
	for _, f := range storageCredentialIdentityFields {
		v := vin.Get(f)
		if !v.IsValid() {
			continue
		}
		if seen != "" {
			return dyn.InvalidValue, fmt.Errorf("storage_credential %q: exactly one identity field allowed, got both %s and %s", key, seen, f)
		}
		seen = f
		pairs = append(pairs, dyn.Pair{
			Key:   dyn.NewValue(f, v.Locations()),
			Value: v,
		})
	}
	if seen == "" {
		return dyn.InvalidValue, fmt.Errorf("storage_credential %q: exactly one identity field (aws_iam_role, azure_managed_identity, azure_service_principal, databricks_gcp_service_account) is required", key)
	}

	appendBoolIfSet(&pairs, vin, "read_only")
	appendBoolIfSet(&pairs, vin, "skip_validation")

	return dyn.NewValue(dyn.NewMappingFromPairs(pairs), vin.Locations()), nil
}

type storageCredentialConverter struct{}

func (storageCredentialConverter) Convert(ctx context.Context, key string, vin dyn.Value, out *Resources) error {
	v, err := convertStorageCredentialResource(ctx, key, vin)
	if err != nil {
		return err
	}
	out.StorageCredential[key] = v
	return nil
}

func init() {
	registerConverter("storage_credentials", storageCredentialConverter{})
}
