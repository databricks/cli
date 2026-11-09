package aircmd

import (
	"archive/tar"
	"compress/gzip"
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"runtime"

	"github.com/databricks/cli/libs/tarpack"
	"golang.org/x/sync/errgroup"
)

// A segmented snapshot is a .tar.gz made of several concatenated gzip members whose
// uncompressed payload is one tar archive. Plain `tar -xzf` reads it as a single
// stream, so launchers that predate the index keep working. The index sidecar
// (<tarball>.idx.json) lets a launcher decompress members in parallel.

const (
	segmentFormat        = "air-segmented-tar-gz"
	segmentFormatVersion = 1
	segmentIndexSuffix   = ".idx.json"
	segmentTargetBytes   = 32 << 20
	segmentMaxMembers    = 16

	tarBlockSize = 512
)

// segmentIndex describes the members of a segmented tarball. Members are contiguous
// and cover [0, Size).
type segmentIndex struct {
	Format    string          `json:"format"`
	Version   int             `json:"version"`
	Component string          `json:"component"`
	Size      int64           `json:"size"`
	Members   []segmentMember `json:"members"`
}

// segmentMember is the byte range of one gzip member in the tarball.
type segmentMember struct {
	Offset int64 `json:"offset"`
	Length int64 `json:"length"`
}

// tarSize estimates the uncompressed tar bytes the entry occupies (header plus padded
// data), matching what tarpack writes for regular files and symlinks. Other file types
// are skipped by tarpack and take no space.
func (f snapshotFile) tarSize() int64 {
	mode := os.FileMode(f.mode)
	switch {
	case mode.IsRegular():
		return tarBlockSize + (f.size+tarBlockSize-1)/tarBlockSize*tarBlockSize
	case mode&os.ModeSymlink != 0:
		return tarBlockSize
	}
	return 0
}

// planSegments returns the index of the first entry of each member, given each
// entry's tar size. Members are contiguous and balanced by bytes: the member count is
// ceil(total/targetBytes) capped at maxMembers, and an entry goes to the member
// containing its starting offset. There is always at least one member.
func planSegments(sizes []int64, targetBytes int64, maxMembers int) []int {
	var total int64
	for _, s := range sizes {
		total += s
	}
	if total == 0 {
		return []int{0}
	}
	n := max(1, min(maxMembers, int((total+targetBytes-1)/targetBytes)))

	starts := []int{0}
	var offset int64
	current := 0
	for i, s := range sizes {
		member := int(offset * int64(n) / total)
		if member != current {
			starts = append(starts, i)
			current = member
		}
		offset += s
	}
	return starts
}

// writeSegmentedTarball writes entries to outputTarball as independently compressed
// gzip members and returns the index of their byte ranges. sizes[i] is the estimated
// tar size of entries[i] and is used only for balancing.
func writeSegmentedTarball(ctx context.Context, component, outputTarball string, entries []tarpack.Entry, sizes []int64, targetBytes int64, maxMembers int) (segmentIndex, error) {
	starts := planSegments(sizes, targetBytes, maxMembers)
	return writeSegments(ctx, component, outputTarball, len(starts), func(ctx context.Context, i int, w io.Writer) error {
		end := len(entries)
		if i+1 < len(starts) {
			end = starts[i+1]
		}
		return writeMemberTar(ctx, w, entries[starts[i]:end], i == len(starts)-1)
	})
}

// writeSegments runs writeMember for n members, spooling each to a temp file with at most
// runtime.NumCPU() in flight, then appends the files to outputTarball in member order. The
// output is assembled only after every member succeeds, so a failed member never leaves a
// partial tarball behind.
func writeSegments(ctx context.Context, component, outputTarball string, n int, writeMember func(ctx context.Context, i int, w io.Writer) error) (segmentIndex, error) {
	dir, err := os.MkdirTemp("", "air-snapshot-members-")
	if err != nil {
		return segmentIndex{}, fmt.Errorf("failed to create snapshot members dir: %w", err)
	}
	defer os.RemoveAll(dir)

	paths := make([]string, n)
	g, gctx := errgroup.WithContext(ctx)
	g.SetLimit(runtime.NumCPU())
	for i := range n {
		paths[i] = filepath.Join(dir, fmt.Sprintf("member-%d", i))
		g.Go(func() error {
			return spoolFile(gctx, paths[i], func(ctx context.Context, w io.Writer) error {
				return writeMember(ctx, i, w)
			})
		})
	}
	if err := g.Wait(); err != nil {
		return segmentIndex{}, err
	}
	return concatMembers(component, outputTarball, paths)
}

