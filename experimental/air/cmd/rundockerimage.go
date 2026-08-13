package aircmd

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/databricks/cli/libs/cmdio"
	"github.com/databricks/cli/libs/log"
	"github.com/databricks/databricks-sdk-go"
)

// imageReadyTimeout is generous: replicating a large image takes minutes.
const imageReadyTimeout = time.Hour

// digestDisplay abbreviates a manifest digest for logs.
func digestDisplay(sha string) string {
	if sha == "" {
		return "digest unknown"
	}
	return shortManifestSHA(sha)
}

func notRegisteredError(dockerImageURL string) error {
	return fmt.Errorf("docker image not registered: %s\nregister it first: databricks experimental air register-image %s", dockerImageURL, dockerImageURL)
}

// prepareDockerImage verifies a run's custom image before submit, so a bad image
// fails here instead of deep in the launch stage where the cause is obscured.
func prepareDockerImage(ctx context.Context, w *databricks.WorkspaceClient, img *dockerImageConfig) error {
	c, err := newImageClient(w)
	if err != nil {
		return err
	}

	if img.wantsLatest() {
		if err := resolveLatestDockerImage(ctx, w, c, img); err != nil {
			return err
		}
	}

	return waitForRegisteredImage(ctx, c, img.URL)
}

// resolveLatestDockerImage re-registers the image so the run picks up the tag's
// newest digest, using the config's credentials when set and otherwise the local
// Docker config.
func resolveLatestDockerImage(ctx context.Context, w *databricks.WorkspaceClient, c *imageClient, img *dockerImageConfig) error {
	// Credentials the user configured are used as-is (discovered=false), so a
	// rejection is not retried anonymously and reports the secret they named.
	creds := imageCredentials{scope: img.CredentialsScope, key: img.CredentialsKey}
	var credErr error
	if creds.scope == "" {
		creds, credErr = discoverCredentials(ctx, w, c, img.URL)
		if credErr != nil {
			log.Debugf(ctx, "could not store local Docker credentials: %v", credErr)
		}
	}

	// Visible at the default log level (WARN): this can block for minutes.
	cmdio.LogString(ctx, fmt.Sprintf("Re-resolving %s against the source registry (tag_policy=latest)...", img.URL))
	if _, _, err := registerWithCredentialFallback(ctx, c, img.URL, creds, imageReadyTimeout); err != nil {
		if !creds.discovered && creds.scope != "" && isAuthError(err) {
			return fmt.Errorf("the credentials in secret %s/%s were rejected for image %q: %w", creds.scope, creds.key, img.URL, err)
		}
		return registrationError(img.URL, err, credErr)
	}
	return nil
}

// waitForRegisteredImage requires an existing registration and blocks while it is
// still importing. Missing or FAILED is an error the user fixes by re-registering.
//
// An AVAILABLE registration whose stored credentials have since lost registry
// access is accepted here and only fails at pod launch. The Python CLI catches
// that with a :validateImageAccess probe (cli/sdk/_submit.py), which is not ported
// deliberately: this check belongs in the backend, so every client gets it and
// the CLI doesn't pay a round trip per submit. Port it there rather than here.
func waitForRegisteredImage(ctx context.Context, c *imageClient, dockerImageURL string) error {
	reg, err := c.getImage(ctx, dockerImageURL)
	if err != nil {
		return err
	}
	if reg == nil {
		return notRegisteredError(dockerImageURL)
	}

	switch reg.Status {
	case imageStatusAvailable:
		log.Infof(ctx, "using image %s (%s)", dockerImageURL, digestDisplay(reg.ManifestSHA256))
		return nil
	case imageStatusFailed:
		msg := reg.StatusMessage
		if msg == "" {
			msg = "unknown error"
		}
		return fmt.Errorf("docker image registration failed: %s\nfix the issue and re-register: databricks experimental air register-image %s", msg, dockerImageURL)
	case imageStatusPending, imageStatusImporting:
		// This wait runs up to imageReadyTimeout, so it must be visible at the
		// default log level (WARN); log.Infof would be silent.
		cmdio.LogString(ctx, fmt.Sprintf("Docker image registration in progress (%s); waiting for it to become available...", reg.Status))
		final, err := c.waitForImageReady(ctx, dockerImageURL, imageReadyTimeout, imagePollInterval)
		if err != nil {
			return err
		}
		cmdio.LogString(ctx, "Image ready ("+digestDisplay(final.ManifestSHA256)+")")
		return nil
	}

	// Unreachable: normalizeStatus maps every unknown state to PENDING.
	return errors.New("unexpected image status " + string(reg.Status))
}
