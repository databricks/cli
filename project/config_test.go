package project

import (
	"fmt"
	"os"
	"path"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestFindProjectRoot(t *testing.T) {
	wd, _ := os.Getwd()
	defer os.Chdir(wd)
	err := os.Chdir("internal/test/a/b/c")
	assert.NoError(t, err)
	root, err := findProjectRoot()
	assert.NoError(t, err)

	assert.Equal(t, fmt.Sprintf("%s/internal/test", wd), root)
}

func TestFindProjectRootInRoot(t *testing.T) {
	wd, _ := os.Getwd()
	defer os.Chdir(wd)
	err := os.Chdir("/tmp")
	assert.NoError(t, err)
	_, err = findProjectRoot()
	assert.EqualError(t, err, "cannot find databricks.yml anywhere")
}

func TestGetGitOrigin(t *testing.T) {
	origin, err := getGitOrigin()
	assert.NoError(t, err)
	assert.Equal(t, "bricks.git", path.Base(origin.Path))
}

func TestLoadProjectConf(t *testing.T) {
	wd, _ := os.Getwd()
	defer os.Chdir(wd)
	os.Chdir("internal/test/a/b/c")

	prj, err := loadProjectConf()
	assert.NoError(t, err)
	assert.Equal(t, "dev", prj.Name)
	assert.True(t, prj.IsDevClusterJustReference())
}