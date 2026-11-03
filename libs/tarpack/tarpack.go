// Package tarpack writes files from the local filesystem into a tar archive in
// Go, as a replacement for shelling out to the `tar` command. The caller
// supplies the destination io.Writer, so compression stays its choice: wrap the
// writer in compress/gzip or klauspost/pgzip (or nothing) — tarpack only emits
// the tar stream.
package tarpack

import (
	"archive/tar"
	"fmt"
	"io"
	"io/fs"
	"os"
)

// Entry is one file to add to the archive.
type Entry struct {
	// Name is the slash-separated name the entry is given in the archive.
	Name string
	// Path is the local filesystem path the content is read from.
	Path string
}

// Write writes entries to w as a tar stream and finalizes it (writes the tar
// footer). It does not close w, so the caller can close a wrapping gzip writer
// afterward. Regular files are copied and symlinks are stored as symlink
// entries; any other file type (e.g. a directory) is skipped, since directories
// are implied by entry names. Each entry's mode and mtime are preserved, which
// matches what `tar` stored for the same file.
func Write(w io.Writer, entries []Entry) error {
	tw := tar.NewWriter(w)
	for _, e := range entries {
		if err := writeEntry(tw, e); err != nil {
			return err
		}
	}
	return tw.Close()
}

func writeEntry(tw *tar.Writer, e Entry) error {
	// Lstat, not Stat, so a symlink is archived as a link rather than followed.
	info, err := os.Lstat(e.Path)
	if err != nil {
		return fmt.Errorf("stat %s: %w", e.Path, err)
	}
	mode := info.Mode()
	if !mode.IsRegular() && mode&fs.ModeSymlink == 0 {
		return nil
	}

	var link string
	if mode&fs.ModeSymlink != 0 {
		link, err = os.Readlink(e.Path)
		if err != nil {
			return fmt.Errorf("readlink %s: %w", e.Path, err)
		}
	}
	hdr, err := tar.FileInfoHeader(info, link)
	if err != nil {
		return fmt.Errorf("tar header for %s: %w", e.Name, err)
	}
	hdr.Name = e.Name
	if err := tw.WriteHeader(hdr); err != nil {
		return fmt.Errorf("write header for %s: %w", e.Name, err)
	}
	if !mode.IsRegular() {
		return nil
	}

	f, err := os.Open(e.Path)
	if err != nil {
		return fmt.Errorf("open %s: %w", e.Path, err)
	}
	defer f.Close()
	if _, err := io.Copy(tw, f); err != nil {
		return fmt.Errorf("write %s: %w", e.Name, err)
	}
	return nil
}
