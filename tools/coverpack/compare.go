package main

// Compare mode (-compare): the coverage of a PR's build against that of a main build, as the
// markdown of a PR comment (.github/workflows/coverage-comment.yml). The files of a PR come from
// its author, so they are read as untrusted data: bounded in size, and every name that reaches
// the markdown is reduced to a safe set of characters.

import (
	"bufio"
	"bytes"
	"cmp"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"maps"
	"os"
	"os/exec"
	"path"
	"regexp"
	"slices"
	"strconv"
	"strings"

	"github.com/klauspost/compress/zstd"
)

const (
	maxPacked       = 64 << 20 // bytes of a file once decompressed; the CLI's are about 5 MB
	maxShownFiles   = 15       // files listed with their uncovered changed lines
	maxShownRanges  = 8        // ranges of lines listed per file
	maxShownPackage = 25       // packages in the table of changed packages
	commentMarker   = "<!-- coverage-report -->"
	maxCells        = 8 // files of a build
)

// The measures of a comment, in the order of its table. Acceptance is the merged profile of
// the acceptance tests (kindAll), unit that of the unit tests, combined either.
const (
	measureCombined = iota
	measureAcceptance
	measureUnit
	measures
)

var measureNames = [measures]string{"Combined", "Acceptance", "Unit"}

type compareOptions struct {
	base, head []string // the files of the cells of each build
	repo       string   // the git checkout the diff is read from
	from, to   string   // the PR's change: from main to the commit the head build tested
	baseDesc   string   // markdown naming each build (trusted: written by the workflow)
	headDesc   string
	output     string
}

// fileCov is a file's coverage in a build, its cells merged.
type fileCov struct {
	sha       [20]byte
	blocks    []block
	acc, unit []byte // bitmaps; nil when the file is not part of that build
	conflict  bool   // the cells measured different versions of the file
}

func (f *fileCov) covered(j int) [measures]bool {
	bit := func(bm []byte) bool { return bm != nil && bm[j/8]>>(j%8)&1 != 0 }
	acc, unit := bit(f.acc), bit(f.unit)
	return [measures]bool{acc || unit, acc, unit}
}

type buildCov map[string]*fileCov

func readPacked(p string) (*decoded, error) {
	f, err := os.Open(p)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	r, err := zstd.NewReader(f, zstd.WithDecoderMaxMemory(maxPacked))
	if err != nil {
		return nil, err
	}
	defer r.Close()
	data, err := io.ReadAll(io.LimitReader(r, maxPacked+1))
	if err != nil {
		return nil, fmt.Errorf("%s: %w", p, err)
	}
	if len(data) > maxPacked {
		return nil, fmt.Errorf("%s: larger than %d bytes", p, maxPacked)
	}
	d, err := decode(data)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", p, err)
	}
	return d, nil
}

// loadBuild merges the cells of a build: a block is covered if any cell covered it.
func loadBuild(paths []string) (buildCov, []map[string]any, error) {
	if len(paths) > maxCells {
		return nil, nil, fmt.Errorf("%d files: more than %d cells", len(paths), maxCells)
	}
	cov := buildCov{}
	var metas []map[string]any
	for _, p := range paths {
		d, err := readPacked(p)
		if err != nil {
			return nil, nil, err
		}
		var meta map[string]any
		if err := json.Unmarshal(d.meta, &meta); err != nil {
			return nil, nil, fmt.Errorf("%s: meta: %w", p, err)
		}
		metas = append(metas, meta)
		for _, prof := range d.profiles {
			if prof.kind != kindAll && prof.kind != kindUnit {
				continue
			}
			for i, l := range prof.files {
				lay := d.layouts[l]
				f := cov[lay.path]
				if f == nil {
					f = &fileCov{sha: lay.sha, blocks: lay.blocks}
					cov[lay.path] = f
				} else if f.sha != lay.sha || len(f.blocks) != len(lay.blocks) {
					f.conflict = true
					continue
				}
				dst := &f.acc
				if prof.kind == kindUnit {
					dst = &f.unit
				}
				bm := d.bitmaps[l][prof.index[i]]
				if *dst == nil {
					*dst = make([]byte, len(bm))
				}
				for k := range bm {
					(*dst)[k] |= bm[k]
				}
			}
		}
	}
	return cov, metas, nil
}

