package resources

import (
	"net/url"
	"strings"

	"github.com/databricks/databricks-sdk-go/marshal"
	"github.com/databricks/databricks-sdk-go/service/catalog"
)

// Schema is a UC schema (a.k.a. database) nested inside a catalog. Embeds the
// SDK's CreateSchema so every API-level field is part of ucm's surface.
type Schema struct {
	catalog.CreateSchema

	// Tags is a key/value map evaluated by ucm's tag-validation mutators.
	// Distinct from the SDK's `properties` (free-form) — Tags is ucm-policy
	// material.
	Tags map[string]string `json:"tags,omitempty"`

	// TagInherit controls whether parent-catalog tags merge into this schema's
	// tags (schema-key wins). nil means inherit (the default). Set false to
	// opt out.
	TagInherit *bool `json:"tag_inherit,omitempty"`

	// Grants exists solely to keep convert.Normalize from dropping nested-form
	// keys during load — FlattenNestedResources zeroes it before any other
	// mutator runs. Always nil after load.
	Grants map[string]*Grant `json:"grants,omitempty"`

	// ID is the deployed resource's terraform-state ID. Populated by
	// statemgmt.Load from the local tfstate; never written from ucm.yml.
	ID string `json:"id,omitempty" ucm:"readonly"`

	// URL is populated by the initialize_urls mutator.
	URL string `json:"url,omitempty" ucm:"readonly"`
}

// InitializeURL sets s.URL iff the schema has been deployed (ID is non-empty).
// Schema ID is the dotted full name catalog.schema; explore URLs use slashes.
func (s *Schema) InitializeURL(baseURL url.URL) {
	if s.ID == "" {
		return
	}
	baseURL.Path = "explore/data/" + strings.ReplaceAll(s.ID, ".", "/")
	s.URL = baseURL.String()
}

func (s *Schema) UnmarshalJSON(b []byte) error {
	return marshal.Unmarshal(b, s)
}

func (s Schema) MarshalJSON() ([]byte, error) {
	return marshal.Marshal(s)
}
