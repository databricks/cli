package resources

import (
	"context"
	"net/url"

	"github.com/databricks/cli/libs/apierrs"
	"github.com/databricks/cli/libs/sdkclients"
	"github.com/databricks/cli/libs/sdktypes/uc/grants/v1"
	"github.com/databricks/cli/libs/utils"
	sdkschemas "github.com/databricks/sdk-go/uc/schemas/v1"

	"github.com/databricks/cli/libs/log"
	"github.com/databricks/cli/libs/workspaceurls"
)

type Schema struct {
	BaseResource
	ID string `json:"id,omitempty" bundle:"readonly"`
	CreateSchema
	// List of grants to apply on this schema.
	Grants []grants.PrivilegeAssignment `json:"grants,omitempty"`
}

func (s *Schema) Exists(ctx context.Context, w *sdkclients.Clients, fullName string) (bool, error) {
	log.Tracef(ctx, "Checking if schema with fullName=%s exists", fullName)

	_, err := w.Schemas.GetSchema(ctx, sdkschemas.GetSchemaRequest{FullNameArg: new(fullName)})
	if err != nil {
		log.Debugf(ctx, "schema with full name %s does not exist: %v", fullName, err)

		if apierrs.IsMissing(err) {
			return false, nil
		}

		return false, err
	}
	return true, nil
}

func (*Schema) ResourceDescription() ResourceDescription {
	return ResourceDescription{
		SingularName:  "schema",
		PluralName:    "schemas",
		SingularTitle: "Schema",
		PluralTitle:   "Schemas",
	}
}

func (s *Schema) InitializeURL(baseURL url.URL) {
	if s.ID == "" {
		return
	}
	s.URL = workspaceurls.ResourceURL(baseURL, "schemas", s.ID)
}

func (s *Schema) GetName() string {
	return utils.Deref(s.Name)
}
