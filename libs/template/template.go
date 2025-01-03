package template

import (
	"context"
	"fmt"
	"strings"

	"github.com/databricks/cli/libs/cmdio"
)

type Template struct {
	// TODO: Make private as much as possible.
	Reader Reader
	Writer Writer

	Name        TemplateName
	Description string
	Aliases     []string
	Hidden      bool
}

// TODO: Make details private?
// TODO: Combine this with the generic template struct?
type NativeTemplate struct {
	Name                string
	Description         string
	Aliases             []string
	GitUrl              string
	Hidden              bool
	IsOwnedByDatabricks bool
}

type TemplateName string

const (
	DefaultPython TemplateName = "default-python"
	DefaultSql    TemplateName = "default-sql"
	DbtSql        TemplateName = "dbt-sql"
	MlopsStacks   TemplateName = "mlops-stacks"
	DefaultPydabs TemplateName = "default-pydabs"
	Custom        TemplateName = "custom"
)

var allTemplates = []Template{
	{
		Name:        DefaultPython,
		Description: "The default Python template for Notebooks / Delta Live Tables / Workflows",
		Reader:      &builtinReader{name: "default-python"},
		Writer:      &writerWithTelemetry{},
	},
	{
		Name:        DefaultSql,
		Description: "The default SQL template for .sql files that run with Databricks SQL",
		Reader:      &builtinReader{name: "default-sql"},
		Writer:      &writerWithTelemetry{},
	},
	{
		Name:        DbtSql,
		Description: "The dbt SQL template (databricks.com/blog/delivering-cost-effective-data-real-time-dbt-and-databricks)",
		Reader:      &builtinReader{name: "dbt-sql"},
		Writer:      &writerWithTelemetry{},
	},
	{
		Name:        MlopsStacks,
		Description: "The Databricks MLOps Stacks template (github.com/databricks/mlops-stacks)",
		Aliases:     []string{"mlops-stack"},
		Reader:      &gitReader{gitUrl: "https://github.com/databricks/mlops-stacks"},
		Writer:      &writerWithTelemetry{},
	},
	{
		Name:        DefaultPydabs,
		Hidden:      true,
		Description: "The default PyDABs template",
		Reader:      &gitReader{gitUrl: "https://databricks.github.io/workflows-authoring-toolkit/pydabs-template.git"},
		Writer:      &writerWithTelemetry{},
	},
	{
		Name:        Custom,
		Description: "Bring your own template",
		Reader:      &failReader{},
		Writer:      &defaultWriter{},
	},
}

func HelpDescriptions() string {
	var lines []string
	for _, template := range allTemplates {
		if template.Name != Custom && !template.Hidden {
			lines = append(lines, fmt.Sprintf("- %s: %s", template.Name, template.Description))
		}
	}
	return strings.Join(lines, "\n")
}

func options() []cmdio.Tuple {
	names := make([]cmdio.Tuple, 0, len(allTemplates))
	for _, template := range allTemplates {
		if template.Hidden {
			continue
		}
		tuple := cmdio.Tuple{
			Name: string(template.Name),
			Id:   template.Description,
		}
		names = append(names, tuple)
	}
	return names
}

// TODO CONTINUE defining the methods that the init command will finally rely on.
func SelectTemplate(ctx context.Context) (TemplateName, error) {
	if !cmdio.IsPromptSupported(ctx) {
		return "", fmt.Errorf("please specify a template")
	}
	description, err := cmdio.SelectOrdered(ctx, options(), "Template to use")
	if err != nil {
		return "", err
	}

	for _, template := range allTemplates {
		if template.Description == description {
			return template.Name, nil
		}
	}

	panic("this should never happen - template not found")
}

func Get(name TemplateName) *Template {
	for _, template := range allTemplates {
		if template.Name == name {
			return &template
		}
	}

	return nil
}
