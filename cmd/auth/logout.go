package auth

import (
	"context"
	"errors"
	"fmt"
	"io/fs"

	"github.com/databricks/cli/libs/auth"
	"github.com/databricks/cli/libs/auth/cache"
	"github.com/databricks/cli/libs/cmdio"
	"github.com/databricks/cli/libs/databrickscfg"
	"github.com/databricks/cli/libs/databrickscfg/profile"
	"github.com/databricks/databricks-sdk-go/config"
	"github.com/spf13/cobra"
)

type logoutSession struct {
	profile        string
	file           config.File
	persistentAuth *auth.PersistentAuth
}

func (l *logoutSession) load(ctx context.Context, profileName string, persistentAuth *auth.PersistentAuth) error {
	l.profile = profileName
	l.persistentAuth = persistentAuth
	iniFile, err := profile.DefaultProfiler.Get(ctx)
	if errors.Is(err, fs.ErrNotExist) {
		return err
	} else if err != nil {
		return fmt.Errorf("cannot parse config file: %w", err)
	}
	l.file = *iniFile
	if err := l.setHostAndAccountIdFromProfile(); err != nil {
		return err
	}
	return nil
}

func (l *logoutSession) setHostAndAccountIdFromProfile() error {
	sectionMap, err := l.getConfigSectionMap()
	if err != nil {
		return err
	}
	if sectionMap["host"] == "" {
		return fmt.Errorf("no host configured for profile %s", l.profile)
	}
	l.persistentAuth.Host = sectionMap["host"]
	l.persistentAuth.AccountID = sectionMap["account_id"]
	return nil
}

func (l *logoutSession) getConfigSectionMap() (map[string]string, error) {
	section, err := l.file.GetSection(l.profile)
	if err != nil {
		return map[string]string{}, fmt.Errorf("profile does not exist in config file: %w", err)
	}
	return section.KeysHash(), nil
}

// clear token from ~/.databricks/token-cache.json
func (l *logoutSession) clearTokenCache(ctx context.Context) error {
	return l.persistentAuth.ClearToken(ctx)
}

// Overrewrite profile to .databrickscfg without fields marked as sensitive
// Other attributes are preserved.
func (l *logoutSession) clearConfigFile(ctx context.Context, sectionMap map[string]string) error {
	return databrickscfg.SaveToProfile(ctx, &config.Config{
		ConfigFile:           l.file.Path(),
		Profile:              l.profile,
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
		Long:  "Clears OAuth token from token-cache and any sensitive value in the config file, if they exist.",
	}

	cmd.RunE = func(cmd *cobra.Command, args []string) error {
		ctx := cmd.Context()
		profileNameFromFlag := cmd.Flag("profile").Value.String()
		// If both [PROFILE] and --profile are provided, return an error.
		if len(args) > 0 && profileNameFromFlag != "" {
			return fmt.Errorf("please only provide a profile as an argument or a flag, not both")
		}
		// Determine the profile name from either args or the flag.
		profileName := profileNameFromFlag
		if len(args) > 0 {
			profileName = args[0]
		}
		// If the user has not specified a profile name, prompt for one.
		if profileName == "" {
			var err error
			profileName, err = promptForProfile(ctx, persistentAuth.ProfileName())
			if err != nil {
				return err
			}
		}
		defer persistentAuth.Close()
		logoutSession := &logoutSession{}
		logoutSession.load(ctx, profileName, persistentAuth)
		configSectionMap, err := logoutSession.getConfigSectionMap()
		if err != nil {
			return err
		}
		err = logoutSession.clearTokenCache(ctx)
		if err != nil {
			if errors.Is(err, cache.ErrNotConfigured) {
				// It is OK to not have OAuth configured. Move on and remove
				// sensitive values from config file (Example PAT)
			} else {
				return err
			}
		}
		if err := logoutSession.clearConfigFile(ctx, configSectionMap); err != nil {
			return err
		}
		cmdio.LogString(ctx, fmt.Sprintf("Profile %s was successfully logged out", profileName))
		return nil
	}
	return cmd
}
