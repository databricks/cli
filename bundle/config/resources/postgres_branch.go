package resources

import (
	"context"
	"net/url"
	"strings"

	"github.com/databricks/cli/libs/log"
	"github.com/databricks/databricks-sdk-go"
	"github.com/databricks/databricks-sdk-go/service/postgres"
)

type PostgresBranch struct {
	BaseResource
	postgres.BranchSpec

	// Parent is the project containing this branch. Format: "projects/{project_id}"
	Parent string `json:"parent,omitempty"`

	// BranchId is the user-specified ID for the branch (becomes part of the hierarchical name).
	// This is specified during creation and becomes part of Name: "projects/{project_id}/branches/{branch_id}"
	BranchId string `json:"branch_id,omitempty"`

	// Name is the hierarchical resource name (output-only). Format: "projects/{project_id}/branches/{branch_id}"
	Name string `json:"name,omitempty" bundle:"readonly"`
}

func (b *PostgresBranch) Exists(ctx context.Context, w *databricks.WorkspaceClient, name string) (bool, error) {
	_, err := w.Postgres.GetBranch(ctx, postgres.GetBranchRequest{Name: name})
	if err != nil {
		log.Debugf(ctx, "postgres branch %s does not exist", name)
		return false, err
	}
	return true, nil
}

func (b *PostgresBranch) ResourceDescription() ResourceDescription {
	return ResourceDescription{
		SingularName:  "postgres_branch",
		PluralName:    "postgres_branches",
		SingularTitle: "Postgres branch",
		PluralTitle:   "Postgres branches",
	}
}

func (b *PostgresBranch) GetName() string {
	// Branches don't have a user-visible name field
	return ""
}

func (b *PostgresBranch) GetURL() string {
	return b.URL
}

func (b *PostgresBranch) InitializeURL(baseURL url.URL) {
	if b.ModifiedStatus == ModifiedStatusCreated {
		return
	}
	if b.Name == "" {
		return
	}
	// Parse: projects/{project_id}/branches/{branch_id}
	parts := strings.Split(b.Name, "/")
	if len(parts) >= 4 {
		projectId := parts[1]
		branchId := parts[3]
		baseURL.Path = "postgres/projects/" + projectId + "/branches/" + branchId
		b.URL = baseURL.String()
	}
}
