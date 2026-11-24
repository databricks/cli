package resources

import (
	"context"
	"net/url"

	"github.com/databricks/cli/libs/log"
	"github.com/databricks/cli/libs/workspaceurls"
	"github.com/databricks/databricks-sdk-go"
	"github.com/databricks/databricks-sdk-go/apierr"
	"github.com/databricks/databricks-sdk-go/marshal"
	"github.com/databricks/databricks-sdk-go/service/domains"
)

// DomainResourceNamePrefix is the prefix of a domain's resource name. The API identifies a
// domain by `domains/{domain_id}`; the bundle uses the bare domain ID as the resource ID.
const DomainResourceNamePrefix = "domains/"

type Domain struct {
	BaseResource
	ID string `json:"id,omitempty" bundle:"readonly"`
	domains.Domain
}

func (d *Domain) UnmarshalJSON(b []byte) error {
	return marshal.Unmarshal(b, d)
}

func (d Domain) MarshalJSON() ([]byte, error) {
	return marshal.Marshal(d)
}

func (*Domain) Exists(ctx context.Context, w *databricks.WorkspaceClient, id string) (bool, error) {
	_, err := w.Domains.GetDomain(ctx, domains.GetDomainRequest{
		Name: DomainResourceNamePrefix + id,
	})
	if err != nil {
		if apierr.IsMissing(err) {
			log.Debugf(ctx, "domain %s does not exist", id)
			return false, nil
		}
		return false, err
	}
	return true, nil
}

func (*Domain) ResourceDescription() ResourceDescription {
	return ResourceDescription{
		SingularName:  "domain",
		PluralName:    "domains",
		SingularTitle: "Domain",
		PluralTitle:   "Domains",
	}
}

func (d *Domain) InitializeURL(baseURL url.URL) {
	if d.ID == "" {
		return
	}
	d.URL = workspaceurls.ResourceURL(baseURL, "domains", d.ID)
}

func (d *Domain) GetName() string {
	return d.TagKey
}
