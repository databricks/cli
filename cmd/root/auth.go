package root

import (
	"context"
	"errors"
	"fmt"
	"os"

	"github.com/databricks/cli/bundle"
	"github.com/databricks/cli/libs/cmdio"
	"github.com/databricks/cli/libs/databrickscfg"
	"github.com/databricks/databricks-sdk-go"
	"github.com/databricks/databricks-sdk-go/config"
	"github.com/databricks/databricks-sdk-go/service/iam"
	"github.com/manifoldco/promptui"
	"github.com/spf13/cobra"
)

// Placeholders to use as unique keys in context.Context.
var workspaceClient int
var accountClient int
var currentUser int

func init() {
	RootCmd.PersistentFlags().StringP("profile", "p", "", "~/.databrickscfg profile")
}

func MustAccountClient(cmd *cobra.Command, args []string) error {
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
		_, profiles, err := databrickscfg.LoadProfiles(
			databrickscfg.DefaultPath,
			databrickscfg.MatchAccountProfiles,
		)
		if err != nil {
			return err
		}
		if len(profiles) == 1 {
			cfg.Profile = profiles[0].Name
		}
	}

TRY_AUTH: // or try picking a config profile dynamically
	a, err := databricks.NewAccountClient((*databricks.Config)(cfg))
	if cmdio.IsInteractive(cmd.Context()) && errors.Is(err, databricks.ErrNotAccountClient) {
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

func MustWorkspaceClient(cmd *cobra.Command, args []string) error {
	cfg := &config.Config{}

	// command-line flag takes precedence over environment variable
	profileFlag := cmd.Flag("profile")
	if profileFlag != nil {
		cfg.Profile = profileFlag.Value.String()
	}

	// try configuring a bundle
	err := TryConfigureBundle(cmd, args)
	if err != nil {
		return err
	}

	// and load the config from there
	currentBundle := bundle.GetOrNil(cmd.Context())
	if currentBundle != nil {
		cfg = currentBundle.WorkspaceClient().Config
	}

TRY_AUTH: // or try picking a config profile dynamically
	ctx := cmd.Context()
	w, err := databricks.NewWorkspaceClient((*databricks.Config)(cfg))
	if err != nil {
		return err
	}
	// get current user identity also to verify validity of configuration
	me, err := w.CurrentUser.Me(ctx)
	if cmdio.IsInteractive(ctx) && errors.Is(err, config.ErrCannotConfigureAuth) {
		profile, err := askForWorkspaceProfile()
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

func transformLoadError(path string, err error) error {
	if os.IsNotExist(err) {
		return fmt.Errorf("no configuration file found at %s; please create one first", path)
	}
	return err
}

func askForWorkspaceProfile() (string, error) {
	path := databrickscfg.DefaultPath
	file, profiles, err := databrickscfg.LoadProfiles(path, databrickscfg.MatchWorkspaceProfiles)
	if err != nil {
		return "", transformLoadError(path, err)
	}
	switch len(profiles) {
	case 0:
		return "", fmt.Errorf("%s does not contain workspace profiles; please create one first", path)
	case 1:
		return profiles[0].Name, nil
	}
	i, _, err := (&promptui.Select{
		Label:             fmt.Sprintf("Workspace profiles defined in %s", file),
		Items:             profiles,
		Searcher:          profiles.SearchCaseInsensitive,
		StartInSearchMode: true,
		Templates: &promptui.SelectTemplates{
			Label:    "{{ . | faint }}",
			Active:   `{{.Name | bold}} ({{.Host|faint}})`,
			Inactive: `{{.Name}}`,
			Selected: `{{ "Using workspace profile" | faint }}: {{ .Name | bold }}`,
		},
		Stdin:  os.Stdin,
		Stdout: os.Stderr,
	}).Run()
	if err != nil {
		return "", err
	}
	return profiles[i].Name, nil
}

func askForAccountProfile() (string, error) {
	path := databrickscfg.DefaultPath
	file, profiles, err := databrickscfg.LoadProfiles(path, databrickscfg.MatchAccountProfiles)
	if err != nil {
		return "", transformLoadError(path, err)
	}
	switch len(profiles) {
	case 0:
		return "", fmt.Errorf("%s does not contain account profiles; please create one first", path)
	case 1:
		return profiles[0].Name, nil
	}
	i, _, err := (&promptui.Select{
		Label:             fmt.Sprintf("Account profiles defined in %s", file),
		Items:             profiles,
		Searcher:          profiles.SearchCaseInsensitive,
		StartInSearchMode: true,
		Templates: &promptui.SelectTemplates{
			Label:    "{{ . | faint }}",
			Active:   `{{.Name | bold}} ({{.AccountID|faint}} {{.Cloud|faint}})`,
			Inactive: `{{.Name}}`,
			Selected: `{{ "Using account profile" | faint }}: {{ .Name | bold }}`,
		},
		Stdin:  os.Stdin,
		Stdout: os.Stderr,
	}).Run()
	if err != nil {
		return "", err
	}
	return profiles[i].Name, nil
}

func SetWorkspaceClient(ctx context.Context, w *databricks.WorkspaceClient) context.Context {
	return context.WithValue(ctx, &workspaceClient, w)
}

func WorkspaceClient(ctx context.Context) *databricks.WorkspaceClient {
	w, ok := ctx.Value(&workspaceClient).(*databricks.WorkspaceClient)
	if !ok {
		panic("cannot get *databricks.WorkspaceClient. Please report it as a bug")
	}
	return w
}

func AccountClient(ctx context.Context) *databricks.AccountClient {
	a, ok := ctx.Value(&accountClient).(*databricks.AccountClient)
	if !ok {
		panic("cannot get *databricks.AccountClient. Please report it as a bug")
	}
	return a
}

func Me(ctx context.Context) *iam.User {
	me, ok := ctx.Value(&currentUser).(*iam.User)
	if !ok {
		panic("cannot get current user. Please report it as a bug")
	}
	return me
}
