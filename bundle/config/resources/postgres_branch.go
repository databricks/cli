package resources

import (
	"context"
	"net/url"

	"github.com/databricks/cli/libs/log"
	"github.com/databricks/cli/libs/workspaceurls"
	"github.com/databricks/databricks-sdk-go"
	"github.com/databricks/databricks-sdk-go/marshal"
	"github.com/databricks/databricks-sdk-go/service/postgres"
)

type PostgresBranchConfig struct {
	postgres.BranchSpec

	// BranchId is the user-specified ID for the branch (becomes part of the hierarchical name).
	// This is specified during creation and becomes part of Name: "projects/{project_id}/branches/{branch_id}"
	BranchId string `json:"branch_id"`

	// Parent is the project containing this branch. Format: "projects/{project_id}"
	Parent string `json:"parent"`

	// ReplaceExisting, when true, takes over an existing branch with the same ID
	// instead of returning ALREADY_EXISTS. Used to manage the implicitly-created
	// production branch of a new project. Input-only: not returned by the GET API.
	ReplaceExisting bool `json:"replace_existing,omitempty"`

	// PurgeOnDelete, when true, hard-deletes the branch on destroy (Purge=true on
	// DeleteBranch). When false or unset, the backend performs a soft delete that
	// can be undone within the branch's retention window. Input-only: not
	// returned by the GET API.
	PurgeOnDelete bool `json:"purge_on_delete,omitempty"`

	// ForceSendFields shadows the embedded BranchSpec.ForceSendFields so the
	// SDK's marshal package tracks zero-value top-level fields (branch_id,
	// parent, replace_existing, purge_on_delete) here instead of polluting
	// BranchSpec.ForceSendFields with names that don't exist in that struct.
	ForceSendFields []string `json:"-" url:"-"`
}

func (c *PostgresBranchConfig) UnmarshalJSON(b []byte) error {
	return marshal.Unmarshal(b, c)
}

func (c *PostgresBranchConfig) MarshalJSON() ([]byte, error) {
	return marshal.Marshal(c)
}

type PostgresBranch struct {
	BaseResource
	PostgresBranchConfig
}

func (b *PostgresBranch) UnmarshalJSON(data []byte) error {
	return marshal.Unmarshal(data, b)
}

func (b PostgresBranch) MarshalJSON() ([]byte, error) {
	return marshal.Marshal(b)
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

// InitializeURL points at the branch's Lakebase page. ID is the branch's
// hierarchical name "projects/{project_id}/branches/{branch_id}".
func (b *PostgresBranch) InitializeURL(baseURL url.URL) {
	if b.ID == "" {
		return
	}
	b.URL = workspaceurls.ResourceURL(baseURL, "postgres_branches", b.ID)
}
