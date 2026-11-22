package main

// Readers for the files that a binary built with -cover writes to GOCOVERDIR. The formats are
// those of Go's internal/coverage packages (defs.go, decodemeta, decodecounter), which code
// outside the standard library cannot import: version 1 of both, the only one there has been.

import (
	"bytes"
	"encoding/binary"
	"errors"
	"fmt"
)

var (
	metaMagic    = []byte{0x00, 0x63, 0x76, 0x6d}
	counterMagic = []byte{0x00, 0x63, 0x77, 0x6d}

	errTruncated = errors.New("truncated")
	errMalformed = errors.New("malformed")
)

const (
	metaFileVersion       = 1
	counterFileVersion    = 1
	metaSymbolHeaderSize  = 44
	counterFileHeaderSize = 32
	counterFooterSize     = 16
	granularityPerBlock   = 1
	flavorRaw             = 1
	flavorULEB128         = 2
)

// cursor reads a byte slice from the front. The first read past the end sets err; reads after
// that return zeros, so a caller checks err once after a run of reads.
type cursor struct {
	b   []byte
	off int
	err error
}

func (c *cursor) take(n int) []byte {
	if c.err != nil {
		return nil
	}
	if n < 0 || n > len(c.b)-c.off {
		c.err = errTruncated
		return nil
	}
	v := c.b[c.off : c.off+n]
	c.off += n
	return v
}

func (c *cursor) seek(off int) {
	if c.err == nil && (off < 0 || off > len(c.b)) {
		c.err = errTruncated
		return
	}
	c.off = off
}

func (c *cursor) u8() uint8 {
	if b := c.take(1); b != nil {
		return b[0]
	}
	return 0
}

func (c *cursor) u32() uint32 {
	if b := c.take(4); b != nil {
		return binary.LittleEndian.Uint32(b)
	}
	return 0
}

func (c *cursor) u64() uint64 {
	if b := c.take(8); b != nil {
		return binary.LittleEndian.Uint64(b)
	}
	return 0
}

func (c *cursor) uleb128() uint64 {
	var v uint64
	for shift := 0; c.err == nil; shift += 7 {
		if c.off >= len(c.b) {
			c.err = errTruncated
			break
		}
		b := c.b[c.off]
		c.off++
		if shift > 63 {
			c.err = errMalformed
			break
		}
		v |= uint64(b&0x7f) << shift
		if b < 0x80 {
			return v
		}
	}
	return 0
}

// count reads a uleb128 count of items that each take at least one byte, so that a corrupt
// count cannot make the caller allocate more than the input could hold.
func (c *cursor) count() int {
	n := c.uleb128()
	if c.err == nil && n > uint64(len(c.b)-c.off) {
		c.err = errMalformed
		return 0
	}
	return int(n)
}

// block is a Go coverage unit: a span of statements that run together. Lines and columns are
// 1-based, columns count bytes, and the end is exclusive.
type block struct {
	file           string // as Go names it: import path of the package, then the file name
	sl, sc, el, ec uint32
	stmts          uint32
}

// pkgMeta is the coverage meta-data of one package: the blocks of each of its functions, in
// the order of the counters.
type pkgMeta struct {
	path   string
	module string
	funcs  [][]block
	ids    [][]int32 // the catalog ids of funcs, filled in once the catalog is known
}

// readMeta returns the packages of a covmeta file, indexed as counter files index them. A
// package is decoded once: cache holds the packages seen so far, by their hash.
func readMeta(data []byte, cache map[[16]byte]*pkgMeta) ([]*pkgMeta, error) {
	c := &cursor{b: data}
	if !bytes.Equal(c.take(4), metaMagic) {
		return nil, errors.New("not a coverage meta-data file")
	}
	if v := c.u32(); v > metaFileVersion {
		return nil, fmt.Errorf("meta-data file version %d, only %d is known", v, metaFileVersion)
	}
	c.u64() // total length
	n := c.u64()
	c.take(16 + 4 + 4 + 1) // hash, string table offset and length (only package payloads have strings), counter mode
	if g := c.u8(); c.err == nil && g != granularityPerBlock {
		return nil, fmt.Errorf("counter granularity %d, only per block is supported", g)
	}
	c.take(6)
	if c.err == nil && n > uint64(len(data))/16 {
		return nil, errMalformed
	}
	offsets := make([]uint64, n)
	for i := range offsets {
		offsets[i] = c.u64()
	}
	pkgs := make([]*pkgMeta, n)
	for i := range pkgs {
		length := c.u64()
		if c.err != nil {
			break
		}
		if offsets[i] > uint64(len(data)) || length > uint64(len(data))-offsets[i] {
			return nil, errTruncated
		}
		p, err := readPackage(data[offsets[i]:offsets[i]+length], cache)
		if err != nil {
			return nil, fmt.Errorf("package %d: %w", i, err)
		}
		pkgs[i] = p
	}
	if c.err != nil {
		return nil, c.err
	}
	return pkgs, nil
}

