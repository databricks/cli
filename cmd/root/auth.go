package root

import (
	"context"
	"errors"
	"fmt"
	"net/http"

	"github.com/databricks/cli/libs/cmdio"
	"github.com/databricks/cli/libs/databrickscfg/profile"
	"github.com/databricks/databricks-sdk-go"
	"github.com/databricks/databricks-sdk-go/config"
	"github.com/manifoldco/promptui"
	"github.com/spf13/cobra"
)

// Placeholders to use as unique keys in context.Context.
var workspaceClient int
var accountClient int
var configUsed int

type ErrNoWorkspaceProfiles struct {
	path string
}

func (e ErrNoWorkspaceProfiles) Error() string {
	return fmt.Sprintf("%s does not contain workspace profiles; please create one by running 'databricks configure'", e.path)
}

type ErrNoAccountProfiles struct {
	path string
}

func (e ErrNoAccountProfiles) Error() string {
	return fmt.Sprintf("%s does not contain account profiles", e.path)
}

func initProfileFlag(cmd *cobra.Command) {
	cmd.PersistentFlags().StringP("profile", "p", "", "~/.databrickscfg profile")
	cmd.RegisterFlagCompletionFunc("profile", profile.ProfileCompletion)
}

func profileFlagValue(cmd *cobra.Command) (string, bool) {
	profileFlag := cmd.Flag("profile")
	if profileFlag == nil {
		return "", false
	}
	value := profileFlag.Value.String()
	return value, value != ""
}

// Helper function to create an account client or prompt once if the given configuration is not valid.
func accountClientOrPrompt(ctx context.Context, cfg *config.Config, allowPrompt bool) (*databricks.AccountClient, error) {
	a, err := databricks.NewAccountClient((*databricks.Config)(cfg))
	if err == nil {
		err = a.Config.Authenticate(emptyHttpRequest(ctx))
	}

	prompt := false
	if allowPrompt && err != nil && cmdio.IsPromptSupported(ctx) {
		// Prompt to select a profile if the current configuration is not an account client.
		prompt = prompt || errors.Is(err, databricks.ErrNotAccountClient)
		// Prompt to select a profile if the current configuration doesn't resolve to a credential provider.
		prompt = prompt || errors.Is(err, config.ErrCannotConfigureAuth)
	}

	if !prompt {
		// If we are not prompting, we can return early.
		return a, err
	}

	// Try picking a profile dynamically if the current configuration is not valid.
	profile, err := AskForAccountProfile(ctx)
	if err != nil {
		return nil, err
	}
	a, err = databricks.NewAccountClient(&databricks.Config{Profile: profile})
	if err == nil {
		err = a.Config.Authenticate(emptyHttpRequest(ctx))
		if err != nil {
			return nil, err
		}
	}
	return a, err
}

func MustAnyClient(cmd *cobra.Command, args []string) (bool, error) {
	// Try to create a workspace client
	werr := MustWorkspaceClient(cmd, args)
	if werr == nil {
		return false, nil
	}

	// If the error is other than "not a workspace client error" or "no workspace profiles",
	// return it because configuration is for workspace client
	// and we don't want to try to create an account client.
	if !errors.Is(werr, databricks.ErrNotWorkspaceClient) && !errors.As(werr, &ErrNoWorkspaceProfiles{}) {
		return false, werr
	}

	// Otherwise, the config used is account client one, so try to create an account client
	aerr := MustAccountClient(cmd, args)
	if errors.As(aerr, &ErrNoAccountProfiles{}) {
		return false, aerr
	}

	return true, aerr
}

func MustAccountClient(cmd *cobra.Command, args []string) error {
	cfg := &config.Config{}

	// The command-line profile flag takes precedence over DATABRICKS_CONFIG_PROFILE.
	pr, hasProfileFlag := profileFlagValue(cmd)
	if hasProfileFlag {
		cfg.Profile = pr
	}

	ctx := cmd.Context()
	ctx = context.WithValue(ctx, &configUsed, cfg)
	cmd.SetContext(ctx)

	profiler := profile.GetProfiler(ctx)

	if cfg.Profile == "" {
		// account-level CLI was not really done before, so here are the assumptions:
		// 1. only admins will have account configured
		// 2. 99% of admins will have access to just one account
		// hence, we don't need to create a special "DEFAULT_ACCOUNT" profile yet
		profiles, err := profiler.LoadProfiles(cmd.Context(), profile.MatchAccountProfiles)
		if err == nil && len(profiles) == 1 {
			cfg.Profile = profiles[0].Name
		}

		// if there is no config file, we don't want to fail and instead just skip it
		if err != nil && !errors.Is(err, profile.ErrNoConfiguration) {
			return err
		}
	}

	allowPrompt := !hasProfileFlag && !shouldSkipPrompt(cmd.Context())
	a, err := accountClientOrPrompt(cmd.Context(), cfg, allowPrompt)
	if err != nil {
		return err
	}

	ctx = context.WithValue(ctx, &accountClient, a)
	cmd.SetContext(ctx)
	return nil
}

