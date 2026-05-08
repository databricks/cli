package auth

import (
	"context"
	"errors"
	"fmt"
	"io/fs"
	"net"
	"net/url"
	"sync"
	"time"

	"github.com/databricks/cli/libs/auth"
	"github.com/databricks/cli/libs/cmdio"
	"github.com/databricks/cli/libs/databrickscfg"
	"github.com/databricks/cli/libs/databrickscfg/profile"
	"github.com/databricks/cli/libs/env"
	"github.com/databricks/cli/libs/log"
	"github.com/databricks/databricks-sdk-go"
	"github.com/databricks/databricks-sdk-go/apierr"
	"github.com/databricks/databricks-sdk-go/config"
	"github.com/spf13/cobra"
	"gopkg.in/ini.v1"
)

// profileStatus is the three-state result of validating a profile, plus a
// fourth state for "validation skipped". Reported as `status` in JSON output.
type profileStatus string

const (
	profileStatusValid       profileStatus = "valid"       // API call succeeded
	profileStatusInvalid     profileStatus = "invalid"     // proven bad: auth/config error
	profileStatusUnknown     profileStatus = "unknown"     // could not determine (network, transient)
	profileStatusUnvalidated profileStatus = "unvalidated" // --skip-validate
)

// profileValidationTimeout bounds the per-profile validation API call so a
// single hung profile (dead host, no route) does not stall the whole listing.
const profileValidationTimeout = 10 * time.Second

type profileMetadata struct {
	Name        string        `json:"name"`
	Host        string        `json:"host,omitempty"`
	AccountID   string        `json:"account_id,omitempty"`
	WorkspaceID string        `json:"workspace_id,omitempty"`
	Cloud       string        `json:"cloud"`
	AuthType    string        `json:"auth_type"`
	Status      profileStatus `json:"status"`
	Error       string        `json:"error,omitempty"`
	Default     bool          `json:"default,omitempty"`

	// Valid is the legacy compat field. Emitted only when Status is conclusively
	// valid or invalid; absent (omitempty) for unknown/unvalidated. Old scripts
	// that branch on `valid: true` keep working; scripts that branch on
	// `valid: false` keep working for proven-bad profiles and now see the field
	// missing for the cases we previously misreported as "false".
	Valid *bool `json:"valid,omitempty"`

	// statusDisplay is the colored cell rendered for the text "Valid" column.
	// Populated by Load; never serialized.
	StatusDisplay string `json:"-"`
}

func (c *profileMetadata) IsEmpty() bool {
	return c.Host == "" && c.AccountID == ""
}

// setStatus records the classification result and keeps the legacy Valid field
// in sync. Valid is left nil for unknown/unvalidated so omitempty drops it.
func (c *profileMetadata) setStatus(ctx context.Context, status profileStatus, msg string) {
	c.Status = status
	c.Error = msg
	switch status {
	case profileStatusValid:
		t := true
		c.Valid = &t
	case profileStatusInvalid:
		f := false
		c.Valid = &f
	case profileStatusUnknown, profileStatusUnvalidated:
		// Leave Valid nil; omitempty drops the legacy field.
	}
	c.StatusDisplay = renderStatusCell(ctx, status)
}

// renderStatusCell formats a profileStatus for the text "Valid" column. We
// keep the values short so the column width stays close to the previous
// YES/NO output.
func renderStatusCell(ctx context.Context, s profileStatus) string {
	switch s {
	case profileStatusValid:
		return cmdio.Green(ctx, "valid")
	case profileStatusInvalid:
		return cmdio.Red(ctx, "invalid")
	case profileStatusUnknown:
		return cmdio.Yellow(ctx, "unknown")
	case profileStatusUnvalidated:
		return "-"
	}
	return "-"
}

// classifyValidationError maps an error returned by the validation API call
// to a (status, message) pair. The profile name is interpolated into auth
// remediation hints. Order matters: timeout is checked first because the
// SDK can wrap context.DeadlineExceeded in url.Error, and APIError sentinels
// before the generic *url.Error / *net.OpError fallthrough.
func classifyValidationError(profileName string, err error) (profileStatus, string) {
	if err == nil {
		return profileStatusValid, ""
	}
	if errors.Is(err, context.DeadlineExceeded) {
		return profileStatusUnknown, "validation timed out"
	}
	if errors.Is(err, apierr.ErrUnauthenticated) {
		return profileStatusInvalid, fmt.Sprintf("authentication failed (token may have expired — try 'databricks auth login -p %s')", profileName)
	}
	if errors.Is(err, apierr.ErrPermissionDenied) {
		return profileStatusInvalid, "credentials lack permission for the validation API call"
	}
	var apiErr *apierr.APIError
	if errors.As(err, &apiErr) && apiErr.StatusCode >= 500 {
		return profileStatusUnknown, fmt.Sprintf("server error: %d", apiErr.StatusCode)
	}
	if reason, ok := networkErrorReason(err); ok {
		return profileStatusUnknown, "could not reach host: " + reason
	}
	return profileStatusUnknown, err.Error()
}

