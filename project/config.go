package project

import (
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"reflect"

	"github.com/databricks/bricks/folders"
	"github.com/databricks/bricks/lib/flavor/mvn"
	"github.com/databricks/bricks/lib/flavor/notebooks"
	"github.com/databricks/bricks/lib/flavor/py"
	"github.com/databricks/bricks/lib/flavor/script"
	"github.com/databricks/databricks-sdk-go/service/clusters"

	"github.com/ghodss/yaml"
)

type Isolation string

const (
	None Isolation = ""
	Soft Isolation = "soft"
)

// ConfigFile is the name of project configuration file
const ConfigFile = "databricks.yml"

type Assertions struct {
	Groups            []string `json:"groups,omitempty"`
	Secrets           []string `json:"secrets,omitempty"`
	ServicePrincipals []string `json:"service_principals,omitempty"`
}

type Config struct {
	Name      string    `json:"name"`              // or do default from folder name?..
	Profile   string    `json:"profile,omitempty"` // rename?
	Isolation Isolation `json:"isolation,omitempty"`

	// development-time vs deployment-time resources
	DevCluster *clusters.ClusterInfo `json:"dev_cluster,omitempty"`

	Maven     *mvn.Maven           `json:"mvn,omitempty"`
	SetupPy   *py.SetupDotPy       `json:"python,omitempty"`
	Script    *script.Script       `json:"script,omitempty"`
	Notebooks *notebooks.Notebooks `json:"notebooks,omitempty"`

	// Assertions defines a list of configurations expected to be applied
	// to the workspace by a higher-privileged user (or service principal)
	// in order for the deploy command to work, as individual project teams
	// in almost all the cases don’t have admin privileges on Databricks
	// workspaces.
	//
	// This configuration simplifies the flexibility of individual project
	// teams, make jobs deployment easier and portable across environments.
	// This configuration block would contain the following entities to be
	// created by administrator users or admin-level automation, like Terraform
	// and/or SCIM provisioning.
	Assertions *Assertions `json:"assertions,omitempty"`

	// Environments contain this project's defined environments.
	// They can be used to differentiate settings and resources between
	// development, staging, production, etc.
	// If not specified, the code below initializes this field with a
	// single default-initialized environment called "development".
	Environments map[string]Environment `json:"environments"`
}

func (c Config) IsDevClusterDefined() bool {
	return reflect.ValueOf(c.DevCluster).IsZero()
}

// IsDevClusterJustReference denotes reference-only clusters.
// This conflicts with Soft isolation. Happens for cost-restricted projects,
// where there's only a single Shared Autoscaling cluster per workspace and
// general users have no ability to create other iteractive clusters.
func (c *Config) IsDevClusterJustReference() bool {
	if c.DevCluster.ClusterName == "" {
		return false
	}
	return reflect.DeepEqual(c.DevCluster, &clusters.ClusterInfo{
		ClusterName: c.DevCluster.ClusterName,
	})
}

// IsDatabricksProject returns true for folders with `databricks.yml`
// in the parent tree
func IsDatabricksProject() bool {
	_, err := findProjectRoot()
	return err == nil
}

func loadProjectConf(root string) (c Config, err error) {
	configFilePath := filepath.Join(root, ConfigFile)

	if _, err := os.Stat(configFilePath); errors.Is(err, os.ErrNotExist) {
		baseDir := filepath.Base(root)
		// If bricks config file is missing we assume the project root dir name
		// as the name of the project
		return validateAndApplyProjectDefaults(Config{Name: baseDir})
	}

	config, err := os.Open(configFilePath)
	if err != nil {
		return
	}
	raw, err := io.ReadAll(config)
	if err != nil {
		return
	}
	err = yaml.Unmarshal(raw, &c)
	if err != nil {
		return
	}
	return validateAndApplyProjectDefaults(c)
}

func validateAndApplyProjectDefaults(c Config) (Config, error) {
	// If no environments are specified, define default environment under default name.
	if c.Environments == nil {
		c.Environments = make(map[string]Environment)
		c.Environments[DefaultEnvironment] = Environment{}
	}
	// defaultCluster := clusters.ClusterInfo{
	// 	NodeTypeID: "smallest",
	// 	SparkVersion: "latest",
	// 	AutoterminationMinutes: 30,
	// }
	return c, nil
}

func findProjectRoot() (string, error) {
	wd, err := os.Getwd()
	if err != nil {
		return "", err
	}
	dir, err := folders.FindDirWithLeaf(wd, ConfigFile)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return "", fmt.Errorf("cannot find %s anywhere", ConfigFile)
		}
	}
	return dir, nil
}
