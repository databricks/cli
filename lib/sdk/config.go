package sdk

import (
	"context"
	"errors"
	"os"

	"github.com/databricks/bricks/lib/ui"
	"github.com/databricks/bricks/project"
	"github.com/databricks/databricks-sdk-go"
	"github.com/databricks/databricks-sdk-go/config"
	"github.com/databricks/databricks-sdk-go/service/scim"
	"github.com/spf13/cobra"
)

// Placeholders to use as unique keys in context.Context.
var workspaceClient int
var accountClient int
var currentUser int

func PreAccountClient(cmd *cobra.Command, args []string) error {
	cfg := &config.Config{}

	// command-line flag can specify the profile in use
	profileFlag := cmd.Flag("profile")
	if profileFlag != nil {
		cfg.Profile = profileFlag.Value.String()
	}

	if cfg.Profile == "" {
		// account-level CLI was not really done before, so here are the assumptions:
		// 1. only admins will have account configured
		// 2. 99% of admins will have access to just one account
		// hence, we don't need to create a special "DEFAULT_ACCOUNT" profile yet
		profiles, err := loadProfiles()
		if err != nil {
			return err
		}
		var items []Profile
		for _, v := range profiles {
			if v.AccountID == "" {
				continue
			}
			items = append(items, v)
		}
		if len(items) == 1 {
			cfg.Profile = items[0].Name
		}
	}

TRY_AUTH: // or try picking a config profile dynamically
	a, err := databricks.NewAccountClient((*databricks.Config)(cfg))
	if ui.Interactive && errors.Is(err, databricks.ErrNotAccountClient) {
		profile, err := askForAccountProfile()
		if err != nil {
			return err
		}
		cfg = &config.Config{Profile: profile}
		goto TRY_AUTH
	}
	if err != nil {
		return err
	}

	cmd.SetContext(context.WithValue(cmd.Context(), &accountClient, a))
	return nil
}

func PreWorkspaceClient(cmd *cobra.Command, args []string) error {
	cfg := &config.Config{}

	// command-line flag takes precedence
	profileFlag := cmd.Flag("profile")
	if profileFlag != nil {
		cfg.Profile = profileFlag.Value.String()
	}

	// otherwise we load profile from databricks.yml
	if cfg.Profile == "" && project.IsDatabricksProject() {
		err := project.Configure(cmd, args)
		if err != nil {
			return err
		}
		prjConfig := project.Get(cmd.Context()).WorkspacesClient().Config
		cfg.Profile = prjConfig.Profile
	}

TRY_AUTH: // or try picking a config profile dynamically
	ctx := cmd.Context()
	w, err := databricks.NewWorkspaceClient((*databricks.Config)(cfg))
	if err != nil {
		return err
	}
	// get current user identity also to verify validity of configuration
	me, err := w.CurrentUser.Me(ctx)
	if ui.Interactive && errors.Is(err, config.ErrCannotConfigureAuth) {
		profile, err := askForWorkspaceProfile()
		if errors.Is(err, os.ErrNotExist) {
			root := cmd.Root()
			root.SetArgs([]string{"configure", "--token"})
			err = root.Execute()
			if err != nil {
				return err
			}
			goto TRY_AUTH
		}
		if err != nil {
			return err
		}
		cfg = &config.Config{Profile: profile}
		goto TRY_AUTH
	}
	if err != nil {
		return err
	}
	ctx = context.WithValue(ctx, &currentUser, me)
	ctx = context.WithValue(ctx, &workspaceClient, w)
	cmd.SetContext(ctx)
	return nil
}

func WorkspaceClient(ctx context.Context) *databricks.WorkspaceClient {
	return ctx.Value(&workspaceClient).(*databricks.WorkspaceClient)
}

func AccountClient(ctx context.Context) *databricks.AccountClient {
	return ctx.Value(&accountClient).(*databricks.AccountClient)
}

func Me(ctx context.Context) *scim.User {
	return ctx.Value(&currentUser).(*scim.User)
}
