package lakebox

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/databricks/databricks-sdk-go"
)

const lakeboxAPIPath = "/api/2.0/lakebox"

// lakeboxAPI wraps raw HTTP calls to the lakebox REST API.
type lakeboxAPI struct {
	w *databricks.WorkspaceClient
}

// createRequest is the JSON body for POST /api/2.0/lakebox.
type createRequest struct {
	PublicKey string `json:"public_key,omitempty"`
}

// createResponse is the JSON body returned by POST /api/2.0/lakebox.
type createResponse struct {
	LakeboxID string `json:"lakebox_id"`
	Status    string `json:"status"`
}

// lakeboxEntry is a single item in the list response.
type lakeboxEntry struct {
	Name   string `json:"name"`
	Status string `json:"status"`
	FQDN   string `json:"fqdn"`
}

// listResponse is the JSON body returned by GET /api/2.0/lakebox.
type listResponse struct {
	Lakeboxes []lakeboxEntry `json:"lakeboxes"`
}

// apiError is the error body returned by the lakebox API.
type apiError struct {
	ErrorCode string `json:"error_code"`
	Message   string `json:"message"`
}

func (e *apiError) Error() string {
	return fmt.Sprintf("%s: %s", e.ErrorCode, e.Message)
}

func newLakeboxAPI(w *databricks.WorkspaceClient) *lakeboxAPI {
	return &lakeboxAPI{w: w}
}

// create calls POST /api/2.0/lakebox with an optional public key.
func (a *lakeboxAPI) create(ctx context.Context, publicKey string) (*createResponse, error) {
	body := createRequest{PublicKey: publicKey}
	jsonBody, err := json.Marshal(body)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	resp, err := a.doRequest(ctx, "POST", lakeboxAPIPath, bytes.NewReader(jsonBody))
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated {
		return nil, parseAPIError(resp)
	}

	var result createResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}
	return &result, nil
}

// list calls GET /api/2.0/lakebox.
func (a *lakeboxAPI) list(ctx context.Context) ([]lakeboxEntry, error) {
	resp, err := a.doRequest(ctx, "GET", lakeboxAPIPath, nil)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, parseAPIError(resp)
	}

	var result listResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}
	return result.Lakeboxes, nil
}

// get calls GET /api/2.0/lakebox/{id}.
func (a *lakeboxAPI) get(ctx context.Context, id string) (*lakeboxEntry, error) {
	resp, err := a.doRequest(ctx, "GET", lakeboxAPIPath+"/"+id, nil)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, parseAPIError(resp)
	}

	var result lakeboxEntry
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}
	return &result, nil
}

// delete calls DELETE /api/2.0/lakebox/{id}.
func (a *lakeboxAPI) delete(ctx context.Context, id string) error {
	resp, err := a.doRequest(ctx, "DELETE", lakeboxAPIPath+"/"+id, nil)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusNoContent {
		return parseAPIError(resp)
	}
	return nil
}

// doRequest makes an authenticated HTTP request to the workspace.
func (a *lakeboxAPI) doRequest(ctx context.Context, method, path string, body io.Reader) (*http.Response, error) {
	host := strings.TrimRight(a.w.Config.Host, "/")
	url := host + path

	req, err := http.NewRequestWithContext(ctx, method, url, body)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	if err := a.w.Config.Authenticate(req); err != nil {
		return nil, fmt.Errorf("failed to authenticate: %w", err)
	}

	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}

	return http.DefaultClient.Do(req)
}

func parseAPIError(resp *http.Response) error {
	body, _ := io.ReadAll(resp.Body)
	var apiErr apiError
	if json.Unmarshal(body, &apiErr) == nil && apiErr.Message != "" {
		return &apiErr
	}
	return fmt.Errorf("HTTP %d: %s", resp.StatusCode, string(body))
}

// registerKeyRequest is the JSON body for POST /api/2.0/lakebox/register-key.
type registerKeyRequest struct {
	PublicKey string `json:"public_key"`
}

// registerKey calls POST /api/2.0/lakebox/register-key.
func (a *lakeboxAPI) registerKey(ctx context.Context, publicKey string) error {
	body := registerKeyRequest{PublicKey: publicKey}
	jsonBody, err := json.Marshal(body)
	if err != nil {
		return fmt.Errorf("failed to marshal request: %w", err)
	}

	resp, err := a.doRequest(ctx, "POST", lakeboxAPIPath+"/register-key", bytes.NewReader(jsonBody))
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return parseAPIError(resp)
	}
	return nil
}

// extractLakeboxID extracts the short ID from a full resource name.
// e.g. "apps/lakebox/instances/happy-panda-1234" -> "happy-panda-1234"
func extractLakeboxID(name string) string {
	parts := strings.Split(name, "/")
	if len(parts) > 0 {
		return parts[len(parts)-1]
	}
	return name
}
