package resources

import "net/url"

// Connection is a UC foreign-catalog connection (the federation link that
// lets a foreign catalog reference tables in MySQL, PostgreSQL, Snowflake,
// etc.). Field names mirror databricks-sdk-go's catalog.CreateConnection.
//
// ConnectionType is a free string matching the SDK enum (e.g. MYSQL,
// POSTGRESQL, SNOWFLAKE, REDSHIFT, BIGQUERY). Options carries the
// connection-specific configuration (host, port, user, password, etc.) and
// must contain at least enough keys for UC to authenticate.
type Connection struct {
	Name           string            `json:"name"`
	ConnectionType string            `json:"connection_type"`
	Options        map[string]string `json:"options"`
	Comment        string            `json:"comment,omitempty"`
	Properties     map[string]string `json:"properties,omitempty"`
	ReadOnly       bool              `json:"read_only,omitempty"`

	// ID is the deployed resource's terraform-state ID. Populated by
	// statemgmt.Load from the local tfstate; never written from ucm.yml.
	ID string `json:"id,omitempty" ucm:"readonly"`

	// URL is populated by the initialize_urls mutator.
	URL string `json:"url,omitempty" ucm:"readonly"`
}

// InitializeURL sets c.URL iff the connection has been deployed
// (ID is non-empty).
func (c *Connection) InitializeURL(baseURL url.URL) {
	if c.ID == "" {
		return
	}
	baseURL.Path = "explore/connections/" + c.Name
	c.URL = baseURL.String()
}
