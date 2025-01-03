package github

import (
	"context"
	"fmt"
	"time"

	"github.com/databricks/cli/cmd/labs/localcache"
	"github.com/databricks/cli/libs/log"
)

const repositoryCacheTTL = 24 * time.Hour

func NewRepositoryCache(org, cacheDir string) *repositoryCache {
	filename := org + "-repositories"
	return &repositoryCache{
		cache: localcache.NewLocalCache[Repositories](cacheDir, filename, repositoryCacheTTL),
		Org:   org,
	}
}

type repositoryCache struct {
	cache localcache.LocalCache[Repositories]
	Org   string
}

func (r *repositoryCache) Load(ctx context.Context) (Repositories, error) {
	return r.cache.Load(ctx, func() (Repositories, error) {
		return getRepositories(ctx, r.Org)
	})
}

// getRepositories is considered to be privata API, as we want the usage to go through a cache
func getRepositories(ctx context.Context, org string) (Repositories, error) {
	var repos Repositories
	log.Debugf(ctx, "Loading repositories for %s from GitHub API", org)
	url := fmt.Sprintf("%s/users/%s/repos", gitHubAPI, org)
	err := httpGetAndUnmarshal(ctx, url, &repos)
	return repos, err
}

type Repositories []ghRepo

type ghRepo struct {
	Name          string   `json:"name"`
	Description   string   `json:"description"`
	Langauge      string   `json:"language"`
	DefaultBranch string   `json:"default_branch"`
	Stars         int      `json:"stargazers_count"`
	IsFork        bool     `json:"fork"`
	IsArchived    bool     `json:"archived"`
	Topics        []string `json:"topics"`
	HtmlURL       string   `json:"html_url"`
	CloneURL      string   `json:"clone_url"`
	SshURL        string   `json:"ssh_url"`
	License       struct {
		Name string `json:"name"`
	} `json:"license"`
}
