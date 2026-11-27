// Command coverpack packs the coverage of a test run into one small file for the CI dashboard
// (format.go).
//
// It reads the coverage directory of the unit tests (-unit, CLI_GOCOVERDIR_UNIT) and that of
// the acceptance tests (-acc, CLI_GOCOVERDIR: a directory per test, with the test name in its
// "name" file), and decodes Go's binary coverage files itself (govcov.go). `go tool covdata`
// would have to run once per test, and each time parse the CLI's meta-data (about 1 MB, of
// which every test directory has a copy); here each distinct meta-data file is parsed once and
// a test costs one pass over its counter files.
//
// Go measures blocks, which never overlap, so the coverage of a file is one bit per block.
// Most tests cover a file the same way as many other tests (the CLI's startup, mostly), so each
// file stores its distinct bitmaps once and a test names, per file it covered, which of them it
// has.
package main

import (
	"bufio"
	"cmp"
	"crypto/sha1"
	"encoding/hex"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"io/fs"
	"maps"
	"math/bits"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"slices"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/klauspost/compress/zstd"
)

func main() {
	var o options
	flag.StringVar(&o.unitDir, "unit", "", "unit coverage directory (CLI_GOCOVERDIR_UNIT)")
	flag.StringVar(&o.accDir, "acc", "", "acceptance coverage directory (CLI_GOCOVERDIR)")
	flag.StringVar(&o.root, "root", ".", "root of the module, where the source files are read")
	flag.StringVar(&o.meta, "meta", "{}", "JSON object for the meta (sha, os, cell, ...)")
	flag.StringVar(&o.output, "o", "", "the .bin.zst to write")
	dumpPath := flag.String("dump", "", "print `FILE` as Go cover profile text instead")
	commitOf := flag.String("commit", "", "print the commit that the build of `FILE` tested instead")
	var c compareOptions
	compareMode := flag.Bool("compare", false, "write the markdown of a PR comment instead (compare.go)")
	flag.Func("base", "with -compare: a `FILE` of the base build (repeated for each cell)", func(s string) error { c.base = append(c.base, s); return nil })
	flag.Func("head", "with -compare: a `FILE` of the PR's build (repeated for each cell)", func(s string) error { c.head = append(c.head, s); return nil })
	flag.StringVar(&c.repo, "repo", ".", "with -compare: the git checkout to read the PR's change from")
	flag.StringVar(&c.from, "from", "", "with -compare: the commit the PR's change starts from")
	flag.StringVar(&c.to, "to", "", "with -compare: the commit the PR's build tested")
	flag.StringVar(&c.baseDesc, "base-desc", "", "with -compare: markdown naming the base build")
	flag.StringVar(&c.headDesc, "head-desc", "", "with -compare: markdown naming the PR's build")
	flag.Parse()

	var err error
	switch {
	case *compareMode:
		c.output = o.output
		err = compare(c)
	case *commitOf != "":
		var commit string
		if commit, err = metaCommit(*commitOf); err == nil {
			fmt.Println(commit)
		}
	case *dumpPath != "":
		err = dump(os.Stdout, *dumpPath)
	case o.output == "":
		err = errors.New("-o is required")
	default:
		err = pack(o)
	}
	if err != nil {
		fmt.Fprintln(os.Stderr, "coverpack:", err)
		os.Exit(1)
	}
}

type options struct {
	unitDir, accDir, root, meta, output string
}

// scanDir adds the meta-data files of dir to metas (by their hash, which names them) and
// returns its counter files.
func scanDir(dir string, metas map[[16]byte]string) ([]string, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, err
	}
	var counters []string
	for _, e := range entries {
		name := e.Name()
		switch {
		case strings.HasPrefix(name, "covmeta."):
			h, err := hex.DecodeString(strings.TrimPrefix(name, "covmeta."))
			if err != nil || len(h) != 16 {
				return nil, fmt.Errorf("%s: unexpected meta-data file name", filepath.Join(dir, name))
			}
			var hash [16]byte
			copy(hash[:], h)
			if _, ok := metas[hash]; !ok {
				metas[hash] = filepath.Join(dir, name)
			}
		case strings.HasPrefix(name, "covcounters."):
			counters = append(counters, filepath.Join(dir, name))
		}
	}
	return counters, nil
}

