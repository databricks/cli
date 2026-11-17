#!/usr/bin/env python3
# /// script
# requires-python = ">=3.12"
# ///
"""
Pack the coverage of a test run into one small binary file, for the CI dashboard.

Reads the unit coverage directory (CLI_GOCOVERDIR_UNIT) and the acceptance coverage
directory (CLI_GOCOVERDIR, one subdirectory per test with the test name in `name`).

Go measures blocks (spans of statements that run together; they never overlap), so a
file's coverage is one bit per block. Most tests cover a file the same way as many other
tests (the CLI's startup, mostly), so each file stores its distinct bitmaps once and a test
names, per file it covered, which of them it has: 3.9k tests have 3M (test, file) pairs but
only 13k distinct bitmaps.

The file is zstd-compressed (`zstd -d --long=27`). Inside, a uvarint is unsigned LEB128:

    "CLICOV1\\n"
    uvarint n, n bytes      meta, JSON: the --meta object plus "go" and "module"
    uvarint L               layouts, sorted by path:
      uvarint n, n bytes      path relative to the module ("bundle/deploy/x.go"; -coverpkg=./... covers only the module)
      20 bytes                git blob sha of the file at the commit (zeros if not tracked)
      uvarint                 line count
      uvarint B               blocks, sorted by start then end; per block:
        uvarint sl - previous sl, uvarint sc, uvarint el - sl, uvarint ec, uvarint statements
                              (as in a Go cover profile: lines and byte columns 1-based, the end exclusive)
    L times                 the distinct bitmaps of each layout, in layout order:
      uvarint P, then P bitmaps of ceil(B / 8) bytes: block i is bit i % 8 of byte i // 8
    uvarint N               profiles:
      1 byte kind             0: all acceptance tests merged, 1: unit tests merged, 2: one acceptance test
      uvarint n, n bytes      test name (t.Name()), empty for kinds 0 and 1
      uvarint F               files, by layout; per file: uvarint layout - previous layout, uvarint bitmap index

Kinds 0 and 1 list every file of their build, covered or not: they are the inventory. A
file that a test (kind 2) does not list was not covered by it if the build has it, else it
is unknown. `coverpack.py --dump FILE` prints the profiles back as Go cover profile text.
"""

import argparse
import json
import os
import subprocess
import sys
import tempfile
from concurrent.futures import ProcessPoolExecutor
from pathlib import Path

MAGIC = b"CLICOV1\n"
KIND_ALL, KIND_UNIT, KIND_TEST = 0, 1, 2

# A single argument of exec is at most 128 KiB on Linux; -i takes the directories as one comma list.
DIRS_PER_MERGE = 500


def covdata():
    """The path of the covdata tool; `go tool covdata` would resolve it again for every test."""
    out = subprocess.run(["go", "tool", "-n", "covdata"], check=True, capture_output=True, text=True)
    return out.stdout.strip()


def textfmt(tool, dirs, out):
    subprocess.run([tool, "textfmt", "-i", ",".join(str(d) for d in dirs), "-o", str(out)], check=True)


def parse_block(text):
    """("file", (sl, sc, el, ec), statements) of `file:sl.sc,el.ec statements`."""
    loc, _, stmts = text.rpartition(" ")
    path, _, span = loc.rpartition(":")
    start, _, end = span.partition(",")
    sl, _, sc = start.partition(".")
    el, _, ec = end.partition(".")
    return path, (int(sl), int(sc), int(el), int(ec)), int(stmts)


def read_profile(path, blocks):
    """Add the blocks of a profile to blocks (`file:span statements` -> covered)."""
    with open(path) as f:
        next(f, None)  # mode: set
        for line in f:
            block, _, count = line.rstrip("\n").rpartition(" ")
            blocks[block] = blocks.get(block, False) or count != "0"
    return blocks


# The block index of the worker processes: `file:span statements` -> (layout, block).
_index = None


def _init_worker(index):
    global _index
    _index = index


def test_bitmaps(tool, d, tmp):
    """{layout: bitmap as int} of the blocks the coverage in d covered."""
    out = Path(tmp) / (d.name + ".txt")
    textfmt(tool, [d], out)
    bitmaps = {}
    with open(out) as f:
        next(f, None)
        for line in f:
            if line.endswith(" 0\n"):
                continue
            layout, block = _index[line.rpartition(" ")[0]]
            bitmaps[layout] = bitmaps.get(layout, 0) | 1 << block
    out.unlink()
    return bitmaps


def has_data(d):
    return any(e.name.startswith("covcounters.") for e in os.scandir(d))


def acceptance_tests(acc_dir):
    """(test name, directory) of every test that ran the CLI, sorted by name."""
    tests = []
    for d in Path(acc_dir).iterdir():
        name = d / "name"
        if d.is_dir() and name.exists() and has_data(d):
            tests.append((name.read_text(), d))
    return sorted(tests)