// spoolFile creates path and fills it with the output of write.
func spoolFile(ctx context.Context, path string, write func(ctx context.Context, w io.Writer) error) error {
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	if err := write(ctx, f); err != nil {
		f.Close()
		return err
	}
	return f.Close()
}

// concatMembers appends the member files at paths, in order, to outputTarball and returns
// the index of their byte ranges.
func concatMembers(component, outputTarball string, paths []string) (segmentIndex, error) {
	out, err := os.Create(outputTarball)
	if err != nil {
		return segmentIndex{}, fmt.Errorf("failed to create tarball: %w", err)
	}
	defer out.Close()

	index := segmentIndex{Format: segmentFormat, Version: segmentFormatVersion, Component: component}
	for _, p := range paths {
		length, err := appendFile(out, p)
		if err != nil {
			return segmentIndex{}, fmt.Errorf("failed to append snapshot member: %w", err)
		}
		index.Members = append(index.Members, segmentMember{Offset: index.Size, Length: length})
		index.Size += length
	}
	if err := out.Close(); err != nil {
		return segmentIndex{}, fmt.Errorf("failed to finalize tarball: %w", err)
	}
	return index, nil
}

// writeMemberTar writes entries to w as one gzip member. Members other than the last
// omit the tar end-of-archive marker.
func writeMemberTar(ctx context.Context, w io.Writer, entries []tarpack.Entry, last bool) error {
	gz, err := gzip.NewWriterLevel(ctxWriter{ctx, w}, gzip.DefaultCompression)
	if err != nil {
		return err
	}
	write := tarpack.WriteFragment
	if last {
		write = tarpack.Write
	}
	if err := write(gz, entries); err != nil {
		return fmt.Errorf("failed to create plain tarball: %w", err)
	}
	if err := gz.Close(); err != nil {
		return fmt.Errorf("failed to finalize gzip: %w", err)
	}
	return nil
}

// writeTarSegments streams the tar archive in tarStream to outputTarball as gzip members. The
// archive size is not known up front, so the caller passes targetBytes, the size at which a
// member closes at the next entry boundary. Once maxMembers members are open, the last one
// takes the rest. Each member is first written as raw tar to a temp file, so the stream is
// never blocked on compression. Members then compress concurrently, at most runtime.NumCPU()
// at a time, and the output is assembled only after all of them succeed.
func writeTarSegments(ctx context.Context, component, outputTarball string, tarStream io.Reader, targetBytes int64, maxMembers int) (segmentIndex, error) {
	dir, err := os.MkdirTemp("", "air-snapshot-members-")
	if err != nil {
		return segmentIndex{}, fmt.Errorf("failed to create snapshot members dir: %w", err)
	}
	defer os.RemoveAll(dir)

	ctx, cancel := context.WithCancel(ctx)
	defer cancel()
	g, gctx := errgroup.WithContext(ctx)
	spool := &segmentSpool{dir: dir, g: g, slots: make(chan struct{}, runtime.NumCPU())}
	streamErr := spool.stream(gctx, tarStream, targetBytes, maxMembers)
	if streamErr != nil {
		// The stream error is the root cause. Waiting only lets the compressors stop.
		cancel()
		_ = g.Wait()
		return segmentIndex{}, streamErr
	}
	if err := g.Wait(); err != nil {
		return segmentIndex{}, err
	}
	return concatMembers(component, outputTarball, spool.members)
}

