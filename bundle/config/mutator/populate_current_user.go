package mutator

import (
	"context"
	"strings"
	"unicode"

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
	if b.Config.Workspace.CurrentUser != nil {
		return nil
	}

	w := b.WorkspaceClient()
	me, err := w.CurrentUser.Me(ctx)
	if err != nil {
		return err
	}

	b.Config.Workspace.CurrentUser = &config.User{
		ShortName: getShortUserName(me.UserName),
		User:      me,
	}
	return nil
}

func replaceNonAlphanumeric(r rune) rune {
	if unicode.IsLetter(r) || unicode.IsDigit(r) {
		return r
	}
	return '_'
}

// Get a short-form username, based on the user's primary email address.
// We leave the full range of unicode letters in tact, but remove all "special" characters,
// including dots, which are not supported in e.g. experiment names.
func getShortUserName(emailAddress string) string {
	local, _, _ := strings.Cut(emailAddress, "@")
	return strings.Map(replaceNonAlphanumeric, local)
}
