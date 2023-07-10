package mutator

import (
	"context"
	"strings"

	"github.com/databricks/cli/bundle"
	"github.com/databricks/cli/bundle/config"
)

type populateCurrentUser struct{}

// PopulateCurrentUser sets the `current_user` property on the workspace.
func PopulateCurrentUser() bundle.Mutator {
	return &populateCurrentUser{}
}

func (m *populateCurrentUser) Name() string {
	return "PopulateCurrentUser"
}

func (m *populateCurrentUser) Apply(ctx context.Context, b *bundle.Bundle) error {
	w := b.WorkspaceClient()
	me, err := w.CurrentUser.Me(ctx)
	if err != nil {
		return err
	}

	b.Config.Workspace.CurrentUser = &config.User{
		ShortName: strings.Split(me.UserName, "@")[0],
		User:      me,
	}
	return nil
}
