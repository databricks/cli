package knowledge

import (
	"context"
	"crypto/rand"
	"fmt"
	"maps"
	"net/http"
	"strings"

	"github.com/databricks/cli/libs/auth"
	"github.com/databricks/databricks-sdk-go"
	"github.com/databricks/databricks-sdk-go/client"
)

const basePath = "/api/2.0/knowledge/"

// Client calls the Knowledge Pages API.
type Client struct {
	client *client.DatabricksClient
}

func NewClient(w *databricks.WorkspaceClient) (*Client, error) {
	c, err := client.New(w.Config)
	if err != nil {
		return nil, err
	}
	return &Client{client: c}, nil
}

func (c *Client) headers() map[string]string {
	headers := map[string]string{
		"Accept":       "application/json",
		"Content-Type": "application/json",
	}
	maps.Copy(headers, auth.WorkspaceIDHeaders(c.client.Config))
	return headers
}

// GetPage returns the page with the given resource name (`pages/{page_id}`).
func (c *Client) GetPage(ctx context.Context, name string) (*Page, error) {
	var page Page
	err := c.client.Do(ctx, http.MethodGet, basePath+name, c.headers(), nil, nil, &page)
	if err != nil {
		return nil, err
	}
	return &page, nil
}

// newRequestID returns a random UUIDv4, the format the API requires of a request ID.
func newRequestID() (string, error) {
	var b [16]byte
	if _, err := rand.Read(b[:]); err != nil {
		return "", err
	}
	b[6] = b[6]&0x0f | 0x40 // version 4
	b[8] = b[8]&0x3f | 0x80 // RFC 4122 variant
	return fmt.Sprintf("%x-%x-%x-%x-%x", b[0:4], b[4:6], b[6:8], b[8:10], b[10:16]), nil
}

// CreatePage creates a page. `domain` and `display_name` are required.
//
// The request carries a request ID, which makes it idempotent: when it is retried after
// its response was lost, the API returns the page the first attempt created instead of
// creating a second one. Draft titles are not unique, so a duplicate could not be told
// apart afterwards.
func (c *Client) CreatePage(ctx context.Context, page Page) (*Page, error) {
	requestID, err := newRequestID()
	if err != nil {
		return nil, err
	}
	query := map[string]any{"request_id": requestID}

	var created Page
	err = c.client.Do(ctx, http.MethodPost, basePath+"pages", c.headers(), query, page, &created)
	if err != nil {
		return nil, err
	}
	return &created, nil
}

// UpdatePage writes the fields of page named by updateMask to the page identified by
// page.Name. The API rejects the `*` wildcard, so every update names its fields: any of
// display_name, domain, body, synonyms, description, state and owner_ids. A field in the
// mask is always sent, so that an empty value clears it.
func (c *Client) UpdatePage(ctx context.Context, page Page, updateMask []string) (*Page, error) {
	body := make(map[string]any, len(updateMask))
	for _, field := range updateMask {
		switch field {
		case "display_name":
			body[field] = page.DisplayName
		case "domain":
			body[field] = page.Domain
		case "body":
			body[field] = page.Body
		case "synonyms":
			// A nil slice marshals to null, which does not clear the list.
			body[field] = append([]string{}, page.Synonyms...)
		case "description":
			body[field] = page.Description
		case "state":
			body[field] = page.State
		case "owner_ids":
			body[field] = append([]string{}, page.OwnerIds...)
		}
	}

	query := map[string]any{"update_mask": strings.Join(updateMask, ",")}

	var updated Page
	err := c.client.Do(ctx, http.MethodPatch, basePath+page.Name, c.headers(), query, body, &updated)
	if err != nil {
		return nil, err
	}
	return &updated, nil
}

// DeletePage deletes the page with the given resource name (`pages/{page_id}`).
func (c *Client) DeletePage(ctx context.Context, name string) error {
	return c.client.Do(ctx, http.MethodDelete, basePath+name, c.headers(), nil, nil, nil)
}
