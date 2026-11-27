package aircmd

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"regexp"
	"strconv"
	"strings"
	"sync"

	"github.com/databricks/cli/cmd/root"
	"github.com/databricks/cli/libs/aitools/agents"
	"github.com/databricks/cli/libs/cmdctx"
	"github.com/databricks/cli/libs/cmdio"
	"github.com/databricks/cli/libs/flags"
	"github.com/databricks/databricks-sdk-go/config"
	"github.com/spf13/cobra"
)

const (
	maxImageRepositoryLength = 255
	maxImageTagLength        = 128
	airImagePlatform         = "linux/amd64"
)

var (
	// DAR uses the OCI repository-component grammar without dots or slashes,
	// which are forbidden in Unity Catalog names.
	artifactNamePattern = sync.OnceValue(func() *regexp.Regexp { return regexp.MustCompile(`^[a-z0-9]+((_|__|-+)[a-z0-9]+)*$`) })
	imageTagPattern     = sync.OnceValue(func() *regexp.Regexp { return regexp.MustCompile(`^[A-Za-z0-9_][A-Za-z0-9_.-]{0,127}$`) })
)

type imagePushOptions struct {
	source   string
	catalog  string
	schema   string
	artifact string
	pull     bool
}

type resolvedImagePushOptions struct {
	source   string
	catalog  string
	schema   string
	artifact string
	tag      string
}

type imagePushDeps struct {
	configureDocker func(context.Context, string) (string, error)
	lookPath        func(string) (string, error)
	runCommand      func(context.Context, io.Reader, io.Writer, io.Writer, string, ...string) error
}

// defaultImagePushDeps connects image-push operations to the real CLI and Docker executables.
func defaultImagePushDeps() imagePushDeps {
	return imagePushDeps{
		configureDocker: configureImageDocker,
		lookPath:        exec.LookPath,
		runCommand:      runImageCommand,
	}
}

// configureImageDocker configures authentication through the CLI and returns the resolved registry host.
func configureImageDocker(ctx context.Context, profile string) (string, error) {
	executable, err := os.Executable()
	if err != nil {
		return "", fmt.Errorf("locate databricks executable: %w", err)
	}

	var output bytes.Buffer
	cmd := exec.CommandContext(ctx, executable, "auth", "docker", "configure", profile)
	cmd.Env = agents.SelfExecEnv(os.Environ())
	cmd.Stdout = &output
	cmd.Stderr = &output
	if err := cmd.Run(); err != nil {
		if details := strings.TrimSpace(output.String()); details != "" {
			return "", fmt.Errorf("run auth docker configure: %w: %s", err, details)
		}
		return "", fmt.Errorf("run auth docker configure: %w", err)
	}

	registryHost, err := registryHostFromDockerConfigureOutput(output.String())
	if err != nil {
		return "", err
	}
	if message := strings.TrimSuffix(output.String(), "\n"); message != "" {
		cmdio.LogString(ctx, message)
	}
	return registryHost, nil
}

// registryHostFromDockerConfigureOutput extracts the DAR host printed by auth docker configure.
func registryHostFromDockerConfigureOutput(output string) (string, error) {
	const prefix = "Configured Docker credential helper for "
	for line := range strings.Lines(output) {
		if registryHost, found := strings.CutPrefix(strings.TrimSpace(line), prefix); found && strings.Contains(registryHost, ".container.") {
			return registryHost, nil
		}
	}
	return "", errors.New("auth docker configure did not report an Artifact Registry host")
}

// newImagesCommand creates the AIR images command group and registers its subcommands.
func newImagesCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "images",
		Short: "Push container images for custom environments",
		RunE:  root.ReportUnknownSubcommand,
	}
	cmd.AddCommand(newImagePushCommand())
	return cmd
}

// newImagePushCommand creates the push command with production dependencies.
func newImagePushCommand() *cobra.Command {
	return newImagePushCommandWithDeps(defaultImagePushDeps())
}