type acceptanceTest struct {
	name     string
	counters []string
}

// acceptanceTests returns the tests that ran the CLI, sorted by name.
func acceptanceTests(dir string, metas map[[16]byte]string) ([]acceptanceTest, error) {
	entries, err := os.ReadDir(dir)
	if errors.Is(err, fs.ErrNotExist) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	var tests []acceptanceTest
	for _, e := range entries {
		d := filepath.Join(dir, e.Name())
		name, err := os.ReadFile(filepath.Join(d, "name"))
		if !e.IsDir() || errors.Is(err, fs.ErrNotExist) {
			continue
		}
		if err != nil {
			return nil, err
		}
		counters, err := scanDir(d, metas)
		if err != nil {
			return nil, err
		}
		if len(counters) > 0 {
			tests = append(tests, acceptanceTest{name: string(name), counters: counters})
		}
	}
	slices.SortFunc(tests, func(a, b acceptanceTest) int { return strings.Compare(a.name, b.name) })
	return tests, nil
}

func compareBlocks(a, b block) int {
	return cmp.Or(cmp.Compare(a.sl, b.sl), cmp.Compare(a.sc, b.sc), cmp.Compare(a.el, b.el), cmp.Compare(a.ec, b.ec), cmp.Compare(a.stmts, b.stmts))
}

// catalog numbers every block of every package: the blocks of a file get consecutive ids, in
// the order of the file, and the files are in the order of their paths.
type catalog struct {
	module   string
	layouts  []*layout
	layoutOf []int32 // block id -> layout
	files    map[string]int
}

func newCatalog(pkgs map[[16]byte]*pkgMeta) (*catalog, error) {
	byFile := map[string]map[block]struct{}{}
	c := &catalog{files: map[string]int{}}
	for _, p := range pkgs {
		if c.module == "" {
			c.module = p.module
		} else if p.module != c.module {
			return nil, fmt.Errorf("packages of modules %q and %q", c.module, p.module)
		}
		for _, units := range p.funcs {
			for _, u := range units {
				if byFile[u.file] == nil {
					byFile[u.file] = map[block]struct{}{}
				}
				byFile[u.file][u] = struct{}{}
			}
		}
	}
	prefix := c.module + "/"
	id := map[block]int32{}
	for _, file := range slices.Sorted(maps.Keys(byFile)) {
		path, ok := strings.CutPrefix(file, prefix)
		if !ok {
			return nil, fmt.Errorf("%s is outside module %s", file, c.module)
		}
		l := &layout{path: path, blocks: slices.SortedFunc(maps.Keys(byFile[file]), compareBlocks), first: len(c.layoutOf)}
		for _, b := range l.blocks {
			id[b] = int32(len(c.layoutOf))
			c.layoutOf = append(c.layoutOf, int32(len(c.layouts)))
		}
		c.files[file] = len(c.layouts)
		c.layouts = append(c.layouts, l)
	}
	for _, p := range pkgs {
		p.ids = make([][]int32, len(p.funcs))
		for f, units := range p.funcs {
			p.ids[f] = make([]int32, len(units))
			for u, b := range units {
				p.ids[f][u] = id[b]
			}
		}
	}
	return c, nil
}

// inventory is the layouts of the files of the packages of the given meta-data files.
func (c *catalog) inventory(metas map[[16]byte]string, pkgsOf map[[16]byte][]*pkgMeta) []int {
	set := map[int]struct{}{}
	for h := range metas {
		for _, p := range pkgsOf[h] {
			for _, units := range p.funcs {
				for _, u := range units {
					set[c.files[u.file]] = struct{}{}
				}
			}
		}
	}
	return slices.Sorted(maps.Keys(set))
}

