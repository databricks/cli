package filer_test

import (
	"context"
	"log/slog"
	"os"
	"testing"

	"github.com/databricks/cli/libs/filer"
	"github.com/databricks/cli/libs/log"
	"github.com/databricks/cli/libs/log/handler"
	"github.com/databricks/cli/libs/sync"
	"github.com/databricks/cli/libs/vfs"
	"github.com/databricks/databricks-sdk-go"
)

func initLogger(t *testing.T, ctx context.Context) context.Context {
	opts := handler.Options{}
	opts.Level = log.LevelTrace
	opts.ReplaceAttr = log.ReplaceAttrFunctions{
		log.ReplaceLevelAttr,
		log.ReplaceSourceAttr,
	}.ReplaceAttr

	h := handler.NewFriendlyHandler(os.Stderr, &opts)

	ctx = log.NewContext(ctx, slog.New(h))
	return ctx
}

func TestExtension(t *testing.T) {
	ctx := initLogger(t, context.Background())

	root := "/Workspace/Users/jingting.lu@databricks.com/dais-cow-bff-5"
	w := databricks.Must(databricks.NewWorkspaceClient())

	// f, err := NewWorkspaceFilesExtensionsClient(w, p)
	// if err != nil {
	// 	t.Fatal(err)
	// }

	// If so, swap out vfs.Path instance of the sync root with one that
	// makes all Workspace File System interactions extension aware.
	p, err := vfs.NewFilerPath(ctx, root, func(path string) (filer.Filer, error) {
		return filer.NewWorkspaceFilesExtensionsClient(w, path)
	})
	if err != nil {
		t.Fatal(err)
	}

	opts := sync.SyncOptions{
		LocalPath:  p,
		RemotePath: "/Workspace/foobar",
		Host:       w.Config.Host,

		Full: false,

		WorkspaceClient: w,
	}

	s, err := sync.New(ctx, opts)
	if err != nil {
		t.Fatal(err)
	}

	s.GetFileList(ctx)

	// entries, err := f.ReadDir(ctx, ".")
	// if err != nil {
	// 	t.Fatal(err)
	// }
	//
	// t.Log(entries)

}
