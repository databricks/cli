package github

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/databricks/cli/libs/log"
)

const (
	gitHubAPI         = "https://api.github.com"
	gitHubUserContent = "https://raw.githubusercontent.com"
)

// Placeholders to use as unique keys in context.Context.
var (
	apiOverride         int
	userContentOverride int
)

func WithApiOverride(ctx context.Context, override string) context.Context {
	return context.WithValue(ctx, &apiOverride, override)
}

func WithUserContentOverride(ctx context.Context, override string) context.Context {
	return context.WithValue(ctx, &userContentOverride, override)
}

var ErrNotFound = errors.New("not found")

type pagedResponse struct {
	Body     []byte
	NextLink string
}

func getBytes(ctx context.Context, method, url string, body io.Reader) ([]byte, error) {
	resp, err := getPagedBytes(ctx, method, url, body)
	if err != nil {
		return nil, err
	}
	return resp.Body, nil
}

func getPagedBytes(ctx context.Context, method, url string, body io.Reader) (*pagedResponse, error) {
	ao, ok := ctx.Value(&apiOverride).(string)
	if ok {
		url = strings.Replace(url, gitHubAPI, ao, 1)
	}
	uco, ok := ctx.Value(&userContentOverride).(string)
	if ok {
		url = strings.Replace(url, gitHubUserContent, uco, 1)
	}
	log.Tracef(ctx, "%s %s", method, url)
	req, err := http.NewRequestWithContext(ctx, "GET", url, body)
	if err != nil {
		return nil, err
	}
	res, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	if res.StatusCode == 404 {
		return nil, ErrNotFound
	}
	if res.StatusCode >= 400 {
		return nil, fmt.Errorf("github request failed: %s", res.Status)
	}
	nextLink := parseNextLink(res.Header.Get("link"))
	defer res.Body.Close()
	bodyBytes, err := io.ReadAll(res.Body)
	if err != nil {
		return nil, err
	}
	return &pagedResponse{
		Body:     bodyBytes,
		NextLink: nextLink,
	}, nil
}

func parseNextLink(linkHeader string) string {
	if linkHeader == "" {
		return ""
	}
	// Pagination and link headers are documented here:
	//   https://docs.github.com/en/rest/using-the-rest-api/using-pagination-in-the-rest-api?apiVersion=2022-11-28#using-link-headers
	// An example link header to handle:
	//   link: <https://api.github.com/repositories/1300192/issues?page=2>; rel="prev", <https://api.github.com/repositories/1300192/issues?page=4>; rel="next", <https://api.github.com/repositories/1300192/issues?page=515>; rel="last", <https://api.github.com/repositories/1300192/issues?page=1>; rel="first"
	links := strings.Split(linkHeader, ",")
	for _, link := range links {
		parts := strings.Split(strings.TrimSpace(link), ";")
		if len(parts) != 2 {
			continue
		}
		if strings.Contains(parts[1], `rel="next"`) {
			urlField := strings.TrimSpace(parts[0])
			if strings.HasPrefix(urlField, "<") && strings.HasSuffix(urlField, ">") {
				url := urlField[1 : len(urlField)-1]
				return url
			}
		}
	}
	return ""
}

func httpGetAndUnmarshal(ctx context.Context, url string, response any) (string, error) {
	raw, err := getPagedBytes(ctx, "GET", url, nil)
	if err != nil {
		return "", err
	}
	return raw.NextLink, json.Unmarshal(raw.Body, response)
}