def git_files(root):
    """{repo-relative path: git blob sha} of the tracked files."""
    out = subprocess.run(["git", "ls-files", "-s"], cwd=root, check=True, capture_output=True, text=True).stdout
    shas = {}
    for line in out.splitlines():
        info, _, path = line.partition("\t")
        shas[path] = bytes.fromhex(info.split()[1])
    return shas


def line_count(path):
    data = path.read_bytes()
    return data.count(b"\n") + (1 if data and not data.endswith(b"\n") else 0)


def uvarint(n, out):
    while n >= 0x80:
        out.append(n & 0x7F | 0x80)
        n >>= 7
    out.append(n)


def put_bytes(data, out):
    uvarint(len(data), out)
    out += data


def pack(meta, root, unit, everything, tests, per_test):
    """The uncompressed file. unit and everything: {`file:span statements`: covered}; per_test: [{layout: bitmap}]."""
    blocks_of = {}  # import path -> {(span, statements)}
    for profile in (unit or {}), everything:
        for block in profile:
            path, span, stmts = parse_block(block)
            blocks_of.setdefault(path, set()).add((span, stmts))
    prefix = meta["module"] + "/"
    outside = [p for p in blocks_of if not p.startswith(prefix)]
    if outside:
        sys.exit(f"coverage of files outside {meta['module']}: {outside[:3]}")
    shas = git_files(root)
    paths = sorted(blocks_of, key=lambda p: p.removeprefix(prefix))
    layouts = [sorted(blocks_of[p]) for p in paths]
    index = {}
    for layout, (path, blocks) in enumerate(zip(paths, layouts, strict=True)):
        for block, ((sl, sc, el, ec), stmts) in enumerate(blocks):
            index[f"{path}:{sl}.{sc},{el}.{ec} {stmts}"] = (layout, block)

    def merged_bitmaps(profile):
        bitmaps = {}
        for block, covered in profile.items():
            layout, i = index[block]
            bitmaps[layout] = bitmaps.get(layout, 0) | (covered << i)
        return bitmaps

    profiles = []  # (kind, name, {layout: bitmap})
    if everything:
        profiles.append((KIND_ALL, "", merged_bitmaps(everything)))
    if unit is not None:
        profiles.append((KIND_UNIT, "", merged_bitmaps(unit)))
    for (name, _), bitmaps in zip(tests, per_test(index), strict=True):
        profiles.append((KIND_TEST, name, bitmaps))

    distinct = [{} for _ in layouts]  # layout -> {bitmap: index}
    for _, _, bitmaps in profiles:
        for layout, bitmap in bitmaps.items():
            distinct[layout].setdefault(bitmap, len(distinct[layout]))

    out = bytearray(MAGIC)
    put_bytes(json.dumps(meta, sort_keys=True).encode(), out)
    uvarint(len(layouts), out)
    for path, blocks in zip(paths, layouts, strict=True):
        rel = path.removeprefix(prefix)
        put_bytes(rel.encode(), out)
        out += shas.get(rel, bytes(20))
        uvarint(line_count(root / rel) if rel in shas else 0, out)
        uvarint(len(blocks), out)
        prev = 0
        for (sl, sc, el, ec), stmts in blocks:
            for n in sl - prev, sc, el - sl, ec, stmts:
                uvarint(n, out)
            prev = sl
    for blocks, bitmaps in zip(layouts, distinct, strict=True):
        uvarint(len(bitmaps), out)
        for bitmap in bitmaps:  # dicts keep the order of the indexes
            out += bitmap.to_bytes((len(blocks) + 7) // 8, "little")
    uvarint(len(profiles), out)
    for kind, name, bitmaps in profiles:
        out.append(kind)
        put_bytes(name.encode(), out)
        uvarint(len(bitmaps), out)
        prev = 0
        for layout in sorted(bitmaps):
            uvarint(layout - prev, out)
            uvarint(distinct[layout][bitmaps[layout]], out)
            prev = layout
    return bytes(out), sum(map(len, distinct))


class Reader:
    def __init__(self, data):
        self.data, self.pos = data, 0

    def uvarint(self):
        n = shift = 0
        while True:
            b = self.data[self.pos]
            self.pos += 1
            n |= (b & 0x7F) << shift
            if b < 0x80:
                return n
            shift += 7

    def take(self, n):
        self.pos += n
        return self.data[self.pos - n : self.pos]

    def bytes(self):
        return self.take(self.uvarint())


def unpack(data):
    """(meta, layouts, profiles): layouts [(path, sha hex, lines, [((sl, sc, el, ec), statements)], [bitmap as int])],
    profiles [(kind, name, {layout: bitmap index})]."""
    r = Reader(data)
    if r.take(len(MAGIC)) != MAGIC:
        raise ValueError("not a coverpack file")
    meta = json.loads(r.bytes())
    layouts = []
    for _ in range(r.uvarint()):
        path, sha, lines = r.bytes().decode(), r.take(20).hex(), r.uvarint()
        blocks, sl = [], 0
        for _ in range(r.uvarint()):
            sl += r.uvarint()
            sc, el, ec, stmts = r.uvarint(), sl + r.uvarint(), r.uvarint(), r.uvarint()
            blocks.append(((sl, sc, el, ec), stmts))
        layouts.append((path, sha, lines, blocks, []))
    for _, _, _, blocks, bitmaps in layouts:
        for _ in range(r.uvarint()):
            bitmaps.append(int.from_bytes(r.take((len(blocks) + 7) // 8), "little"))
    profiles = []
    for _ in range(r.uvarint()):
        kind, name, files, layout = r.take(1)[0], r.bytes().decode(), {}, 0
        for _ in range(r.uvarint()):
            layout += r.uvarint()
            files[layout] = r.uvarint()
        profiles.append((kind, name, files))
    return meta, layouts, profiles


def dump(path):
    data = subprocess.run(["zstd", "-d", "--long=27", "-c", path], check=True, capture_output=True).stdout
    meta, layouts, profiles = unpack(data)
    prefix = meta["module"] + "/"
    out = sys.stdout
    out.write(f"# coverage v1 {json.dumps(meta, sort_keys=True)}\n")
    for kind, name, files in profiles:
        out.write(["# all\n", "# unit\n", f"# test {name}\n"][kind])
        for layout, bitmap_index in files.items():
            path, _, _, blocks, bitmaps = layouts[layout]
            bitmap = bitmaps[bitmap_index]
            for i, ((sl, sc, el, ec), stmts) in enumerate(blocks):
                covered = bitmap >> i & 1
                if covered or kind != KIND_TEST:
                    out.write(f"{prefix}{path}:{sl}.{sc},{el}.{ec} {stmts} {covered}\n")


def main():
    parser = argparse.ArgumentParser(description=__doc__, formatter_class=argparse.RawDescriptionHelpFormatter)
    parser.add_argument("--unit", help="unit coverage directory (CLI_GOCOVERDIR_UNIT)")
    parser.add_argument("--acc", help="acceptance coverage directory (CLI_GOCOVERDIR)")
    parser.add_argument("--meta", default="{}", help="JSON object for the meta (sha, os, cell, ...)")
    parser.add_argument("-o", "--output", help="the .bin.zst to write")
    parser.add_argument("--dump", metavar="FILE", help="print FILE as Go cover profile text instead")
    args = parser.parse_args()
    if args.dump:
        dump(args.dump)
        return
    if not args.output:
        parser.error("-o is required")

    root = Path(
        subprocess.run(
            ["git", "rev-parse", "--show-toplevel"], check=True, capture_output=True, text=True
        ).stdout.strip()
    )
    tool = covdata()
    meta = json.loads(args.meta)
    meta["go"] = subprocess.run(["go", "env", "GOVERSION"], check=True, capture_output=True, text=True).stdout.strip()
    meta["module"] = subprocess.run(
        ["go", "list", "-m"], cwd=root, check=True, capture_output=True, text=True
    ).stdout.strip()
    tests = acceptance_tests(args.acc) if args.acc and Path(args.acc).is_dir() else []

    with tempfile.TemporaryDirectory() as tmp:
        tmp = Path(tmp)
        unit = None
        if args.unit and Path(args.unit).is_dir() and has_data(args.unit):
            textfmt(tool, [args.unit], tmp / "unit.txt")
            unit = read_profile(tmp / "unit.txt", {})
        everything = {}
        dirs = [d for _, d in tests]
        for i in range(0, len(dirs), DIRS_PER_MERGE):
            textfmt(tool, dirs[i : i + DIRS_PER_MERGE], tmp / "all.txt")
            read_profile(tmp / "all.txt", everything)

        def per_test(index):
            # Processes, not threads: parsing the profile of each test is most of the time.
            with ProcessPoolExecutor(os.cpu_count(), initializer=_init_worker, initargs=(index,)) as pool:
                return list(pool.map(test_bitmaps, [tool] * len(dirs), dirs, [tmp] * len(dirs), chunksize=4))

        data, nbitmaps = pack(meta, root, unit, everything, tests, per_test)

    zstd = subprocess.run(["zstd", "-q", "-f", "-19", "--long=27", "-T0", "-o", args.output], input=data)
    if zstd.returncode != 0:
        sys.exit("zstd failed")
    print(
        f"{args.output}: {len(tests)} acceptance tests, {len(everything)} acceptance blocks, "
        f"{len(unit or ())} unit blocks, {nbitmaps} distinct file bitmaps, "
        f"{len(data)} bytes, {os.path.getsize(args.output)} compressed",
        file=sys.stderr,
    )


if __name__ == "__main__":
    main()
