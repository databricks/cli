package aircmd

import (
	"bytes"
	"context"
	"errors"
	"io"
	"strings"
	"testing"

	"github.com/databricks/cli/libs/cmdctx"
	"github.com/databricks/cli/libs/cmdio"
	"github.com/databricks/cli/libs/flags"
	"github.com/databricks/databricks-sdk-go"
	"github.com/databricks/databricks-sdk-go/config"
	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type recordedImageCommand struct {
	executable string
	args       []string
}

func newImagePushTestCommand(t *testing.T, profile string) (*cobra.Command, *bytes.Buffer, *bytes.Buffer) {
	t.Helper()
	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}
	ctx := cmdio.InContext(t.Context(), cmdio.NewIO(t.Context(), flags.OutputText, strings.NewReader(""), stdout, stderr, "", ""))
	ctx = cmdctx.SetWorkspaceClient(ctx, &databricks.WorkspaceClient{Config: &config.Config{Profile: profile}})
	cmd := &cobra.Command{Use: "push"}
	cmd.SetContext(ctx)
	cmd.SetIn(strings.NewReader(""))
	cmd.SetOut(stdout)
	cmd.SetErr(stderr)
	return withOutput(cmd, flags.OutputText), stdout, stderr
}

func TestRunImagePushTagsAndPushesLocalImage(t *testing.T) {
	cmd, stdout, stderr := newImagePushTestCommand(t, "workspace")
	var commands []recordedImageCommand
	deps := imagePushDeps{
		configureDocker: func(_ context.Context, profile string) (string, error) {
			assert.Equal(t, "workspace", profile)
			return "123.container.us-west-2.cloud.databricks.test", nil
		},
		lookPath: func(name string) (string, error) {
			assert.Equal(t, "docker", name)
			return "/usr/bin/docker", nil
		},
		runCommand: func(_ context.Context, _ io.Reader, out, _ io.Writer, executable string, args ...string) error {
			commands = append(commands, recordedImageCommand{executable: executable, args: args})
			switch strings.Join(args[:2], " ") {
			case "image ls":
				_, err := io.WriteString(out, "image-id\n")
				return err
			case "image inspect":
				_, err := io.WriteString(out, "linux/amd64\n")
				return err
			}
			return nil
		},
	}

	err := runImagePush(cmd, &imagePushOptions{
		source:   "nvidia/cuda:13.4.1",
		catalog:  "main",
		schema:   "training",
		artifact: "cuda",
	}, deps)
	require.NoError(t, err)

	target := "123.container.us-west-2.cloud.databricks.test/main.training.cuda:13.4.1"
	assert.Equal(t, []recordedImageCommand{
		{executable: "/usr/bin/docker", args: []string{"image", "ls", "--quiet", "nvidia/cuda:13.4.1"}},
		{executable: "/usr/bin/docker", args: []string{"image", "inspect", "--format", "{{.Os}}/{{.Architecture}}", "nvidia/cuda:13.4.1"}},
		{executable: "/usr/bin/docker", args: []string{"tag", "nvidia/cuda:13.4.1", target}},
		{executable: "/usr/bin/docker", args: []string{"push", target}},
	}, commands)
	assert.Equal(t,
		"Now you can set `environment.unity_catalog_image = main.training.cuda:13.4.1` in your YAML config and create a workload with `databricks air run -f config.yaml`.\n",
		stdout.String(),
	)
	stderrLines := strings.Split(stderr.String(), "\n")
	assert.Equal(t, "Warning: This feature is in Preview. APIs may change, and the workspace must enable the Preview features.", stderrLines[0])
	assert.Contains(t, stderr.String(), "Pushed image")
}

func TestRegistryHostFromDockerConfigureOutput(t *testing.T) {
	output := `Configured Docker credential helper for 123.container.us-west-2.cloud.databricks.test
Updated Docker config: /tmp/docker/config.json
Installed Docker credential helper: /tmp/bin/docker-credential-databricks`

	host, err := registryHostFromDockerConfigureOutput(output)
	require.NoError(t, err)
	assert.Equal(t, "123.container.us-west-2.cloud.databricks.test", host)

	_, err = registryHostFromDockerConfigureOutput("Docker authentication configured")
	assert.ErrorContains(t, err, "did not report an Artifact Registry host")
}