// newImagePushCommandWithDeps defines push flags and handlers with injectable dependencies.
func newImagePushCommandWithDeps(deps imagePushDeps) *cobra.Command {
	var opts imagePushOptions
	cmd := &cobra.Command{
		Use:   "push",
		Short: "Push a container image to Databricks Artifact Registry",
		Long: `For feature status and requirements, see:
  AWS: https://docs.databricks.com/aws/en/machine-learning/ai-runtime/connecting/cli/docker-images
  Azure: https://learn.microsoft.com/azure/databricks/machine-learning/ai-runtime/connecting/cli/docker-images

Push a container image to Databricks Artifact Registry under the specified Unity Catalog catalog and schema.

The command configures Docker's Databricks credential helper, automatically detects
the registry region, and pushes the source image to catalog.schema.artifact:tag.
Omitted image details are prompted for when the terminal is interactive.

AIR requires linux/amd64 images. A compatible local image is reused; missing images
are pulled for linux/amd64. Use --pull to refresh an existing local image.
The destination --artifact ARTIFACT[:TAG] defaults to the source image name and tag.`,
		Args:    root.NoArgs,
		PreRunE: root.MustWorkspaceClient,
		RunE: func(cmd *cobra.Command, _ []string) error {
			return runImagePush(cmd, &opts, deps)
		},
	}

	cmd.Flags().StringVarP(&opts.source, "source", "s", "", "Source container image to push (NAME[:TAG] or NAME@DIGEST)")
	cmd.Flags().StringVar(&opts.catalog, "catalog", "", "Destination Unity Catalog catalog; use lowercase letters and digits separated by one or two underscores or one or more hyphens")
	cmd.Flags().StringVar(&opts.schema, "schema", "", "Destination Unity Catalog schema; use lowercase letters and digits separated by one or two underscores or one or more hyphens")
	cmd.Flags().StringVar(&opts.artifact, "artifact", "", "Destination ARTIFACT[:TAG]; defaults to the source image name and tag (latest if untagged); maximum 255 characters for catalog.schema.artifact and 128 for the tag")
	cmd.Flags().BoolVar(&opts.pull, "pull", false, "Pull the source image even when it is already available locally")
	return cmd
}

// formatImageSize renders a byte count in decimal MB or GB, the units docker images uses.
func formatImageSize(sizeBytes int64) string {
	if sizeBytes < 1e9 {
		return fmt.Sprintf("%.1f MB", float64(sizeBytes)/1e6)
	}
	return fmt.Sprintf("%.1f GB", float64(sizeBytes)/1e9)
}

// logCompressedImageSize prints the compressed size and layer count from the pushed manifest of the
// given platform. The registry stores compressed layers; the local image store only knows the uncompressed size.
func logCompressedImageSize(ctx context.Context, deps imagePushDeps, dockerPath, target, platform string) {
	var manifest, errOut bytes.Buffer
	if err := deps.runCommand(ctx, nil, &manifest, &errOut, dockerPath, "manifest", "inspect", "--verbose", target); err != nil {
		detail, _, _ := strings.Cut(strings.TrimSpace(errOut.String()), "\n")
		if detail == "" {
			detail = err.Error()
		}
		cmdio.LogString(ctx, fmt.Sprintf("Warning: could not read the compressed size of %s: %s", target, detail))
		return
	}
	sizeBytes, layers, err := compressedImageSize(manifest.Bytes(), platform)
	if err != nil {
		cmdio.LogString(ctx, fmt.Sprintf("Warning: could not read the compressed size of %s: %v", target, err))
		return
	}
	cmdio.LogString(ctx, fmt.Sprintf("Compressed image size: %s (layers: %d).", formatImageSize(sizeBytes), layers))
}

type manifestLayers struct {
	Layers []struct {
		Size int64 `json:"size"`
	} `json:"layers"`
}

type verboseManifest struct {
	Descriptor struct {
		Platform *struct {
			OS           string `json:"os"`
			Architecture string `json:"architecture"`
		} `json:"platform"`
	}
	SchemaV2Manifest *manifestLayers
	OCIManifest      *manifestLayers
}

// compressedImageSize sums the layer sizes in `docker manifest inspect --verbose` output. That
// output is one object for an image manifest and an array for an index, whose entry for platform is used.
// The JSON is Docker CLI's ImageManifest (https://github.com/docker/cli/blob/master/cli/manifest/types/types.go),
// printed by https://github.com/docker/cli/blob/master/cli/command/manifest/inspect.go.
func compressedImageSize(output []byte, platform string) (int64, int, error) {
	output = bytes.TrimSpace(output)
	var entries []verboseManifest
	if bytes.HasPrefix(output, []byte("[")) {
		if err := json.Unmarshal(output, &entries); err != nil {
			return 0, 0, err
		}
	} else {
		var entry verboseManifest
		if err := json.Unmarshal(output, &entry); err != nil {
			return 0, 0, err
		}
		entries = []verboseManifest{entry}
	}
	for _, e := range entries {
		if p := e.Descriptor.Platform; p != nil && p.OS+"/"+p.Architecture != platform {
			continue
		}
		m := e.OCIManifest
		if m == nil {
			m = e.SchemaV2Manifest
		}
		if m == nil || len(m.Layers) == 0 {
			continue
		}
		var sizeBytes int64
		for _, l := range m.Layers {
			sizeBytes += l.Size
		}
		return sizeBytes, len(m.Layers), nil
	}
	return 0, 0, fmt.Errorf("no %s image manifest with layers", platform)
}

