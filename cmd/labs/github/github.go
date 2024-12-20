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

func getBytes(ctx context.Context, method, url string, body io.Reader) ([]byte, error) {
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
	defer res.Body.Close()
	return io.ReadAll(res.Body)
}

func httpGetAndUnmarshal(ctx context.Context, url string, response any) error {
	raw, err := getBytes(ctx, "GET", url, nil)
	if err != nil {
		return err
	}
	return json.Unmarshal(raw, response)
}
