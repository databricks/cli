package mutator

import (
	"context"
	"strings"

	"github.com/databricks/cli/bundle"
	"github.com/databricks/cli/bundle/config"
	"github.com/databricks/cli/libs/tags"
	"github.com/databricks/cli/libs/textutil"
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
	if b.Config.Workspace.CurrentUser != nil {
		return nil
	}

	w := b.WorkspaceClient()
	me, err := w.CurrentUser.Me(ctx)
	if err != nil {
		return err
	}

	b.Config.Workspace.CurrentUser = &config.User{
		ShortName: GetShortUserName(me.UserName),
		User:      me,
	}

	// Configure tagging object now that we know we have a valid client.
	b.Tagging = tags.ForCloud(w.Config)

	return nil
}

// Get a short-form username, based on the user's primary email address.
// We leave the full range of unicode letters in tact, but remove all "special" characters,
// including dots, which are not supported in e.g. experiment names.
func GetShortUserName(userName string) string {
	local, _, _ := strings.Cut(emailAddress, "@")
	return textutil.NormalizeString(local)
}