type counts struct {
	stmts   int
	covered [measures]int
}

func (c *counts) add(o counts) {
	c.stmts += o.stmts
	for m := range c.covered {
		c.covered[m] += o.covered[m]
	}
}

func (c *counts) pct(m int) float64 {
	if c.stmts == 0 {
		return 0
	}
	return 100 * float64(c.covered[m]) / float64(c.stmts)
}

// packages counts the statements of each package (a directory of the module) of a build.
func (b buildCov) packages() map[string]counts {
	out := map[string]counts{}
	for p, f := range b {
		var c counts
		for j, blk := range f.blocks {
			c.stmts += int(blk.stmts)
			for m, ok := range f.covered(j) {
				if ok {
					c.covered[m] += int(blk.stmts)
				}
			}
		}
		pkg := out[path.Dir(p)]
		pkg.add(c)
		out[path.Dir(p)] = pkg
	}
	return out
}

func total(pkgs map[string]counts) counts {
	var c counts
	for _, p := range pkgs {
		c.add(p)
	}
	return c
}

// addedLines reads a `git diff -U0`: the added lines of each file, by their number in the new
// version, and every file the diff touches. A hunk's body is skipped by its counts, so a line
// of it that looks like a header is not read as one.
func addedLines(diff io.Reader) (map[string][]int, map[string]bool, error) {
	added, touched := map[string][]int{}, map[string]bool{}
	var file string
	body := 0 // lines left in the current hunk
	afterOld := false
	s := bufio.NewScanner(diff)
	s.Buffer(nil, 16<<20)
	for s.Scan() {
		line := s.Text()
		if body > 0 {
			if !strings.HasPrefix(line, `\`) { // "\ No newline at end of file"
				body--
			}
			continue
		}
		wasOld := afterOld
		afterOld = false
		switch {
		case strings.HasPrefix(line, "--- "):
			afterOld = true
			if p, ok := strings.CutPrefix(line[4:], "a/"); ok {
				touched[p] = true
			}
		case wasOld && strings.HasPrefix(line, "+++ "):
			file = ""
			if p, ok := strings.CutPrefix(line[4:], "b/"); ok {
				file = p
				touched[p] = true
			}
		case strings.HasPrefix(line, "@@ "):
			oldCount, start, count, err := hunk(line)
			if err != nil {
				return nil, nil, err
			}
			body = oldCount + count
			for l := start; l < start+count && file != ""; l++ {
				added[file] = append(added[file], l)
			}
		}
	}
	return added, touched, s.Err()
}

// hunk reads "@@ -a,b +c,d @@": b, c and d (a count is 1 when left out).
func hunk(line string) (oldCount, start, count int, err error) {
	fields := strings.Fields(line)
	if len(fields) < 3 || !strings.HasPrefix(fields[1], "-") || !strings.HasPrefix(fields[2], "+") {
		return 0, 0, 0, fmt.Errorf("hunk %q", line)
	}
	span := func(spec string) (int, int, error) {
		first, n, found := strings.Cut(spec, ",")
		if !found {
			n = "1"
		}
		a, err1 := strconv.Atoi(first)
		b, err2 := strconv.Atoi(n)
		if err := cmp.Or(err1, err2); err != nil || b < 0 || b > 1<<24 {
			return 0, 0, fmt.Errorf("hunk %q", line)
		}
		return a, b, nil
	}
	_, oldCount, err = span(fields[1][1:])
	if err != nil {
		return 0, 0, 0, err
	}
	start, count, err = span(fields[2][1:])
	return oldCount, start, count, err
}

// lineStates is, per wanted line that has code, whether any of its code ran, per measure.
func lineStates(f *fileCov, wanted []int) map[int][measures]bool {
	want := map[int]bool{}
	for _, l := range wanted {
		want[l] = true
	}
	lo, hi := slices.Min(wanted), slices.Max(wanted)
	out := map[int][measures]bool{}
	for j, blk := range f.blocks {
		if blk.stmts == 0 {
			continue
		}
		last := int(blk.el)
		if blk.ec <= 1 && blk.el > blk.sl {
			last-- // the block ends before the first byte of its last line
		}
		cov := f.covered(j)
		for l := max(int(blk.sl), lo); l <= min(last, hi); l++ {
			if !want[l] {
				continue
			}
			s := out[l]
			for m := range s {
				s[m] = s[m] || cov[m]
			}
			out[l] = s
		}
	}
	return out
}

type patchFile struct {
	path      string
	uncovered []int // lines with code that nothing ran (combined)
}

type patch struct {
	lines    int // changed lines that have code
	covered  [measures]int
	files    []patchFile     // with uncovered lines, the most first
	skipped  int             // changed files whose coverage is of another version of them
	packages map[string]bool // that the PR changes a Go file of
}

func git(repo string, args ...string) ([]byte, error) {
	cmd := exec.Command("git", append([]string{"-C", repo}, args...)...)
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	out, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("git %s: %w: %s", strings.Join(args, " "), err, strings.TrimSpace(stderr.String()))
	}
	return out, nil
}

// blobs is the git blob sha of each file of a commit.
func blobs(repo, commit string) (map[string][20]byte, error) {
	out, err := git(repo, "ls-tree", "-r", "-z", commit)
	if err != nil {
		return nil, err
	}
	result := map[string][20]byte{}
	for entry := range bytes.SplitSeq(out, []byte{0}) {
		// "<mode> blob <sha>\t<path>"
		meta, p, ok := bytes.Cut(entry, []byte{'\t'})
		fields := bytes.Fields(meta)
		if !ok || len(fields) != 3 || string(fields[1]) != "blob" {
			continue
		}
		var sha [20]byte
		if _, err := hex.Decode(sha[:], fields[2]); err == nil {
			result[string(p)] = sha
		}
	}
	return result, nil
}

func patchCoverage(head buildCov, o compareOptions) (patch, error) {
	var p patch
	diff, err := git(o.repo, "-c", "core.quotePath=false", "diff", "-U0", "--no-color", "--no-ext-diff", "--no-renames", "--src-prefix=a/", "--dst-prefix=b/", o.from, o.to, "--", "*.go")
	if err != nil {
		return p, err
	}
	added, touched, err := addedLines(bytes.NewReader(diff))
	if err != nil {
		return p, err
	}
	p.packages = map[string]bool{}
	for file := range touched {
		p.packages[path.Dir(file)] = true
	}
	shas, err := blobs(o.repo, o.to)
	if err != nil {
		return p, err
	}
	for _, file := range slices.Sorted(maps.Keys(added)) {
		f := head[file]
		if f == nil {
			continue // not in the build: a test, or not built on this OS
		}
		if f.conflict || f.sha != shas[file] {
			p.skipped++
			continue
		}
		states := lineStates(f, added[file])
		pf := patchFile{path: file}
		for _, l := range added[file] {
			s, ok := states[l]
			if !ok {
				continue
			}
			p.lines++
			for m, ran := range s {
				if ran {
					p.covered[m]++
				}
			}
			if !s[measureCombined] {
				pf.uncovered = append(pf.uncovered, l)
			}
		}
		if len(pf.uncovered) > 0 {
			p.files = append(p.files, pf)
		}
	}
	slices.SortStableFunc(p.files, func(a, b patchFile) int { return cmp.Compare(len(b.uncovered), len(a.uncovered)) })
	return p, nil
}

var unsafeName = regexp.MustCompile(`[^A-Za-z0-9_./+=-]`)

// safeName is a path for inside backticks in markdown.
func safeName(s string) string {
	if len(s) > 200 {
		s = s[:200] + "..."
	}
	return unsafeName.ReplaceAllString(s, "?")
}

func thousands(n int) string {
	s := strconv.Itoa(max(n, -n))
	for i := len(s) - 3; i > 0; i -= 3 {
		s = s[:i] + "," + s[i:]
	}
	if n < 0 {
		return "-" + s
	}
	return s
}

func pct(part, whole int) string {
	if whole == 0 {
		return "–"
	}
	return fmt.Sprintf("%.1f%%", 100*float64(part)/float64(whole))
}

func delta(pp float64) string {
	if pp > -0.005 && pp < 0.005 {
		return "0.00 pp"
	}
	return fmt.Sprintf("%+.2f pp", pp)
}

// ranges writes lines (ascending) as "3-5, 9".
func ranges(lines []int, limit int) string {
	var parts []string
	i := 0
	for i < len(lines) {
		j := i
		for j+1 < len(lines) && lines[j+1] == lines[j]+1 {
			j++
		}
		if len(parts) == limit {
			parts = append(parts, "...")
			break
		}
		if i == j {
			parts = append(parts, strconv.Itoa(lines[i]))
		} else {
			parts = append(parts, fmt.Sprintf("%d-%d", lines[i], lines[j]))
		}
		i = j + 1
	}
	return strings.Join(parts, ", ")
}

var cellName = regexp.MustCompile(`^[a-z0-9]{1,32}$`)

// cells is the names of the cells of a build, sorted: "?" for one that is not a plain name.
func cells(metas []map[string]any) []string {
	var names []string
	for _, m := range metas {
		c, _ := m["cell"].(string)
		if !cellName.MatchString(c) {
			c = "?"
		}
		names = append(names, c)
	}
	slices.Sort(names)
	return names
}

func render(w io.Writer, o compareOptions, base, head map[string]counts, metas []map[string]any, p patch) {
	fmt.Fprintln(w, commentMarker)
	fmt.Fprintln(w, "### Coverage")
	fmt.Fprintln(w)
	fmt.Fprintf(w, "Linux (`%s`), statements covered by the unit and the acceptance tests. PR: %s.", strings.Join(cells(metas), "`, `"), o.headDesc)
	if base != nil {
		fmt.Fprintf(w, " Base: %s.", o.baseDesc)
	}
	fmt.Fprintln(w)
	fmt.Fprintln(w)

	ht := total(head)
	if base != nil {
		bt := total(base)
		fmt.Fprintln(w, "| | Changed lines | Total | Δ vs base |")
		fmt.Fprintln(w, "|---|---|---|---|")
		for m, name := range measureNames {
			fmt.Fprintf(w, "| %s | %s | %.1f%% | %s |\n", name, patchCell(p, m), ht.pct(m), delta(ht.pct(m)-bt.pct(m)))
		}
	} else {
		fmt.Fprintln(w, "| | Changed lines | Total |")
		fmt.Fprintln(w, "|---|---|---|")
		for m, name := range measureNames {
			fmt.Fprintf(w, "| %s | %s | %.1f%% |\n", name, patchCell(p, m), ht.pct(m))
		}
	}
	fmt.Fprintln(w)
	if base == nil {
		fmt.Fprintln(w, "No coverage of main was found at or before this PR's base, so there is nothing to compare the totals with.")
		fmt.Fprintln(w)
	}
	if p.skipped > 0 {
		files := "files are"
		if p.skipped == 1 {
			files = "file is"
		}
		fmt.Fprintf(w, "%d changed %s left out: the build measured another version.\n\n", p.skipped, files)
	}

	if len(p.files) > 0 {
		missed := 0
		for _, f := range p.files {
			missed += len(f.uncovered)
		}
		fmt.Fprintf(w, "<details><summary>%s changed lines that no test ran, in %d files</summary>\n\n", thousands(missed), len(p.files))
		for i, f := range p.files {
			if i == maxShownFiles {
				fmt.Fprintf(w, "- and %d more files\n", len(p.files)-maxShownFiles)
				break
			}
			fmt.Fprintf(w, "- `%s`: %s\n", safeName(f.path), ranges(f.uncovered, maxShownRanges))
		}
		fmt.Fprintln(w, "\n</details>")
		fmt.Fprintln(w)
	}

	if base != nil {
		renderPackages(w, base, head, p.packages)
	}
}

func patchCell(p patch, m int) string {
	if p.lines == 0 {
		return "–"
	}
	return fmt.Sprintf("%s (%s of %s)", pct(p.covered[m], p.lines), thousands(p.covered[m]), thousands(p.lines))
}

// renderPackages lists the packages that the PR changes and whose statements or combined
// coverage changed, the most newly uncovered statements first. Other packages can differ too,
// from the commits of main that the base build lacks, but that is not the PR's doing.
func renderPackages(w io.Writer, base, head map[string]counts, changed map[string]bool) {
	type row struct {
		name             string
		base, head       counts
		uncoveredChanged int
	}
	var rows []row
	for name := range maps.Keys(head) {
		rows = append(rows, row{name: name, base: base[name], head: head[name]})
	}
	for name := range maps.Keys(base) {
		if _, ok := head[name]; !ok {
			rows = append(rows, row{name: name, base: base[name]})
		}
	}
	rows = slices.DeleteFunc(rows, func(r row) bool {
		return !changed[r.name] || r.base.stmts == r.head.stmts && r.base.covered[measureCombined] == r.head.covered[measureCombined]
	})
	if len(rows) == 0 {
		return
	}
	for i := range rows {
		r := &rows[i]
		r.uncoveredChanged = (r.head.stmts - r.head.covered[measureCombined]) - (r.base.stmts - r.base.covered[measureCombined])
	}
	slices.SortFunc(rows, func(a, b row) int {
		return cmp.Or(cmp.Compare(max(b.uncoveredChanged, -b.uncoveredChanged), max(a.uncoveredChanged, -a.uncoveredChanged)), strings.Compare(a.name, b.name))
	})
	fmt.Fprintf(w, "<details><summary>%d changed packages</summary>\n\n", len(rows))
	fmt.Fprintln(w, "| Package | Statements | Combined | Δ | Uncovered statements |")
	fmt.Fprintln(w, "|---|---|---|---|---|")
	for i, r := range rows {
		if i == maxShownPackage {
			fmt.Fprintf(w, "\nand %d more packages\n", len(rows)-maxShownPackage)
			break
		}
		name := "`" + safeName(r.name) + "`"
		if r.name == "." {
			name = "(module root)"
		}
		combined, d := pct(r.head.covered[measureCombined], r.head.stmts), "new"
		switch {
		case r.head.stmts == 0:
			combined, d = "–", "removed"
		case r.base.stmts > 0:
			d = delta(r.head.pct(measureCombined) - r.base.pct(measureCombined))
		}
		fmt.Fprintf(w, "| %s | %s (%+d) | %s | %s | %+d |\n", name, thousands(r.head.stmts), r.head.stmts-r.base.stmts, combined, d, r.uncoveredChanged)
	}
	fmt.Fprintln(w, "\n</details>")
}

var hexSha = regexp.MustCompile(`^[0-9a-f]{40}$`)

func compare(o compareOptions) error {
	if len(o.head) == 0 {
		return errors.New("-head is required")
	}
	if !hexSha.MatchString(o.from) || !hexSha.MatchString(o.to) {
		return errors.New("-from and -to must be full commit shas")
	}
	head, metas, err := loadBuild(o.head)
	if err != nil {
		return err
	}
	var base map[string]counts
	if len(o.base) > 0 {
		b, baseMetas, err := loadBuild(o.base)
		if err != nil {
			return err
		}
		if !slices.Equal(cells(baseMetas), cells(metas)) {
			return fmt.Errorf("the base build has cells %v, the PR's %v", cells(baseMetas), cells(metas))
		}
		base = b.packages()
	}
	p, err := patchCoverage(head, o)
	if err != nil {
		return err
	}
	var out bytes.Buffer
	render(&out, o, base, head.packages(), metas, p)
	if o.output == "" {
		_, err = os.Stdout.Write(out.Bytes())
		return err
	}
	return os.WriteFile(o.output, out.Bytes(), 0o644)
}

// metaCommit is the commit a packed file says its build tested.
func metaCommit(p string) (string, error) {
	d, err := readPacked(p)
	if err != nil {
		return "", err
	}
	var meta struct {
		Commit string `json:"commit"`
	}
	if err := json.Unmarshal(d.meta, &meta); err != nil {
		return "", err
	}
	if !hexSha.MatchString(meta.Commit) {
		return "", errors.New("the file names no commit")
	}
	return meta.Commit, nil
}