// Helper function to create a workspace client or prompt once if the given configuration is not valid.
func workspaceClientOrPrompt(ctx context.Context, cfg *config.Config, allowPrompt bool) (*databricks.WorkspaceClient, error) {
	w, err := databricks.NewWorkspaceClient((*databricks.Config)(cfg))
	if err == nil {
		err = w.Config.Authenticate(emptyHttpRequest(ctx))
	}

	prompt := false
	if allowPrompt && err != nil && cmdio.IsPromptSupported(ctx) {
		// Prompt to select a profile if the current configuration is not a workspace client.
		prompt = prompt || errors.Is(err, databricks.ErrNotWorkspaceClient)
		// Prompt to select a profile if the current configuration doesn't resolve to a credential provider.
		prompt = prompt || errors.Is(err, config.ErrCannotConfigureAuth)
	}

	if !prompt {
		// If we are not prompting, we can return early.
		return w, err
	}

	// Try picking a profile dynamically if the current configuration is not valid.
	profile, err := AskForWorkspaceProfile(ctx)
	if err != nil {
		return nil, err
	}
	w, err = databricks.NewWorkspaceClient(&databricks.Config{Profile: profile})
	if err == nil {
		err = w.Config.Authenticate(emptyHttpRequest(ctx))
		if err != nil {
			return nil, err
		}
	}
	return w, err
}

func MustWorkspaceClient(cmd *cobra.Command, args []string) error {
	cfg := &config.Config{}

	// The command-line profile flag takes precedence over DATABRICKS_CONFIG_PROFILE.
	profile, hasProfileFlag := profileFlagValue(cmd)
	if hasProfileFlag {
		cfg.Profile = profile
	}

	ctx := cmd.Context()
	ctx = context.WithValue(ctx, &configUsed, cfg)
	cmd.SetContext(ctx)

	// Try to load a bundle configuration if we're allowed to by the caller (see `./auth_options.go`).
	if !shouldSkipLoadBundle(cmd.Context()) {
		b, diags := TryConfigureBundle(cmd)
		if err := diags.Error(); err != nil {
			return err
		}

		if b != nil {
			ctx = context.WithValue(ctx, &configUsed, b.Config.Workspace.Config())
			cmd.SetContext(ctx)
			client, err := b.InitializeWorkspaceClient()
			if err != nil {
				return err
			}
			cfg = client.Config
		}
	}

	allowPrompt := !hasProfileFlag && !shouldSkipPrompt(cmd.Context())
	w, err := workspaceClientOrPrompt(cmd.Context(), cfg, allowPrompt)
	if err != nil {
		return err
	}

	ctx = context.WithValue(ctx, &workspaceClient, w)
	cmd.SetContext(ctx)
	return nil
}

func SetWorkspaceClient(ctx context.Context, w *databricks.WorkspaceClient) context.Context {
	return context.WithValue(ctx, &workspaceClient, w)
}

func SetAccountClient(ctx context.Context, a *databricks.AccountClient) context.Context {
	return context.WithValue(ctx, &accountClient, a)
}

func AskForWorkspaceProfile(ctx context.Context) (string, error) {
	profiler := profile.GetProfiler(ctx)
	path, err := profiler.GetPath(ctx)
	if err != nil {
		return "", fmt.Errorf("cannot determine Databricks config file path: %w", err)
	}
	profiles, err := profiler.LoadProfiles(ctx, profile.MatchWorkspaceProfiles)
	if err != nil {
		return "", err
	}
	switch len(profiles) {
	case 0:
		return "", ErrNoWorkspaceProfiles{path: path}
	case 1:
		return profiles[0].Name, nil
	}
	i, _, err := cmdio.RunSelect(ctx, &promptui.Select{
		Label:             fmt.Sprintf("Workspace profiles defined in %s", path),
		Items:             profiles,
		Searcher:          profiles.SearchCaseInsensitive,
		StartInSearchMode: true,
		Templates: &promptui.SelectTemplates{
			Label:    "{{ . | faint }}",
			Active:   `{{.Name | bold}} ({{.Host|faint}})`,
			Inactive: `{{.Name}}`,
			Selected: `{{ "Using workspace profile" | faint }}: {{ .Name | bold }}`,
		},
	})
	if err != nil {
		return "", err
	}
	return profiles[i].Name, nil
}

func AskForAccountProfile(ctx context.Context) (string, error) {
	profiler := profile.GetProfiler(ctx)
	path, err := profiler.GetPath(ctx)
	if err != nil {
		return "", fmt.Errorf("cannot determine Databricks config file path: %w", err)
	}
	profiles, err := profiler.LoadProfiles(ctx, profile.MatchAccountProfiles)
	if err != nil {
		return "", err
	}
	switch len(profiles) {
	case 0:
		return "", ErrNoAccountProfiles{path}
	case 1:
		return profiles[0].Name, nil
	}
	i, _, err := cmdio.RunSelect(ctx, &promptui.Select{
		Label:             fmt.Sprintf("Account profiles defined in %s", path),
		Items:             profiles,
		Searcher:          profiles.SearchCaseInsensitive,
		StartInSearchMode: true,
		Templates: &promptui.SelectTemplates{
			Label:    "{{ . | faint }}",
			Active:   `{{.Name | bold}} ({{.AccountID|faint}} {{.Cloud|faint}})`,
			Inactive: `{{.Name}}`,
			Selected: `{{ "Using account profile" | faint }}: {{ .Name | bold }}`,
		},
	})
	if err != nil {
		return "", err
	}
	return profiles[i].Name, nil
}

// To verify that a client is configured correctly, we pass an empty HTTP request
// to a client's `config.Authenticate` function. Note: this functionality
// should be supported by the SDK itself.
func emptyHttpRequest(ctx context.Context) *http.Request {
	req, err := http.NewRequestWithContext(ctx, "", "", nil)
	if err != nil {
		panic(err)
	}
	return req
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

func ConfigUsed(ctx context.Context) *config.Config {
	cfg, ok := ctx.Value(&configUsed).(*config.Config)
	if !ok {
		panic("cannot get *config.Config. Please report it as a bug")
	}
	return cfg
}
