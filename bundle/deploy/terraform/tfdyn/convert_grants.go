package tfdyn

import (
	"context"

	"github.com/databricks/cli/bundle/internal/tf/schema"
	"github.com/databricks/cli/libs/dyn"
)

func convertGrantsResource(ctx context.Context, vin dyn.Value) *schema.ResourceGrants {
	grants, ok := vin.Get("grants").AsSequence()
	if !ok || len(grants) == 0 {
		return nil
	}

	resource := &schema.ResourceGrants{}
	for _, permission := range grants {
		principal, _ := permission.Get("principal").AsString()
		v, _ := permission.Get("privileges").AsSequence()

		// Turn privileges into a slice of strings.
		var privileges []string
		for _, privilege := range v {
			str, ok := privilege.AsString()
			if !ok {
				continue
			}

			privileges = append(privileges, str)
		}

		resource.Grant = append(resource.Grant, schema.ResourceGrantsGrant{
			Principal:  principal,
			Privileges: privileges,
		})
	}

	return resource
}