func TestRunImagePushPullsMissingImage(t *testing.T) {
	cmd, _, _ := newImagePushTestCommand(t, "workspace")
	var commands []recordedImageCommand
	deps := imagePushDeps{
		configureDocker: func(context.Context, string) (string, error) {
			return "123.container.us-west-2.cloud.databricks.test", nil
		},
		lookPath: func(string) (string, error) { return "docker", nil },
		runCommand: func(_ context.Context, _ io.Reader, out, _ io.Writer, executable string, args ...string) error {
			commands = append(commands, recordedImageCommand{executable: executable, args: args})
			if args[0] == "image" && args[1] == "inspect" {
				_, err := io.WriteString(out, "linux/amd64\n")
				return err
			}
			return nil
		},
	}

	err := runImagePush(cmd, &imagePushOptions{
		source:   "example/image:v1",
		catalog:  "main",
		schema:   "training",
		artifact: "artifact:v2",
	}, deps)
	require.NoError(t, err)

	require.Len(t, commands, 5)
	assert.Equal(t, []string{"pull", "--platform", "linux/amd64", "example/image:v1"}, commands[1].args)
}

func TestRunImagePushRequiresWorkspaceProfile(t *testing.T) {
	cmd, _, _ := newImagePushTestCommand(t, "")
	deps := imagePushDeps{}

	err := runImagePush(cmd, &imagePushOptions{
		source:   "example/image:v1",
		catalog:  "main",
		schema:   "training",
		artifact: "artifact:v1",
	}, deps)
	assert.ErrorContains(t, err, "air images push requires a workspace profile")
}

func TestResolveImagePushOptionsRequiresFlagsWithoutPrompt(t *testing.T) {
	ctx := cmdio.MockDiscard(t.Context())
	_, err := resolveImagePushOptions(ctx, &imagePushOptions{})
	assert.ErrorContains(t, err, "--source is required when prompting is unavailable")
}

func TestResolveImagePushOptionsValidatesCatalogAndSchema(t *testing.T) {
	ctx := cmdio.MockDiscard(t.Context())
	tests := []struct {
		name    string
		catalog string
		schema  string
		want    string
	}{
		{name: "uppercase catalog", catalog: "Main", schema: "training", want: "invalid catalog"},
		{name: "leading hyphen", catalog: "-catalog", schema: "training", want: "invalid catalog"},
		{name: "mixed separators", catalog: "main", schema: "team_-training", want: "invalid schema"},
		{name: "triple underscore", catalog: "main", schema: "team___training", want: "invalid schema"},
		{name: "uppercase schema", catalog: "main", schema: "Training", want: "invalid schema"},
		{name: "dotted schema", catalog: "main", schema: "team.training", want: "invalid schema"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := resolveImagePushOptions(ctx, &imagePushOptions{
				source:   "example/image:v1",
				catalog:  tt.catalog,
				schema:   tt.schema,
				artifact: "artifact:v1",
			})
			assert.ErrorContains(t, err, tt.want)
		})
	}
}

func TestResolveArtifactAndTagValidation(t *testing.T) {
	ctx := cmdio.MockDiscard(t.Context())
	tests := []struct {
		name  string
		image string
		want  string
	}{
		{name: "slash", image: "team/artifact:v1", want: "invalid artifact name"},
		{name: "dot", image: "artifact.name:v1", want: "invalid artifact name"},
		{name: "uppercase artifact", image: "Artifact:v1", want: "invalid artifact name"},
		{name: "leading separator", image: "_artifact:v1", want: "invalid artifact name"},
		{name: "trailing separator", image: "artifact-:v1", want: "invalid artifact name"},
		{name: "three underscores", image: "artifact___name:v1", want: "invalid artifact name"},
		{name: "space", image: "artifact:bad tag", want: "invalid image tag"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, _, err := resolveArtifactAndTag(ctx, tt.image, "", "latest")
			assert.ErrorContains(t, err, tt.want)
		})
	}
}