// darLimitationsURL returns the DAR limitations docs for the workspace's cloud.
func darLimitationsURL(cfg *config.Config) string {
	switch {
	case cfg.IsAzure():
		return "https://learn.microsoft.com/azure/databricks/artifact-registry/#limitations"
	case cfg.IsGcp():
		return "https://docs.databricks.com/gcp/en/artifact-registry/#limitations"
	default:
		return "https://docs.databricks.com/aws/en/artifact-registry/#limitations"
	}
}

// runImagePush validates inputs, prepares a compatible source image, and pushes it to DAR.
func runImagePush(cmd *cobra.Command, opts *imagePushOptions, deps imagePushDeps) error {
	ctx := cmd.Context()
	w := cmdctx.WorkspaceClient(ctx)

	resolved, err := resolveImagePushOptions(ctx, opts)
	if err != nil {
		return err
	}
	if w.Config.Profile == "" {
		return errors.New("air images push requires a workspace profile so Docker can refresh credentials; authenticate with 'databricks auth login' and pass --profile")
	}
	dockerPath, err := deps.lookPath("docker")
	if err != nil {
		return fmt.Errorf("find Docker on PATH: %w", err)
	}

	unityCatalogReference := fmt.Sprintf("%s.%s.%s:%s", resolved.catalog, resolved.schema, resolved.artifact, resolved.tag)
	cmdio.LogString(ctx, fmt.Sprintf("Pushing %s as %s using profile %s", resolved.source, unityCatalogReference, w.Config.Profile))

	registryHost, err := deps.configureDocker(ctx, w.Config.Profile)
	if err != nil {
		return fmt.Errorf("configure Docker authentication: %w", err)
	}
	target := fmt.Sprintf("%s/%s", registryHost, unityCatalogReference)

	pull := opts.pull
	if !pull {
		exists, err := imageExistsLocally(ctx, deps.runCommand, dockerPath, resolved.source, cmd.ErrOrStderr())
		if err != nil {
			return err
		}
		pull = !exists
	}
	if pull {
		cmdio.LogProgress(ctx, "Pulling source image "+resolved.source+" for "+airImagePlatform)
		if err := deps.runCommand(ctx, cmd.InOrStdin(), cmd.ErrOrStderr(), cmd.ErrOrStderr(), dockerPath, "pull", "--platform", airImagePlatform, resolved.source); err != nil {
			return fmt.Errorf("pull source image %s: %w", resolved.source, err)
		}
	} else {
		cmdio.LogString(ctx, "Using local source image "+resolved.source)
	}

	var inspected bytes.Buffer
	// {{with .Config}} keeps an image with no config from failing the template (https://docs.docker.com/engine/cli/formatting/).
	// WorkingDir comes last because it is the only field that may itself contain the separator.
	if err := deps.runCommand(ctx, nil, &inspected, cmd.ErrOrStderr(), dockerPath, "image", "inspect", "--format", "{{.Os}}/{{.Architecture}}|{{.Size}}|{{with .Config}}{{.WorkingDir}}{{end}}", resolved.source); err != nil {
		return fmt.Errorf("inspect source image %s: %w", resolved.source, err)
	}
	platform, rest, _ := strings.Cut(strings.TrimSpace(inspected.String()), "|")
	size, workingDir, _ := strings.Cut(rest, "|")
	if platform != airImagePlatform {
		return fmt.Errorf("source image %s has platform %q; AIR requires %s: use --pull to fetch a compatible image, or rebuild the source for %s", resolved.source, platform, airImagePlatform, airImagePlatform)
	}
	// Sizes are informational only: a size that can't be read never blocks the push.
	if sizeBytes, err := strconv.ParseInt(size, 10, 64); err == nil {
		cmdio.LogString(ctx, fmt.Sprintf("Source image %s is %s uncompressed.", resolved.source, formatImageSize(sizeBytes)))
	} else {
		cmdio.LogString(ctx, fmt.Sprintf("Warning: could not read the uncompressed size of source image %s (got %q).", resolved.source, size))
	}
	// With no WORKDIR or WORKDIR /, relative paths don't depend on a directory the image chose.
	if workingDir != "" && workingDir != "/" {
		cmdio.LogString(ctx, fmt.Sprintf("Warning: source image %s sets WORKDIR %s. Use absolute paths in your command and scripts. See how AI Runtime command paths work:\n"+
			"  AWS: https://docs.databricks.com/aws/en/machine-learning/ai-runtime/connecting/cli/yaml-config#command\n"+
			"  Azure: https://learn.microsoft.com/azure/databricks/machine-learning/ai-runtime/connecting/cli/yaml-config#command", resolved.source, workingDir))
	}

	cmdio.LogProgress(ctx, "Tagging image as "+target)
	if err := deps.runCommand(ctx, cmd.InOrStdin(), cmd.ErrOrStderr(), cmd.ErrOrStderr(), dockerPath, "tag", resolved.source, target); err != nil {
		return fmt.Errorf("tag source image %s: %w", resolved.source, err)
	}
	// DAR sets and may change its own limits, so the CLI links them instead of enforcing one.
	limitsURL := darLimitationsURL(w.Config)
	cmdio.LogProgress(ctx, "Pushing image to "+target)
	if err := deps.runCommand(ctx, cmd.InOrStdin(), cmd.ErrOrStderr(), cmd.ErrOrStderr(), dockerPath, "push", target); err != nil {
		// A push can fail for many reasons, so the link is offered without naming size as the cause.
		cmdio.LogString(ctx, "If the registry rejected the image for its size, see the Databricks Artifact Registry image limits: "+limitsURL)
		return fmt.Errorf("push image %s: %w", target, err)
	}

	cmdio.LogString(ctx, "Pushed image.")
	logCompressedImageSize(ctx, deps, dockerPath, target, platform)
	cmdio.LogString(ctx, "For Databricks Artifact Registry image limits, see "+limitsURL)
	if root.OutputType(cmd) == flags.OutputJSON {
		return renderEnvelope(ctx, struct {
			UnityCatalogImage string `json:"unity_catalog_image"`
			RegistryImage     string `json:"registry_image"`
		}{UnityCatalogImage: unityCatalogReference, RegistryImage: target})
	}
	_, err = fmt.Fprintf(
		cmd.OutOrStdout(),
		"Now you can set `environment.unity_catalog_image = %s` in your YAML config and create a workload with `databricks air run -f config.yaml`.\n",
		unityCatalogReference,
	)
	return err
}

