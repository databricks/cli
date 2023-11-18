package github

import (
	"context"
	"fmt"

	"github.com/databricks/cli/libs/log"
)

func ReadFileFromRef(ctx context.Context, org, repo, ref, file string) ([]byte, error) {
	log.Debugf(ctx, "Reading %s@%s from %s/%s", file, ref, org, repo)
	url := fmt.Sprintf("%s/%s/%s/%s/%s", gitHubUserContent, org, repo, ref, file)
	return getBytes(ctx, "GET", url, nil)
}

func DownloadZipball(ctx context.Context, org, repo, ref string) ([]byte, error) {
	log.Debugf(ctx, "Downloading zipball for %s from %s/%s", ref, org, repo)
	zipballURL := fmt.Sprintf("%s/repos/%s/%s/zipball/%s", gitHubAPI, org, repo, ref)
	return getBytes(ctx, "GET", zipballURL, nil)
}
