package template

import (
	"context"
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

	_, err := r.Resolve(context.Background())
	assert.EqualError(t, err, "only one of tag or branch can be specified")
}

func TestTemplateResolverErrorsWhenPromptingIsNotSupported(t *testing.T) {
	r := Resolver{}
	ctx := cmdio.MockDiscard(context.Background())

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

			tmpl, err := r.Resolve(context.Background())
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

		tmpl, err := r.Resolve(context.Background())
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

	tmpl, err := r.Resolve(context.Background())
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

	tmpl, err := r.Resolve(context.Background())
	require.NoError(t, err)

	assert.Equal(t, Custom, tmpl.name)

	// Assert reader configuration
	assert.Equal(t, "/custom/path", tmpl.Reader.(*localReader).path)

	// Assert writer configuration
	assert.Equal(t, "/config/file", tmpl.Writer.(*defaultWriter).configPath)
}

func TestBundleInitIsRepoUrl(t *testing.T) {
	assert.True(t, isRepoUrl("git@github.com:databricks/cli.git"))
	assert.True(t, isRepoUrl("https://github.com/databricks/cli.git"))

	assert.False(t, isRepoUrl("./local"))
	assert.False(t, isRepoUrl("foo"))
}