// resolveImagePushOptions fills in missing inputs and validates the full Unity Catalog destination.
func resolveImagePushOptions(
	ctx context.Context,
	opts *imagePushOptions,
) (resolvedImagePushOptions, error) {
	source, err := promptImagePushValue(ctx, opts.source, "Source image", "", "--source")
	if err != nil {
		return resolvedImagePushOptions{}, err
	}
	catalog, err := promptImagePushValue(ctx, opts.catalog, "Unity Catalog catalog", "", "--catalog")
	if err != nil {
		return resolvedImagePushOptions{}, err
	}
	if !artifactNamePattern().MatchString(catalog) {
		return resolvedImagePushOptions{}, fmt.Errorf("invalid catalog %q: use lowercase letters and digits separated by one or two underscores or one or more hyphens", catalog)
	}
	schema, err := promptImagePushValue(ctx, opts.schema, "Unity Catalog schema", "", "--schema")
	if err != nil {
		return resolvedImagePushOptions{}, err
	}
	if !artifactNamePattern().MatchString(schema) {
		return resolvedImagePushOptions{}, fmt.Errorf("invalid schema %q: use lowercase letters and digits separated by one or two underscores or one or more hyphens", schema)
	}

	defaultArtifact, defaultTag := sourceImageDefaults(source)
	artifact, tag, err := resolveArtifactAndTag(ctx, opts.artifact, defaultArtifact, defaultTag)
	if err != nil {
		return resolvedImagePushOptions{}, err
	}
	// Docker limits the repository path, including both dots but excluding the registry and tag.
	if len(catalog)+len(schema)+len(artifact)+2 > maxImageRepositoryLength {
		return resolvedImagePushOptions{}, fmt.Errorf("invalid image repository name: catalog.schema.artifact must be %d characters or less", maxImageRepositoryLength)
	}

	return resolvedImagePushOptions{
		source:   source,
		catalog:  catalog,
		schema:   schema,
		artifact: artifact,
		tag:      tag,
	}, nil
}

