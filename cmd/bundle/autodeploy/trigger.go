package autodeploy

import (
	"context"
	"errors"
	"fmt"
	"net/url"
	"path"
	"path/filepath"
	"strings"

	"github.com/databricks/cli/bundle"
	"github.com/databricks/cli/bundle/config"
	"github.com/databricks/cli/cmd/bundle/utils"
	"github.com/databricks/cli/libs/gitwebhooks"
	"github.com/databricks/databricks-sdk-go/listing"
	"github.com/spf13/cobra"
)

// trigger identifies the auto-deploy trigger of a bundle target.
type trigger struct {
	repoURL string

	// bundlePath is the repository-relative path to the bundle's configuration file.
	bundlePath string
	target     string
}

// Auto-deploy infers only these providers. Any other host takes --git-provider, since
// a host name does not tell, for example, GitHub Enterprise from GitLab self-managed.
var providersByHost = map[string]gitwebhooks.GitProvider{
	"github.com":    gitwebhooks.GitProviderGitHub,
	"dev.azure.com": gitwebhooks.GitProviderAzureDevOpsServices,
}

// resolveTrigger reads the trigger of the bundle's target from its Git details.
func resolveTrigger(b *bundle.Bundle) (trigger, error) {
	git := b.Config.Bundle.Git
	if b.InGitRepo == nil || !*b.InGitRepo || git.OriginURL == "" {
		return trigger{}, errors.New("auto-deploy needs the bundle in a Git repository with an origin remote. To add one, run: git remote add origin <url>")
	}
	// The service keys webhooks on the HTTPS URL and clones over HTTPS. It cannot parse the
	// scp-like SSH form, and an ssh:// URL would be a different webhook than the HTTPS one.
	if u, err := url.Parse(git.OriginURL); err != nil || u.Scheme != "https" {
		return trigger{}, fmt.Errorf("auto-deploy needs the HTTPS URL of the repository, but the origin is %s. Set bundle.git.origin_url to the HTTPS URL", git.OriginURL)
	}

	configFile, err := config.FileNames.FindInPath(b.BundleRootPath)
	if err != nil {
		return trigger{}, err
	}

	return trigger{
		repoURL:    git.OriginURL,
		bundlePath: path.Join(git.BundleRootPath, filepath.Base(configFile)),
		target:     b.Config.Bundle.Target,
	}, nil
}

// loadTrigger loads the bundle and returns the trigger of its target, with the API to look it up.
func loadTrigger(cmd *cobra.Command) (trigger, *gitwebhooks.API, error) {
	b, err := utils.ProcessBundle(cmd, utils.ProcessOptions{})
	if err != nil {
		return trigger{}, nil, err
	}
	t, err := resolveTrigger(b)
	if err != nil {
		return trigger{}, nil, err
	}
	api, err := gitwebhooks.New(b.WorkspaceClient(cmd.Context()))
	return t, api, err
}

// bundleFilter matches the triggers of every target of the bundle.
func (t trigger) bundleFilter() string {
	return fmt.Sprintf("repo_url = %s AND trigger_config.dabs_deploy.bundle_yml_repo_path = %s", quote(t.repoURL), quote(t.bundlePath))
}

// targetFilter matches the trigger of the bundle's target. The service has at most one.
func (t trigger) targetFilter() string {
	return t.bundleFilter() + " AND trigger_config.dabs_deploy.deployment_mode = " + quote(t.target)
}

// quote quotes s as an AIP-160 string. Not strconv.Quote: the filter takes only \" and \\ as escapes.
func quote(s string) string {
	return `"` + strings.NewReplacer(`\`, `\\`, `"`, `\"`).Replace(s) + `"`
}

// findTrigger returns the trigger of the bundle's target, or an error if it has none.
func findTrigger(ctx context.Context, api *gitwebhooks.API, t trigger) (*gitwebhooks.WebhookTrigger, error) {
	triggers, err := listing.ToSlice(ctx, api.ListWebhookTriggers(ctx, gitwebhooks.ListWebhookTriggersRequest{Filter: t.targetFilter()}))
	if err != nil {
		return nil, err
	}
	if len(triggers) == 0 {
		// The service lists only the triggers the caller created, unless the caller is an admin.
		return nil, fmt.Errorf("target %s has no auto-deploy trigger that you can view. To create one, run: databricks bundle auto-deploy create -t %s", t.target, t.target)
	}
	return &triggers[0], nil
}

// resolveProvider returns the provider flag if set, or the one inferred from the repository URL.
func resolveProvider(repoURL string, flag gitwebhooks.GitProvider) (gitwebhooks.GitProvider, error) {
	if flag != "" {
		return flag, nil
	}
	provider, ok := inferProvider(repoURL)
	if !ok {
		return "", fmt.Errorf("cannot infer the Git provider of %s. Pass it with --git-provider", repoURL)
	}
	return provider, nil
}

func inferProvider(repoURL string) (gitwebhooks.GitProvider, bool) {
	u, err := url.Parse(repoURL)
	if err != nil {
		return "", false
	}
	provider, ok := providersByHost[strings.ToLower(u.Hostname())]
	return provider, ok
}
