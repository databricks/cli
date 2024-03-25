package auth

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/databricks/cli/cmd/root"
	"github.com/databricks/cli/libs/cmdio"
	"github.com/databricks/cli/libs/flags"
	"github.com/databricks/databricks-sdk-go/config"
	"github.com/spf13/cobra"
)

var authTemplate = `{{"Workspace:" | bold}} {{.Details.Host}}
{{if .Username }}{{"User:" | bold}} {{.Username}}{{end}}
{{"Authenticated with:" | bold}} {{.Details.AuthType}}
{{if .AccountID }}{{"Account ID:" | bold}} {{.AccountID}}{{end}}
-----
` + configurationTemplate

var errorTemplate = `Unable to authenticate: {{.Error}}
-----
` + configurationTemplate

const configurationTemplate = `Configuration:
  {{- $details := .Details}}
  {{- range $k, $v := $details.Configuration }}
  {{if $v.AuthTypeMismatch}}~{{else}}✓{{end}} {{$k | bold}}: {{$v.Value}}
  {{- if and (not (eq $v.Source.String "dynamic configuration")) $v.Source }}
  {{- " (from" | italic}} {{$v.Source.String | italic}}
  {{- if $v.AuthTypeMismatch}}, {{ "not used for auth type " | red | italic }}{{$details.AuthType | red | italic}}{{end}})
  {{- end}}
  {{- end}}
`

func newDescribeCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "describe",
		Short: "Describes which auth credentials is being used by the CLI and identify their source",
	}

	var showSensitive bool
	var isAccount bool
	cmd.Flags().BoolVar(&showSensitive, "sensitive", false, "Include sensitive fields like passwords and tokens in the output")
	cmd.Flags().BoolVar(&isAccount, "account", false, "Describe the account client instead of the workspace client")

	cmd.RunE = func(cmd *cobra.Command, args []string) error {
		ctx := cmd.Context()
		var status *authStatus
		var err error
		if isAccount {
			status, err = getAccountAuthStatus(cmd, args, showSensitive, func(cmd *cobra.Command, args []string) (*config.Config, error) {
				err := root.MustAccountClient(cmd, args)
				return root.ConfigUsed(cmd.Context()), err
			})
		} else {
			status, err = getWorkspaceAuthStatus(cmd, args, showSensitive, func(cmd *cobra.Command, args []string) (*config.Config, error) {
				err := root.MustWorkspaceClient(cmd, args)
				return root.ConfigUsed(cmd.Context()), err
			})
		}

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

type tryAuth func(cmd *cobra.Command, args []string) (*config.Config, error)

func getWorkspaceAuthStatus(cmd *cobra.Command, args []string, showSensitive bool, fn tryAuth) (*authStatus, error) {
	cfg, err := fn(cmd, args)
	ctx := cmd.Context()
	if err != nil {
		return &authStatus{
			Status:  "error",
			Error:   err,
			Details: getAuthDetails(cmd, cfg, showSensitive),
		}, nil
	}

	w := root.WorkspaceClient(ctx)
	me, err := w.CurrentUser.Me(ctx)
	if err != nil {
		return nil, err
	}

	status := authStatus{
		Status:   "success",
		Details:  getAuthDetails(cmd, w.Config, showSensitive),
		Username: me.UserName,
	}

	return &status, nil
}

func getAccountAuthStatus(cmd *cobra.Command, args []string, showSensitive bool, fn tryAuth) (*authStatus, error) {
	cfg, err := fn(cmd, args)
	ctx := cmd.Context()
	if err != nil {
		return &authStatus{
			Status:  "error",
			Error:   err,
			Details: getAuthDetails(cmd, cfg, showSensitive),
		}, nil
	}

	a := root.AccountClient(ctx)
	status := authStatus{
		Status:    "success",
		Details:   getAuthDetails(cmd, a.Config, showSensitive),
		AccountID: a.Config.AccountID,
		Username:  a.Config.Username,
	}

	return &status, nil
}

func render(ctx context.Context, cmd *cobra.Command, status *authStatus, template string) error {
	switch root.OutputType(cmd) {
	case flags.OutputText:
		return cmdio.RenderWithTemplate(ctx, status, "", template)
	case flags.OutputJSON:
		buf, err := json.MarshalIndent(status, "", "  ")
		if err != nil {
			return err
		}
		cmd.OutOrStdout().Write(buf)
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

	return details
}
