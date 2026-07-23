package resources

import (
	"context"
	"fmt"
	"net/url"
	"time"

	"github.com/databricks/cli/libs/log"
	"github.com/databricks/cli/libs/workspaceurls"
	"github.com/databricks/databricks-sdk-go"
	"github.com/databricks/databricks-sdk-go/apierr"
	"github.com/databricks/databricks-sdk-go/marshal"
	"github.com/databricks/databricks-sdk-go/service/catalog"
)

type Secret struct {
	BaseResource

	// The name of the catalog where the schema and the secret reside.
	CatalogName string `json:"catalog_name"`

	// The name of the schema where the secret resides.
	SchemaName string `json:"schema_name"`

	// The name of the secret, relative to its parent schema.
	Name string `json:"name"`

	// The secret value to store. This field must be a variable reference (e.g., ${var.my_secret})
	// to prevent leaking secrets in configuration files. Plain text values are not allowed.
	// The maximum size is 60 KiB (pre-encryption).
	Value string `json:"value" bundle:"sensitive"`

	// User-provided free-form text description of the secret.
	Comment string `json:"comment,omitempty"`

	// The owner of the secret. Defaults to the creating principal on creation.
	// Can be updated to transfer ownership of the secret to another principal.
	Owner string `json:"owner,omitempty"`

	// User-provided expiration time of the secret. This field indicates when
	// the secret should no longer be used and may be displayed as a warning in
	// the UI. It is purely informational and does not trigger any automatic
	// actions or affect the secret's lifecycle.
	ExpireTime *time.Time `json:"expire_time,omitempty"`

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
