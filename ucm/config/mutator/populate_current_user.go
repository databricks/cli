package mutator

import (
	"context"

	"github.com/databricks/cli/libs/diag"
	"github.com/databricks/cli/libs/iamutil"
	"github.com/databricks/cli/ucm"
	"github.com/databricks/cli/ucm/config"
)

type populateCurrentUser struct{}

// PopulateCurrentUser sets Workspace.CurrentUser by resolving a workspace
// client from Workspace.{Host,Profile} and calling CurrentUser.Me().
//
// Idempotent: skips the network call if CurrentUser is already set — so tests
// can pre-seed a fake user and short-circuit the real resolution.
//
// Mirrors bundle/config/mutator.PopulateCurrentUser. Errors surface as
// diagnostics and abort the mutator chain; parity with DAB: `ucm validate`
// requires reachable workspace creds.
func PopulateCurrentUser() ucm.Mutator { return &populateCurrentUser{} }

func (m *populateCurrentUser) Name() string { return "PopulateCurrentUser" }

func (m *populateCurrentUser) Apply(ctx context.Context, u *ucm.Ucm) diag.Diagnostics {
	if u.CurrentUser != nil {
		return nil
	}

	w, err := u.WorkspaceClientE()
	if err != nil {
		return diag.FromErr(err)
	}

	me, err := w.CurrentUser.Me(ctx)
	if err != nil {
		return diag.FromErr(err)
	}

	u.CurrentUser = &config.User{
		ShortName:          iamutil.GetShortUserName(me),
		DomainFriendlyName: iamutil.GetShortUserDomainFriendlyName(me),
		User:               me,
	}
	return nil
}
