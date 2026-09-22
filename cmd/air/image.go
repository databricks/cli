package aircmd

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"regexp"
	"strings"

	"github.com/databricks/cli/cmd/root"
	"github.com/databricks/cli/libs/cmdctx"
	"github.com/databricks/cli/libs/cmdio"
	"github.com/databricks/databricks-sdk-go"
	"github.com/databricks/databricks-sdk-go/config"
	"github.com/spf13/cobra"
)

const (
	maxArtifactNameLength = 255
	maxImageTagLength     = 128
)

var (
	catalogSchemaPattern = regexp.MustCompile(`^[a-z0-9_]+$`)
	artifactNamePattern  = regexp.MustCompile(`^[a-z0-9]+((_|__|-+)[a-z0-9]+)*$`)
	imageTagPattern      = regexp.MustCompile(`^[A-Za-z0-9_][A-Za-z0-9_.-]{0,127}$`)
)

type imageSetupOptions struct {
	source  string
	catalog string
	schema  string
	image   string
	region  string
	pull    bool
}

type resolvedImageSetupOptions struct {
	source   string
	catalog  string
	schema   string
	artifact string
	tag      string
	region   string
}

type imageSetupDeps struct {
	configureDocker func(context.Context, string, string) (string, error)
	lookPath        func(string) (string, error)
	resolveRegion   func(context.Context, *databricks.WorkspaceClient) (string, error)
	runCommand      func(context.Context, io.Reader, io.Writer, io.Writer, string, ...string) error
}

func defaultImageSetupDeps() imageSetupDeps {
	return imageSetupDeps{
		configureDocker: configureImageDocker,
		lookPath:        exec.LookPath,
		resolveRegion:   resolveImageRegion,
		runCommand:      runImageCommand,
	}
}

func configureImageDocker(ctx context.Context, profile, region string) (string, error) {
	executable, err := os.Executable()
	if err != nil {
		return "", fmt.Errorf("locate databricks executable: %w", err)
	}

	var output bytes.Buffer
	cmd := exec.CommandContext(ctx, executable, "auth", "docker", "configure", profile, "--region", region)
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

func registryHostFromDockerConfigureOutput(output string) (string, error) {
	const prefix = "Configured Docker credential helper for "
	for line := range strings.Lines(output) {
		if registryHost, found := strings.CutPrefix(strings.TrimSpace(line), prefix); found && strings.Contains(registryHost, ".container.") {
			return registryHost, nil
		}
	}
	return "", errors.New("auth docker configure did not report an Artifact Registry host")
}

func newImageCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "image",
		Short: "Manage container images for AI Runtime",
		RunE:  root.ReportUnknownSubcommand,
	}
	cmd.AddCommand(newImageSetupCommand())
	return cmd
}

func newImageSetupCommand() *cobra.Command {
	return newImageSetupCommandWithDeps(defaultImageSetupDeps())
}

func newImageSetupCommandWithDeps(deps imageSetupDeps) *cobra.Command {
	var opts imageSetupOptions
	cmd := &cobra.Command{
		Use:   "setup",
		Short: "Configure Docker authentication and push an image to Databricks Artifact Registry",
		Long: `Configure Docker authentication and push a container image to Databricks Artifact Registry.

The command detects the workspace region, configures Docker's Databricks credential
helper, and pushes the source image to catalog.schema.artifact:tag. Omitted image
details are prompted for when the terminal is interactive.`,
		Args: root.NoArgs,
		PreRunE: func(cmd *cobra.Command, args []string) error {
			if opts.region != "" {
				profileFlag := cmd.Flag("profile")
				if profileFlag != nil && profileFlag.Value.String() != "" {
					w := &databricks.WorkspaceClient{Config: &config.Config{Profile: profileFlag.Value.String()}}
					cmd.SetContext(cmdctx.SetWorkspaceClient(cmd.Context(), w))
					return nil
				}
			}
			return root.MustWorkspaceClient(cmd, args)
		},
		RunE: func(cmd *cobra.Command, _ []string) error {
			return runImageSetup(cmd, &opts, deps)
		},
	}

	cmd.Flags().StringVarP(&opts.source, "source", "s", "", "Source container image to push")
	cmd.Flags().StringVar(&opts.catalog, "catalog", "", "Unity Catalog catalog for the image; use lowercase letters, digits, and underscores")
	cmd.Flags().StringVar(&opts.schema, "schema", "", "Unity Catalog schema for the image; use lowercase letters, digits, and underscores")
	cmd.Flags().StringVar(&opts.image, "image", "", "Artifact name and optional tag in Databricks Artifact Registry; artifact names must be 255 characters or less and tags must be 128 characters or less")
	cmd.Flags().StringVar(&opts.region, "region", "", "Workspace region; detected automatically when omitted")
	cmd.Flags().BoolVar(&opts.pull, "pull", false, "Pull the source image even when it is already available locally")
	return cmd
}

