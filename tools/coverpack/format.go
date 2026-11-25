package main

// The packed file, zstd-compressed. Inside, a uvarint is unsigned LEB128 (encoding/binary's
// uvarint):
//
//	"CLICOV1\n"
//	uvarint n, n bytes      meta, JSON: the -meta object plus "go", "module" and "commit" (the commit that was
//	                        checked out and tested; for a PR, GitHub's merge of it into the base branch)
//	uvarint L               layouts, sorted by path:
//	  uvarint n, n bytes      path relative to the module ("bundle/deploy/x.go")
//	  20 bytes                git blob sha of the file as read (zeros if it was not found)
//	  uvarint                 line count
//	  uvarint B               blocks, sorted by start, then end; per block:
//	    uvarint sl - previous sl, uvarint sc, uvarint el - sl, uvarint ec, uvarint statements
//	                          (as in a Go cover profile: lines and byte columns 1-based, the end exclusive)
//	L times                 the distinct bitmaps of each layout, in layout order:
//	  uvarint P, then P bitmaps of ceil(B / 8) bytes: block i is bit i % 8 of byte i / 8
//	uvarint N               profiles:
//	  1 byte kind             0: all acceptance tests merged, 1: unit tests merged, 2: one acceptance test
//	  uvarint n, n bytes      test name (t.Name()), empty for kinds 0 and 1
//	  uvarint F               files, by layout; per file: uvarint layout - previous layout, uvarint bitmap index
//
// Kinds 0 and 1 list every file of their build, covered or not: they are its inventory. A file
// that a test (kind 2) does not list was not covered by it if the build has the file, else it is
// unknown.

import (
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"math"
)

const magic = "CLICOV1\n"

// maxDecodedBlocks bounds what decode allocates for a file from elsewhere; the CLI has about 130k.
const maxDecodedBlocks = 2_000_000

const (
	kindAll  = 0
	kindUnit = 1
	kindTest = 2
)

type layout struct {
	path   string // relative to the module
	sha    [20]byte
	lines  int
	blocks []block
	first  int // catalog id of blocks[0]; the ids of a layout are consecutive
}

type profile struct {
	kind  byte
	name  string
	files []int // layouts, ascending
	bits  bitset
}

type bitset []uint64

func newBitset(n int) bitset { return make(bitset, (n+63)/64) }

func (b bitset) set(i int32)    { b[i>>6] |= 1 << (i & 63) }
func (b bitset) has(i int) bool { return b[i>>6]>>(i&63)&1 != 0 }

func (b bitset) or(o bitset) {
	for i := range b {
		b[i] |= o[i]
	}
}

// bitmap is the bits of a layout, in the order of the file.
func (b bitset) bitmap(l *layout) []byte {
	out := make([]byte, (len(l.blocks)+7)/8)
	for i := range l.blocks {
		if b.has(l.first + i) {
			out[i/8] |= 1 << (i % 8)
		}
	}
	return out
}

// encode returns the uncompressed file and the number of distinct bitmaps in it.
func encode(meta []byte, layouts []*layout, profiles []profile) ([]byte, int) {
	distinct := make([]map[string]int, len(layouts))
	order := make([][][]byte, len(layouts))
	index := make([][]int, len(profiles))
	for p, prof := range profiles {
		for _, l := range prof.files {
			bm := prof.bits.bitmap(layouts[l])
			if distinct[l] == nil {
				distinct[l] = map[string]int{}
			}
			i, ok := distinct[l][string(bm)]
			if !ok {
				i = len(order[l])
				distinct[l][string(bm)] = i
				order[l] = append(order[l], bm)
			}
			index[p] = append(index[p], i)
		}
	}

	out := []byte(magic)
	put := func(v int) { out = binary.AppendUvarint(out, uint64(v)) }
	putBytes := func(b []byte) { put(len(b)); out = append(out, b...) }

	putBytes(meta)
	put(len(layouts))
	for _, l := range layouts {
		putBytes([]byte(l.path))
		out = append(out, l.sha[:]...)
		put(l.lines)
		put(len(l.blocks))
		prev := uint32(0)
		for _, b := range l.blocks {
			put(int(b.sl - prev))
			put(int(b.sc))
			put(int(b.el - b.sl))
			put(int(b.ec))
			put(int(b.stmts))
			prev = b.sl
		}
	}
	total := 0
	for l := range layouts {
		put(len(order[l]))
		for _, bm := range order[l] {
			out = append(out, bm...)
		}
		total += len(order[l])
	}
	put(len(profiles))
	for p, prof := range profiles {
		out = append(out, prof.kind)
		putBytes([]byte(prof.name))
		put(len(prof.files))
		prev := 0
		for i, l := range prof.files {
			put(l - prev)
			put(index[p][i])
			prev = l
		}
	}
	return out, total
}