// promptImagePushValue prompts for a missing value or uses its default when prompting is unavailable.
func promptImagePushValue(ctx context.Context, value, label, defaultValue, flag string) (string, error) {
	if value != "" {
		return value, nil
	}
	if !cmdio.IsPromptSupported(ctx) {
		if defaultValue != "" {
			return defaultValue, nil
		}
		return "", fmt.Errorf("%s is required when prompting is unavailable", flag)
	}
	value, err := cmdio.Ask(ctx, label, defaultValue)
	if err != nil {
		return "", err
	}
	if value == "" {
		return "", fmt.Errorf("%s is required", flag)
	}
	return value, nil
}

// resolveArtifactAndTag resolves the destination artifact and tag, then validates their naming rules.
func resolveArtifactAndTag(ctx context.Context, image, defaultArtifact, defaultTag string) (string, string, error) {
	artifact := image
	tag := defaultTag
	if image == "" {
		var err error
		artifact, err = promptImagePushValue(ctx, "", "Artifact name", defaultArtifact, "--artifact")
		if err != nil {
			return "", "", err
		}
		if cmdio.IsPromptSupported(ctx) {
			tag, err = cmdio.Ask(ctx, "Tag", defaultTag)
			if err != nil {
				return "", "", err
			}
		}
	} else if before, after, found := strings.Cut(image, ":"); found {
		artifact = before
		tag = after
	}

	if !artifactNamePattern().MatchString(artifact) {
		return "", "", fmt.Errorf("invalid artifact name %q: use lowercase letters and digits separated by one or two underscores or one or more hyphens", artifact)
	}
	if tag == "" {
		tag = "latest"
	}
	if len(tag) > maxImageTagLength {
		return "", "", fmt.Errorf("invalid image tag: must be %d characters or less", maxImageTagLength)
	}
	if !imageTagPattern().MatchString(tag) {
		return "", "", fmt.Errorf("invalid image tag %q", tag)
	}
	return artifact, tag, nil
}

// sourceImageDefaults extracts artifact/tag defaults, leaving the artifact empty when its name is invalid.
func sourceImageDefaults(source string) (string, string) {
	source, _, _ = strings.Cut(source, "@")
	repository := source
	tag := "latest"
	lastSlash := strings.LastIndex(source, "/")
	if lastColon := strings.LastIndex(source, ":"); lastColon > lastSlash {
		repository = source[:lastColon]
		tag = source[lastColon+1:]
	}
	artifact := repository[strings.LastIndex(repository, "/")+1:]
	if !artifactNamePattern().MatchString(artifact) {
		artifact = ""
	}
	return artifact, tag
}

// imageExistsLocally checks Docker's local image list and propagates operational failures.
func imageExistsLocally(
	ctx context.Context,
	runCommand func(context.Context, io.Reader, io.Writer, io.Writer, string, ...string) error,
	dockerPath, source string,
	errOut io.Writer,
) (bool, error) {
	// Listing succeeds with empty output for a missing image, but fails for
	// operational errors such as an unavailable daemon. Inspect cannot distinguish them.
	// https://docs.docker.com/reference/cli/docker/image/ls/
	// Match Docker's implicit latest tag rather than listing every tag of a repository.
	if !strings.Contains(source, "@") && !strings.Contains(source[strings.LastIndex(source, "/")+1:], ":") {
		source += ":latest"
	}
	var output bytes.Buffer
	if err := runCommand(ctx, nil, &output, errOut, dockerPath, "image", "ls", "--quiet", source); err != nil {
		return false, fmt.Errorf("check local source image %s (ensure Docker is running and accessible): %w", source, err)
	}
	return strings.TrimSpace(output.String()) != "", nil
}

// runImageCommand runs a subprocess with the supplied context and I/O streams.
func runImageCommand(ctx context.Context, in io.Reader, out, errOut io.Writer, executable string, args ...string) error {
	cmd := exec.CommandContext(ctx, executable, args...)
	cmd.Stdin = in
	cmd.Stdout = out
	cmd.Stderr = errOut
	return cmd.Run()
}