// covered is the layouts that have a bit set.
func (c *catalog) covered(b bitset) []int {
	var files []int
	for w, word := range b {
		for word != 0 {
			l := int(c.layoutOf[w*64+bits.TrailingZeros64(word)])
			if len(files) == 0 || files[len(files)-1] != l {
				files = append(files, l)
			}
			word &= word - 1
		}
	}
	return files
}

// readSources fills in the git blob sha and the line count of each layout, and returns how
// many files were not found.
func readSources(root string, layouts []*layout) int {
	missing := 0
	for _, l := range layouts {
		data, err := os.ReadFile(filepath.Join(root, filepath.FromSlash(l.path)))
		if err != nil {
			missing++
			continue
		}
		h := sha1.New()
		h.Write([]byte("blob " + strconv.Itoa(len(data)) + "\x00"))
		h.Write(data)
		copy(l.sha[:], h.Sum(nil))
		l.lines = strings.Count(string(data), "\n")
		if len(data) > 0 && data[len(data)-1] != '\n' {
			l.lines++
		}
	}
	return missing
}

// parallel runs f(i) for i in [0, n) on all CPUs and returns the errors joined.
func parallel(n int, f func(i int) error) error {
	errs := make([]error, n)
	var next atomic.Int64
	var wg sync.WaitGroup
	for range runtime.NumCPU() {
		wg.Go(func() {
			for i := int(next.Add(1)) - 1; i < n; i = int(next.Add(1)) - 1 {
				errs[i] = f(i)
			}
		})
	}
	wg.Wait()
	return errors.Join(errs...)
}

