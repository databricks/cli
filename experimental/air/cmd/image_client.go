package aircmd

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/databricks/databricks-sdk-go"
	"github.com/databricks/databricks-sdk-go/apierr"
	"github.com/databricks/databricks-sdk-go/client"
)

// imagesAPIPath is the AI Compute Manager image service, called with a raw
// client.Do because the SDK does not model it. The ":get" and ":checkImageAccess"
// verbs are literal path suffixes the backend expects, not query-style actions.
const imagesAPIPath = "/api/2.0/ai-compute-manager/images"

// imageStatus is the lifecycle state of a registered image, as reported in the
// API response's "state" field.
type imageStatus string

const (
	imageStatusPending   imageStatus = "PENDING"
	imageStatusImporting imageStatus = "IMPORTING"
	imageStatusAvailable imageStatus = "AVAILABLE"
	imageStatusFailed    imageStatus = "FAILED"
)

// errImageUploadFailed marks a terminal FAILED upload, so callers can classify
// it as permanent rather than a retryable transient error.
var errImageUploadFailed = errors.New("image upload failed")

// errImageWaitTimeout marks the poll giving up before the image became
// AVAILABLE. A later run may find it ready, so callers classify it as transient.
var errImageWaitTimeout = errors.New("image did not become AVAILABLE")

// imageRegistration is a registered image with its status and metadata.
type imageRegistration struct {
	DockerImageURL string      `json:"docker_image_url"`
	Status         imageStatus `json:"-"`
	StatusMessage  string      `json:"status_message"`
	ManifestSHA256 string      `json:"manifest_sha256"`
	// State is the raw wire field; Status is derived from it via normalizeStatus
	// so an unknown value degrades to PENDING rather than an invalid enum.
	State string `json:"state"`
}

// normalizeStatus maps the raw "state" field to a known status, defaulting to
// PENDING for absent or unrecognized values (matching the Python client).
func (r *imageRegistration) normalizeStatus() {
	switch imageStatus(r.State) {
	case imageStatusPending, imageStatusImporting, imageStatusAvailable, imageStatusFailed:
		r.Status = imageStatus(r.State)
	default:
		r.Status = imageStatusPending
	}
}

// normalizeDockerImageURL canonicalizes a container image URL for consistent
// hashing. It prepends docker.io/ only for short-form URLs (no explicit
// registry); a registry is identified by a "." in the host portion of the first
// path component (e.g. docker.io, nvcr.io, registry.gitlab.com).
func normalizeDockerImageURL(imageURL string) string {
	url := strings.TrimSpace(imageURL)
	parts := strings.Split(url, "/")

	// A registry hostname contains a dot. Check only the host portion (before any
	// ":") of the first component so version tags like "ubuntu:22.04" are not
	// mistaken for a registry hostname.
	if !strings.Contains(strings.Split(parts[0], ":")[0], ".") {
		if len(parts) == 1 {
			// Bare name (e.g. "ubuntu", "ubuntu:latest") — a Docker Hub official image.
			url = "docker.io/library/" + url
		} else {
			// User/org image (e.g. "pytorch/pytorch:2.0.0") — just add the registry.
			url = "docker.io/" + url
		}
	}

	// When a digest is present it takes precedence per the OCI spec — strip any tag.
	if idx := strings.Index(url, "@"); idx != -1 {
		repo, digest := url[:idx], url[idx+1:]
		lastSlash := strings.LastIndex(repo, "/")
		if colon := strings.Index(repo[lastSlash+1:], ":"); colon != -1 {
			repo = repo[:lastSlash+1+colon]
		}
		return repo + "@" + digest
	}

	// No digest and no tag on the final component — default to :latest.
	last := url[strings.LastIndex(url, "/")+1:]
	if !strings.Contains(last, ":") {
		url += ":latest"
	}
	return url
}

// imageClient calls the AI Compute Manager image service.
type imageClient struct {
	api *client.DatabricksClient
}

