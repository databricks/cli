package resources

import (
	"context"
	"fmt"
	"net/url"

	"github.com/databricks/cli/libs/log"
	"github.com/databricks/cli/libs/workspaceurls"
	"github.com/databricks/databricks-sdk-go"
	"github.com/databricks/databricks-sdk-go/apierr"
	"github.com/databricks/databricks-sdk-go/marshal"
	"github.com/databricks/databricks-sdk-go/service/catalog"
)

type Secret struct {
	BaseResource
	catalog.Secret

	// List of grants to apply on this secret.
	Grants []catalog.PrivilegeAssignment `json:"grants,omitempty"`
}

func (s *Secret) UnmarshalJSON(b []byte) error {
	return marshal.Unmarshal(b, s)
}

func (s *Secret) MarshalJSON() ([]byte, error) {
	return marshal.Marshal(s)
}

func (s *Secret) Exists(ctx context.Context, w *databricks.WorkspaceClient, fullName string) (bool, error) {
	log.Tracef(ctx, "Checking if secret with fullName=%s exists", fullName)

	_, err := w.SecretsUc.GetSecret(ctx, catalog.GetSecretRequest{
		FullName: fullName,
	})
	if err != nil {
		log.Debugf(ctx, "secret with full name %s does not exist: %v", fullName, err)

		if apierr.IsMissing(err) {
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

func (s *Secret) GetURL() string {
	return s.URL
}

func (s *Secret) GetName() string {
	if s.ID != "" {
		return s.ID
	}
	return fmt.Sprintf("%s.%s.%s", s.CatalogName, s.SchemaName, s.Name)
}
