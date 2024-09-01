package auth

import (
	"context"
	"errors"
	"fmt"
	"io/fs"

	"github.com/databricks/cli/libs/auth/cache"
	"github.com/databricks/cli/libs/auth"
	"github.com/databricks/cli/libs/cmdio"
	"github.com/databricks/cli/libs/databrickscfg"
	"github.com/databricks/cli/libs/databrickscfg/profile"
	"github.com/databricks/databricks-sdk-go/config"
	"github.com/spf13/cobra"
)

type LogoutSession struct {
	Profile        string
	File           config.File
	PersistentAuth *auth.PersistentAuth
}

func (l *LogoutSession) load(ctx context.Context, profileName string, persistentAuth *auth.PersistentAuth) error {
	l.Profile = profileName
	l.PersistentAuth = persistentAuth
	iniFile, err := profile.DefaultProfiler.Get(ctx)
	if errors.Is(err, fs.ErrNotExist) {
		return err
	} else if err != nil {
		return fmt.Errorf("cannot parse config file: %w", err)
	}
	l.File = *iniFile
	return nil
}

func (l *LogoutSession) getConfigSetionMap() (map[string]string, error) {
	section, err := l.File.GetSection(l.Profile)
	if err != nil {
		return map[string]string{}, fmt.Errorf("profile does not exist in config file: %w", err)
	}
	return section.KeysHash(), nil
}

// clear token from ~/.databricks/token-cache.json
func (l *LogoutSession) clearTokenCache(ctx context.Context) error {
	return l.PersistentAuth.ClearToken(ctx)
}

// Overrewrite profile to .databrickscfg without fields marked as sensitive
// Other attributes are preserved.
func (l *LogoutSession) clearConfigFile(ctx context.Context, sectionMap map[string]string) error {
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

func newLogoutCommand(persistentAuth *auth.PersistentAuth) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "logout [PROFILE]",
		Short: "Logout from specified profile",
	}

	cmd.RunE = func(cmd *cobra.Command, args []string) error {
		ctx := cmd.Context()
		profileName := cmd.Flag("profile").Value.String()
		// If the user has not specified a profile name, prompt for one.
		if profileName == "" {
			var err error
			profileName, err = promptForProfile(ctx, persistentAuth.ProfileName())
			if err != nil {
				return err
			}
		}
		// Set the host and account-id based on the provided arguments and flags.
		err := setHostAndAccountId(ctx, profileName, persistentAuth, args)
		if err != nil {
			return err
		}
		defer persistentAuth.Close()
		LogoutSession := &LogoutSession{}
		LogoutSession.load(ctx, profileName, persistentAuth)
		configSectionMap, err := LogoutSession.getConfigSetionMap()
		if err != nil {
			return err
		}
		err = LogoutSession.clearTokenCache(ctx)
		if err != nil {
			if errors.Is(err, cache.ErrNotConfigured) {
				// It is OK to not have OAuth configured. Move on and remove
				// sensitive values example PAT from config file
			} else {
				return err
			}
		}
		if err := LogoutSession.clearConfigFile(ctx, configSectionMap); err != nil {
			return err
		}
		cmdio.LogString(ctx, fmt.Sprintf("Profile %s was successfully logged out", profileName))
		return nil
	}
	return cmd
}
