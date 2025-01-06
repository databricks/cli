package template

import (
	"context"
	"fmt"
	"strings"

	"github.com/databricks/cli/libs/cmdio"
	"github.com/databricks/cli/libs/filer"
)

type Template struct {
	Reader Reader
	Writer Writer

	Id          TemplateId
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

type TemplateId string

const (
	DefaultPython TemplateId = "default-python"
	DefaultSql    TemplateId = "default-sql"
	DbtSql        TemplateId = "dbt-sql"
	MlopsStacks   TemplateId = "mlops-stacks"
	DefaultPydabs TemplateId = "default-pydabs"
	Custom        TemplateId = "custom"
)

var allTemplates = []Template{
	{
		Id:          DefaultPython,
		Description: "The default Python template for Notebooks / Delta Live Tables / Workflows",
		Reader:      &builtinReader{name: "default-python"},
		Writer:      &writerWithTelemetry{},
	},
	{
		Id:          DefaultSql,
		Description: "The default SQL template for .sql files that run with Databricks SQL",
		Reader:      &builtinReader{name: "default-sql"},
		Writer:      &writerWithTelemetry{},
	},
	{
		Id:          DbtSql,
		Description: "The dbt SQL template (databricks.com/blog/delivering-cost-effective-data-real-time-dbt-and-databricks)",
		Reader:      &builtinReader{name: "dbt-sql"},
		Writer:      &writerWithTelemetry{},
	},
	{
		Id:          MlopsStacks,
		Description: "The Databricks MLOps Stacks template (github.com/databricks/mlops-stacks)",
		Aliases:     []string{"mlops-stack"},
		Reader:      &gitReader{gitUrl: "https://github.com/databricks/mlops-stacks"},
		Writer:      &writerWithTelemetry{},
	},
	{
		Id:          DefaultPydabs,
		Hidden:      true,
		Description: "The default PyDABs template",
		Reader:      &gitReader{gitUrl: "https://databricks.github.io/workflows-authoring-toolkit/pydabs-template.git"},
		Writer:      &writerWithTelemetry{},
	},
	{
		Id:          Custom,
		Description: "Bring your own template",
		Reader:      &failReader{},
		Writer:      &defaultWriter{},
	},
}

func HelpDescriptions() string {
	var lines []string
	for _, template := range allTemplates {
		if template.Id != Custom && !template.Hidden {
			lines = append(lines, fmt.Sprintf("- %s: %s", template.Id, template.Description))
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
			Name: string(template.Id),
			Id:   template.Description,
		}
		names = append(names, tuple)
	}
	return names
}

// TODO CONTINUE defining the methods that the init command will finally rely on.
func PromptForTemplateId(ctx context.Context, ref, templateDir string) (TemplateId, error) {
	if !cmdio.IsPromptSupported(ctx) {
		return "", fmt.Errorf("please specify a template")
	}
	description, err := cmdio.SelectOrdered(ctx, options(), "Template to use")
	if err != nil {
		return "", err
	}

	for _, template := range allTemplates {
		if template.Description == description {
			return template.Id, nil
		}
	}

	panic("this should never happen - template not found")
}

func (tmpl *Template) InitializeWriter(configPath string, outputFiler filer.Filer) {
	tmpl.Writer.Initialize(tmpl.Reader, configPath, outputFiler)
}

func (tmpl *Template) SetReader(r Reader) {
	tmpl.Reader = r
}

func Get(id TemplateId) *Template {
	for _, template := range allTemplates {
		if template.Id == id {
			return &template
		}
	}

	return nil
}
