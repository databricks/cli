package template

import (
	"testing"

	"github.com/databricks/cli/libs/cmdio"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestTemplateResolverBothTagAndBranch(t *testing.T) {
	r := Resolver{
		Tag:    "tag",
		Branch: "branch",
	}

	_, err := r.Resolve(t.Context())
	assert.EqualError(t, err, "only one of tag or branch can be specified")
}

func TestTemplateResolverErrorsWhenPromptingIsNotSupported(t *testing.T) {
	r := Resolver{}
	ctx := cmdio.MockDiscard(t.Context())

	_, err := r.Resolve(ctx)
	assert.EqualError(t, err, "prompting is not supported. Please specify the path, name or URL of the template to use")
}

func TestTemplateResolverForDefaultTemplates(t *testing.T) {
	for _, name := range []string{
		"default-python",
		"default-sql",
		"dbt-sql",
	} {
		t.Run(name, func(t *testing.T) {
			r := Resolver{
				TemplatePathOrUrl: name,
			}

			tmpl, err := r.Resolve(t.Context())
			require.NoError(t, err)

			assert.Equal(t, &builtinReader{name: name}, tmpl.Reader)
			assert.IsType(t, &writerWithFullTelemetry{}, tmpl.Writer)
		})
	}

	t.Run("mlops-stacks", func(t *testing.T) {
		r := Resolver{
			TemplatePathOrUrl: "mlops-stacks",
			ConfigFile:        "/config/file",
		}

		tmpl, err := r.Resolve(t.Context())
		require.NoError(t, err)

		// Assert reader and writer configuration
		assert.Equal(t, "https://github.com/databricks/mlops-stacks", tmpl.Reader.(*gitReader).gitUrl)
		assert.Equal(t, "/config/file", tmpl.Writer.(*writerWithFullTelemetry).configPath)
	})
}

func TestTemplateResolverForCustomUrl(t *testing.T) {
	r := Resolver{
		TemplatePathOrUrl: "https://www.example.com/abc",
		Tag:               "tag",
		TemplateDir:       "/template/dir",
		ConfigFile:        "/config/file",
	}

	tmpl, err := r.Resolve(t.Context())
	require.NoError(t, err)

	assert.Equal(t, Custom, tmpl.name)

	// Assert reader configuration
	assert.Equal(t, "https://www.example.com/abc", tmpl.Reader.(*gitReader).gitUrl)
	assert.Equal(t, "tag", tmpl.Reader.(*gitReader).ref)
	assert.Equal(t, "/template/dir", tmpl.Reader.(*gitReader).templateDir)

	// Assert writer configuration
	assert.Equal(t, "/config/file", tmpl.Writer.(*defaultWriter).configPath)
}

func TestTemplateResolverForCustomPath(t *testing.T) {
	r := Resolver{
		TemplatePathOrUrl: "/custom/path",
		ConfigFile:        "/config/file",
	}

	tmpl, err := r.Resolve(t.Context())
	require.NoError(t, err)

	assert.Equal(t, Custom, tmpl.name)

	// Assert reader configuration
	assert.Equal(t, "/custom/path", tmpl.Reader.(*localReader).path)

	// Assert writer configuration
	assert.Equal(t, "/config/file", tmpl.Writer.(*defaultWriter).configPath)
}

func TestBundleInitIsGitRepoUrl(t *testing.T) {
	// Supported
	assert.True(t, IsGitRepoUrl("git@github.com:databricks/cli.git"))
	assert.True(t, IsGitRepoUrl("https://github.com/databricks/cli.git"))
	assert.True(t, IsGitRepoUrl("ssh://user@company.ghe.com/databricks/cli.git"))

	// Unsupported
	assert.False(t, IsGitRepoUrl("git://github.com/databricks/cli.git"))
	assert.False(t, IsGitRepoUrl("http://github.com/databricks/cli.git"))
	assert.False(t, IsGitRepoUrl("ftp://github.com/databricks/cli.git"))
	assert.False(t, IsGitRepoUrl("ftps://github.com/databricks/cli.git"))

	// Not git repos
	assert.False(t, IsGitRepoUrl("./local"))
	assert.False(t, IsGitRepoUrl("foo"))
	assert.False(t, IsGitRepoUrl("github.com/databricks/cli.git"))
}

func TestResolveReader(t *testing.T) {
	t.Run("builtin template", func(t *testing.T) {
		reader, isGit, err := ResolveReader("default-python", "", "")
		require.NoError(t, err)
		assert.False(t, isGit)
		assert.Equal(t, &builtinReader{name: "default-python"}, reader)
	})

	for _, url := range []string{
		"https://github.com/example/repo",
		"ssh://git@github.com/example/repo",
		"git@github.com:example/repo",
	} {
		t.Run("git URL "+url, func(t *testing.T) {
			reader, isGit, err := ResolveReader(url, "/template", "v1.0")
			require.NoError(t, err)
			assert.True(t, isGit)
			gitReader := reader.(*gitReader)
			assert.Equal(t, url, gitReader.gitUrl)
			assert.Equal(t, "/template", gitReader.templateDir)
			assert.Equal(t, "v1.0", gitReader.ref)
		})
	}

	for _, url := range []string{
		"http://github.com/example/repo",
		"git://github.com/example/repo",
		"ftp://github.com/example/repo",
		"ftps://github.com/example/repo",
	} {
		t.Run("unsupported protocol "+url, func(t *testing.T) {
			_, _, err := ResolveReader(url, "", "")
			assert.ErrorContains(t, err, "unsupported protocol")
		})
	}

	t.Run("local path", func(t *testing.T) {
		reader, isGit, err := ResolveReader("/local/path", "", "")
		require.NoError(t, err)
		assert.False(t, isGit)
		assert.Equal(t, "/local/path", reader.(*localReader).path)
	})
}
