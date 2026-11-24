package dresources

import (
	"context"
	"strings"

	"github.com/databricks/cli/bundle/config/resources"
	"github.com/databricks/cli/libs/knowledge"
	"github.com/databricks/cli/libs/structs/structpath"
	"github.com/databricks/databricks-sdk-go"
)

// Unity Catalog page.
// API: https://docs.databricks.com/api/workspace/knowledge
type ResourcePage struct {
	client *knowledge.Client
}

func (*ResourcePage) New(client *databricks.WorkspaceClient) *ResourcePage {
	// The client is nil when only the types of the resource are inspected (refschema).
	if client == nil {
		return &ResourcePage{client: nil}
	}

	pages, err := knowledge.NewClient(client)
	if err != nil {
		panic(err)
	}
	return &ResourcePage{client: pages}
}

// PrepareState trims the title, as the API does. Otherwise a padded title would differ
// from what the API returns and show as changed on every plan.
func (*ResourcePage) PrepareState(input *resources.Page) *knowledge.Page {
	state := input.Page
	state.DisplayName = strings.TrimSpace(state.DisplayName)
	return &state
}

func (r *ResourcePage) DoRead(ctx context.Context, id string) (*knowledge.Page, error) {
	return r.client.GetPage(ctx, knowledge.PageResourceNamePrefix+id)
}

func (r *ResourcePage) DoCreate(ctx context.Context, config *knowledge.Page) (string, *knowledge.Page, error) {
	page, err := r.client.CreatePage(ctx, *config)
	if err != nil {
		return "", nil, err
	}
	return strings.TrimPrefix(page.Name, knowledge.PageResourceNamePrefix), page, nil
}

var (
	pageDomainPath   = structpath.MustParsePath("domain")
	pageStatePath    = structpath.MustParsePath("state")
	pageOwnerIdsPath = structpath.MustParsePath("owner_ids")
)

// pageUpdateMask returns the fields to write on update. The API has no wildcard, so the
// content fields are always written. Three fields are an action rather than content and
// are written only when the plan changes them:
//   - state: writing PUBLISHED publishes a new version of the page, also when the page is
//     already published.
//   - domain: moving a page to another domain can be disabled for the account, and needs
//     access to the destination.
//   - owner_ids: changing the owner needs more access than editing the page.
func pageUpdateMask(entry *PlanEntry) []string {
	mask := []string{"display_name", "body", "synonyms", "description"}
	if entry.Changes.HasChange(pageDomainPath) {
		mask = append(mask, "domain")
	}
	if entry.Changes.HasChange(pageStatePath) {
		mask = append(mask, "state")
	}
	if entry.Changes.HasChange(pageOwnerIdsPath) {
		mask = append(mask, "owner_ids")
	}
	return mask
}

func (r *ResourcePage) DoUpdate(ctx context.Context, id string, config *knowledge.Page, entry *PlanEntry) (*knowledge.Page, error) {
	page := *config
	page.Name = knowledge.PageResourceNamePrefix + id
	return r.client.UpdatePage(ctx, page, pageUpdateMask(entry))
}

func (r *ResourcePage) DoDelete(ctx context.Context, id string, _ *knowledge.Page) error {
	return r.client.DeletePage(ctx, knowledge.PageResourceNamePrefix+id)
}
