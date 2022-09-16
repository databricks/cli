package sync

import (
	"context"
	"encoding/json"
	"log"
	"os"
	"path/filepath"

	"github.com/databricks/bricks/cmd/root"
	"github.com/databricks/bricks/project"
	"github.com/ghodss/yaml"
	"github.com/hashicorp/terraform-exec/tfexec"
	"github.com/imdario/mergo"
	"github.com/spf13/cobra"
)

func resourcesToTerraform(in map[string]interface{}) map[string]interface{} {
	var out = map[string]interface{}{}
	for k, v := range in {
		out["databricks_"+k] = v
	}
	return map[string]interface{}{
		"resource": out,
	}
}

var deployCmd = &cobra.Command{
	Use:   "deploy",
	Short: "Deploys as project",

	PreRunE: project.Configure,
	RunE: func(cmd *cobra.Command, args []string) error {
		// ctx := cmd.Context()
		// wsc := project.Get(ctx).WorkspacesClient()

		prj := project.Get(cmd.Context())

		// Find all

		files, err := filepath.Glob(filepath.Join(prj.Root(), "resource*.yml"))
		if err != nil {
			return err
		}

		env := "dev"

		var out = map[string]interface{}{
			"terraform": map[string]interface{}{
				"required_providers": map[string]interface{}{
					"databricks": map[string]interface{}{
						"source":  "databricks/databricks",
						"version": ">= 1.0.0",
					},
				},
			},
			"provider": map[string]interface{}{
				"databricks": []map[string]interface{}{
					{
						"profile": prj.Environment().Workspace.Profile,
					},
				},
			},
			"resource": map[string]interface{}{},
		}

		for _, f := range files {
			var contents struct {
				Environments map[string]struct {
					Resources map[string]interface{} `yaml:"resources"`
				} `yaml:"environments"`
				Resources map[string]interface{} `yaml:"resources"`
			}

			raw, err := os.ReadFile(f)
			if err != nil {
				return err
			}

			err = yaml.Unmarshal(raw, &contents)
			if err != nil {
				return err
			}

			var base = map[string]interface{}{}

			if contents.Environments != nil {
				if r, ok := contents.Environments[env]; ok {
					err = mergo.Merge(&base, r.Resources)
					if err != nil {
						panic(err)
					}
				}
			}

			// Merge in the resources
			err = mergo.Merge(&base, &contents.Resources)
			if err != nil {
				panic(err)
			}

			// TODO check that these resources haven't been defined in out yet (must be unique across files)

			// raw, err = json.MarshalIndent(base, "", "  ")
			// if err != nil {
			// 	panic(err)
			// }

			// log.Printf("[INFO] %s", string(raw))

			err = mergo.Merge(&out, resourcesToTerraform(base))
			if err != nil {
				panic(err)
			}

		}

		// Perform any string interpolation / string templating

		// TODO Make sure dist/env exists...

		f, err := os.Create(filepath.Join(prj.Root(), "dist", env, "main.tf.json"))
		if err != nil {
			return err
		}

		enc := json.NewEncoder(f)
		err = enc.Encode(out)
		if err != nil {
			return err
		}

		// installer := &releases.ExactVersion{
		// 	Product: product.Terraform,
		// 	Version: version.Must(version.NewVersion("1.0.6")),
		// }

		// execPath, err := installer.Install(context.Background())
		// if err != nil {
		// 	log.Fatalf("error installing Terraform: %s", err)
		// }

		runtf := true
		if runtf {
			execPath := "/opt/homebrew/bin/terraform"
			log.Printf("[INFO] tf exec path: %s", execPath)

			workingDir := filepath.Join(prj.Root(), "dist", env)
			tf, err := tfexec.NewTerraform(workingDir, execPath)
			if err != nil {
				log.Fatalf("[ERROR] error running NewTerraform: %s", err)
			}

			err = tf.Init(context.Background(), tfexec.Upgrade(true))
			if err != nil {
				log.Fatalf("[ERROR] error running Init: %s", err)
			}

			err = tf.Apply(context.Background())
			if err != nil {
				log.Fatalf("[ERROR] error running apply: %s", err)
			}
		}

		return nil
	},
}

func init() {
	root.RootCmd.AddCommand(deployCmd)
	// interval = syncCmd.Flags().Duration("interval", 1*time.Second, "project files polling interval")
	// remotePath = syncCmd.Flags().String("remote-path", "", "remote path to store repo in. eg: /Repos/me@example.com/test-repo")

	// flag := pflag.StringP("environment", "e", "", "Environment to use")
	deployCmd.Flags().StringP("environment", "e", "", "Environment to use")
}
