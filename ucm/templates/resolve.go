package templates

import (
	"context"
	"io"
	"io/fs"
	"os"
	"path/filepath"

	"github.com/databricks/cli/libs/cmdio"
)

// ResolveBuiltinOrPassthrough returns (pathOrUrl, cleanup, err). If the input
// matches a built-in ucm template, the embedded FS is extracted to a temp dir
// and that path is returned; otherwise the original value is passed through
// for the shared resolver to handle as a local path or git URL.
//
// Used by cmd/ucm/init.go before constructing libs/template.Resolver.
// Extraction to disk is the simplest way to plug the ucm-owned embed.FS into
// libs/template without forking the resolver — template.Resolver already
// understands local directories via NewLocalReader.
func ResolveBuiltinOrPassthrough(ctx context.Context, input string) (string, func(), error) {
	noop := func() {}

	// Interactive selection from the built-in list when no argument is given.
	if input == "" {
		if !cmdio.IsPromptSupported(ctx) {
			// Nothing to expand; let the shared resolver emit its own error.
			return "", noop, nil
		}
		options := make([]cmdio.Tuple, 0, len(List()))
		for _, b := range List() {
			options = append(options, cmdio.Tuple{Name: b.Name, Id: b.Description})
		}
		name, err := cmdio.SelectOrdered(ctx, options, "Template to use")
		if err != nil {
			return "", noop, err
		}
		// SelectOrdered returns the Id (description); map back to the name.
		input = descriptionToName(name)
	}

	reader := Lookup(input)
	if reader == nil {
		return input, noop, nil
	}

	dir, err := extractBuiltin(ctx, reader, input)
	if err != nil {
		return "", noop, err
	}
	return dir, func() { _ = os.RemoveAll(dir) }, nil
}

// descriptionToName reverses List's (name -> description) mapping so
// cmdio.SelectOrdered's choice can be routed back to the embed entry.
func descriptionToName(description string) string {
	for _, b := range List() {
		if b.Description == description || b.Name == description {
			return b.Name
		}
	}
	return description
}

// extractBuiltin writes the embedded template filesystem to a newly created
// temp directory so template.Resolver.Resolve can consume it as a local path.
func extractBuiltin(ctx context.Context, reader interface {
	SchemaFS(context.Context) (fs.FS, error)
}, name string) (string, error) {
	src, err := reader.SchemaFS(ctx)
	if err != nil {
		return "", err
	}

	dir, err := os.MkdirTemp("", "ucm-init-"+name+"-*")
	if err != nil {
		return "", err
	}

	if err := copyFS(src, dir); err != nil {
		_ = os.RemoveAll(dir)
		return "", err
	}
	return dir, nil
}

// copyFS copies a read-only filesystem tree to a destination on disk.
func copyFS(src fs.FS, dst string) error {
	return fs.WalkDir(src, ".", func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		target := filepath.Join(dst, filepath.FromSlash(path))
		if d.IsDir() {
			return os.MkdirAll(target, 0o755)
		}
		in, err := src.Open(path)
		if err != nil {
			return err
		}
		defer in.Close()
		out, err := os.Create(target)
		if err != nil {
			return err
		}
		defer out.Close()
		if _, err := io.Copy(out, in); err != nil {
			return err
		}
		return nil
	})
}
