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

// imagePolicy is the image resolution policy: reuse a cached image or always
// re-check the source registry for a newer digest.
type imagePolicy string

const (
	imagePolicyAuto   imagePolicy = "auto"
	imagePolicyLatest imagePolicy = "latest"
)

// parseImagePolicy resolves a --tag-policy string to an imagePolicy.
func parseImagePolicy(value string) (imagePolicy, error) {
	switch p := imagePolicy(strings.ToLower(strings.TrimSpace(value))); p {
	case imagePolicyAuto, imagePolicyLatest:
		return p, nil
	default:
		return "", fmt.Errorf("invalid image tag policy %q: valid options are auto, latest", value)
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
		scope           string
		key             string
		interactiveAuth bool
		tagPolicy       string
		timeoutMinutes  int
	)

	cmd := &cobra.Command{
		Use:   "register-image IMAGE_URL",
		Args:  root.ExactArgs(1),
		Short: "Mirror a Docker image into the workspace registry",
	}

	cmd.Flags().StringVar(&scope, "scope", "", "Databricks secret scope holding registry credentials")
	cmd.Flags().StringVar(&key, "key", "", "Databricks secret key holding registry credentials")
	cmd.Flags().BoolVarP(&interactiveAuth, "interactive-authenticate", "i", false, "Prompt for registry credentials and store them as a secret")
	cmd.Flags().StringVar(&tagPolicy, "tag-policy", "auto", "Image resolution policy: auto or latest")
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

		policy, err := parseImagePolicy(tagPolicy)
		if err != nil {
			return renderError(ctx, cmd, "INVALID_ARGS", "PERMANENT", false, err)
		}

		if (scope != "") != (key != "") {
			return renderError(ctx, cmd, "INVALID_ARGS", "PERMANENT", false,
				errors.New("both --scope and --key must be provided together"))
		}
		if interactiveAuth && (scope != "" || key != "") {
			return renderError(ctx, cmd, "INVALID_ARGS", "PERMANENT", false,
				errors.New("--interactive-authenticate cannot be used with --scope/--key"))
		}

		// Interactive credential setup and local Docker credential discovery are
		// not ported yet; reject rather than silently ignore the flag.
		if interactiveAuth {
			return renderError(ctx, cmd, "INVALID_ARGS", "PERMANENT", false,
				errors.New("--interactive-authenticate is not yet supported"))
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

		updated, sha, err := resolveImage(ctx, c, dockerImageURL, policy, scope, key, timeout)
		if err != nil {
			return renderError(ctx, cmd, "REGISTRATION_FAILED", "TRANSIENT", true, err)
		}

		return renderRegisterResult(ctx, cmd, dockerImageURL, scope, key,
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

// resolveImage registers an image according to policy and waits for it to
// become AVAILABLE, returning whether the stored image changed and its final
// manifest digest. It flattens the Python ImagePolicyHandler: an existing
// AVAILABLE image is reused under AUTO, while LATEST (or a non-AVAILABLE status)
// re-registers to pick up a newer digest. CreateImage is idempotent.
func resolveImage(ctx context.Context, c *imageClient, dockerImageURL string, policy imagePolicy, scope, key string, timeout time.Duration) (updated bool, sha string, err error) {
	existing, err := c.getImage(ctx, dockerImageURL)
	if err != nil {
		return false, "", err
	}

	if existing != nil {
		// Reuse the cached image unless it isn't AVAILABLE or LATEST forces a
		// re-check against the source registry.
		if existing.Status == imageStatusAvailable && policy != imagePolicyLatest {
			return false, existing.ManifestSHA256, nil
		}

		cachedSHA := existing.ManifestSHA256
		reg, err := createAndWait(ctx, c, dockerImageURL, scope, key, timeout)
		if err != nil {
			return false, "", err
		}
		newSHA := reg.ManifestSHA256
		shaChanged := cachedSHA != "" && newSHA != "" && cachedSHA != newSHA
		if newSHA == "" {
			newSHA = cachedSHA
		}
		// A FAILED image that now succeeds counts as updated even if the digest
		// is unchanged (or was never recorded).
		return shaChanged || existing.Status == imageStatusFailed, newSHA, nil
	}

	reg, err := createAndWait(ctx, c, dockerImageURL, scope, key, timeout)
	if err != nil {
		return false, "", err
	}
	// Re-read to ensure the digest is populated after the background upload.
	if final, err := c.getImage(ctx, dockerImageURL); err == nil && final != nil && final.ManifestSHA256 != "" {
		return true, final.ManifestSHA256, nil
	}
	return true, reg.ManifestSHA256, nil
}

// createAndWait registers the image and, if it isn't immediately AVAILABLE,
// polls until it becomes ready.
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
func renderRegisterResult(ctx context.Context, cmd *cobra.Command, dockerImageURL, scope, key string, result registerImageResult) error {
	if root.OutputType(cmd) != flags.OutputText {
		return renderEnvelope(ctx, result)
	}

	out := cmd.OutOrStdout()
	sha := "unknown"
	if result.ManifestSHA256 != "" {
		sha = shortManifestSHA(result.ManifestSHA256)
	}

	switch {
	case result.Cached && !result.ImageUpdated && scope == "":
		fmt.Fprintf(out, "Image already registered and available: %s\n", sha)
	case result.ImageUpdated:
		fmt.Fprintf(out, "Image registered: %s\n", sha)
	default:
		fmt.Fprintf(out, "Using cached image: %s\n", sha)
	}

	fmt.Fprintln(out, "\nTo use this image in your training config:")
	fmt.Fprintln(out, "  environment:")
	fmt.Fprintln(out, "    docker_image:")
	fmt.Fprintf(out, "      url: %s\n", dockerImageURL)

	if scope != "" {
		fmt.Fprintln(out, "\nTo reuse these credentials:")
		fmt.Fprintf(out, "  air register-image %s --scope %s --key %s\n", dockerImageURL, scope, key)
	}
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
