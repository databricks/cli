package aircmd

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/databricks/cli/cmd/root"
	"github.com/databricks/cli/libs/cmdctx"
	"github.com/databricks/cli/libs/flags"
	"github.com/databricks/cli/libs/log"
	"github.com/databricks/databricks-sdk-go"
	"github.com/databricks/databricks-sdk-go/apierr"
	"github.com/databricks/databricks-sdk-go/service/iam"
	"github.com/spf13/cobra"
)

// imagePollInterval is how often waitForImageReady polls for a status change.
const imagePollInterval = 5 * time.Second

// validateTagPolicy checks the deprecated --tag-policy value. Registration
// always re-checks the source registry, so "latest" and the empty default are
// no-ops. "auto" is rejected rather than silently remapped: it used to reuse a
// cached image, so honoring it as always-re-check would be a hidden change.
func validateTagPolicy(value string) error {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "", "latest":
		return nil
	case "auto":
		return errors.New("--tag-policy auto is no longer supported: auto mode was removed and registration now always checks the source registry for the latest digest; omit the flag or use --tag-policy latest")
	default:
		return fmt.Errorf("invalid image tag policy %q: the only supported value is latest", value)
	}
}

// registerImageResult is the JSON payload for `air register-image`. It mirrors
// the Python CLI's success shape so existing consumers keep working.
type registerImageResult struct {
	DockerImageURL string `json:"docker_image_url"`
	ManifestSHA256 string `json:"manifest_sha256"`
	Status         string `json:"status"`
	ImageUpdated   bool   `json:"image_updated"`
	Cached         bool   `json:"cached"`
}

func newRegisterImageCommand() *cobra.Command {
	var (
		tagPolicy      string
		timeoutMinutes int
	)

	cmd := &cobra.Command{
		Use:   "register-image IMAGE_URL",
		Args:  root.ExactArgs(1),
		Short: "Mirror a Docker image into the workspace registry",
		Long: `Mirror a Docker image into the workspace registry.

Credentials for private images are discovered from your local Docker
configuration (run ` + "`docker login`" + ` first); there are no credential flags.`,
	}

	// Registration always re-checks the source registry for the latest digest.
	// --tag-policy is kept only for backward compatibility (accepts "latest").
	cmd.Flags().StringVar(&tagPolicy, "tag-policy", "", "Deprecated and ignored; registration always checks the source registry for the latest digest")
	_ = cmd.Flags().MarkHidden("tag-policy")
	cmd.Flags().IntVar(&timeoutMinutes, "timeout-minutes", 60, "Timeout to wait for the image to become available")

	// Resolve and authenticate the workspace client up front so an auth failure
	// fails fast here, before any image is registered or polled.
	cmd.PreRunE = func(cmd *cobra.Command, args []string) error {
		err := root.MustWorkspaceClient(cmd, args)
		if err == nil || errors.Is(err, root.ErrAlreadyPrinted) {
			return err
		}
		return authError(cmd.Context(), cmd, err)
	}

	cmd.RunE = func(cmd *cobra.Command, args []string) error {
		ctx := cmd.Context()

		dockerImageURL := strings.TrimSpace(args[0])
		if dockerImageURL == "" {
			return renderError(ctx, cmd, "INVALID_ARGS", "PERMANENT", false,
				errors.New("IMAGE_URL cannot be empty"))
		}

		if err := validateTagPolicy(tagPolicy); err != nil {
			return renderError(ctx, cmd, "INVALID_ARGS", "PERMANENT", false, err)
		}

		if timeoutMinutes <= 0 {
			return renderError(ctx, cmd, "INVALID_ARGS", "PERMANENT", false,
				fmt.Errorf("--timeout-minutes must be positive, got %d", timeoutMinutes))
		}

		w := cmdctx.WorkspaceClient(ctx)

		// Validate authentication against the workspace before registering
		// anything: MustWorkspaceClient only attaches credentials, so without this
		// a bad credential surfaces as a confusing mid-flow failure.
		if _, err := w.CurrentUser.Me(ctx, iam.MeRequest{}); err != nil {
			return authError(ctx, cmd, err)
		}

		c, err := newImageClient(w)
		if err != nil {
			return renderError(ctx, cmd, "INTERNAL_ERROR", "TRANSIENT", true, err)
		}

		timeout := time.Duration(timeoutMinutes) * time.Minute

		// credErr is not fatal on its own: registration proceeds without
		// credentials (a public image still succeeds) and registrationError reports
		// it only if the registry rejects anonymous access.
		creds, credErr := discoverCredentials(ctx, w, c, dockerImageURL)
		if credErr != nil {
			log.Debugf(ctx, "could not store local Docker credentials: %v", credErr)
		}

		updated, sha, err := registerWithCredentialFallback(ctx, c, dockerImageURL, creds, timeout)
		if err != nil {
			kind, retryable := classifyRegistrationError(err)
			return renderError(ctx, cmd, "REGISTRATION_FAILED", kind, retryable,
				registrationError(dockerImageURL, err, credErr))
		}

		return renderRegisterResult(ctx, cmd, dockerImageURL,
			registerImageResult{
				DockerImageURL: dockerImageURL,
				ManifestSHA256: sha,
				Status:         string(imageStatusAvailable),
				ImageUpdated:   updated,
				Cached:         !updated,
			})
	}

	return cmd
}

