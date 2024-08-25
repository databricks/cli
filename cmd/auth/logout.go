package auth

import (
	"context"
	"errors"
	"fmt"
	"io/fs"

	"github.com/databricks/cli/libs/auth/cache"
	"github.com/databricks/cli/libs/cmdio"
	"github.com/databricks/cli/libs/databrickscfg"
	"github.com/databricks/cli/libs/databrickscfg/profile"
	"github.com/databricks/databricks-sdk-go/config"
	"github.com/spf13/cobra"
)

type Logout struct {
	Profile string
	File    config.File
	Cache   cache.TokenCache
}

func (l *Logout) load(ctx context.Context, profileName string) error {
	l.Profile = profileName
	l.Cache = cache.GetTokenCache(ctx)
	iniFile, err := profile.DefaultProfiler.Get(ctx)
	if errors.Is(err, fs.ErrNotExist) {
		return err
	} else if err != nil {
		return fmt.Errorf("cannot parse config file: %w", err)
	}
	l.File = *iniFile
	return nil
}

func (l *Logout) getSetionMap() (map[string]string, error) {
	section, err := l.File.GetSection(l.Profile)
	if err != nil {
		return map[string]string{}, fmt.Errorf("profile does not exist in config file: %w", err)
	}
	return section.KeysHash(), nil
}

// clear token from ~/.databricks/token-cache.json
func (l *Logout) clearTokenCache(key string) error {
	return l.Cache.DeleteKey(key)
}

// Overrewrite profile to .databrickscfg without fields marked as sensitive
// Other attributes are preserved.
func (l *Logout) clearConfigFile(ctx context.Context, sectionMap map[string]string) error {
	return databrickscfg.SaveToProfile(ctx, &config.Config{
		ConfigFile:           l.File.Path(),
		Profile:              l.Profile,
		Host:                 sectionMap["host"],
		ClusterID:            sectionMap["cluster_id"],
		WarehouseID:          sectionMap["warehouse_id"],
		ServerlessComputeID:  sectionMap["serverless_compute_id"],
		AccountID:            sectionMap["account_id"],
		Username:             sectionMap["username"],
		GoogleServiceAccount: sectionMap["google_service_account"],
		AzureResourceID:      sectionMap["azure_workspace_resource_id"],
		AzureClientID:        sectionMap["azure_client_id"],
		AzureTenantID:        sectionMap["azure_tenant_id"],
		AzureEnvironment:     sectionMap["azure_environment"],
		AzureLoginAppID:      sectionMap["azure_login_app_id"],
		ClientID:             sectionMap["client_id"],
		AuthType:             sectionMap["auth_type"],
	})
}

func newLogoutCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "logout [PROFILE]",
		Short: "Logout from specified profile",
	}

	cmd.RunE = func(cmd *cobra.Command, args []string) error {
		ctx := cmd.Context()
		var profileName string
		if len(args) < 1 {
			profileName = cmd.Flag("profile").Value.String()
		} else {
			profileName = args[0]
		}
		logout := &Logout{}
		logout.load(ctx, profileName)
		sectionMap, err := logout.getSetionMap()
		if err != nil {
			return err
		}
		if err := logout.clearTokenCache(sectionMap["host"]); err != nil {
			return err
		}
		if err := logout.clearConfigFile(ctx, sectionMap); err != nil {
			return err
		}
		cmdio.LogString(ctx, fmt.Sprintf("Profile %s was successfully logged out", profileName))
		return nil
	}
	return cmd
}