// decoded is a packed file read back.
type decoded struct {
	meta     []byte
	layouts  []*layout
	bitmaps  [][][]byte // per layout
	profiles []decodedProfile
}

type decodedProfile struct {
	kind  byte
	name  string
	files []int // layouts, ascending
	index []int // bitmap of each file
}

func decode(data []byte) (*decoded, error) {
	c := &cursor{b: data}
	if string(c.take(len(magic))) != magic {
		return nil, errors.New("not a coverpack file")
	}
	str := func() []byte { return c.take(c.count()) }
	d := &decoded{meta: str()}
	d.layouts = make([]*layout, c.count())
	blocks := 0
	for i := range d.layouts {
		l := &layout{path: string(str())}
		copy(l.sha[:], c.take(20))
		l.lines = int(c.uleb128())
		n := c.count()
		if blocks += n; blocks > maxDecodedBlocks {
			return nil, errMalformed
		}
		l.blocks = make([]block, n)
		sl := uint64(0)
		for j := range l.blocks {
			sl += c.uleb128()
			sc := c.uleb128()
			el := sl + c.uleb128()
			ec, stmts := c.uleb128(), c.uleb128()
			// The lines of a block are within its file, which compare walks (compare.go).
			if c.err == nil && (el > math.MaxUint32 || l.lines > 0 && el > uint64(l.lines)) {
				return nil, errMalformed
			}
			l.blocks[j] = block{file: l.path, sl: uint32(sl), sc: uint32(sc), el: uint32(el), ec: uint32(ec), stmts: uint32(stmts)}
		}
		d.layouts[i] = l
	}
	d.bitmaps = make([][][]byte, len(d.layouts))
	for i, l := range d.layouts {
		n, size := c.count(), (len(l.blocks)+7)/8
		// count bounds n by the bytes left, but a bitmap of a file without blocks has none.
		if size == 0 && n > 1 || size > 0 && n > (len(c.b)-c.off)/size {
			return nil, errMalformed
		}
		d.bitmaps[i] = make([][]byte, n)
		for j := range d.bitmaps[i] {
			d.bitmaps[i][j] = c.take(size)
		}
	}
	// A profile takes at least 3 bytes: its kind, the length of its name and its file count.
	n := c.count()
	if n > (len(c.b)-c.off)/3 {
		return nil, errMalformed
	}
	d.profiles = make([]decodedProfile, n)
	for i := range d.profiles {
		if c.err != nil {
			return nil, c.err
		}
		p := decodedProfile{kind: c.u8(), name: string(str())}
		n, l := c.count(), 0
		for range n {
			l += int(c.uleb128())
			p.files = append(p.files, l)
			p.index = append(p.index, int(c.uleb128()))
		}
		d.profiles[i] = p
	}
	if c.err != nil {
		return nil, c.err
	}
	for _, p := range d.profiles {
		for i, l := range p.files {
			if l < 0 || l >= len(d.layouts) || p.index[i] < 0 || p.index[i] >= len(d.bitmaps[l]) {
				return nil, errMalformed
			}
		}
	}
	return d, nil
}

// dumpText writes the profiles as Go cover profile text, each after a "# all", "# unit" or
// "# test NAME" line. A test lists its covered blocks only.
func dumpText(w io.Writer, d *decoded, module string) error {
	if _, err := fmt.Fprintf(w, "# coverage %s\n", d.meta); err != nil {
		return err
	}
	for _, p := range d.profiles {
		header := []string{"# all", "# unit"}
		h := "# test " + p.name
		if int(p.kind) < len(header) {
			h = header[p.kind]
		}
		if _, err := fmt.Fprintln(w, h); err != nil {
			return err
		}
		for i, l := range p.files {
			bm := d.bitmaps[l][p.index[i]]
			for j, b := range d.layouts[l].blocks {
				covered := bm[j/8] >> (j % 8) & 1
				if covered == 0 && p.kind == kindTest {
					continue
				}
				if _, err := fmt.Fprintf(w, "%s/%s:%d.%d,%d.%d %d %d\n", module, b.file, b.sl, b.sc, b.el, b.ec, b.stmts, covered); err != nil {
					return err
				}
			}
		}
	}
	return nil
}