// imageCredentials is a secret reference passed to registration. discovered
// marks it as auto-discovered from the local Docker config rather than
// configured by the user, which decides whether a rejection may be retried
// anonymously.
type imageCredentials struct {
	scope      string
	key        string
	discovered bool
}

// discoverCredentials resolves registry credentials from the local Docker config
// and stores them in a per-user secret, returning the reference for registration.
// It first probes whether the image is public: if so, no credentials are stored
// (avoiding a throwaway secret). Returns empty credentials when the image is
// public or no local credentials exist, both with a nil error. A non-nil error
// means credentials were found but could not be stored (e.g. the user lacks
// permission to create a secret scope); it is advisory, so the caller can still
// attempt an anonymous registration and report this as the cause if that fails.
func discoverCredentials(ctx context.Context, w *databricks.WorkspaceClient, c *imageClient, dockerImageURL string) (creds imageCredentials, err error) {
	// readDockerCredentials keys off the registry host, so it needs the normalized
	// URL (e.g. bare "ubuntu" resolves to the Docker Hub host).
	normalized := normalizeDockerImageURL(dockerImageURL)

	// Resolve local creds once (a cheap file read, or one credential-helper call),
	// so the no-`docker login` case pays for nothing and helpers aren't invoked
	// twice.
	username, password, ok := readDockerCredentials(ctx, normalized)
	if !ok {
		return imageCredentials{}, nil
	}

	if public := c.checkImageAccess(ctx, dockerImageURL); public != nil && *public {
		log.Infof(ctx, "image is publicly accessible; skipping local Docker credentials")
		return imageCredentials{}, nil
	}

	scope, key, err := storeDockerCredentials(ctx, w, normalized, username, password)
	if err != nil {
		return imageCredentials{}, err
	}
	log.Infof(ctx, "using Docker credentials from local config (stored as %s/%s)", scope, key)
	return imageCredentials{scope: scope, key: key, discovered: true}, nil
}

// isAuthError reports whether err is an authentication or permission failure,
// used to decide whether stale auto-discovered credentials warrant an anonymous
// retry. Unlike the Python CLI's substring match on "401"/"403", this keys off
// the SDK's typed sentinels.
func isAuthError(err error) bool {
	return errors.Is(err, apierr.ErrUnauthenticated) || errors.Is(err, apierr.ErrPermissionDenied)
}

// classifyRegistrationError maps a registration failure to the error envelope's
// kind and retryable flag. Only errors we can positively identify as transient
// (rate limits, server-side blips, a poll timeout) are retryable; auth,
// not-found, bad-request, conflict, a terminal FAILED upload, and any
// unclassified error default to permanent, so a consumer never retries a request
// that can't succeed.
func classifyRegistrationError(err error) (kind string, retryable bool) {
	switch {
	case errors.Is(err, errImageWaitTimeout),
		errors.Is(err, apierr.ErrTooManyRequests),
		errors.Is(err, apierr.ErrTemporarilyUnavailable),
		errors.Is(err, apierr.ErrInternalError),
		errors.Is(err, apierr.ErrDeadlineExceeded):
		return "TRANSIENT", true
	default:
		return "PERMANENT", false
	}
}

