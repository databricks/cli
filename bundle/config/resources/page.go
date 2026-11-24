package resources

import (
	"context"
	"net/url"

	"github.com/databricks/cli/libs/knowledge"
	"github.com/databricks/cli/libs/log"
	"github.com/databricks/cli/libs/workspaceurls"
	"github.com/databricks/databricks-sdk-go"
	"github.com/databricks/databricks-sdk-go/apierr"
	"github.com/databricks/databricks-sdk-go/marshal"
)

type Page struct {
	BaseResource
	ID string `json:"id,omitempty" bundle:"readonly"`
	knowledge.Page

	// FilePath points to a local Markdown file with the body of the page. Its contents are
	// inlined into body during deployment. It is not part of knowledge.Page because it is
	// not stored in the resource state.
	FilePath string `json:"file_path,omitempty"`
}

func (p *Page) UnmarshalJSON(b []byte) error {
	return marshal.Unmarshal(b, p)
}

func (p Page) MarshalJSON() ([]byte, error) {
	return marshal.Marshal(p)
}

func (*Page) Exists(ctx context.Context, w *databricks.WorkspaceClient, id string) (bool, error) {
	client, err := knowledge.NewClient(w)
	if err != nil {
		return false, err
	}
	_, err = client.GetPage(ctx, knowledge.PageResourceNamePrefix+id)
	if err != nil {
		if apierr.IsMissing(err) {
			log.Debugf(ctx, "page %s does not exist", id)
			return false, nil
		}
		return false, err
	}
	return true, nil
}

func (*Page) ResourceDescription() ResourceDescription {
	return ResourceDescription{
		SingularName:  "page",
		PluralName:    "pages",
		SingularTitle: "Page",
		PluralTitle:   "Pages",
	}
}

func (p *Page) InitializeURL(baseURL url.URL) {
	if p.ID == "" {
		return
	}
	p.URL = workspaceurls.ResourceURL(baseURL, "pages", p.ID)
}

func (p *Page) GetName() string {
	return p.DisplayName
}
