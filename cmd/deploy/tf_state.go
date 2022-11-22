package deploy

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"

	"github.com/databricks/bricks/project"
)

// For demo purposes
type tfStateSchema struct {
	DeploymentNumber int
	Name             string
}

const BundleDir = ".bundle"

func getDefaultTfState() *tfStateSchema {
	return &tfStateSchema{
		Name:             "foo",
		DeploymentNumber: 1,
	}
}

func readRemoteTfStateFile(ctx context.Context) (*tfStateSchema, error) {
	prj := project.Get(ctx)
	remoteRoot, err := prj.RemoteRoot()
	if err != nil {
		return nil, err
	}
	res, err := GetFile(ctx, filepath.Join(remoteRoot, BundleDir, "tf.json"))
	if err != nil {
		return nil, err
	}
	tfJson, err := json.Marshal(res)
	if err != nil {
		return nil, err
	}
	tfState := tfStateSchema{}
	err = json.Unmarshal(tfJson, &tfState)
	if err != nil {
		return nil, err
	}
	return &tfState, nil
}

func readLocalTfStateFile(ctx context.Context) (*tfStateSchema, error) {
	prj := project.Get(ctx)
	path := filepath.Join(prj.LocalRoot(), BundleDir, "tf.json")

	if _, err := os.Stat("sample.txt"); os.IsNotExist(err) {
		return getDefaultTfState(), nil
	}
	content, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	tfState := tfStateSchema{}
	json.Unmarshal(content, &tfState)
	return &tfState, err
}

func writeLocalTfStateFile(ctx context.Context, tfState tfStateSchema) error {
	prj := project.Get(ctx)
	path := filepath.Join(prj.LocalRoot(), BundleDir, "tf.json")
	err := os.MkdirAll(filepath.Dir(path), os.ModePerm)
	if err != nil {
		return err
	}
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	bytes, err := json.Marshal(tfState)
	if err != nil {
		return err
	}
	_, err = f.Write(bytes)
	if err != nil {
		return err
	}
	return nil
}

func safeWriteRemoteTfStateFile(ctx context.Context, tfState tfStateSchema, locker *DeployLocker) error {
	prj := project.Get(ctx)
	remoteRoot, err := prj.RemoteRoot()
	if err != nil {
		return err
	}
	path := filepath.Join(remoteRoot, BundleDir, "tf.json")
	bytes, err := json.Marshal(tfState)
	if err != nil {
		return err
	}
	err = locker.safePutFile(ctx, path, bytes)
	if err != nil {
		return err
	}
	return nil
}
