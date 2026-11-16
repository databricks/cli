#!/usr/bin/env python3
# /// script
# requires-python = ">=3.12"
# ///
"""
Pack the coverage of a test run into one zstd-compressed text file, for the CI dashboard.

Reads the unit coverage directory (CLI_GOCOVERDIR_UNIT) and the acceptance coverage
directory (CLI_GOCOVERDIR, one subdirectory per test with the test name in `name`) and
writes sections, each starting with a `# ` line, the rest being lines of a Go cover
profile (`file:sl.sc,el.ec nstmt count`):

    # coverage v1 {"sha": ..., ...}     the arguments of --meta, as JSON
    # all                               the acceptance tests merged: every block of the CLI, zero counts included
    # unit                              the unit tests: every block of every package they link, zero counts included
    # test TestAccept/bundle/x/DMS=true one per acceptance test, sorted by name: the blocks it covered (count > 0)

A section with `mode: set` in front of its lines is a profile that `go tool cover` reads.

Most tests cover the same blocks (the CLI's startup), in the same order, so the text is
large (hundreds of MB) and repetitive. It is compressed with zstd --long=27: its 128 MiB
window sees the repeats between tests, which gzip's 32 KiB window does not (50 MB with
gzip -9 against 1 MB with zstd -3 --long=27 for 358 tests). Read it back with
`zstd -d --long=27`.
"""

import argparse
import json
import os
import shutil
import subprocess
import sys
import tempfile
from concurrent.futures import ThreadPoolExecutor
from pathlib import Path

# A single argument of exec is at most 128 KiB on Linux; -i takes the directories as one comma list.
DIRS_PER_MERGE = 500


def covdata():
    """The path of the covdata tool; `go tool covdata` would resolve it again for every test."""
    out = subprocess.run(["go", "tool", "-n", "covdata"], check=True, capture_output=True, text=True)
    return out.stdout.strip()


def textfmt(tool, dirs, out):
    subprocess.run([tool, "textfmt", "-i", ",".join(str(d) for d in dirs), "-o", str(out)], check=True)


def read_profile(path, blocks):
    """Add the blocks of a profile to blocks (block -> count, the larger count wins)."""
    with open(path) as f:
        next(f, None)  # mode: set
        for line in f:
            block, _, count = line.rstrip("\n").rpartition(" ")
            blocks[block] = max(blocks.get(block, 0), int(count))
    return blocks


def covered_lines(tool, d, out):
    """Write the lines of the blocks that the coverage in d covered to out."""
    full = out.with_suffix(".full")
    textfmt(tool, [d], full)
    with open(out, "wb") as f:
        # grep exits 1 when nothing matches: a test that ran the CLI but covered nothing is not an error.
        r = subprocess.run(["grep", "-v", "-e", " 0$", "-e", "^mode: ", str(full)], stdout=f)
        if r.returncode > 1:
            sys.exit(f"grep failed on {full}")
    full.unlink()


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


def write_blocks(out, header, blocks):
    out.write(f"# {header}\n".encode())
    out.write("".join(f"{b} {blocks[b]}\n" for b in sorted(blocks)).encode())


def main():
    parser = argparse.ArgumentParser(description=__doc__, formatter_class=argparse.RawDescriptionHelpFormatter)
    parser.add_argument("--unit", help="unit coverage directory (CLI_GOCOVERDIR_UNIT)")
    parser.add_argument("--acc", help="acceptance coverage directory (CLI_GOCOVERDIR)")
    parser.add_argument("--meta", default="{}", help="JSON object for the first line (sha, os, cell, ...)")
    parser.add_argument("-o", "--output", required=True, help="the .txt.zst to write")
    args = parser.parse_args()

    tool = covdata()
    meta = json.loads(args.meta)
    meta["go"] = subprocess.run(["go", "env", "GOVERSION"], check=True, capture_output=True, text=True).stdout.strip()
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

        outs = [tmp / f"test{i}.txt" for i in range(len(tests))]
        with ThreadPoolExecutor(max_workers=os.cpu_count()) as pool:
            list(pool.map(lambda i: covered_lines(tool, tests[i][1], outs[i]), range(len(tests))))

        zstd = subprocess.Popen(
            ["zstd", "-q", "-f", "-3", "--long=27", "-T0", "-o", args.output], stdin=subprocess.PIPE
        )
        out = zstd.stdin
        out.write(f"# coverage v1 {json.dumps(meta, sort_keys=True)}\n".encode())
        if tests:
            write_blocks(out, "all", everything)
        if unit is not None:
            write_blocks(out, "unit", unit)
        for (name, _), path in zip(tests, outs, strict=True):
            out.write(f"# test {name}\n".encode())
            with open(path, "rb") as f:
                shutil.copyfileobj(f, out)
        out.close()
        if zstd.wait() != 0:
            sys.exit("zstd failed")

    print(
        f"{args.output}: {len(tests)} acceptance tests, {len(everything)} acceptance blocks, "
        f"{len(unit or ())} unit blocks, {os.path.getsize(args.output)} bytes",
        file=sys.stderr,
    )


if __name__ == "__main__":
    main()