func TestResolveArtifactAndTagLength(t *testing.T) {
	ctx := cmdio.MockDiscard(t.Context())
	tests := []struct {
		name      string
		image     string
		wantError string
	}{
		{name: "255 character artifact", image: strings.Repeat("a", 255) + ":v1"},
		{name: "256 character artifact", image: strings.Repeat("a", 256) + ":v1", wantError: "invalid artifact name: must be 255 characters or less"},
		{name: "128 character tag", image: "artifact:" + strings.Repeat("a", 128)},
		{name: "129 character tag", image: "artifact:" + strings.Repeat("a", 129), wantError: "invalid image tag: must be 128 characters or less"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, _, err := resolveArtifactAndTag(ctx, tt.image, "", "latest")
			if tt.wantError == "" {
				require.NoError(t, err)
				return
			}
			assert.ErrorContains(t, err, tt.wantError)
		})
	}
}

func TestSourceImageDefaults(t *testing.T) {
	tests := []struct {
		source       string
		wantArtifact string
		wantTag      string
	}{
		{source: "nvidia/cuda:13.4.1", wantArtifact: "cuda", wantTag: "13.4.1"},
		{source: "registry.example.test:5000/team/image", wantArtifact: "image", wantTag: "latest"},
		{source: "team/image.with-dot:v1", wantArtifact: "", wantTag: "v1"},
		{source: "team/image@sha256:abc123", wantArtifact: "image", wantTag: "latest"},
		{source: "team/image:v1@sha256:abc123", wantArtifact: "image", wantTag: "v1"},
	}
	for _, tt := range tests {
		t.Run(tt.source, func(t *testing.T) {
			artifact, tag := sourceImageDefaults(tt.source)
			assert.Equal(t, tt.wantArtifact, artifact)
			assert.Equal(t, tt.wantTag, tag)
		})
	}
}

func TestResolveArtifactAndTagAcceptsSupportedSeparators(t *testing.T) {
	ctx := cmdio.MockDiscard(t.Context())
	for _, artifact := range []string{"artifact_name", "artifact__name", "artifact---name"} {
		t.Run(artifact, func(t *testing.T) {
			gotArtifact, gotTag, err := resolveArtifactAndTag(ctx, artifact+":Tag_1.2-rc", "", "latest")
			require.NoError(t, err)
			assert.Equal(t, artifact, gotArtifact)
			assert.Equal(t, "Tag_1.2-rc", gotTag)
		})
	}
}

func TestImageExistsLocally(t *testing.T) {
	for _, tt := range []struct {
		name      string
		source    string
		wantQuery string
		output    string
		err       error
		want      bool
	}{
		{name: "missing", source: "team/image:v1", wantQuery: "team/image:v1"},
		{name: "present", source: "team/image:v1", wantQuery: "team/image:v1", output: "image-id\n", want: true},
		{name: "implicit latest", source: "team/image", wantQuery: "team/image:latest"},
		{name: "registry port", source: "registry.example.test:5000/team/image", wantQuery: "registry.example.test:5000/team/image:latest"},
		{name: "digest", source: "team/image@sha256:abc123", wantQuery: "team/image@sha256:abc123"},
		{name: "daemon error", source: "team/image:v1", wantQuery: "team/image:v1", err: errors.New("daemon unavailable")},
		{name: "canceled", source: "team/image:v1", wantQuery: "team/image:v1", err: context.Canceled},
	} {
		t.Run(tt.name, func(t *testing.T) {
			run := func(_ context.Context, _ io.Reader, out, _ io.Writer, executable string, args ...string) error {
				assert.Equal(t, "docker", executable)
				assert.Equal(t, []string{"image", "ls", "--quiet", tt.wantQuery}, args)
				_, err := io.WriteString(out, tt.output)
				require.NoError(t, err)
				return tt.err
			}
			got, err := imageExistsLocally(t.Context(), run, "docker", tt.source, io.Discard)
			if tt.err != nil {
				require.ErrorIs(t, err, tt.err)
			} else {
				require.NoError(t, err)
			}
			assert.Equal(t, tt.want, got)
		})
	}
}
