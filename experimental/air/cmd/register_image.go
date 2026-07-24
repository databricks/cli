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

		updated, sha, err := resolveImage(ctx, c, dockerImageURL, timeout)
		if err != nil {
			return renderError(ctx, cmd, "REGISTRATION_FAILED", "TRANSIENT", true, err)
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

// resolveImage always re-registers the image and waits for it to become
// AVAILABLE, returning whether the stored digest changed and the final digest.
// CreateImage is idempotent. The prior registration is fetched only to detect a
// digest change; its status is not consulted.
func resolveImage(ctx context.Context, c *imageClient, dockerImageURL string, timeout time.Duration) (updated bool, sha string, err error) {
	existing, err := c.getImage(ctx, dockerImageURL)
	if err != nil {
		return false, "", err
	}

	reg, err := createAndWait(ctx, c, dockerImageURL, timeout)
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
// Credentials are omitted; private-image support is added with local Docker
// credential discovery in a later phase.
func createAndWait(ctx context.Context, c *imageClient, dockerImageURL string, timeout time.Duration) (*imageRegistration, error) {
	reg, err := c.createImage(ctx, dockerImageURL, "", "")
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