// newImageClient builds an imageClient from an authenticated workspace client.
func newImageClient(w *databricks.WorkspaceClient) (*imageClient, error) {
	api, err := client.New(w.Config)
	if err != nil {
		return nil, fmt.Errorf("failed to create API client: %w", err)
	}
	return &imageClient{api: api}, nil
}

// do issues one request against the image service, decoding the response into
// out. The image URL is normalized by the caller.
func (c *imageClient) do(ctx context.Context, method, endpoint string, query map[string]any, body, out any) error {
	return c.api.Do(ctx, method, imagesAPIPath+endpoint, nil, query, body, out)
}

// createImage registers a Docker image, optionally with registry credentials
// from a Databricks secret. CreateImage is idempotent: re-registering reconciles
// the stored status against the backing image entity.
func (c *imageClient) createImage(ctx context.Context, dockerImageURL, credentialsScope, credentialsKey string) (*imageRegistration, error) {
	body := map[string]any{"docker_image_url": normalizeDockerImageURL(dockerImageURL)}
	if credentialsScope != "" && credentialsKey != "" {
		body["credentials_scope"] = credentialsScope
		body["credentials_key"] = credentialsKey
	}

	var resp struct {
		Image *imageRegistration `json:"image"`
		imageRegistration
	}
	if err := c.do(ctx, http.MethodPost, "", nil, body, &resp); err != nil {
		return nil, fmt.Errorf("failed to register image: %w", err)
	}

	// The response may wrap the registration under "image" or inline it.
	reg := resp.Image
	if reg == nil {
		reg = &resp.imageRegistration
	}
	reg.normalizeStatus()
	return reg, nil
}

// getImage returns the registration for an image, or nil if it is not registered.
func (c *imageClient) getImage(ctx context.Context, dockerImageURL string) (*imageRegistration, error) {
	query := map[string]any{"docker_image_url": normalizeDockerImageURL(dockerImageURL)}
	var reg imageRegistration
	if err := c.do(ctx, http.MethodGet, ":get", query, nil, &reg); err != nil {
		if errors.Is(err, apierr.ErrNotFound) {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to get image: %w", err)
	}
	reg.normalizeStatus()
	return &reg, nil
}

// checkImageAccess reports whether an image is publicly pullable without
// credentials. It returns nil when the answer can't be determined — e.g. the
// manager region does not expose this RPC — so callers can treat nil as unknown.
func (c *imageClient) checkImageAccess(ctx context.Context, dockerImageURL string) *bool {
	query := map[string]any{"docker_image_url": normalizeDockerImageURL(dockerImageURL)}
	var resp struct {
		PubliclyAccessible *bool `json:"publicly_accessible"`
	}
	if err := c.do(ctx, http.MethodGet, ":checkImageAccess", query, nil, &resp); err != nil {
		return nil
	}
	return resp.PubliclyAccessible
}

// waitForImageReady polls getImage until the image is AVAILABLE. Callers should
// call createImage first so the status is reconciled before polling begins.
func (c *imageClient) waitForImageReady(ctx context.Context, dockerImageURL string, timeout, pollInterval time.Duration) (*imageRegistration, error) {
	deadline := time.Now().Add(timeout)
	for {
		reg, err := c.getImage(ctx, dockerImageURL)
		if err != nil {
			return nil, err
		}
		if reg == nil {
			return nil, fmt.Errorf("image registration not found: %s", dockerImageURL)
		}

		switch reg.Status {
		case imageStatusAvailable:
			return reg, nil
		case imageStatusFailed:
			msg := reg.StatusMessage
			if msg == "" {
				msg = "unknown error"
			}
			return nil, fmt.Errorf("%w: %s", errImageUploadFailed, msg)
		case imageStatusPending, imageStatusImporting:
			// Still uploading; fall through to sleep and poll again.
		}

		if time.Now().After(deadline) {
			return nil, fmt.Errorf("%w within %s", errImageWaitTimeout, timeout)
		}
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case <-time.After(pollInterval):
		}
	}
}
