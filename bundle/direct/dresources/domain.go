package dresources

import (
	"context"
	"slices"

	"github.com/databricks/cli/bundle/config/resources"
	"github.com/databricks/databricks-sdk-go"
	"github.com/databricks/databricks-sdk-go/common/types/fieldmask"
	"github.com/databricks/databricks-sdk-go/service/domains"
)

// Discover domain.
// API: https://docs.databricks.com/api/workspace/domains
// Terraform: databricks_domain
type ResourceDomain struct {
	client *databricks.WorkspaceClient
}

func (*ResourceDomain) New(client *databricks.WorkspaceClient) *ResourceDomain {
	return &ResourceDomain{client: client}
}

func (*ResourceDomain) PrepareState(input *resources.Domain) *domains.Domain {
	return &input.Domain
}

func (r *ResourceDomain) DoRead(ctx context.Context, id string) (*domains.Domain, error) {
	return r.client.Domains.GetDomain(ctx, domains.GetDomainRequest{
		Name: resources.DomainResourceNamePrefix + id,
	})
}

func (r *ResourceDomain) DoCreate(ctx context.Context, config *domains.Domain) (string, *domains.Domain, error) {
	domain, err := r.client.Domains.CreateDomain(ctx, domains.CreateDomainRequest{
		Domain: *config,
		// The API generates the ID of the domain.
		DomainId:        "",
		ForceSendFields: nil,
	})
	if err != nil {
		return "", nil, err
	}
	return domain.DomainId, domain, nil
}

// domainUpdateMask returns the fields to write on update. The API rejects the `*` wildcard
// because it expands to the immutable fields. Those are left out:
//   - tag_key and parent_domain_id are immutable; a change recreates the domain.
//   - domain_id, name, effective_draft, create_time and update_time are output only.
//
// draft is written only when the config sets it. Unset means the API decides, and writing
// the zero value would publish a domain that is a draft by an account default or was set to
// draft outside of the bundle. The API returns the resolved value as effective_draft and
// never returns draft, so draft itself is input only.
func domainUpdateMask(config *domains.Domain) []string {
	mask := []string{
		"subtitle",
		"description",
		"icon",
		"technical_owner_ids",
		"business_owner_ids",
	}
	if config.Draft || slices.Contains(config.ForceSendFields, "Draft") {
		mask = append(mask, "draft")
	}
	return mask
}

func (r *ResourceDomain) DoUpdate(ctx context.Context, id string, config *domains.Domain, _ *PlanEntry) (*domains.Domain, error) {
	return r.client.Domains.UpdateDomain(ctx, domains.UpdateDomainRequest{
		Domain:     *config,
		Name:       resources.DomainResourceNamePrefix + id,
		UpdateMask: *fieldmask.New(domainUpdateMask(config)),
	})
}

func (r *ResourceDomain) DoDelete(ctx context.Context, id string, _ *domains.Domain) error {
	return r.client.Domains.DeleteDomain(ctx, domains.DeleteDomainRequest{
		Name: resources.DomainResourceNamePrefix + id,
		// Without force the API refuses to delete a domain that still has pages. The pages
		// of the bundle are deleted before their domain; pages that are not part of the
		// bundle are not the bundle's to delete.
		Force:           false,
		ForceSendFields: nil,
	})
}
