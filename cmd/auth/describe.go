package auth

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/databricks/cli/cmd/root"
	"github.com/databricks/cli/libs/cmdctx"
	"github.com/databricks/cli/libs/cmdio"
	"github.com/databricks/cli/libs/flags"
	"github.com/databricks/databricks-sdk-go/config"
	"github.com/spf13/cobra"
)

var authTemplate = `{{"Host:" | bold}} {{.Status.Details.Host}}
{{- if .Status.AccountID}}
{{"Account ID:" | bold}} {{.Status.AccountID}}
{{- end}}
{{- if .Status.Username}}
{{"User:" | bold}} {{.Status.Username}}
{{- end}}
{{"Authenticated with:" | bold}} {{.Status.Details.AuthType}}
-----
` + configurationTemplate

var errorTemplate = `Unable to authenticate: {{.Status.Error}}
-----
` + configurationTemplate

const configurationTemplate = `Current configuration:
  {{- $details := .Status.Details}}
  {{- range $a := .ConfigAttributes}}
  {{- $k := $a.Name}}
  {{- if index $details.Configuration $k}}
  {{- $v := index $details.Configuration $k}}
  {{if $v.AuthTypeMismatch}}~{{else}}✓{{end}} {{$k | bold}}: {{$v.Value}}
  {{- if not (eq $v.Source.String "dynamic configuration")}}
  {{- " (from" | italic}} {{$v.Source.String | italic}}
  {{- if $v.AuthTypeMismatch}}, {{ "not used for auth type " | red | italic }}{{$details.AuthType | red | italic}}{{end}})
  {{- end}}
  {{- end}}
  {{- end}}
`

func newDescribeCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "describe",
		Short: "Describes the credentials and the source of those credentials, being used by the CLI to authenticate",
	}

	var showSensitive bool
	cmd.Flags().BoolVar(&showSensitive, "sensitive", false, "Include sensitive fields like passwords and tokens in the output")

	cmd.RunE = func(cmd *cobra.Command, args []string) error {
		ctx := cmd.Context()
		var status *authStatus
		var err error
		status, err = getAuthStatus(cmd, args, showSensitive, func(cmd *cobra.Command, args []string) (*config.Config, bool, error) {
			isAccount, err := root.MustAnyClient(cmd, args)
			return cmdctx.ConfigUsed(cmd.Context()), isAccount, err
		})
		if err != nil {
			return err
		}

		if status.Error != nil {
			return render(ctx, cmd, status, errorTemplate)
		}

		return render(ctx, cmd, status, authTemplate)
	}

	return cmd
}

type tryAuth func(cmd *cobra.Command, args []string) (*config.Config, bool, error)

func getAuthStatus(cmd *cobra.Command, args []string, showSensitive bool, fn tryAuth) (*authStatus, error) {
	cfg, isAccount, err := fn(cmd, args)
	ctx := cmd.Context()
	if err != nil {
		return &authStatus{
			Status:  "error",
			Error:   err,
			Details: getAuthDetails(cmd, cfg, showSensitive),
		}, nil
	}

	if isAccount {
		a := cmdctx.AccountClient(ctx)

		// Doing a simple API call to check if the auth is valid
		_, err := a.Workspaces.List(ctx)
		if err != nil {
			return &authStatus{
				Status:  "error",
				Error:   err,
				Details: getAuthDetails(cmd, cfg, showSensitive),
			}, nil
		}

		status := authStatus{
			Status:    "success",
			Details:   getAuthDetails(cmd, a.Config, showSensitive),
			AccountID: a.Config.AccountID,
			Username:  a.Config.Username,
		}

		return &status, nil
	}

	w := cmdctx.WorkspaceClient(ctx)
	me, err := w.CurrentUser.Me(ctx)
	if err != nil {
		return &authStatus{
			Status:  "error",
			Error:   err,
			Details: getAuthDetails(cmd, cfg, showSensitive),
		}, nil
	}

	status := authStatus{
		Status:   "success",
		Details:  getAuthDetails(cmd, w.Config, showSensitive),
		Username: me.UserName,
	}

	return &status, nil
}

func render(ctx context.Context, cmd *cobra.Command, status *authStatus, template string) error {
	switch root.OutputType(cmd) {
	case flags.OutputText:
		return cmdio.RenderWithTemplate(ctx, map[string]any{
			"Status":           status,
			"ConfigAttributes": config.ConfigAttributes,
		}, "", template)
	case flags.OutputJSON:
		buf, err := json.MarshalIndent(status, "", "  ")
		if err != nil {
			return err
		}
		_, err = cmd.OutOrStdout().Write(buf)
		if err != nil {
			return err
		}
	default:
		return fmt.Errorf("unknown output type %s", root.OutputType(cmd))
	}

	return nil
}

type authStatus struct {
	Status    string             `json:"status"`
	Error     error              `json:"error,omitempty"`
	Username  string             `json:"username,omitempty"`
	AccountID string             `json:"account_id,omitempty"`
	Details   config.AuthDetails `json:"details"`
}

func getAuthDetails(cmd *cobra.Command, cfg *config.Config, showSensitive bool) config.AuthDetails {
	var opts []config.AuthDetailsOptions
	if showSensitive {
		opts = append(opts, config.ShowSensitive)
	}
	details := cfg.GetAuthDetails(opts...)

	for k, v := range details.Configuration {
		if k == "profile" && cmd.Flag("profile").Changed {
			v.Source = config.Source{Type: config.SourceType("flag"), Name: "--profile"}
		}

		if k == "host" && cmd.Flag("host").Changed {
			v.Source = config.Source{Type: config.SourceType("flag"), Name: "--host"}
		}
	}

	// If profile is not set explicitly, default to "default"
	if _, ok := details.Configuration["profile"]; !ok {
		profile := cfg.Profile
		if profile == "" {
			profile = "default"
		}
		details.Configuration["profile"] = &config.AttrConfig{Value: profile, Source: config.Source{Type: config.SourceDynamicConfig}}
	}

	// Unset source for databricks_cli_path because it can't be overridden anyway
	if v, ok := details.Configuration["databricks_cli_path"]; ok {
		v.Source = config.Source{Type: config.SourceDynamicConfig}
	}

	return details
}
