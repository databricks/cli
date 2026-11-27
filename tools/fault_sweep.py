#!/usr/bin/env python3
"""Check that acceptance tests and the CLI survive a single backend error.

1. Runs the selected tests --count-runs times with -faultlog to learn how many requests each
   one makes (the max over the runs). Tests that fail in any of these runs, or make no
   requests, are left out.
2. Each round picks, for every remaining test, a random request number up to that max and a
   random fault kind, and runs the tests with that -faultplan. A test that passes is run again
   with the same fault, so a pass is not luck of timing. The CLI should retry the failed
   request, and the test should pass unchanged: retry warnings are suppressed by
   SuppressLines.warn_retry and the failed attempt is dropped from out.requests.txt.
3. Writes results.jsonl, logs of the failures and summary.md to --out.

Example:
    tools/fault_sweep.py --run 'TestAccept/bundle/resources/jobs' --rounds 2 --out /tmp/sweep
"""

import argparse
import collections
import json
import random
import re
import subprocess
import sys
import time
from pathlib import Path

# Keep in sync with RequestFaultKinds in libs/testserver/request_fault.go.
KINDS = ["429", "500", "502", "503", "504", "504-after", "reset", "reset-after", "eof-after"]

REPO = Path(__file__).resolve().parent.parent


def go_test(args, run, faultlog, faultplan, json_out):
    cmd = [
        "go",
        "test",
        "./acceptance",
        "-count=1",
        "-json",
        "-timeout",
        args.timeout,
        "-run",
        run,
        "-faultlog",
        str(faultlog),
    ]
    if faultplan:
        cmd += ["-faultplan", str(faultplan)]
    cmd += args.go_args
    print("+ " + " ".join(cmd), file=sys.stderr, flush=True)
    start = time.time()
    with open(json_out, "w") as f:
        subprocess.run(cmd, cwd=REPO, stdout=f, stderr=subprocess.STDOUT, check=False)
    print(f"  took {time.time() - start:.0f}s", file=sys.stderr, flush=True)
    return read_jsonl(faultlog)


def read_jsonl(path):
    if not path.exists():
        return []
    return [json.loads(line) for line in path.read_text().splitlines() if line.strip()]


def test_outputs(json_out):
    """Return the go test output of each test, from a -json run."""
    outputs = collections.defaultdict(list)
    for line in Path(json_out).read_text().splitlines():
        try:
            event = json.loads(line)
        except json.JSONDecodeError:
            continue
        if event.get("Action") == "output" and event.get("Test"):
            outputs[event["Test"]].append(event["Output"])
    return {name: "".join(lines) for name, lines in outputs.items()}


def failure_snippet(output):
    """Pick the lines of a failed test's log that say what broke."""
    lines = [line.strip() for line in output.splitlines()]
    diff = [
        line
        for line in lines
        if line[:1] in ("+", "-")
        and line not in ("+", "-")
        and not line.startswith(("+++", "---", "--- FAIL", "--- PASS", "--- SKIP"))
    ]
    if diff:
        return " | ".join(diff[:4])
    errors = [line for line in lines if "Error" in line or "error" in line]
    return " | ".join(errors[:3])


ID_SEGMENT = re.compile(r"^(\d+|[0-9a-f]{16,}|[0-9a-f-]{36})$")


def endpoint(method, path):
    """Generalize a request path so faults in the same API group together."""
    if "/Workspace/" in path:
        path = path[: path.index("/Workspace/")] + "/Workspace/..."
    segments = ["{id}" if ID_SEGMENT.match(s) else s for s in path.split("/")]
    return method + " " + "/".join(segments)


def sanitize(name):
    return re.sub(r"[^A-Za-z0-9_.=-]+", "_", name)


def count_requests(args, out):
    """Return {test: max requests} for tests that passed every counting run, and the exclusions."""
    max_requests = {}
    failed = set()
    for i in range(args.count_runs):
        entries = go_test(args, args.run, out / f"count-{i}.jsonl", None, out / f"count-{i}.json")
        for e in entries:
            if e["skipped"]:
                continue
            if e["failed"]:
                failed.add(e["test"])
            max_requests[e["test"]] = max(max_requests.get(e["test"], 0), e["requests"])
    no_requests = {t for t, n in max_requests.items() if n == 0}
    eligible = {t: n for t, n in max_requests.items() if t not in failed and n > 0}
    return eligible, sorted(failed), sorted(no_requests)


def run_round(args, out, rnd, eligible, rng):
    plan = {test: {"at": rng.randint(1, n), "kind": rng.choice(args.kinds)} for test, n in sorted(eligible.items())}
    results = {}
    passed = set()
    for attempt in (1, 2):
        if attempt == 2:
            plan = {t: f for t, f in plan.items() if t in passed}
            if not plan:
                break
        tag = f"round-{rnd}-run-{attempt}"
        plan_path = out / f"{tag}.plan.json"
        plan_path.write_text(json.dumps(plan, indent=1))
        json_out = out / f"{tag}.json"
        entries = go_test(args, args.run, out / f"{tag}.jsonl", plan_path, json_out)
        outputs = None
        seen = set()
        for e in entries:
            test = e["test"]
            if test not in plan:
                continue
            seen.add(test)
            fired = e.get("fired")
            result = results.setdefault(
                test,
                {"test": test, "round": rnd, "planned": plan[test], "fired": fired, "requests": e["requests"]},
            )
            if attempt == 1 and not fired:
                result["outcome"] = "not-fired"
                continue
            if e["failed"]:
                if outputs is None:
                    outputs = test_outputs(json_out)
                log = outputs.get(test, "")
                log_path = out / "logs" / f"{tag}.{sanitize(test)}.txt"
                log_path.parent.mkdir(exist_ok=True)
                log_path.write_text(log)
                result["outcome"] = "fail" if attempt == 1 else "fail-on-rerun"
                result["fired"] = fired or result["fired"]
                result["snippet"] = failure_snippet(log)
                result["log"] = str(log_path.relative_to(out))
            elif attempt == 1:
                passed.add(test)
                result["outcome"] = "pass-once"
            else:
                result["outcome"] = "pass"
        for test in set(plan) - seen:
            # No log entry: the run was cut short, e.g. by -timeout.
            results.setdefault(test, {"test": test, "round": rnd, "planned": plan[test]})["outcome"] = "no-result"
    return list(results.values())


