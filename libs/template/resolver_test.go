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
	assert.EqualError(t, err, "only one of --tag or --branch can be specified")
}

func TestTemplateResolverErrorsWhenPromptingIsNotSupported(t *testing.T) {
	r := Resolver{}
	ctx := cmdio.MockContext(context.Background())

	_, err := r.Resolve(ctx)
	assert.EqualError(t, err, "prompting is not supported. Please specify the path, name or URL of the template to use")
}

func TestTemplateResolverErrorWhenUserSelectsCustom(t *testing.T) {
	r := Resolver{
		TemplatePathOrUrl: "custom",
	}

	_, err := r.Resolve(context.Background())
	assert.EqualError(t, err, "custom template selected")
}

func TestTemplateResolverForDefaultTemplates(t *testing.T) {
	for _, name := range []string{
		"default-python",
		"default-sql",
		"dbt-sql",
	} {
		r := Resolver{
			TemplatePathOrUrl: name,
		}

		tmpl, err := r.Resolve(context.Background())
		require.NoError(t, err)

		assert.Equal(t, &builtinReader{name: name}, tmpl.Reader)
		assert.IsType(t, &writerWithTelemetry{}, tmpl.Writer)
	}

	r := Resolver{
		TemplatePathOrUrl: "mlops-stacks",
		ConfigFile:        "/config/file",
	}

	tmpl, err := r.Resolve(context.Background())
	require.NoError(t, err)

	// Assert reader and writer configuration
	assert.Equal(t, "https://github.com/databricks/mlops-stacks", tmpl.Reader.(*gitReader).gitUrl)
	assert.Equal(t, "/config/file", tmpl.Writer.(*writerWithTelemetry).configPath)
}

func TestTemplateResolverForCustomTemplate(t *testing.T) {
	r := Resolver{
		TemplatePathOrUrl: "https://www.example.com/abc",
		Tag:               "tag",
		TemplateDir:       "/template/dir",
		ConfigFile:        "/config/file",
	}

	tmpl, err := r.Resolve(context.Background())
	require.NoError(t, err)

	// Assert reader configuration
	assert.Equal(t, "https://www.example.com/abc", tmpl.Reader.(*gitReader).gitUrl)
	assert.Equal(t, "tag", tmpl.Reader.(*gitReader).ref)
	assert.Equal(t, "/template/dir", tmpl.Reader.(*gitReader).templateDir)

	// Assert writer configuration
	assert.Equal(t, "/config/file", tmpl.Writer.(*defaultWriter).configPath)
}
