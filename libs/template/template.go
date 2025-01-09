package template

import (
	"context"
	"fmt"
	"slices"
	"strings"

	"github.com/databricks/cli/libs/cmdio"
	"github.com/databricks/cli/libs/git"
)

type Template struct {
	Reader Reader
	Writer Writer

	name        TemplateName
	description string
	aliases     []string
	hidden      bool
}

type TemplateName string

const (
	DefaultPython TemplateName = "default-python"
	DefaultSql    TemplateName = "default-sql"
	DbtSql        TemplateName = "dbt-sql"
	MlopsStacks   TemplateName = "mlops-stacks"
	DefaultPydabs TemplateName = "default-pydabs"

	// Custom represents any template that is not one of the above default
	// templates. It's a catch all for any custom templates that customers provide
	// as a path or URL argument.
	Custom TemplateName = "custom..."
)

var allTemplates = []Template{
	{
		name:        DefaultPython,
		description: "The default Python template for Notebooks / Delta Live Tables / Workflows",
		Reader:      &builtinReader{name: "default-python"},
		Writer:      &writerWithFullTelemetry{},
	},
	{
		name:        DefaultSql,
		description: "The default SQL template for .sql files that run with Databricks SQL",
		Reader:      &builtinReader{name: "default-sql"},
		Writer:      &writerWithFullTelemetry{},
	},
	{
		name:        DbtSql,
		description: "The dbt SQL template (databricks.com/blog/delivering-cost-effective-data-real-time-dbt-and-databricks)",
		Reader:      &builtinReader{name: "dbt-sql"},
		Writer:      &writerWithFullTelemetry{},
	},
	{
		name:        MlopsStacks,
		description: "The Databricks MLOps Stacks template (github.com/databricks/mlops-stacks)",
		aliases:     []string{"mlops-stack"},
		Reader:      &gitReader{gitUrl: "https://github.com/databricks/mlops-stacks", cloneFunc: git.Clone},
		Writer:      &writerWithFullTelemetry{},
	},
	{
		name:        DefaultPydabs,
		hidden:      true,
		description: "The default PyDABs template",
		Reader:      &gitReader{gitUrl: "https://databricks.github.io/workflows-authoring-toolkit/pydabs-template.git", cloneFunc: git.Clone},
		Writer:      &writerWithFullTelemetry{},
	},
	{
		name:        Custom,
		description: "Bring your own template",
		// Reader is determined at runtime based on the user input.
		Reader: nil,
		Writer: &defaultWriter{},
	},
}

func HelpDescriptions() string {
	var lines []string
	for _, template := range allTemplates {
		if template.name != Custom && !template.hidden {
			lines = append(lines, fmt.Sprintf("- %s: %s", template.name, template.description))
		}
	}
	return strings.Join(lines, "\n")
}

func options() []cmdio.Tuple {
	names := make([]cmdio.Tuple, 0, len(allTemplates))
	for _, template := range allTemplates {
		if template.hidden {
			continue
		}
		tuple := cmdio.Tuple{
			Name: string(template.name),
			Id:   template.description,
		}
		names = append(names, tuple)
	}
	return names
}

func SelectTemplate(ctx context.Context) (TemplateName, error) {
	if !cmdio.IsPromptSupported(ctx) {
		return "", fmt.Errorf("prompting is not supported. Please specify the path, name or URL of the template to use")
	}
	description, err := cmdio.SelectOrdered(ctx, options(), "Template to use")
	if err != nil {
		return "", err
	}

	for _, template := range allTemplates {
		if template.description == description {
			return template.name, nil
		}
	}

	panic("this should never happen - template not found")
}

func Get(name TemplateName) *Template {
	for _, template := range allTemplates {
		if template.name == name || slices.Contains(template.aliases, string(name)) {
			return &template
		}
	}

	return nil
}
