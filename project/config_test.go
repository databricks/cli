package project

import (
	"fmt"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestFindProjectRoot(t *testing.T) {
	wd, _ := os.Getwd()
	defer os.Chdir(wd)
	err := os.Chdir("testdata/a/b/c")
	assert.NoError(t, err)
	root, err := findProjectRoot()
	assert.NoError(t, err)

	assert.Equal(t, fmt.Sprintf("%s/testdata", wd), root)
}

func TestFindProjectRootInRoot(t *testing.T) {
	wd, _ := os.Getwd()
	defer os.Chdir(wd)
	err := os.Chdir("/tmp")
	assert.NoError(t, err)
	_, err = findProjectRoot()
	assert.EqualError(t, err, "cannot find databricks.yml anywhere")
}

func TestLoadProjectConf(t *testing.T) {
	wd, _ := os.Getwd()
	defer os.Chdir(wd)
	os.Chdir("testdata/a/b/c")

	prj, err := loadProjectConf()
	assert.NoError(t, err)
	assert.Equal(t, "dev", prj.Name)
	assert.True(t, prj.IsDevClusterJustReference())
}
