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

func TestTemplateResolverForBundleExamples(t *testing.T) {
	tests := []struct {
		url         string
		templateDir string
		wantName    string
	}{
		{"https://github.com/databricks/bundle-examples", "contrib/templates/dbt-factory", "bundle-examples/contrib/templates/dbt-factory"},
		// Equivalent template directories canonicalize to the same identifier.
		{"https://github.com/databricks/bundle-examples", "./contrib/templates/dbt-factory", "bundle-examples/contrib/templates/dbt-factory"},
		{"https://github.com/databricks/bundle-examples", "contrib/templates/dbt-factory/.", "bundle-examples/contrib/templates/dbt-factory"},
		// Without a template directory we can only attribute the init to the repo.
		{"https://github.com/databricks/bundle-examples", "", "bundle-examples"},
	}

	for _, tc := range tests {
		t.Run(tc.url+"|"+tc.templateDir, func(t *testing.T) {
			r := Resolver{
				TemplatePathOrUrl: tc.url,
				TemplateDir:       tc.templateDir,
			}

			tmpl, err := r.Resolve(t.Context())
			require.NoError(t, err)

			// bundle-examples is a first-party repo, so we record verbose telemetry
			// (template name and enum args) for it.
			w, ok := tmpl.Writer.(*writerWithFullTelemetry)
			require.True(t, ok, "expected a full-telemetry writer")
			assert.Equal(t, tc.wantName, string(w.name))
			assert.Equal(t, tc.url, tmpl.Reader.(*gitReader).gitUrl)
		})
	}
}

func TestTemplateResolverForBundleExamplesRejectsEscapingTemplateDir(t *testing.T) {
	// A template directory that escapes the cloned repo would load an arbitrary
	// local template, so it must not be recorded as first-party.
	for _, dir := range []string{
		"../../../local-template",
		"/etc/local-template",
	} {
		t.Run(dir, func(t *testing.T) {
			r := Resolver{
				TemplatePathOrUrl: "https://github.com/databricks/bundle-examples",
				TemplateDir:       dir,
			}

			tmpl, err := r.Resolve(t.Context())
			require.NoError(t, err)
			assert.IsType(t, &defaultWriter{}, tmpl.Writer)
		})
	}
}

func TestIsBundleExamplesRepo(t *testing.T) {
	// Recognized across the common Git URL forms.
	for _, url := range []string{
		"https://github.com/databricks/bundle-examples",
		"https://github.com/databricks/bundle-examples.git",
		"https://github.com/databricks/bundle-examples/",
		"git@github.com:databricks/bundle-examples.git",
		"ssh://git@github.com/databricks/bundle-examples.git",
		"ssh://git@github.com:22/databricks/bundle-examples.git",
		// GitHub owner/repo names are case-insensitive.
		"https://github.com/Databricks/Bundle-Examples",
	} {
		assert.True(t, isBundleExamplesRepo(url), url)
	}

	// Other repos, forks, hosts, owners and local paths must not match.
	for _, url := range []string{
		"https://github.com/databricks/cli",
		"https://github.com/someone/bundle-examples",
		"https://github.com/databricks/bundle-examples-fork",
		"https://gitlab.com/databricks/bundle-examples",
		"/local/bundle-examples",
		// An "@" embedded in the path must not be mistaken for the host: this
		// clones from attacker.example, not github.com.
		"https://attacker.example/foo@github.com/databricks/bundle-examples",
	} {
		assert.False(t, isBundleExamplesRepo(url), url)
	}
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