// segmentSpool writes members as raw tar files and compresses each one in the background.
type segmentSpool struct {
	dir     string
	g       *errgroup.Group
	slots   chan struct{}
	members []string // compressed member paths, in order
}

// rawMember is a member being written as raw tar, counting its bytes.
type rawMember struct {
	raw        *os.File
	rawPath    string
	memberPath string
	tar        *tar.Writer
	bytes      int64
}

func (m *rawMember) Write(p []byte) (int, error) {
	n, err := m.raw.Write(p)
	m.bytes += int64(n)
	return n, err
}

// stream copies the entries of tarStream into members, closing a member at an entry boundary
// once it holds targetBytes, and never opening more than maxMembers.
func (s *segmentSpool) stream(ctx context.Context, tarStream io.Reader, targetBytes int64, maxMembers int) error {
	cur, err := s.open()
	if err != nil {
		return err
	}
	tr := tar.NewReader(tarStream)
	for {
		hdr, err := tr.Next()
		if errors.Is(err, io.EOF) {
			break
		}
		if err != nil {
			return fmt.Errorf("failed to read git archive: %w", err)
		}
		if cur.bytes >= targetBytes && len(s.members) < maxMembers {
			if err := s.close(ctx, cur, false); err != nil {
				return err
			}
			if cur, err = s.open(); err != nil {
				return err
			}
		}
		if err := cur.tar.WriteHeader(hdr); err != nil {
			return fmt.Errorf("failed to write snapshot entry %s: %w", hdr.Name, err)
		}
		if _, err := io.Copy(cur.tar, tr); err != nil {
			return fmt.Errorf("failed to write snapshot entry %s: %w", hdr.Name, err)
		}
	}
	return s.close(ctx, cur, true)
}

// open starts the next member as a raw tar file.
func (s *segmentSpool) open() (*rawMember, error) {
	i := len(s.members)
	raw, err := os.Create(filepath.Join(s.dir, fmt.Sprintf("raw-%d", i)))
	if err != nil {
		return nil, fmt.Errorf("failed to create snapshot member: %w", err)
	}
	memberPath := filepath.Join(s.dir, fmt.Sprintf("member-%d", i))
	s.members = append(s.members, memberPath)
	m := &rawMember{raw: raw, rawPath: raw.Name(), memberPath: memberPath}
	m.tar = tar.NewWriter(m)
	return m, nil
}

// close finishes a raw member and queues its compression. Only the last member gets the
// end-of-archive marker.
func (s *segmentSpool) close(ctx context.Context, m *rawMember, last bool) error {
	if last {
		if err := m.tar.Close(); err != nil {
			return fmt.Errorf("failed to finish snapshot tar: %w", err)
		}
	} else if err := m.tar.Flush(); err != nil {
		return fmt.Errorf("failed to finish snapshot member: %w", err)
	}
	if err := m.raw.Close(); err != nil {
		return fmt.Errorf("failed to finish snapshot member: %w", err)
	}
	s.g.Go(func() error {
		s.slots <- struct{}{}
		defer func() { <-s.slots }()
		return compressRaw(ctx, m.rawPath, m.memberPath)
	})
	return nil
}

// compressRaw gzips the raw tar at rawPath into the member file at memberPath.
func compressRaw(ctx context.Context, rawPath, memberPath string) error {
	raw, err := os.Open(rawPath)
	if err != nil {
		return err
	}
	defer raw.Close()
	return spoolFile(ctx, memberPath, func(ctx context.Context, w io.Writer) error {
		gz, err := gzip.NewWriterLevel(ctxWriter{ctx, w}, gzip.DefaultCompression)
		if err != nil {
			return err
		}
		if _, err := io.Copy(gz, raw); err != nil {
			return fmt.Errorf("failed to create plain tarball: %w", err)
		}
		return gz.Close()
	})
}

// appendFile copies the file at path onto the end of w and returns the bytes copied.
func appendFile(w io.Writer, path string) (int64, error) {
	f, err := os.Open(path)
	if err != nil {
		return 0, err
	}
	defer f.Close()
	return io.Copy(w, f)
}
