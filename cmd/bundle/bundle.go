package bundle

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"

	"github.com/databricks/databricks-sdk-go/workspaces"
)

type bundleByShreyas struct {
	workspacesClient *workspaces.WorkspacesClient
	localRoot        string
	remoteRoot       string
	env              string
	locker           *DeployLocker
	user             string
}

func CreateBundle(env, localRoot, remoteRoot string) *bundleByShreyas {
	return &bundleByShreyas{
		workspacesClient: workspaces.New(),
		localRoot:        localRoot,
		remoteRoot:       remoteRoot,
		env:              env,
	}
}

func (b *bundleByShreyas) Locker() (*DeployLocker, error) {
	if b.locker != nil {
		return b.locker, nil
	}
	user, err := b.User()
	if err != nil {
		return nil, err
	}
	newLocker, err := CreateLocker(user, false, b.remoteRoot)
	if err != nil {
		return nil, err
	}
	b.locker = newLocker
	return b.locker, nil
}

func (b *bundleByShreyas) User() (string, error) {
	if b.user != "" {
		return b.user, nil
	}
	user, err := b.workspacesClient.CurrentUser.Me(context.Background())
	if err != nil {
		return "", err
	}
	b.user = user.UserName
	return b.user, nil
}

func (b *bundleByShreyas) cacheDir() string {
	return filepath.Join(b.localRoot, ".databricks/bundle")
}

func (b *bundleByShreyas) tfHclPath() string {
	return filepath.Join(b.cacheDir(), b.env, "main.tf")
}

func (b *bundleByShreyas) tfStateRemotePath() string {
	return filepath.Join(b.remoteRoot, ".bundle", "terraform.tfstate")
}

func (b *bundleByShreyas) tfStateLocalPath() string {
	return filepath.Join(b.cacheDir(), b.env, "terraform.tfstate")
}

func (b *bundleByShreyas) exportTerraformState(ctx context.Context) error {
	res, err := GetFileContent(ctx, b.workspacesClient, b.tfStateRemotePath())
	if err != nil && strings.Contains(err.Error(), "File not found.") {
		return nil
	}
	if err != nil {
		if strings.Contains(err.Error(), "File not found.") {
			return nil
		} else {
			return err
		}
	}
	bytes, err := json.MarshalIndent(res, "", "  ")
	if err != nil {
		return err
	}
	err = os.MkdirAll(b.cacheDir(), os.ModeDir)
	if err != nil {
		return err
	}
	f, err := os.OpenFile(b.tfStateLocalPath(), os.O_CREATE|os.O_TRUNC|os.O_WRONLY, os.ModePerm)
	if err != nil {
		return err
	}
	_, err = f.Write(bytes)
	return err
}

func (b *bundleByShreyas) importTerraformState(ctx context.Context) error {
	l, err := b.Locker()
	if err != nil {
		return err
	}
	bytes, err := os.ReadFile(b.tfStateLocalPath())
	// TODO: does every terraform apply create a state file?
	// ie. should we throw and error ignore if a state file is missing
	if err != nil {
		return err
	}
	return l.safePutFile(ctx, b.workspacesClient, b.tfStateRemotePath(), bytes)
}

func (b *bundleByShreyas) Lock(ctx context.Context) error {
	l, err := b.Locker()
	if err != nil {
		return err
	}
	return l.Lock(ctx, b.workspacesClient)
}

func (b *bundleByShreyas) Unlock(ctx context.Context) error {
	l, err := b.Locker()
	if err != nil {
		return err
	}
	return l.Unlock(ctx, b.workspacesClient)
}
