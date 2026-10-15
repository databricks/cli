// Package apiclient issues hand-written REST calls with the CLI's resolved
// authentication, for endpoints that have no sdk-go client (or need raw access,
// e.g. `databricks api`).
package apiclient

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"maps"
	"net/http"
	"net/url"
	"reflect"
	"slices"
	"strings"

	"github.com/databricks/databricks-sdk-go/config"
	"github.com/databricks/sdk-go/core/apierr"
)

// ErrHTMLContent is returned when a JSON response was expected but the server
// returned an HTML page, which usually means the host is not a Databricks API.
var ErrHTMLContent = errors.New("received HTML response instead of JSON")

type Client struct {
	Config *config.Config

	host       string
	httpClient *http.Client
}

// New returns a client for cfg. cfg must already be resolved.
func New(cfg *config.Config) (*Client, error) {
	if err := cfg.EnsureResolved(); err != nil {
		return nil, err
	}
	return &Client{
		Config:     cfg,
		host:       strings.TrimSuffix(cfg.CanonicalHostName(), "/"),
		httpClient: HTTPClient(cfg),
	}, nil
}

// Do sends a request to path (relative to the host, or an absolute URL).
//
// For GET, DELETE and HEAD, request is encoded into the query string; for
// other methods it is the body: []byte, string and io.Reader are sent as is,
// anything else is JSON-encoded. queryParams are always added to the query.
//
// response may be nil, *[]byte, *bytes.Buffer, *io.ReadCloser (caller closes),
// or a value to JSON-decode into. Non-2xx responses are returned as
// *apierr.APIError.
func (c *Client) Do(ctx context.Context, method, path string, headers map[string]string, queryParams map[string]any, request, response any) error {
	reqURL, err := c.url(path)
	if err != nil {
		return err
	}
	query := reqURL.Query()
	if err := addQuery(query, queryParams); err != nil {
		return err
	}

	var body io.Reader
	contentType := ""
	if request != nil {
		switch method {
		case http.MethodGet, http.MethodDelete, http.MethodHead:
			if err := addQuery(query, request); err != nil {
				return err
			}
		default:
			switch v := request.(type) {
			case []byte:
				body = bytes.NewReader(v)
			case string:
				body = strings.NewReader(v)
			case io.Reader:
				body = v
			default:
				data, err := json.Marshal(request)
				if err != nil {
					return fmt.Errorf("request marshal: %w", err)
				}
				body = bytes.NewReader(data)
				contentType = "application/json"
			}
		}
	}
	reqURL.RawQuery = encodeQuery(query)

	req, err := http.NewRequestWithContext(ctx, method, reqURL.String(), body)
	if err != nil {
		return err
	}
	if contentType != "" {
		req.Header.Set("Content-Type", contentType)
	}
	switch response.(type) {
	case nil, *bytes.Buffer, *io.ReadCloser, *[]byte:
	default:
		req.Header.Set("Accept", "application/json")
	}
	for k, v := range headers {
		req.Header.Set(k, v)
	}
	if err := c.Config.Authenticate(req); err != nil {
		return err
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return err
	}
	if rc, ok := response.(*io.ReadCloser); ok && resp.StatusCode < 400 {
		*rc = resp.Body
		return nil
	}
	defer resp.Body.Close()
	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("failed to read response body: %w", err)
	}
	if apiErr := apierr.FromHTTPError(resp.StatusCode, resp.Header, respBody); apiErr != nil {
		return apiErr
	}
	return decodeResponse(respBody, response)
}

func (c *Client) url(path string) (*url.URL, error) {
	if strings.HasPrefix(path, "http://") || strings.HasPrefix(path, "https://") {
		return url.Parse(path)
	}
	if !strings.HasPrefix(path, "/") {
		path = "/" + path
	}
	return url.Parse(c.host + path)
}

func decodeResponse(body []byte, response any) error {
	switch v := response.(type) {
	case nil:
		return nil
	case *bytes.Buffer:
		_, err := v.Write(body)
		return err
	case *[]byte:
		*v = body
		return nil
	}
	if len(body) == 0 {
		return nil
	}
	if err := json.Unmarshal(body, response); err != nil {
		if syntaxErr, ok := errors.AsType[*json.SyntaxError](err); ok && syntaxErr != nil && strings.HasPrefix(string(body), "<") {
			return ErrHTMLContent
		}
		return fmt.Errorf("failed to unmarshal response body: %w", err)
	}
	return nil
}

// addQuery adds v (a map, url.Values, or a JSON-encodable struct) to query.
// Zero values are skipped, matching what the APIs expect for optional fields.
func addQuery(query url.Values, v any) error {
	switch x := v.(type) {
	case nil:
		return nil
	case url.Values:
		for k, vs := range x {
			for _, s := range vs {
				query.Add(k, s)
			}
		}
		return nil
	case map[string]string:
		for _, k := range slices.Sorted(maps.Keys(x)) {
			if x[k] != "" {
				query.Add(k, x[k])
			}
		}
		return nil
	case map[string]any:
		// Like the previous SDK: nil values are skipped, everything else
		// (including zero values such as 0 or false) is formatted with %v.
		for _, k := range slices.Sorted(maps.Keys(x)) {
			if v, ok := derefValue(x[k]); ok {
				query.Add(k, fmt.Sprintf("%v", v))
			}
		}
		return nil
	}
	data, err := json.Marshal(v)
	if err != nil {
		return err
	}
	dec := json.NewDecoder(bytes.NewReader(data))
	dec.UseNumber()
	var decoded any
	if err := dec.Decode(&decoded); err != nil {
		return err
	}
	if decoded == nil {
		return nil
	}
	m, ok := decoded.(map[string]any)
	if !ok {
		return fmt.Errorf("unsupported query parameters: %T", v)
	}
	flatten(query, "", m)
	return nil
}

// derefValue follows pointers (optional fields of request types are pointers)
// and reports false for nil.
func derefValue(v any) (any, bool) {
	rv := reflect.ValueOf(v)
	for rv.Kind() == reflect.Pointer {
		if rv.IsNil() {
			return nil, false
		}
		rv = rv.Elem()
	}
	if !rv.IsValid() {
		return nil, false
	}
	return rv.Interface(), true
}

func flatten(query url.Values, prefix string, m map[string]any) {
	for k, v := range m {
		var key string
		if prefix != "" {
			key = prefix + "." + k
		} else {
			key = k
		}
		switch x := v.(type) {
		case nil:
		case map[string]any:
			flatten(query, key, x)
		case []any:
			for _, item := range x {
				query.Add(key, fmt.Sprint(item))
			}
		case string:
			if x != "" {
				query.Add(key, x)
			}
		case bool:
			if x {
				query.Add(key, "true")
			}
		case json.Number:
			if x.String() != "0" {
				query.Add(key, x.String())
			}
		default:
			query.Add(key, fmt.Sprint(x))
		}
	}
}

// encodeQuery is url.Values.Encode with spaces as %20, as the old SDK did.
func encodeQuery(q url.Values) string {
	return strings.ReplaceAll(q.Encode(), "+", "%20")
}
