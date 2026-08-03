package aircmd

import (
	"context"
	"errors"
	"fmt"
	"time"

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
	scope, key := img.CredentialsScope, img.CredentialsKey
	var credErr error
	if scope == "" {
		scope, key, credErr = discoverCredentials(ctx, w, c, img.URL)
		if credErr != nil {
			log.Debugf(ctx, "could not store local Docker credentials: %v", credErr)
		}
	}

	log.Infof(ctx, "re-resolving %s against the source registry (tag_policy=latest)", img.URL)
	if _, _, err := registerWithCredentialFallback(ctx, c, img.URL, scope, key, imageReadyTimeout); err != nil {
		return registrationError(img.URL, err, credErr)
	}
	return nil
}

// waitForRegisteredImage requires an existing registration and blocks while it is
// still importing. Missing or FAILED is an error the user fixes by re-registering.
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
		log.Infof(ctx, "docker image registration in progress (%s); waiting for it to become available", reg.Status)
		final, err := c.waitForImageReady(ctx, dockerImageURL, imageReadyTimeout, imagePollInterval)
		if err != nil {
			return err
		}
		log.Infof(ctx, "image ready (%s)", digestDisplay(final.ManifestSHA256))
		return nil
	}

	// Unreachable: normalizeStatus maps every unknown state to PENDING.
	return errors.New("unexpected image status " + string(reg.Status))
}
