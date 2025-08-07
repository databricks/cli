package template

import (
	"context"
	"errors"
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
	DefaultPython          TemplateName = "default-python"
	DefaultSql             TemplateName = "default-sql"
	LakeflowPipelines      TemplateName = "lakeflow-pipelines"
	CLIPipelines           TemplateName = "cli-pipelines"
	DbtSql                 TemplateName = "dbt-sql"
	MlopsStacks            TemplateName = "mlops-stacks"
	DefaultPydabs          TemplateName = "default-pydabs"
	Custom                 TemplateName = "custom"
	ExperimentalJobsAsCode TemplateName = "experimental-jobs-as-code"
)

var databricksTemplates = []Template{
	{
		name:        DefaultPython,
		description: "The default Python template for Notebooks / Delta Live Tables / Workflows",
		Reader:      &builtinReader{name: string(DefaultPython)},
		Writer:      &writerWithFullTelemetry{defaultWriter: defaultWriter{name: DefaultPython}},
	},
	{
		name:        DefaultSql,
		description: "The default SQL template for .sql files that run with Databricks SQL",
		Reader:      &builtinReader{name: string(DefaultSql)},
		Writer:      &writerWithFullTelemetry{defaultWriter: defaultWriter{name: DefaultSql}},
	},
	{
		name:        LakeflowPipelines,
		hidden:      true,
		description: "The default template for Lakeflow Declarative Pipelines",
		Reader:      &builtinReader{name: string(LakeflowPipelines)},
		Writer:      &writerWithFullTelemetry{defaultWriter: defaultWriter{name: LakeflowPipelines}},
	},
	{
		name:        CLIPipelines,
		hidden:      true,
		description: "The default template for CLI pipelines",
		Reader:      &builtinReader{name: string(CLIPipelines)},
		Writer:      &writerWithFullTelemetry{defaultWriter: defaultWriter{name: CLIPipelines}},
	},
	{
		name:        DbtSql,
		description: "The dbt SQL template (databricks.com/blog/delivering-cost-effective-data-real-time-dbt-and-databricks)",
		Reader:      &builtinReader{name: string(DbtSql)},
		Writer:      &writerWithFullTelemetry{defaultWriter: defaultWriter{name: DbtSql}},
	},
	{
		name:        MlopsStacks,
		description: "The Databricks MLOps Stacks template (github.com/databricks/mlops-stacks)",
		aliases:     []string{"mlops-stack"},
		Reader:      &gitReader{gitUrl: "https://github.com/databricks/mlops-stacks", cloneFunc: git.Clone},
		Writer:      &writerWithFullTelemetry{defaultWriter: defaultWriter{name: MlopsStacks}},
	},
	{
		name:        DefaultPydabs,
		hidden:      true,
		description: "The default PyDABs template",
		Reader:      &gitReader{gitUrl: "https://databricks.github.io/workflows-authoring-toolkit/pydabs-template.git", cloneFunc: git.Clone},
		Writer:      &writerWithFullTelemetry{defaultWriter: defaultWriter{name: DefaultPydabs}},
	},
	{
		name:        ExperimentalJobsAsCode,
		description: "Jobs as code template (experimental)",
		Reader:      &builtinReader{name: string(ExperimentalJobsAsCode)},
		Writer:      &writerWithFullTelemetry{defaultWriter: defaultWriter{name: ExperimentalJobsAsCode}},
	},
}

func HelpDescriptions() string {
	var lines []string
	for _, template := range databricksTemplates {
		if template.name != Custom && !template.hidden {
			lines = append(lines, fmt.Sprintf("- %s: %s", template.name, template.description))
		}
	}
	return strings.Join(lines, "\n")
}

var customTemplateDescription = "Bring your own template"

func options() []cmdio.Tuple {
	names := make([]cmdio.Tuple, 0, len(databricksTemplates))
	for _, template := range databricksTemplates {
		if template.hidden {
			continue
		}
		tuple := cmdio.Tuple{
			Name: string(template.name),
			Id:   template.description,
		}
		names = append(names, tuple)
	}

	names = append(names, cmdio.Tuple{
		Name: "custom...",
		Id:   customTemplateDescription,
	})
	return names
}

func SelectTemplate(ctx context.Context) (TemplateName, error) {
	if !cmdio.IsPromptSupported(ctx) {
		return "", errors.New("prompting is not supported. Please specify the path, name or URL of the template to use")
	}
	description, err := cmdio.SelectOrdered(ctx, options(), "Template to use")
	if err != nil {
		return "", err
	}

	if description == customTemplateDescription {
		return TemplateName(""), ErrCustomSelected
	}

	for _, template := range databricksTemplates {
		if template.description == description {
			return template.name, nil
		}
	}

	return "", fmt.Errorf("template with description %s not found", description)
}

func GetDatabricksTemplate(name TemplateName) *Template {
	for _, template := range databricksTemplates {
		if template.name == name || slices.Contains(template.aliases, string(name)) {
			return &template
		}
	}

	return nil
}
