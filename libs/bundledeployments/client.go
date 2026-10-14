// Package bundledeployments is a client for the bundle deployments API
// (/api/2.0/bundle/...), which sdk-go does not provide.
package bundledeployments

import (
	"encoding/json"
	"fmt"
	"net/url"
	"strings"

	"github.com/databricks/cli/libs/apiclient"
)

// Client calls the bundle deployments API.
type Client struct {
	bundleDeploymentsImpl
}

func New(c *apiclient.Client) *Client {
	return &Client{bundleDeploymentsImpl{client: c}}
}

func addWorkspaceID(c *apiclient.Client, headers map[string]string) {
	c.AddWorkspaceIDHeader(headers)
}

// FieldMask is a list of field paths, encoded as a comma-separated string.
type FieldMask struct {
	Paths []string
}

func (f FieldMask) MarshalJSON() ([]byte, error) {
	return json.Marshal(strings.Join(f.Paths, ","))
}

func (f *FieldMask) UnmarshalJSON(data []byte) error {
	if f == nil {
		return fmt.Errorf("FieldMask.UnmarshalJSON on nil pointer")
	}
	var s string
	if err := json.Unmarshal(data, &s); err != nil {
		return err
	}
	if s != "" {
		f.Paths = strings.Split(s, ",")
	} else {
		f.Paths = []string{}
	}
	return nil
}

func (f *FieldMask) EncodeValues(key string, v *url.Values) error {
	if f == nil || len(f.Paths) == 0 {
		return nil
	}
	v.Set(key, strings.Join(f.Paths, ","))
	return nil
}
