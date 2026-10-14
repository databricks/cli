package resources

import (
	"context"
	"fmt"
	"github.com/databricks/cli/libs/apierrs"
	"github.com/databricks/cli/libs/sdkclients"
	"github.com/databricks/cli/libs/sdktypes/uc/grants/v1"
	"github.com/databricks/cli/libs/sdktypes/uc/secrets/v1"
	"github.com/databricks/cli/libs/utils"
	sdksecretsuc "github.com/databricks/sdk-go/uc/secrets/v1"
	"net/url"

	"github.com/databricks/cli/libs/log"
	"github.com/databricks/cli/libs/workspaceurls"
)

type Secret struct {
	BaseResource
	ID string `json:"id,omitempty" bundle:"readonly"`
	secrets.Secret

	// List of grants to apply on this secret.
	Grants []grants.PrivilegeAssignment `json:"grants,omitempty"`
}

func (s *Secret) Exists(ctx context.Context, w *sdkclients.Clients, fullName string) (bool, error) {
	log.Tracef(ctx, "Checking if secret with fullName=%s exists", fullName)

	_, err := w.SecretsUc.GetSecret(ctx, sdksecretsuc.GetSecretRequest{
		FullName: new(fullName),
	})
	if err != nil {
		log.Debugf(ctx, "secret with full name %s does not exist: %v", fullName, err)

		if apierrs.IsMissing(err) {
			return false, nil
		}

		return false, err
	}
	return true, nil
}

func (*Secret) ResourceDescription() ResourceDescription {
	return ResourceDescription{
		SingularName:  "secret",
		PluralName:    "secrets",
		SingularTitle: "Secret",
		PluralTitle:   "Secrets",
	}
}

func (s *Secret) InitializeURL(baseURL url.URL) {
	if s.ID == "" {
		return
	}
	s.URL = workspaceurls.ResourceURL(baseURL, "secrets", s.ID)
}

func (s *Secret) GetName() string {
	if s.ID != "" {
		return s.ID
	}
	return fmt.Sprintf("%s.%s.%s", utils.Deref(s.CatalogName), utils.Deref(s.SchemaName), utils.Deref(s.Name))
}
