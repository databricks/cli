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

type profileMetadata struct {
	Name        string `json:"name"`
	Host        string `json:"host,omitempty"`
	AccountID   string `json:"account_id,omitempty"`
	WorkspaceID string `json:"workspace_id,omitempty"`
	Cloud       string `json:"cloud"`
	AuthType    string `json:"auth_type"`
	Valid       bool   `json:"valid"`
	Default     bool   `json:"default,omitempty"`
}

func (c *profileMetadata) IsEmpty() bool {
	return c.Host == "" && c.AccountID == ""
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
		return
	}

	configType := auth.ResolveConfigType(cfg)
	if configType != cfg.ConfigType() {
		log.Debugf(ctx, "Profile %q: overrode config type from %s to %s (SPOG host)", c.Name, cfg.ConfigType(), configType)
	}

	callCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	switch configType {
	case config.AccountConfig:
		a, err := databricks.NewAccountClient((*databricks.Config)(cfg))
		if err != nil {
			return
		}
		_, err = a.Workspaces.List(callCtx)
		c.Host = cfg.Host
		c.AuthType = cfg.AuthType
		if err != nil {
			return
		}
		c.Valid = true
	case config.WorkspaceConfig:
		w, err := databricks.NewWorkspaceClient((*databricks.Config)(cfg))
		if err != nil {
			return
		}
		_, err = w.CurrentUser.Me(callCtx, iam.MeRequest{})
		c.Host = cfg.Host
		c.AuthType = cfg.AuthType
		if err != nil {
			return
		}
		c.Valid = true
	case config.InvalidConfig:
		// Invalid configuration, skip validation
		return
	}
}

func newProfilesCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "profiles",
		Short: "Lists profiles from ~/.databrickscfg",
		Annotations: map[string]string{
			"template": cmdio.Heredoc(`
			{{header "Name"}}	{{header "Host"}}	{{header "Valid"}}
			{{range .Profiles}}{{.Name | green}}{{if .Default}} (Default){{end}}	{{.Host|cyan}}	{{bool .Valid}}
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
