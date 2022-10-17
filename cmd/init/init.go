package init

import (
	"embed"
	"fmt"
	"os"
	"path"

	"github.com/databricks/bricks/cmd/prompt"
	"github.com/databricks/bricks/cmd/root"
	"github.com/databricks/bricks/project"
	"github.com/ghodss/yaml"
	"github.com/spf13/cobra"
)

//go:embed templates
var templates embed.FS

// initCmd represents the init command
var initCmd = &cobra.Command{
	Use:   "init",
	Short: "Project starter templates",
	Long:  `Generate project templates`,

	RunE: func(cmd *cobra.Command, args []string) error {
		if project.IsDatabricksProject() {
			return fmt.Errorf("this path is already a Databricks project")
		}
		profileChoice, err := getConnectionProfile()
		if err != nil {
			return err
		}
		wd, _ := os.Getwd()
		q := prompt.Questions{
			prompt.Text{
				Key:   "name",
				Label: "Project name",
				Default: func(res prompt.Results) string {
					return path.Base(wd)
				},
				Callback: func(ans prompt.Answer, config *project.Config, res prompt.Results) {
					config.Name = ans.Value
				},
			},
			*profileChoice,
			prompt.Choice{Key: "language", Label: "Project language", Answers: prompt.Answers{
				{
					Value:    "Python",
					Details:  "Machine learning and data engineering focused projects",
					Callback: nil,
				},
				{
					Value:    "Scala",
					Details:  "Data engineering focused projects with strong typing",
					Callback: nil,
				},
			}},
			prompt.Choice{Key: "isolation", Label: "Deployment isolation", Answers: prompt.Answers{
				{
					Value:    "None",
					Details:  "Use shared Databricks workspace resources for all project team members",
					Callback: nil,
				},
				{
					Value:   "Soft",
					Details: "Prepend prefixes to each team member's deployment",
					Callback: func(
						ans prompt.Answer, config *project.Config, res prompt.Results) {
						config.Isolation = project.Soft
					},
				},
			}},
			// DBR selection
			// Choice{"cloud", "Cloud", Answers{
			// 	{"AWS", "Amazon Web Services", nil},
			// 	{"Azure", "Microsoft Azure Cloud", nil},
			// 	{"GCP", "Google Cloud Platform", nil},
			// }},
			// Choice{"ci", "Continuous Integration", Answers{
			// 	{"None", "Do not create continuous integration configuration", nil},
			// 	{"GitHub Actions", "Create .github/workflows/push.yml configuration", nil},
			// 	{"Azure DevOps", "Create basic build and test pipelines", nil},
			// }},
			// Choice{"ide", "Integrated Development Environment", Answers{
			// 	{"None", "Do not create templates for IDE", nil},
			// 	{"VSCode", "Create .devcontainer and other useful things", nil},
			// 	{"PyCharm", "Create project conf and other things", nil},
			// }},
		}
		res := prompt.Results{}
		err = q.Ask(res)
		if err != nil {
			return err
		}
		var config project.Config
		for _, ans := range res {
			if ans.Callback == nil {
				continue
			}
			ans.Callback(ans, &config, res)
		}
		raw, err := yaml.Marshal(config)
		if err != nil {
			return err
		}
		newConfig, err := os.Create(fmt.Sprintf("%s/%s", wd, project.ConfigFile))
		if err != nil {
			return err
		}
		_, err = newConfig.Write(raw)
		if err != nil {
			return err
		}

		// Create .gitignore if absent
		gitIgnoreFile, err := os.OpenFile(fmt.Sprintf("%s/%s", wd, project.GitIgnoreFile),
			os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
		if err != nil {
			return err
		}
		defer gitIgnoreFile.Close()

		// Append .databricks to the end of .gitignore file
		_, err = gitIgnoreFile.WriteString("\n/.databricks/")
		if err != nil {
			return err
		}

		d, err := templates.ReadDir(".")
		if err != nil {
			return err
		}
		for _, v := range d {
			cmd.Printf("template found: %v", v.Name())
		}
		cmd.Print("Config initialized!")
		return err
	},
}

func init() {
	root.RootCmd.AddCommand(initCmd)
}