func pack(o options) error {
	start := time.Now()
	var meta map[string]any
	if err := json.Unmarshal([]byte(o.meta), &meta); err != nil {
		return fmt.Errorf("-meta: %w", err)
	}

	unitMetas, accMetas := map[[16]byte]string{}, map[[16]byte]string{}
	var unitCounters []string
	if o.unitDir != "" {
		var err error
		unitCounters, err = scanDir(o.unitDir, unitMetas)
		if err != nil && !errors.Is(err, fs.ErrNotExist) {
			return err
		}
	}
	var tests []acceptanceTest
	if o.accDir != "" {
		var err error
		if tests, err = acceptanceTests(o.accDir, accMetas); err != nil {
			return err
		}
	}

	cache := map[[16]byte]*pkgMeta{}
	pkgsOf := map[[16]byte][]*pkgMeta{}
	for _, metas := range []map[[16]byte]string{unitMetas, accMetas} {
		for h, path := range metas {
			if _, ok := pkgsOf[h]; ok {
				continue
			}
			data, err := os.ReadFile(path)
			if err != nil {
				return err
			}
			if pkgsOf[h], err = readMeta(data, cache); err != nil {
				return fmt.Errorf("%s: %w", path, err)
			}
		}
	}
	cat, err := newCatalog(cache)
	if err != nil {
		return err
	}
	metaTime := time.Since(start)

	// One job per unit counter file (the unit tests write hundreds into one directory) and one
	// per acceptance test.
	jobs := make([][]string, 0, len(unitCounters)+len(tests))
	for _, f := range unitCounters {
		jobs = append(jobs, []string{f})
	}
	for _, t := range tests {
		jobs = append(jobs, t.counters)
	}
	results := make([]bitset, len(jobs))
	err = parallel(len(jobs), func(i int) error {
		results[i] = newBitset(len(cat.layoutOf))
		for _, path := range jobs[i] {
			data, err := os.ReadFile(path)
			if err != nil {
				return err
			}
			var unknown error
			err = readCounters(data, func(h [16]byte, pkg, fn uint32, counters []uint32) {
				pkgs := pkgsOf[h]
				if int(pkg) >= len(pkgs) || int(fn) >= len(pkgs[pkg].ids) {
					unknown = fmt.Errorf("counters of function %d of package %d of meta-data %x, which it does not have", fn, pkg, h)
					return
				}
				ids := pkgs[pkg].ids[fn]
				for j, v := range counters {
					if v != 0 && j < len(ids) {
						results[i].set(ids[j])
					}
				}
			})
			if err = cmp.Or(err, unknown); err != nil {
				return fmt.Errorf("%s: %w", path, err)
			}
		}
		return nil
	})
	if err != nil {
		return err
	}
	countersTime := time.Since(start) - metaTime

	var profiles []profile
	accInventory, unitInventory := cat.inventory(accMetas, pkgsOf), cat.inventory(unitMetas, pkgsOf)
	if len(tests) > 0 {
		all := newBitset(len(cat.layoutOf))
		for _, b := range results[len(unitCounters):] {
			all.or(b)
		}
		profiles = append(profiles, profile{kind: kindAll, files: accInventory, bits: all})
	}
	if len(unitCounters) > 0 {
		unit := newBitset(len(cat.layoutOf))
		for _, b := range results[:len(unitCounters)] {
			unit.or(b)
		}
		profiles = append(profiles, profile{kind: kindUnit, files: unitInventory, bits: unit})
	}
	for i, t := range tests {
		b := results[len(unitCounters)+i]
		profiles = append(profiles, profile{kind: kindTest, name: t.name, files: cat.covered(b), bits: b})
	}

	missing := readSources(o.root, cat.layouts)
	meta["go"] = runtime.Version()
	meta["module"] = cat.module
	commit, err := exec.Command("git", "-C", o.root, "rev-parse", "HEAD").Output()
	if err != nil {
		return fmt.Errorf("git rev-parse HEAD in %s: %w", o.root, err)
	}
	meta["commit"] = strings.TrimSpace(string(commit))
	metaBytes, err := json.Marshal(meta)
	if err != nil {
		return err
	}
	data, distinct := encode(metaBytes, cat.layouts, profiles)
	compressed, err := writeCompressed(o.output, data)
	if err != nil {
		return err
	}

	blocksIn := func(inventory []int) int {
		n := 0
		for _, l := range inventory {
			n += len(cat.layouts[l].blocks)
		}
		return n
	}
	fmt.Fprintf(os.Stderr, "%s: %d acceptance tests, %d acceptance blocks, %d unit blocks, %d distinct file bitmaps, "+
		"%d bytes, %d compressed; %d meta-data files, %d counter files, %d sources not found; "+
		"meta-data %s, counters %s, total %s\n",
		o.output, len(tests), blocksIn(accInventory), blocksIn(unitInventory), distinct,
		len(data), compressed, len(pkgsOf), len(unitCounters)+countFiles(tests), missing,
		metaTime.Round(time.Millisecond), countersTime.Round(time.Millisecond), time.Since(start).Round(time.Millisecond))
	return nil
}

func countFiles(tests []acceptanceTest) int {
	n := 0
	for _, t := range tests {
		n += len(t.counters)
	}
	return n
}

func writeCompressed(path string, data []byte) (int64, error) {
	f, err := os.Create(path)
	if err != nil {
		return 0, err
	}
	w, err := zstd.NewWriter(f, zstd.WithEncoderLevel(zstd.SpeedBestCompression))
	if err == nil {
		_, err = w.Write(data)
		err = cmp.Or(err, w.Close())
	}
	var size int64
	if fi, statErr := f.Stat(); statErr == nil {
		size = fi.Size()
	}
	return size, cmp.Or(err, f.Close())
}

func dump(out io.Writer, path string) error {
	d, err := readPacked(path)
	if err != nil {
		return err
	}
	var meta struct {
		Module string `json:"module"`
	}
	if err := json.Unmarshal(d.meta, &meta); err != nil {
		return err
	}
	w := bufio.NewWriter(out)
	if err := dumpText(w, d, meta.Module); err != nil {
		return err
	}
	return w.Flush()
}
