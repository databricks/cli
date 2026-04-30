package api

import (
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"regexp"
	"strings"

	"github.com/databricks/cli/cmd/root"
	"github.com/databricks/cli/libs/auth"
	"github.com/databricks/cli/libs/cmdio"
	"github.com/databricks/cli/libs/flags"
	"github.com/databricks/databricks-sdk-go/client"
	"github.com/databricks/databricks-sdk-go/config"
	"github.com/spf13/cobra"
)

const (
	// orgIDHeader is the workspace routing identifier sent on workspace-scope
	// requests against unified hosts. Generated SDK service methods set this
	// per-call when cfg.WorkspaceID is populated; we mirror the same idiom.
	orgIDHeader = "X-Databricks-Org-Id"

	accountIDPlaceholder = "{account_id}"
)

// accountSegmentRe matches a non-empty segment immediately after "accounts/",
// anchored at the start of the path or after a "/". Account-ID shape is
// deliberately opaque; the workspace-proxy list carves out SDK proxies that
// also live under /accounts/.
var accountSegmentRe = regexp.MustCompile(`(^|/)accounts/[^/]+`)

func New() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "api",
		Short: "Perform Databricks API call",
	}

	cmd.AddCommand(
		makeCommand(http.MethodGet),
		makeCommand(http.MethodHead),
		makeCommand(http.MethodPost),
		makeCommand(http.MethodPut),
		makeCommand(http.MethodPatch),
		makeCommand(http.MethodDelete),
	)

	return cmd
}

func makeCommand(method string) *cobra.Command {
	var (
		payload         flags.JsonFlag
		forceAccount    bool
		workspaceIDFlag string
	)

	command := &cobra.Command{
		Use:   strings.ToLower(method) + " PATH",
		Args:  root.ExactArgs(1),
		Short: fmt.Sprintf("Perform %s request", method),
		RunE: func(cmd *cobra.Command, args []string) error {
			path := args[0]

			var request any
			diags := payload.Unmarshal(&request)
			if diags.HasError() {
				return diags.Error()
			}

			cfg := &config.Config{}

			// command-line flag can specify the profile in use
			profileFlag := cmd.Flag("profile")
			if profileFlag != nil {
				cfg.Profile = profileFlag.Value.String()
			}

			api, err := client.New(cfg)
			if err != nil {
				return err
			}

			path, err = substituteAccountID(path, cfg.AccountID, cfg.Profile)
			if err != nil {
				return err
			}

			orgID, err := resolveOrgID(
				forceAccount,
				workspaceIDFlag,
				cmd.Flags().Changed("workspace-id"),
				normalizeWorkspaceID(cfg.WorkspaceID),
				path,
			)
			if err != nil {
				return err
			}

			headers := map[string]string{"Content-Type": "application/json"}
			if orgID != "" {
				headers[orgIDHeader] = orgID
			}

			var response any
			err = api.Do(cmd.Context(), method, path, headers, nil, request, &response)
			if err != nil {
				return err
			}
			return cmdio.Render(cmd.Context(), response)
		},
	}

	command.Flags().Var(&payload, "json", `either inline JSON string or @path/to/file.json with request body`)
	command.Flags().BoolVar(&forceAccount, "account", false,
		"Treat this call as account-scoped (skip the workspace routing identifier). Mutually exclusive with --workspace-id.")
	command.Flags().StringVar(&workspaceIDFlag, "workspace-id", "",
		"Override the workspace routing identifier on this call. Mutually exclusive with --account.")
	return command
}

// normalizeWorkspaceID strips the CLI-only WorkspaceIDNone sentinel so the
// SDK's idiomatic "if cfg.WorkspaceID != \"\"" check produces the right call
// shape. The CLI persists "none" in .databrickscfg to mark profiles where the
// user explicitly skipped workspace selection; the SDK does not know about
// this sentinel and would otherwise send the literal "none" as a routing
// identifier.
func normalizeWorkspaceID(workspaceID string) string {
	if workspaceID == auth.WorkspaceIDNone {
		return ""
	}
	return workspaceID
}

// substituteAccountID replaces "{account_id}" placeholders in path with the
// resolved account ID. Errors with an actionable message if the placeholder
// is present but no account_id is configured for the active profile.
func substituteAccountID(path, accountID, profile string) (string, error) {
	if !strings.Contains(path, accountIDPlaceholder) {
		return path, nil
	}
	if accountID == "" {
		return "", fmt.Errorf(
			"path contains %s but no account_id is set on profile %q "+
				"(set account_id in ~/.databrickscfg, export DATABRICKS_ACCOUNT_ID, "+
				"or replace %s in the path)",
			accountIDPlaceholder, profile, accountIDPlaceholder)
	}
	return strings.ReplaceAll(path, accountIDPlaceholder, accountID), nil
}

// hasAccountSegment reports whether path is an account-scope API. The match
// runs on URL.Path, so query strings and fragments containing "/accounts/"
// can't trigger a false positive. Returns false for paths that match a known
// workspace-routed proxy from the proxy path tables.
func hasAccountSegment(rawPath string) (bool, error) {
	u, err := url.Parse(rawPath)
	if err != nil {
		return false, fmt.Errorf("parse path: %w", err)
	}
	p := u.Path
	if isWorkspaceProxyPath(p) {
		return false, nil
	}
	return accountSegmentRe.MatchString(p), nil
}

// resolveOrgID picks the value (if any) for the workspace routing identifier
// based on flags, the resolved profile, and the path shape. Returns "" when
// no header should be sent.
func resolveOrgID(
	forceAccount bool,
	workspaceIDFlag string,
	workspaceIDFlagSet bool,
	cfgWorkspaceID string,
	path string,
) (string, error) {
	if forceAccount && workspaceIDFlagSet {
		return "", errors.New("--account and --workspace-id are mutually exclusive")
	}
	if forceAccount {
		return "", nil
	}
	if workspaceIDFlagSet {
		if workspaceIDFlag == "" {
			return "", errors.New("--workspace-id requires a value; use --account to scope this call to the account API")
		}
		return workspaceIDFlag, nil
	}
	isAccount, err := hasAccountSegment(path)
	if err != nil {
		return "", err
	}
	if isAccount {
		return "", nil
	}
	return cfgWorkspaceID, nil
}