func runImageSetup(cmd *cobra.Command, opts *imageSetupOptions, deps imageSetupDeps) error {
	ctx := cmd.Context()
	cmdio.LogString(ctx, "Warning: This feature is in Preview. APIs may change, and the workspace must enable the Preview features.")
	w := cmdctx.WorkspaceClient(ctx)

	resolved, err := resolveImageSetupOptions(ctx, w, opts, deps.resolveRegion)
	if err != nil {
		return err
	}
	if w.Config.Profile == "" {
		return errors.New("air image setup requires a workspace profile so Docker can refresh credentials; authenticate with 'databricks auth login' and pass --profile")
	}
	dockerPath, err := deps.lookPath("docker")
	if err != nil {
		return fmt.Errorf("find Docker on PATH: %w", err)
	}

	registryHost, err := deps.configureDocker(ctx, w.Config.Profile, resolved.region)
	if err != nil {
		return fmt.Errorf("configure Docker authentication: %w", err)
	}

	target := fmt.Sprintf("%s/%s.%s.%s:%s", registryHost, resolved.catalog, resolved.schema, resolved.artifact, resolved.tag)
	if opts.pull || !imageExistsLocally(ctx, deps.runCommand, dockerPath, resolved.source) {
		cmdio.LogProgress(ctx, "Pulling source image "+resolved.source)
		if err := deps.runCommand(ctx, cmd.InOrStdin(), cmd.OutOrStdout(), cmd.ErrOrStderr(), dockerPath, "pull", resolved.source); err != nil {
			return fmt.Errorf("pull source image %s: %w", resolved.source, err)
		}
	}

	cmdio.LogProgress(ctx, "Tagging image as "+target)
	if err := deps.runCommand(ctx, cmd.InOrStdin(), cmd.OutOrStdout(), cmd.ErrOrStderr(), dockerPath, "tag", resolved.source, target); err != nil {
		return fmt.Errorf("tag source image %s: %w", resolved.source, err)
	}
	cmdio.LogProgress(ctx, "Pushing image to "+target)
	if err := deps.runCommand(ctx, cmd.InOrStdin(), cmd.OutOrStdout(), cmd.ErrOrStderr(), dockerPath, "push", target); err != nil {
		return fmt.Errorf("push image %s: %w", target, err)
	}

	unityCatalogReference := fmt.Sprintf("%s.%s.%s:%s", resolved.catalog, resolved.schema, resolved.artifact, resolved.tag)
	cmdio.LogString(ctx, "Pushed image.")
	_, err = fmt.Fprintf(
		cmd.OutOrStdout(),
		"Now you can set `environment.unity_catalog_image = %s` in your YAML config and create a workload with `databricks air run -f config.yaml`.\n",
		unityCatalogReference,
	)
	return err
}

