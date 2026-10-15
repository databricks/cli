package resources

import (
	"context"
	"net/url"

	"github.com/databricks/cli/libs/sdkclients"
	"github.com/databricks/cli/libs/sdktypes/postgres/v1"
	sdkpostgres "github.com/databricks/sdk-go/postgres/v1"

	"github.com/databricks/cli/libs/log"
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
	PurgeOnDelete *bool `json:"purge_on_delete,omitempty"`
}

type PostgresBranch struct {
	BaseResource
	ID string `json:"id,omitempty" bundle:"readonly"`
	PostgresBranchConfig
}

func (b *PostgresBranch) Exists(ctx context.Context, w *sdkclients.Clients, name string) (bool, error) {
	_, err := w.Postgres.GetBranch(ctx, sdkpostgres.GetBranchRequest{Name: new(name)})
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
	// Branches have no name field of their own; the resource name is the ID
	// ("projects/{project_id}/branches/{branch_id}").
	return b.ID
}

func (b *PostgresBranch) GetURL() (string, bool) {
	// The IDs in the API do not (yet) map to IDs in the web UI.
	return "", false
}

func (b *PostgresBranch) InitializeURL(_ url.URL) {
	// The IDs in the API do not (yet) map to IDs in the web UI.
}
