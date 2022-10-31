package project

import (
	"context"
	"encoding/base64"
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/databricks/bricks/lib/flavor"
	"github.com/databricks/databricks-sdk-go/databricks/apierr"
	"github.com/databricks/databricks-sdk-go/service/workspace"
)

var b64 = base64.StdEncoding

func (p *project) LocalArtifacts(ctx context.Context) (flavor.Artifacts, error) {
	for _, f := range p.flavors {
		if !f.Detected(p) {
			continue
		}
		arts, err := f.LocalArtifacts(ctx, p)
		if err != nil {
			return nil, err
		}
		p.artifacts = append(p.artifacts, arts...)
	}
	return p.artifacts, nil
}

type preparable interface {
	Prepare(ctx context.Context, prj flavor.Project, status func(string)) error
}

func (p *project) Prepare(ctx context.Context, status func(string)) error {
	for _, f := range p.flavors {
		if !f.Detected(p) {
			continue
		}
		prep, ok := f.(preparable)
		if !ok {
			continue
		}
		err := prep.Prepare(ctx, p, status)
		if err != nil {
			return fmt.Errorf("prepare: %w", err)
		}
	}
	return nil
}

type buildable interface {
	Build(ctx context.Context, prj flavor.Project, status func(string)) error
}

func (p *project) Build(ctx context.Context, status func(string)) error {
	for _, f := range p.flavors {
		if !f.Detected(p) {
			continue
		}
		b, ok := f.(buildable)
		if !ok {
			continue
		}
		// TODO: compare last modified artifact vs last modified remote
		// TODO: compare last modified artifact vs last modified fileset
		err := b.Build(ctx, p, status)
		if err != nil {
			return fmt.Errorf("build: %w", err)
		}
	}
	return nil
}

func (p *project) Upload(ctx context.Context, status func(string)) error {
	for _, a := range p.artifacts {
		err := p.upload(ctx, a, status)
		if err != nil {
			return fmt.Errorf("upload: %w", err)
		}
	}
	return nil
}

func (p *project) upload(ctx context.Context, a flavor.Artifact, status func(string)) error {
	kind, local, remote := p.remotePath(a)
	if remote == "" {
		return nil
	}
	localBasename := filepath.Base(local)
	remoteDir := filepath.Dir(remote)
	status(fmt.Sprintf("Uploading %s to %s", localBasename, remoteDir))
	file, err := os.Open(local)
	if err != nil {
		return fmt.Errorf("open: %w", err)
	}
	defer file.Close()
	if kind != flavor.LocalNotebook {
		return p.wsc.Dbfs.Overwrite(ctx, remote, file)
	}
	format, ok := a.NotebookInfo()
	if !ok {
		return fmt.Errorf("unknown notebook: %s", a)
	}
	raw, err := io.ReadAll(file)
	if err != nil {
		return fmt.Errorf("read: %w", err)
	}
	_, err = p.wsc.Workspace.GetStatusByPath(ctx, remoteDir)
	if apierr.IsMissing(err) {
		err = p.wsc.Workspace.MkdirsByPath(ctx, remoteDir)
		if err != nil {
			return err
		}
	}
	return p.wsc.Workspace.Import(ctx, workspace.Import{
		Path:      remote,
		Overwrite: format.Overwrite,
		Format:    format.Format,
		Language:  format.Language,
		Content:   b64.EncodeToString(raw),
	})
}

func (p *project) remotePath(a flavor.Artifact) (flavor.Kind, string, string) {
	libsFmt := fmt.Sprintf("dbfs:/FileStore/%%s/%s/%%s", p.DeploymentIsolationPrefix())
	kind, loc := a.KindAndLocation()
	switch kind {
	case flavor.LocalJar:
		return kind, loc, fmt.Sprintf(libsFmt, "jars", filepath.Base(loc))
	case flavor.LocalWheel:
		return kind, loc, fmt.Sprintf(libsFmt, "wheels", filepath.Base(loc))
	case flavor.LocalEgg:
		return kind, loc, fmt.Sprintf(libsFmt, "eggs", filepath.Base(loc))
	case flavor.LocalNotebook:
		me, err := p.Me()
		if err != nil {
			panic(err)
		}
		return kind, loc, fmt.Sprintf("/Users/%s/%s/%s",
			me.UserName, p.config.Name, a.Notebook.RemoteRelative)
	default:
		return kind, loc, ""
	}
}