// networkErrorReason returns the short reason from a *url.Error / *net.OpError
// chain (skipping the URL prefix that url.Error.Error() prepends). The second
// return is false when err is not a network error.
func networkErrorReason(err error) (string, bool) {
	var urlErr *url.Error
	if errors.As(err, &urlErr) && urlErr.Err != nil {
		return urlErr.Err.Error(), true
	}
	var netErr *net.OpError
	if errors.As(err, &netErr) {
		return netErr.Error(), true
	}
	return "", false
}

func (c *profileMetadata) Load(ctx context.Context, configFilePath string, skipValidate bool) {
	timeoutSeconds := int(profileValidationTimeout / time.Second)
	cfg := &config.Config{
		Loaders:           []config.Loader{config.ConfigFile},
		ConfigFile:        configFilePath,
		Profile:           c.Name,
		DatabricksCliPath: env.Get(ctx, "DATABRICKS_CLI_PATH"),

		// Bound the SDK's per-request and total-retry budgets to the same
		// per-profile ceiling. EnsureResolved fetches host metadata via the
		// SDK's retrier, which defaults to 5 minutes — without this, a single
		// dead host stalls the listing well past the validation context's
		// timeout (the EnsureResolved call uses context.Background internally,
		// so our context.WithTimeout below cannot reach it).
		HTTPTimeoutSeconds:  timeoutSeconds,
		RetryTimeoutSeconds: timeoutSeconds,
	}
	_ = cfg.EnsureResolved()
	if cfg.IsAws() {
		c.Cloud = "aws"
	} else if cfg.IsAzure() {
		c.Cloud = "azure"
	} else if cfg.IsGcp() {
		c.Cloud = "gcp"
	}

	if skipValidate {
		c.Host = cfg.CanonicalHostName()
		c.AuthType = cfg.AuthType
		c.setStatus(ctx, profileStatusUnvalidated, "")
		return
	}

	configType := auth.ResolveConfigType(cfg)
	if configType != cfg.ConfigType() {
		log.Debugf(ctx, "Profile %q: overrode config type from %s to %s (SPOG host)", c.Name, cfg.ConfigType(), configType)
	}

	if configType == config.InvalidConfig {
		c.setStatus(ctx, profileStatusInvalid, "profile fields conflict (e.g. workspace and account configured together)")
		return
	}

	callCtx, cancel := context.WithTimeout(ctx, profileValidationTimeout)
	defer cancel()

	var err error
	switch configType {
	case config.AccountConfig:
		var a *databricks.AccountClient
		a, err = databricks.NewAccountClient((*databricks.Config)(cfg))
		if err == nil {
			_, err = a.Workspaces.List(callCtx)
		}
	case config.WorkspaceConfig:
		var w *databricks.WorkspaceClient
		w, err = databricks.NewWorkspaceClient((*databricks.Config)(cfg))
		if err == nil {
			_, err = w.CurrentUser.Me(callCtx)
		}
	case config.InvalidConfig:
		// Handled above with an early return; listed here for switch exhaustiveness.
	}

	c.Host = cfg.Host
	c.AuthType = cfg.AuthType

	status, msg := classifyValidationError(c.Name, err)
	c.setStatus(ctx, status, msg)
}

func newProfilesCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "profiles",
		Short: "Lists profiles from ~/.databrickscfg",
		Annotations: map[string]string{
			"template": cmdio.Heredoc(`
			{{header "Name"}}	{{header "Host"}}	{{header "Valid"}}
			{{range .Profiles}}{{.Name | green}}{{if .Default}} (Default){{end}}	{{.Host|cyan}}	{{.StatusDisplay}}
			{{end}}`),
		},
	}

	var skipValidate bool
	cmd.Flags().BoolVar(&skipValidate, "skip-validate", false, "Whether to skip validating the profiles")

	cmd.RunE = func(cmd *cobra.Command, args []string) error {
		var profiles []*profileMetadata
		iniFile, err := profile.DefaultProfiler.Get(cmd.Context())
		if errors.Is(err, fs.ErrNotExist) {
			// return empty list for non-configured machines
			iniFile = &config.File{
				File: &ini.File{},
			}
		} else if err != nil {
			return fmt.Errorf("cannot parse config file: %w", err)
		}

		defaultProfile := databrickscfg.GetConfiguredDefaultProfileFrom(iniFile)

		var wg sync.WaitGroup
		for _, v := range iniFile.Sections() {
			hash := v.KeysHash()
			profile := &profileMetadata{
				Name:        v.Name(),
				Host:        hash["host"],
				AccountID:   hash["account_id"],
				WorkspaceID: hash["workspace_id"],
				Default:     v.Name() == defaultProfile,
			}
			if profile.IsEmpty() {
				continue
			}
			wg.Go(func() {
				ctx := cmd.Context()
				t := time.Now()
				profile.Load(ctx, iniFile.Path(), skipValidate)
				log.Debugf(ctx, "Profile %q took %s to load", profile.Name, time.Since(t))
			})
			profiles = append(profiles, profile)
		}
		wg.Wait()
		return cmdio.Render(cmd.Context(), struct {
			Profiles []*profileMetadata `json:"profiles"`
		}{profiles})
	}

	return cmd
}