func readPackage(b []byte, cache map[[16]byte]*pkgMeta) (*pkgMeta, error) {
	c := &cursor{b: b}
	c.u32() // length
	c.u32() // package name
	pathIndex, moduleIndex := c.u32(), c.u32()
	var hash [16]byte
	copy(hash[:], c.take(16))
	c.take(4)
	c.u32() // number of files
	numFuncs := int(c.u32())
	if c.err != nil {
		return nil, c.err
	}
	if p := cache[hash]; p != nil {
		return p, nil
	}

	c.seek(metaSymbolHeaderSize + 4*numFuncs)
	strs := make([]string, c.count())
	for i := range strs {
		strs[i] = string(c.take(c.count()))
	}
	str := func(i uint64) string {
		if i >= uint64(len(strs)) {
			if c.err == nil {
				c.err = errMalformed
			}
			return ""
		}
		return strs[i]
	}

	// The seek past the function offsets above bounds numFuncs by the file size.
	if c.err != nil {
		return nil, c.err
	}
	p := &pkgMeta{path: str(uint64(pathIndex)), module: str(uint64(moduleIndex)), funcs: make([][]block, numFuncs)}
	for f := range p.funcs {
		c.seek(metaSymbolHeaderSize + 4*f)
		c.seek(int(c.u32()))
		units := make([]block, c.count())
		c.uleb128() // function name
		file := str(c.uleb128())
		for u := range units {
			units[u] = block{
				file:  file,
				sl:    uint32(c.uleb128()),
				sc:    uint32(c.uleb128()),
				el:    uint32(c.uleb128()),
				ec:    uint32(c.uleb128()),
				stmts: uint32(c.uleb128()),
			}
		}
		p.funcs[f] = units
	}
	if c.err != nil {
		return nil, c.err
	}
	cache[hash] = p
	return p, nil
}

// readCounters calls visit for every function with counters in a covcounters file: the hash
// of its meta-data file, the package index in it, the function index and the counters, one per
// block. The counters slice is reused between calls.
func readCounters(data []byte, visit func(metaHash [16]byte, pkg, fn uint32, counters []uint32)) error {
	c := &cursor{b: data}
	if !bytes.Equal(c.take(4), counterMagic) {
		return errors.New("not a coverage counter file")
	}
	if v := c.u32(); v > counterFileVersion {
		return fmt.Errorf("counter file version %d, only %d is known", v, counterFileVersion)
	}
	var hash [16]byte
	copy(hash[:], c.take(16))
	flavor, bigEndian := c.u8(), c.u8() != 0
	c.take(6)
	if c.err != nil || len(data) < counterFileHeaderSize+counterFooterSize {
		return errTruncated
	}

	footer := &cursor{b: data[len(data)-counterFooterSize:]}
	if !bytes.Equal(footer.take(4), counterMagic) {
		return errors.New("bad counter file footer")
	}
	footer.take(4)
	segments := footer.u32()

	var value func() uint32
	switch {
	case flavor == flavorULEB128:
		value = func() uint32 { return uint32(c.uleb128()) }
	case flavor == flavorRaw && bigEndian:
		value = func() uint32 {
			if b := c.take(4); b != nil {
				return binary.BigEndian.Uint32(b)
			}
			return 0
		}
	case flavor == flavorRaw:
		value = c.u32
	default:
		return fmt.Errorf("unknown counter flavor %d", flavor)
	}

	counters := make([]uint32, 0, 1024)
	for range segments {
		if c.err != nil {
			break
		}
		funcs := c.u64()
		strTabLen, argsLen := c.u32(), c.u32()
		c.take(int(strTabLen) + int(argsLen)) // the args (os.Args, GOOS, GOARCH) are not needed
		if r := c.off % 4; r != 0 {
			c.take(4 - r)
		}
		for i := uint64(0); i < funcs && c.err == nil; i++ {
			n, pkg, fn := value(), value(), value()
			counters = counters[:0]
			for j := uint32(0); j < n && c.err == nil; j++ {
				counters = append(counters, value())
			}
			if c.err == nil {
				visit(hash, pkg, fn, counters)
			}
		}
		c.take(counterFooterSize)
	}
	return c.err
}