func resolveImageSetupOptions(
	ctx context.Context,
	w *databricks.WorkspaceClient,
	opts *imageSetupOptions,
	resolveRegion func(context.Context, *databricks.WorkspaceClient) (string, error),
) (resolvedImageSetupOptions, error) {
	source, err := promptImageSetupValue(ctx, opts.source, "Source image", "", "--source")
	if err != nil {
		return resolvedImageSetupOptions{}, err
	}
	catalog, err := promptImageSetupValue(ctx, opts.catalog, "Unity Catalog catalog", "", "--catalog")
	if err != nil {
		return resolvedImageSetupOptions{}, err
	}
	if !catalogSchemaPattern.MatchString(catalog) {
		return resolvedImageSetupOptions{}, fmt.Errorf("invalid catalog %q: use only lowercase letters, digits, and underscores", catalog)
	}
	schema, err := promptImageSetupValue(ctx, opts.schema, "Unity Catalog schema", "", "--schema")
	if err != nil {
		return resolvedImageSetupOptions{}, err
	}
	if !catalogSchemaPattern.MatchString(schema) {
		return resolvedImageSetupOptions{}, fmt.Errorf("invalid schema %q: use only lowercase letters, digits, and underscores", schema)
	}

	defaultArtifact, defaultTag := sourceImageDefaults(source)
	artifact, tag, err := resolveArtifactAndTag(ctx, opts.image, defaultArtifact, defaultTag)
	if err != nil {
		return resolvedImageSetupOptions{}, err
	}
	region, err := resolveImageSetupRegion(ctx, w, opts.region, resolveRegion)
	if err != nil {
		return resolvedImageSetupOptions{}, err
	}

	return resolvedImageSetupOptions{
		source:   source,
		catalog:  catalog,
		schema:   schema,
		artifact: artifact,
		tag:      tag,
		region:   region,
	}, nil
}

func promptImageSetupValue(ctx context.Context, value, label, defaultValue, flag string) (string, error) {
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

func resolveArtifactAndTag(ctx context.Context, image, defaultArtifact, defaultTag string) (string, string, error) {
	artifact := image
	tag := defaultTag
	if image == "" {
		var err error
		artifact, err = promptImageSetupValue(ctx, "", "Artifact name", defaultArtifact, "--image")
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

	if len(artifact) > maxArtifactNameLength {
		return "", "", fmt.Errorf("invalid artifact name: must be %d characters or less", maxArtifactNameLength)
	}
	if !artifactNamePattern.MatchString(artifact) {
		return "", "", fmt.Errorf("invalid artifact name %q: use lowercase letters and digits separated by one or two underscores or one or more hyphens", artifact)
	}
	if tag == "" {
		tag = "latest"
	}
	if len(tag) > maxImageTagLength {
		return "", "", fmt.Errorf("invalid image tag: must be %d characters or less", maxImageTagLength)
	}
	if !imageTagPattern.MatchString(tag) {
		return "", "", fmt.Errorf("invalid image tag %q", tag)
	}
	return artifact, tag, nil
}

func sourceImageDefaults(source string) (string, string) {
	repository := source
	tag := "latest"
	lastSlash := strings.LastIndex(source, "/")
	if lastColon := strings.LastIndex(source, ":"); lastColon > lastSlash {
		repository = source[:lastColon]
		tag = source[lastColon+1:]
	}
	artifact := repository[strings.LastIndex(repository, "/")+1:]
	if !artifactNamePattern.MatchString(artifact) {
		artifact = ""
	}
	return artifact, tag
}

func resolveImageSetupRegion(
	ctx context.Context,
	w *databricks.WorkspaceClient,
	region string,
	resolveRegion func(context.Context, *databricks.WorkspaceClient) (string, error),
) (string, error) {
	if region != "" {
		return region, nil
	}
	region, err := resolveRegion(ctx, w)
	if err == nil && region != "" {
		return region, nil
	}
	if !cmdio.IsPromptSupported(ctx) {
		if err == nil {
			err = errors.New("metastore summary did not include a region")
		}
		return "", fmt.Errorf("could not detect the workspace region; pass --region: %w", err)
	}
	return promptImageSetupValue(ctx, "", "Workspace region", "", "--region")
}

func resolveImageRegion(ctx context.Context, w *databricks.WorkspaceClient) (string, error) {
	summary, err := w.Metastores.Summary(ctx)
	if err != nil {
		return "", fmt.Errorf("get metastore summary: %w", err)
	}
	if summary.Region == "" {
		return "", errors.New("metastore summary did not include a region")
	}
	return summary.Region, nil
}

func imageExistsLocally(
	ctx context.Context,
	runCommand func(context.Context, io.Reader, io.Writer, io.Writer, string, ...string) error,
	dockerPath, source string,
) bool {
	return runCommand(ctx, nil, io.Discard, io.Discard, dockerPath, "image", "inspect", source) == nil
}

func runImageCommand(ctx context.Context, in io.Reader, out, errOut io.Writer, executable string, args ...string) error {
	cmd := exec.CommandContext(ctx, executable, args...)
	cmd.Stdin = in
	cmd.Stdout = out
	cmd.Stderr = errOut
	return cmd.Run()
}
