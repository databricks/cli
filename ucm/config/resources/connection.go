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

	// URL is populated by the initialize_urls mutator.
	URL string `json:"url,omitempty" ucm:"readonly"`
}

func (c *Connection) InitializeURL(baseURL url.URL) {
	if c.Name == "" {
		return
	}
	baseURL.Path = "explore/connections/" + c.Name
	c.URL = baseURL.String()
}
