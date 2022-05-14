package init

import (
	"embed"
	"fmt"
	"os"
	"path"

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
		q := Questions{
			Text{"name", "Project name", func(res Results) string {
				return path.Base(wd)
			}, func(ans Answer, prj *project.Project, res Results) {
				prj.Name = ans.Value
			}},
			*profileChoice,
			Choice{"language", "Project language", Answers{
				{"Python", "Machine learning and data engineering focused projects", nil},
				{"Scala", "Data engineering focused projects with strong typing", nil},
			}},
			Choice{"isolation", "Deployment isolation", Answers{
				{"None", "Use shared Databricks workspace resources for all project team members", nil},
				{"Soft", "Prepend prefixes to each team member's deployment", func(
					ans Answer, prj *project.Project, res Results) {
					prj.Isolation = project.Soft
				}},
			}},
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
		res := Results{}
		err = q.Ask(res)
		if err != nil {
			return err
		}
		var prj project.Project
		for _, ans := range res {
			if ans.Callback == nil {
				continue
			}
			ans.Callback(ans, &prj, res)
		}
		raw, err := yaml.Marshal(prj)
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