// registrationError wraps an auth failure with actionable guidance. When
// credentials were found locally but could not be stored, credErr is the real
// cause — telling the user to `docker login` would be wrong, since they already
// have working credentials.
func registrationError(dockerImageURL string, err, credErr error) error {
	if !isAuthError(err) {
		return err
	}
	if credErr != nil {
		return fmt.Errorf("image %q requires credentials, and the credentials found in your local Docker config could not be stored: %w", dockerImageURL, credErr)
	}
	return fmt.Errorf("image %q was not found or requires credentials: run `docker login` for its registry, then retry: %w", dockerImageURL, err)
}

// registerWithCredentialFallback registers the image and, if auto-discovered
// credentials are rejected as an auth failure, retries once anonymously so a
// public image isn't blocked by stale local creds (e.g. a revoked PAT from an old
// `docker login`). Credentials the user configured explicitly are never retried
// away: they asked for those specifically, so the rejection is the real answer.
func registerWithCredentialFallback(ctx context.Context, c *imageClient, dockerImageURL string, creds imageCredentials, timeout time.Duration) (updated bool, sha string, err error) {
	updated, sha, err = resolveImage(ctx, c, dockerImageURL, creds.scope, creds.key, timeout)
	if err != nil && creds.discovered && isAuthError(err) {
		log.Warnf(ctx, "Docker credentials discovered from your local config were rejected (%v); retrying without credentials in case the image is public", err)
		return resolveImage(ctx, c, dockerImageURL, "", "", timeout)
	}
	return updated, sha, err
}

// resolveImage always re-registers the image and waits for it to become
// AVAILABLE, returning whether the stored digest changed and the final digest.
// CreateImage is idempotent. The prior registration is fetched only to detect a
// digest change; its status is not consulted.
func resolveImage(ctx context.Context, c *imageClient, dockerImageURL, scope, key string, timeout time.Duration) (updated bool, sha string, err error) {
	existing, err := c.getImage(ctx, dockerImageURL)
	if err != nil {
		return false, "", err
	}

	reg, err := createAndWait(ctx, c, dockerImageURL, scope, key, timeout)
	if err != nil {
		return false, "", err
	}

	newSHA := reg.ManifestSHA256
	// Re-read to pick up a digest populated by the background upload.
	if final, err := c.getImage(ctx, dockerImageURL); err == nil && final != nil && final.ManifestSHA256 != "" {
		newSHA = final.ManifestSHA256
	}

	// A first-time registration, or a changed digest, counts as updated.
	if existing == nil {
		return true, newSHA, nil
	}
	cachedSHA := existing.ManifestSHA256
	if newSHA == "" {
		newSHA = cachedSHA
	}
	return cachedSHA != "" && newSHA != "" && cachedSHA != newSHA, newSHA, nil
}

// createAndWait registers the image and polls until it becomes AVAILABLE.
func createAndWait(ctx context.Context, c *imageClient, dockerImageURL, scope, key string, timeout time.Duration) (*imageRegistration, error) {
	reg, err := c.createImage(ctx, dockerImageURL, scope, key)
	if err != nil {
		return nil, err
	}
	if reg.Status == imageStatusAvailable {
		return reg, nil
	}
	return c.waitForImageReady(ctx, dockerImageURL, timeout, imagePollInterval)
}

// renderRegisterResult prints the result as a JSON envelope or human-readable
// text, matching the Python CLI's output.
func renderRegisterResult(ctx context.Context, cmd *cobra.Command, dockerImageURL string, result registerImageResult) error {
	if root.OutputType(cmd) != flags.OutputText {
		return renderEnvelope(ctx, result)
	}

	out := cmd.OutOrStdout()
	sha := "unknown"
	if result.ManifestSHA256 != "" {
		sha = shortManifestSHA(result.ManifestSHA256)
	}

	if result.ImageUpdated {
		fmt.Fprintf(out, "Image registered: %s\n", sha)
	} else {
		fmt.Fprintf(out, "Image already up to date: %s\n", sha)
	}

	fmt.Fprintln(out, "\nTo use this image in your training config:")
	fmt.Fprintln(out, "  environment:")
	fmt.Fprintln(out, "    docker_image:")
	fmt.Fprintf(out, "      url: %s\n", dockerImageURL)
	return nil
}

// shortManifestSHA truncates a manifest digest to its first 16 characters for
// display, matching the Python CLI.
func shortManifestSHA(sha string) string {
	if len(sha) <= 16 {
		return sha
	}
	return sha[:16] + "..."
}
