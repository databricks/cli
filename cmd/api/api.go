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
	"github.com/databricks/cli/libs/databrickscfg"
	"github.com/databricks/cli/libs/env"
	"github.com/databricks/cli/libs/flags"
	"github.com/databricks/databricks-sdk-go/client"
	"github.com/databricks/databricks-sdk-go/config"
	"github.com/spf13/cobra"
)

const (
	// workspaceIDHeader is the workspace routing identifier sent on
	// workspace-scope requests against unified hosts. Generated SDK service
	// methods set this per-call when cfg.WorkspaceID is populated; we mirror
	// the same idiom. The gateway also accepts the legacy X-Databricks-Org-Id
	// header for rollback safety.
	workspaceIDHeader = "X-Databricks-Workspace-Id"

	// orgIDQueryParam and workspaceIDQueryParam are the SPOG
	// (single-page-of-glass) URL convention used by the Databricks UI:
	// "?o=<workspace-id>" or "?w=<workspace-id>" identifies the workspace a
	// URL targets. When present on the path, we treat it as a per-call
	// override for the workspace routing identifier so that pasted SPOG URLs
	// route correctly without requiring --workspace-id. "w" is the new
	// spelling that matches the X-Databricks-Workspace-Id header; "o" stays
	// accepted for URLs already pasted from older UI builds, shell history,
	// or committed databricks.yml files. "o" takes precedence when both are
	// present to preserve the meaning of existing URLs.
	orgIDQueryParam       = "o"
	workspaceIDQueryParam = "w"
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

			// Resolve the profile mirroring MustWorkspaceClient precedence:
			// 1. --profile flag, 2. DATABRICKS_CONFIG_PROFILE env var (the SDK
			// also reads it, but setting cfg.Profile here keeps any error
			// messages we render referring to the same name), 3.
			// [__settings__].default_profile in the config file.
			if profileFlag := cmd.Flag("profile"); profileFlag != nil {
				cfg.Profile = profileFlag.Value.String()
			}
			if cfg.Profile == "" {
				cfg.Profile = env.Get(cmd.Context(), "DATABRICKS_CONFIG_PROFILE")
			}
			if cfg.Profile == "" {
				cfg.Profile = databrickscfg.ResolveDefaultProfile(cmd.Context())
			}

			auth.NormalizeDatabricksConfigFromEnv(cmd.Context(), cfg)

			api, err := client.New(cfg)
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
				headers[workspaceIDHeader] = orgID
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

// extractWorkspaceIDFromQuery returns the workspace ID encoded in the path's
// query string (the SPOG URL convention). It checks "o" first, then "w";
// returns "" if neither is present or non-empty.
func extractWorkspaceIDFromQuery(rawPath string) (string, error) {
	u, err := url.Parse(rawPath)
	if err != nil {
		return "", fmt.Errorf("parse path: %w", err)
	}
	q := u.Query()
	if v := q.Get(orgIDQueryParam); v != "" {
		return v, nil
	}
	return q.Get(workspaceIDQueryParam), nil
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
	workspaceIDFromQuery, err := extractWorkspaceIDFromQuery(path)
	if err != nil {
		return "", err
	}
	if workspaceIDFromQuery != "" {
		return workspaceIDFromQuery, nil
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
