package autodeploy

import (
	"errors"
	"fmt"
	"net/url"
	"path"
	"path/filepath"
	"strings"

	"github.com/databricks/cli/bundle"
	"github.com/databricks/cli/bundle/config"
	"github.com/databricks/cli/libs/gitwebhooks"
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
