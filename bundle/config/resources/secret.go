package resources

import (
	"context"
	"fmt"
	"net/url"

	"github.com/databricks/databricks-sdk-go"
	"github.com/databricks/databricks-sdk-go/apierr"
	"github.com/databricks/databricks-sdk-go/marshal"
	"github.com/databricks/databricks-sdk-go/service/workspace"
)

type Secret struct {
	BaseResource

	// The name of the secret scope containing the secret.
	Scope string `json:"scope"`

	// A unique name to identify the secret.
	Key string `json:"key"`

	// The string value of the secret. Only one of string_value or bytes_value can be specified.
	StringValue string `json:"string_value,omitempty"`

	// The base64-encoded bytes value of the secret. Only one of string_value or bytes_value can be specified.
	BytesValue string `json:"bytes_value,omitempty"`
}

func (s *Secret) UnmarshalJSON(b []byte) error {
	return marshal.Unmarshal(b, s)
}

func (s Secret) MarshalJSON() ([]byte, error) {
	return marshal.Marshal(s)
}

func (s Secret) Exists(ctx context.Context, w *databricks.WorkspaceClient, id string) (bool, error) {
	// Secrets don't have a direct "get" API - we can check if it exists by trying to get its metadata
	// The GetSecret API returns the secret metadata (not the value)
	_, err := w.Secrets.GetSecret(ctx, workspace.GetSecretRequest{
		Scope: s.Scope,
		Key:   s.Key,
	})
	if err != nil {
		if apierr.IsMissing(err) {
			return false, nil
		}
		return false, err
	}
	return true, nil
}

func (s Secret) ResourceDescription() ResourceDescription {
	return ResourceDescription{
		SingularName:  "secret",
		PluralName:    "secrets",
		SingularTitle: "Secret",
		PluralTitle:   "Secrets",
	}
}

func (s Secret) GetName() string {
	if s.ID != "" {
		return s.ID
	}
	// Return a composite name of scope/key
	return fmt.Sprintf("%s/%s", s.Scope, s.Key)
}

func (s Secret) GetURL() string {
	// Secrets do not have a URL in the Databricks UI
	return ""
}

func (s Secret) InitializeURL(_ url.URL) {
	// Secrets do not have a URL
}
