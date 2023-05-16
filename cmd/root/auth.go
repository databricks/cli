package root

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/databricks/cli/bundle"
	"github.com/databricks/cli/libs/cmdio"
	"github.com/databricks/databricks-sdk-go"
	"github.com/databricks/databricks-sdk-go/config"
	"github.com/databricks/databricks-sdk-go/service/iam"
	"github.com/manifoldco/promptui"
	"github.com/spf13/cobra"
	"gopkg.in/ini.v1"
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
		profiles, err := loadProfiles()
		if err != nil {
			return err
		}
		var items []profile
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

type profile struct {
	Name      string
	Host      string
	AccountID string
}

func (p profile) Cloud() string {
	if strings.Contains(p.Host, ".azuredatabricks.net") {
		return "Azure"
	}
	if strings.Contains(p.Host, "gcp.databricks.com") {
		return "GCP"
	}
	return "AWS"
}

func loadProfiles() (profiles []profile, err error) {
	homedir, err := os.UserHomeDir()
	if err != nil {
		return nil, fmt.Errorf("cannot find homedir: %w", err)
	}
	file := filepath.Join(homedir, ".databrickscfg")
	iniFile, err := ini.Load(file)
	if err != nil {
		return
	}
	for _, v := range iniFile.Sections() {
		all := v.KeysHash()
		host, ok := all["host"]
		if !ok {
			// invalid profile
			continue
		}
		profiles = append(profiles, profile{
			Name:      v.Name(),
			Host:      host,
			AccountID: all["account_id"],
		})
	}
	return profiles, nil
}

func askForWorkspaceProfile() (string, error) {
	profiles, err := loadProfiles()
	if err != nil {
		return "", err
	}
	var items []profile
	for _, v := range profiles {
		if v.AccountID != "" {
			continue
		}
		items = append(items, v)
	}
	label := "~/.databrickscfg profile"
	i, _, err := (&promptui.Select{
		Label: label,
		Items: items,
		Templates: &promptui.SelectTemplates{
			Active:   `{{.Name | bold}} ({{.Host|faint}})`,
			Inactive: `{{.Name}}`,
			Selected: fmt.Sprintf(`{{ "%s" | faint }}: {{ .Name | bold }}`, label),
		},
		Stdin: os.Stdin,
	}).Run()
	if err != nil {
		return "", err
	}
	return items[i].Name, nil
}

func askForAccountProfile() (string, error) {
	profiles, err := loadProfiles()
	if err != nil {
		return "", err
	}
	var items []profile
	for _, v := range profiles {
		if v.AccountID == "" {
			continue
		}
		items = append(items, v)
	}
	if len(items) == 1 {
		return items[0].Name, nil
	}
	label := "~/.databrickscfg profile"
	i, _, err := (&promptui.Select{
		Label: label,
		Items: items,
		Templates: &promptui.SelectTemplates{
			Active:   `{{.Name | bold}} ({{.AccountID|faint}} {{.Cloud|faint}})`,
			Inactive: `{{.Name}}`,
			Selected: fmt.Sprintf(`{{ "%s" | faint }}: {{ .Name | bold }}`, label),
		},
		Stdin: os.Stdin,
	}).Run()
	if err != nil {
		return "", err
	}
	return items[i].Name, nil
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
