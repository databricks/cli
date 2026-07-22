package auth

import (
	"context"
	"errors"
	"fmt"
	"io/fs"
	"sync"
	"time"

	"github.com/databricks/cli/libs/auth"
	"github.com/databricks/cli/libs/cmdio"
	"github.com/databricks/cli/libs/databrickscfg"
	"github.com/databricks/cli/libs/databrickscfg/profile"
	"github.com/databricks/cli/libs/env"
	"github.com/databricks/cli/libs/log"
	"github.com/databricks/databricks-sdk-go"
	"github.com/databricks/databricks-sdk-go/config"
	"github.com/databricks/databricks-sdk-go/service/iam"
	"github.com/spf13/cobra"
	"gopkg.in/ini.v1"
)

// profileValidationTimeout bounds each per-profile validation so a host the SDK
// retries — connection refused, connect/TLS timeout, retriable 5xx — cannot
// stall the whole listing. Without it a single such host blocks `auth profiles`
// for the SDK's default retry budget (~5 minutes). Hosts that fail DNS are not
// retriable and already fail fast, so this only bounds the retriable cases.
const profileValidationTimeout = 5 * time.Second

// profileStatus is the three-state result of listing a profile. It drives the
// YES/NO/?? cell in the text table. Only YES (validated) and NO (validation
// failed) reflect an actual check; ?? means validation was skipped.
type profileStatus string

const (
	profileStatusValid   profileStatus = "valid"   // validation succeeded
	profileStatusInvalid profileStatus = "invalid" // validation failed (any error)
	profileStatusSkipped profileStatus = "skipped" // --skip-validate; not checked
)

type profileMetadata struct {
	Name        string `json:"name"`
	Host        string `json:"host,omitempty"`
	AccountID   string `json:"account_id,omitempty"`
	WorkspaceID string `json:"workspace_id,omitempty"`
	Cloud       string `json:"cloud"`
	AuthType    string `json:"auth_type"`

	// Valid is true only when validation conclusively succeeded. ValidReason
	// explains a false result: the full underlying error when validation failed,
	// or a "skipped" note under --skip-validate. It is empty only on success.
	// The text table shows only YES/NO/?? (via StatusDisplay); the reason detail
	// is reserved for --output json.
	Valid       bool   `json:"valid"`
	ValidReason string `json:"valid_reason,omitempty"`
	Default     bool   `json:"default,omitempty"`

	// StatusDisplay is the colored YES/NO/?? cell for the text table. Not
	// serialized; the JSON form uses Valid + ValidReason instead.
	StatusDisplay string `json:"-"`
}

func (c *profileMetadata) IsEmpty() bool {
	return c.Host == "" && c.AccountID == ""
}

// setStatus records the validation result: Valid is true only for a
// conclusively valid profile, and reason explains a non-valid result (the
// underlying error, or a skip note) for JSON output.
func (c *profileMetadata) setStatus(ctx context.Context, status profileStatus, reason string) {
	c.Valid = status == profileStatusValid
	c.ValidReason = reason
	c.StatusDisplay = renderStatusCell(ctx, status)
}

// renderStatusCell formats a profileStatus for the text "Valid" column:
// YES (green, validated) / NO (red, validation failed) / ?? (grey, skipped).
func renderStatusCell(ctx context.Context, s profileStatus) string {
	switch s {
	case profileStatusValid:
		return cmdio.Green(ctx, "YES")
	case profileStatusInvalid:
		return cmdio.Red(ctx, "NO")
	case profileStatusSkipped:
		return cmdio.HiBlack(ctx, "??")
	}
	return cmdio.HiBlack(ctx, "??")
}

func (c *profileMetadata) Load(ctx context.Context, configFilePath string, skipValidate bool, timeout time.Duration) {
	timeoutSeconds := int(timeout / time.Second)
	cfg := &config.Config{
		Loaders:           []config.Loader{config.ConfigFile},
		ConfigFile:        configFilePath,
		Profile:           c.Name,
		DatabricksCliPath: env.Get(ctx, "DATABRICKS_CLI_PATH"),

		// Bound the SDK's per-request and total-retry budgets to the same
		// per-profile ceiling. EnsureResolved fetches host metadata via the
		// SDK's retrier, which defaults to 5 minutes — and it runs on
		// context.Background internally, so the context.WithTimeout below on the
		// validation call cannot reach it. Without these a single unreachable
		// host stalls the listing well past the validation timeout.
		HTTPTimeoutSeconds:  timeoutSeconds,
		RetryTimeoutSeconds: timeoutSeconds,
	}
	if skipValidate {
		// EnsureResolved fetches <host>/.well-known/databricks-config to enrich
		// the config, so without this stub a skip-validate listing still makes
		// one network call per profile (and warns when offline). Resolve from
		// the config file alone; cloud detection falls back to the host pattern.
		cfg.HostMetadataResolver = func(context.Context, string) (*config.HostMetadata, error) {
			return nil, nil
		}
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
		c.setStatus(ctx, profileStatusSkipped, "validation skipped (--skip-validate)")
		return
	}

	configType := auth.ResolveConfigType(cfg)
	if configType != cfg.ConfigType() {
		log.Debugf(ctx, "Profile %q: overrode config type from %s to %s (SPOG host)", c.Name, cfg.ConfigType(), configType)
	}

	callCtx, cancel := context.WithTimeout(ctx, timeout)
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
			_, err = w.CurrentUser.Me(callCtx, iam.MeRequest{})
		}
	case config.InvalidConfig:
		c.setStatus(ctx, profileStatusInvalid, "profile fields conflict (e.g. workspace and account configured together)")
		return
	}

	c.Host = cfg.Host
	c.AuthType = cfg.AuthType

	// Any validation error means NO; the full error is preserved in ValidReason
	// for --output json. The text table shows only the YES/NO/?? cell.
	if err != nil {
		c.setStatus(ctx, profileStatusInvalid, err.Error())
		return
	}
	c.setStatus(ctx, profileStatusValid, "")
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
				profile.Load(ctx, iniFile.Path(), skipValidate, profileValidationTimeout)
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
