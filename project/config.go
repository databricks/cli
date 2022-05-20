package project

import (
	"errors"
	"fmt"
	"io/ioutil"
	"net/url"
	"os"
	"path"
	"reflect"
	"strings"

	"github.com/databrickslabs/terraform-provider-databricks/clusters"
	"github.com/ghodss/yaml"
	gitUrls "github.com/whilp/git-urls"
	"gopkg.in/ini.v1"
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

type Project struct {
	Name      string    `json:"name"`              // or do default from folder name?..
	Profile   string    `json:"profile,omitempty"` // rename?
	Isolation Isolation `json:"isolation,omitempty"`

	// development-time vs deployment-time resources
	DevCluster *clusters.Cluster `json:"dev_cluster,omitempty"`

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
}

func (p Project) IsDevClusterDefined() bool {
	return reflect.ValueOf(p.DevCluster).IsZero()
}

// IsDevClusterJustReference denotes reference-only clusters.
// This conflicts with Soft isolation. Happens for cost-restricted projects,
// where there's only a single Shared Autoscaling cluster per workspace and
// general users have no ability to create other iteractive clusters.
func (p *Project) IsDevClusterJustReference() bool {
	if p.DevCluster.ClusterName == "" {
		return false
	}
	return reflect.DeepEqual(p.DevCluster, &clusters.Cluster{
		ClusterName: p.DevCluster.ClusterName,
	})
}

// IsDatabricksProject returns true for folders with `databricks.yml`
// in the parent tree
func IsDatabricksProject() bool {
	_, err := findProjectRoot()
	return err == nil
}

func loadProjectConf() (prj Project, err error) {
	root, err := findProjectRoot()
	if err != nil {
		return
	}
	config, err := os.Open(fmt.Sprintf("%s/%s", root, ConfigFile))
	if err != nil {
		return
	}
	raw, err := ioutil.ReadAll(config)
	if err != nil {
		return
	}
	err = yaml.Unmarshal(raw, &prj)
	if err != nil {
		return
	}
	return validateAndApplyProjectDefaults(prj)
}

func validateAndApplyProjectDefaults(prj Project) (Project, error) {
	// defaultCluster := clusters.Cluster{
	// 	NodeTypeID: "smallest",
	// 	SparkVersion: "latest",
	// 	AutoterminationMinutes: 30,
	// }
	return prj, nil
}

func findProjectRoot() (string, error) {
	return findDirWithLeaf(ConfigFile)
}

// finds the original git repository the project is cloned from, so that
// we could automatically verify if this project is checked out in repos
// home folder of the user according to recommended best practices. Can
// also be used to determine a good enough default project name.
func getGitOrigin() (*url.URL, error) {
	root, err := findDirWithLeaf(".git")
	if err != nil {
		return nil, err
	}
	file := fmt.Sprintf("%s/.git/config", root)
	gitConfig, err := ini.Load(file)
	if err != nil {
		return nil, err
	}
	section := gitConfig.Section(`remote "origin"`)
	if section == nil {
		return nil, fmt.Errorf("remote `origin` is not defined in %s", file)
	}
	url := section.Key("url")
	if url == nil {
		return nil, fmt.Errorf("git origin url is not defined")
	}
	return gitUrls.Parse(url.Value())
}

// GitRepositoryName returns repository name as last path entry from detected
// git repository up the tree or returns error if it fails to do so.
func GitRepositoryName() (string, error) {
	origin, err := getGitOrigin()
	if err != nil {
		return "", err
	}
	base := path.Base(origin.Path)
	return strings.ReplaceAll(base, ".git", ""), nil
}

func findDirWithLeaf(leaf string) (string, error) {
	dir, err := os.Getwd()
	if err != nil {
		return "", fmt.Errorf("cannot find $PWD: %s", err)
	}
	for {
		_, err = os.Stat(fmt.Sprintf("%s/%s", dir, leaf))
		if errors.Is(err, os.ErrNotExist) {
			// TODO: test on windows
			next := path.Dir(dir)
			if dir == next { // or stop at $HOME?..
				return "", fmt.Errorf("cannot find %s anywhere", leaf)
			}
			dir = next
			continue
		}
		if err != nil {
			return "", err
		}
		return dir, nil
	}
}