def write_summary(args, out, eligible, baseline_failed, no_requests, results):
    by_kind = collections.defaultdict(collections.Counter)
    for r in results:
        by_kind[r["planned"]["kind"]][r["outcome"]] += 1

    failures = [r for r in results if r["outcome"].startswith("fail")]
    by_endpoint = collections.defaultdict(list)
    for r in failures:
        f = r["fired"]
        by_endpoint[(endpoint(f["method"], f["path"]), r["planned"]["kind"])].append(r)

    lines = [
        "# Fault sweep summary",
        "",
        f"Selection: `-run {args.run}`, {args.count_runs} counting runs, {args.rounds} round(s), seed {args.seed}.",
        "",
        f"- Tests with requests that passed every counting run: {len(eligible)}",
        f"- Left out, failed a counting run (flaky or broken without faults): {len(baseline_failed)}",
        f"- Left out, made no requests: {len(no_requests)}",
        "",
        "Outcomes: `pass` passed twice with the fault, `fail` failed with it, `fail-on-rerun` passed once and",
        "failed the second time, `not-fired` made fewer requests than the chosen number, `no-result` was cut short.",
        "",
        "| Kind | Injected | pass | fail | fail-on-rerun | not-fired | no-result |",
        "|---|---|---|---|---|---|---|",
    ]
    total = collections.Counter()
    for kind in args.kinds:
        c = by_kind[kind]
        total.update(c)
        injected = sum(c.values()) - c["not-fired"]
        lines.append(
            f"| {kind} | {injected} | {c['pass']} | {c['fail']} | {c['fail-on-rerun']} | {c['not-fired']} | {c['no-result']} |"
        )
    injected = sum(total.values()) - total["not-fired"]
    lines.append(
        f"| **all** | {injected} | {total['pass']} | {total['fail']} | {total['fail-on-rerun']} | {total['not-fired']} | {total['no-result']} |"
    )

    lines += [
        "",
        "## Failures by endpoint and kind",
        "",
        "| Endpoint | Kind | Failures | Example test |",
        "|---|---|---|---|",
    ]
    for (ep, kind), rs in sorted(by_endpoint.items(), key=lambda item: (-len(item[1]), item[0])):
        lines.append(f"| `{ep}` | {kind} | {len(rs)} | {rs[0]['test']} |")

    lines += ["", "## Failures", "", "| Test | Kind | Request | Outcome | Snippet |", "|---|---|---|---|---|"]
    for r in sorted(failures, key=lambda r: r["test"]):
        f = r["fired"]
        snippet = r.get("snippet", "").replace("|", "\\|")[:300]
        lines.append(
            f"| {r['test']} | {f['kind']} | {f['at']} `{f['method']} {f['path']}` | {r['outcome']} | `{snippet}` |"
        )

    if baseline_failed:
        lines += ["", "## Left out: failed a counting run", ""]
        lines += [f"- {t}" for t in baseline_failed]

    path = out / "summary.md"
    path.write_text("\n".join(lines) + "\n")
    return path


def main():
    parser = argparse.ArgumentParser(description=__doc__, formatter_class=argparse.RawDescriptionHelpFormatter)
    parser.add_argument("--run", default="^TestAccept$", help="go test -run selection (default: every acceptance test)")
    parser.add_argument("--count-runs", type=int, default=3, help="runs to learn request counts (default: 3)")
    parser.add_argument("--rounds", type=int, default=1, help="rounds of injection per test (default: 1)")
    parser.add_argument("--kinds", default=",".join(KINDS), help="comma-separated fault kinds (default: all)")
    parser.add_argument("--seed", type=int, default=None, help="random seed (default: random)")
    parser.add_argument("--timeout", default="4h", help="go test -timeout for each run (default: 4h)")
    parser.add_argument("--out", required=True, type=Path, help="directory for logs, results.jsonl and summary.md")
    parser.add_argument("go_args", nargs="*", help="extra go test arguments, after --")
    args = parser.parse_args()

    args.kinds = args.kinds.split(",")
    unknown = set(args.kinds) - set(KINDS)
    if unknown:
        parser.error(f"unknown kinds {sorted(unknown)}, expected some of {KINDS}")
    if args.seed is None:
        args.seed = random.randrange(1 << 32)
    rng = random.Random(args.seed)
    args.out.mkdir(parents=True, exist_ok=True)

    eligible, baseline_failed, no_requests = count_requests(args, args.out)
    print(
        f"{len(eligible)} tests to inject into; left out {len(baseline_failed)} failing, {len(no_requests)} without requests",
        file=sys.stderr,
    )

    results = []
    for rnd in range(1, args.rounds + 1):
        results += run_round(args, args.out, rnd, eligible, rng)

    with open(args.out / "results.jsonl", "w") as f:
        f.writelines(json.dumps(r) + "\n" for r in results)
    print(write_summary(args, args.out, eligible, baseline_failed, no_requests, results))


if __name__ == "__main__":
    main()
