#!/usr/bin/env python3
"""Reproduce the customer-reported state from DECO-28299.

The state: a live, healthy `databricks ssh connect` session whose shells nonetheless get
EPERM on /Workspace and /Volumes, while /dbfs and the REST API keep working.

Mechanism under test: the bootstrap notebook is both the PR_SET_CHILD_SUBREAPER root and
the PID registered with WSFS/UC. A tmux server started in session A survives session A's
teardown and reparents away from that PID. Session B is a brand-new healthy run, but a
shell inside the surviving tmux server has no registered ancestor, so both mounts deny it.
Reconnecting never helps, because `tmux new` reuses the poisoned server.

It also tells that fault apart from the other one in the same Slack thread (the wsfs
metadata-flush-queue latch, ENOENT on close(), not ours), so it doubles as a triage tool.

THIS DELIBERATELY ORPHANS A PROCESS ON THE DRIVER. Use a disposable test cluster. The tmux
socket is private to the run, so a real tmux server is never touched, and the orphan is
killed on exit unless --keep.

    repro_orphaned_tmux_wsfs.py --cluster <id> [--cli PATH] [--outdir DIR] [--keep]
    repro_orphaned_tmux_wsfs.py --self-test    # check the verdict logic, no cluster needed

Exit codes: 0 reproduced, 1 not reproduced, 2 setup failure, 3 a different fault (wsfs).
"""

import argparse
import base64
import json
import os
import re
import subprocess
import sys
import time
from pathlib import Path

MOUNTS = ("Workspace", "WorkspaceUser", "Volumes", "dbfs", "local_disk0")
REPRODUCED, NOT_REPRODUCED, SETUP_FAILED, OTHER_FAULT = 0, 1, 2, 3

# The CLI writes progress to stderr and the remote command's output to stdout; sentinels
# keep extraction working even if that stops being true.
BEGIN, END = "<<<REPRO_JSON", "REPRO_JSON>>>"

# Runs on the driver, stdlib only. Re-invoked by its own path for `inner`, so the probe
# running inside tmux is *the same file* — no second serialization step that could bake
# results in at write time.
REMOTE_SRC = r'''
import argparse, errno, json, os, shlex, shutil, stat, subprocess, sys, time
from pathlib import Path

SESSION = "repro"
BEGIN, END = "<<<REPRO_JSON", "REPRO_JSON>>>"
SELF = os.path.abspath(sys.argv[0])
ERRNOS = {errno.EPERM: "EPERM", errno.ENOENT: "ENOENT", errno.EACCES: "EACCES"}


def classify(path):
    """Errno of a real open+close. stat() alone can succeed off a stale dentry cache while
    close() fails, which is precisely the wsfs flush-queue signature."""
    def name(exc):
        return ERRNOS.get(exc.errno, f"ERRNO{exc.errno}")

    try:
        st = os.stat(path)
    except OSError as exc:
        return name(exc)
    flags = os.O_RDONLY | (os.O_DIRECTORY if stat.S_ISDIR(st.st_mode) else 0)
    try:
        fd = os.open(path, flags)
    except OSError as exc:
        return f"{name(exc)}(open)"
    try:
        os.close(fd)
    except OSError as exc:
        return f"{name(exc)}(close)"
    return "OK"


def mount_targets():
    targets = {"Workspace": "/Workspace", "Volumes": "/Volumes", "dbfs": "/dbfs", "local_disk0": "/local_disk0"}
    try:
        users = sorted(Path("/Workspace/Users").iterdir())
    except OSError:
        users = []
    if users:
        targets["WorkspaceUser"] = str(users[0])
    return targets


def ancestry(pid, limit=25):
    """Walk /proc upward. Not `ps --ppid`, which counts the ps process itself and produced
    a phantom survivor during PR #6387."""
    chain = []
    for _ in range(limit):
        try:
            raw = Path(f"/proc/{pid}/stat").read_text()
        except OSError:
            chain.append({"pid": pid, "gone": True})
            break
        # comm can contain spaces and parens, so split after the LAST ')'.
        ppid = int(raw[raw.rindex(")") + 2 :].split()[1])
        try:
            cmd = Path(f"/proc/{pid}/cmdline").read_bytes().replace(b"\0", b" ").decode(errors="replace").strip()
        except OSError:
            cmd = ""
        chain.append({"pid": pid, "ppid": ppid, "cmd": cmd[:110] or "[kernel]"})
        if pid == 1:
            break
        pid = ppid
    return chain


def pgrep(pattern):
    return [int(x) for x in subprocess.run(["pgrep", "-f", pattern], capture_output=True, text=True).stdout.split()]


def latch_events(path="/databricks/data/logs/wsfs.log"):
    """Count wsfs flush-queue latch events. Non-zero means the ES-2160568 fault is on this
    driver, which is NOT what this script reproduces."""
    needle = b"Marking queue as in error state"
    try:
        total, carry = 0, b""
        with open(path, "rb") as fh:
            while chunk := fh.read(1 << 20):
                total += (carry + chunk).count(needle)
                carry = chunk[-(len(needle) - 1) :]
        return total
    except OSError:
        return None


def probe():
    return {
        "pid": os.getpid(),
        "notebook_pids": pgrep("db_ipykernel_launcher"),
        "server_pids": pgrep("databricks_cli"),
        "mounts": {name: classify(path) for name, path in mount_targets().items()},
        "ancestry": ancestry(os.getpid()),
        "latch_events": latch_events(),
    }


def tmux(sock, *args):
    return subprocess.run(["tmux", "-L", sock, *args], capture_output=True, text=True)


def run_inner(sock, workdir, tag, timeout=30.0):
    """Run the probe INSIDE the tmux server and read its JSON back. send-keys types into
    the shell tmux owns, so the probe inherits the tmux server's ancestry. Output lands on
    local disk, never on the mount under test."""
    out, done = Path(workdir, f"inner_{tag}.json"), Path(workdir, f"inner_{tag}.done")
    for path in (out, done):
        path.unlink(missing_ok=True)
    tmux(sock, "send-keys", "-t", SESSION,
         f"python3 {shlex.quote(SELF)} inner --out {shlex.quote(str(out))}; touch {shlex.quote(str(done))}", "Enter")
    deadline = time.time() + timeout
    while not done.exists():
        if time.time() > deadline:
            return {"fatal": f"inner probe did not finish within {timeout}s"}
        time.sleep(0.5)
    try:
        return json.loads(out.read_text())
    except (OSError, ValueError) as exc:
        return {"fatal": f"could not read inner probe output: {exc}"}


def mode_inner(args):
    Path(args.out).write_text(json.dumps(probe()))
    return None  # written to a file; nothing on stdout


def mode_baseline(args):
    if not shutil.which("tmux"):
        return {"fatal": "tmux is not installed on the driver"}
    started = tmux(args.sock, "new-session", "-d", "-s", SESSION, "exec bash --norc -i")
    if started.returncode != 0:
        return {"fatal": f"tmux new-session failed: {started.stderr.strip()}"}
    time.sleep(2)
    pid = tmux(args.sock, "display-message", "-p", "#{pid}").stdout.strip()
    if not pid.isdigit():
        return {"fatal": "could not read the tmux server pid"}
    return {
        "tmux_version": tmux(args.sock, "-V").stdout.strip(),
        "tmux_pid": int(pid),
        "tmux_ancestry": ancestry(int(pid)),
        "inner": run_inner(args.sock, args.dir, "a"),
    }


def mode_survivor(args):
    result = {"control": probe()}
    if tmux(args.sock, "list-sessions").returncode != 0:
        result["fatal"] = f"tmux -L {args.sock} is gone; nothing survived teardown"
        return result
    pid = tmux(args.sock, "display-message", "-p", "#{pid}").stdout.strip()
    result["tmux_pid"] = int(pid) if pid.isdigit() else None
    result["tmux_ancestry"] = ancestry(int(pid)) if pid.isdigit() else []
    result["inner"] = run_inner(args.sock, args.dir, "b")
    # Same mount namespace, different caller ancestry: success here while the survivor
    # fails means the denial is about who is asking, not about the mount.
    notebooks = pgrep("db_ipykernel_launcher")
    if notebooks and shutil.which("nsenter"):
        rc = subprocess.run(["nsenter", "-t", str(notebooks[0]), "-m", "--", "stat", "/Workspace"],
                            capture_output=True).returncode
        result["nsenter_as_notebook"] = {"pid": notebooks[0], "workspace_ok": rc == 0}
    return result


def mode_cleanup(args):
    killed = tmux(args.sock, "kill-server").returncode == 0
    shutil.rmtree(args.dir, ignore_errors=True)
    return {"tmux_killed": killed}


parser = argparse.ArgumentParser()
sub = parser.add_subparsers(dest="mode", required=True)
for name in ("baseline", "survivor", "cleanup"):
    p = sub.add_parser(name)
    p.add_argument("--sock", required=True)
    p.add_argument("--dir", required=True)
sub.add_parser("inner").add_argument("--out", required=True)

args = parser.parse_args()
payload = {"baseline": mode_baseline, "survivor": mode_survivor, "cleanup": mode_cleanup, "inner": mode_inner}[
    args.mode
](args)
if payload is not None:
    print(BEGIN)
    print(json.dumps(payload, indent=2))
    print(END)
'''


def say(msg):
    print(f"\n\033[1m== {msg}\033[0m" if not os.environ.get("NO_COLOR") else f"\n== {msg}")


def info(msg):
    print(f"   {msg}")


def verdict(control, survivor):
    """Decide what the two probes mean. Pure, so --self-test can exercise it."""
    eperm = [m for m in ("Workspace", "Volumes") if survivor.get(m, "").startswith("EPERM")]
    enoent = [m for m, v in survivor.items() if v.startswith("ENOENT")]
    control_ok = all(control.get(m, "").startswith("OK") for m in ("Workspace", "Volumes"))
    dbfs_ok = survivor.get("dbfs", "").startswith("OK")

    if enoent and not eperm:
        code, body = (
            OTHER_FAULT,
            [
                "RESULT: NOT REPRODUCED - errno is ENOENT, not EPERM.",
                "That is the wsfs metadata-flush-queue latch (ES-2160568 family), a different",
                "fault from DECO-28299. Check latch_events and read wsfs.log.",
            ],
        )
    elif len(eperm) == 2 and control_ok and dbfs_ok:
        code, body = (
            REPRODUCED,
            [
                "RESULT: REPRODUCED.",
                "A shell inside the surviving tmux server gets EPERM on both /Workspace and",
                "/Volumes, while /dbfs works and a fresh shell in the same live session is",
                "healthy. That is caller-ancestry denial, i.e. the DECO-28299 state.",
            ],
        )
    elif eperm:
        code, body = (
            NOT_REPRODUCED,
            [
                f"RESULT: PARTIAL - EPERM on {', '.join(eperm)} only.",
                "Ancestry denial is happening but not on both mounts; read the ancestry dumps",
                "before drawing conclusions.",
            ],
        )
    else:
        code, body = (
            NOT_REPRODUCED,
            [
                "RESULT: NOT REPRODUCED - the surviving tmux server still reads both mounts.",
                "Either the subreaper adopted it, or teardown did not kill the notebook.",
            ],
        )

    table = ["mount          control-shell        surviving-tmux", "-" * 56]
    table += [f"{m:<14} {control.get(m, '-'):<20} {survivor.get(m, '-')}" for m in MOUNTS]
    return code, [*table, "", *body]


def remote(opts, label, mode, delay=None):
    """Run one mode of the remote program over its own fresh tunnel session. Each call
    submits its own bootstrap run, which is the point: baseline and survivor must run
    under different notebooks."""
    # base64 is [A-Za-z0-9+/=] only, so this needs no shell quoting.
    src = base64.b64encode(REMOTE_SRC.encode()).decode()
    cmd = (
        f"mkdir -p {opts.rdir} && printf %s {src} | base64 -d > {opts.rdir}/probe.py && "
        f"python3 {opts.rdir}/probe.py {mode} --sock {opts.sock} --dir {opts.rdir}"
    )
    argv = [
        opts.cli,
        "ssh",
        "connect",
        f"--cluster={opts.cluster}",
        f"--shutdown-delay={delay or opts.shutdown_delay}",
        "--",
        cmd,
    ]
    proc = subprocess.run(argv, capture_output=True, text=True)
    (opts.outdir / f"{label}.stdout").write_text(proc.stdout)
    (opts.outdir / f"{label}.stderr").write_text(proc.stderr)

    if BEGIN in proc.stdout and END in proc.stdout:
        body = proc.stdout.split(BEGIN, 1)[1].split(END, 1)[0]
        try:
            result = json.loads(body)
        except ValueError as exc:
            result = {"fatal": f"malformed JSON from the driver: {exc}"}
    else:
        result = {"fatal": f"no JSON payload in the remote output; see {label}.stderr"}

    runs = re.findall(r"run ID: (\d+)", proc.stderr)
    result.setdefault("run_id", runs[-1] if runs else None)
    return result


def run_state(opts, run_id):
    proc = subprocess.run([opts.cli, "jobs", "get-run", run_id, "--output", "json"], capture_output=True, text=True)
    try:
        state = json.loads(proc.stdout).get("state") or {}
    except ValueError:
        return "UNKNOWN"
    return state.get("life_cycle_state") or state.get("state") or "UNKNOWN"


def wait_terminal(opts, run_id, timeout=600.0):
    deadline, last = time.time() + timeout, "UNKNOWN"
    while time.time() < deadline:
        last = run_state(opts, run_id)
        if last in {"TERMINATED", "INTERNAL_ERROR", "SKIPPED"}:
            return last
        info(f"run {run_id} is {last} ...")
        time.sleep(10)
    return last


def show(label, snapshot):
    info(f"{label}:")
    for name in MOUNTS:
        if (value := snapshot.get("mounts", {}).get(name)) is not None:
            info(f"    {name:<14} {value}")
    if snapshot.get("latch_events"):
        info(f"    ! wsfs latch events in wsfs.log: {snapshot['latch_events']} (that is the OTHER fault)")
    for hop in snapshot.get("ancestry", [])[:8]:
        info(
            f"    {hop['pid']:<8} <gone>"
            if hop.get("gone")
            else f"    {hop['pid']:<8} ppid={hop['ppid']:<8} {hop['cmd'][:70]}"
        )


def self_test():
    healthy = dict.fromkeys(MOUNTS, "OK")
    both = {"Workspace": "EPERM", "Volumes": "EPERM", "dbfs": "OK"}
    cases = [
        ("reproduced", healthy, both, REPRODUCED),
        ("wsfs latch", healthy, {"Workspace": "ENOENT(close)", "Volumes": "OK", "dbfs": "OK"}, OTHER_FAULT),
        ("workspace only", healthy, {"Workspace": "EPERM", "Volumes": "OK", "dbfs": "OK"}, NOT_REPRODUCED),
        ("nothing broken", healthy, healthy, NOT_REPRODUCED),
        ("dbfs broken too", healthy, {**both, "dbfs": "EPERM"}, NOT_REPRODUCED),
        # A control shell that is already broken means the run proves nothing.
        ("broken baseline", {"Workspace": "EPERM", "Volumes": "OK"}, both, NOT_REPRODUCED),
    ]
    failures = [name for name, ctl, sur, want in cases if verdict(ctl, sur)[0] != want]
    for name, ctl, sur, want in cases:
        got = verdict(ctl, sur)[0]
        print(f"  {'ok  ' if got == want else 'FAIL'} {name:<16} expected {want}, got {got}")
    print("\nself-test passed" if not failures else f"\nself-test FAILED: {', '.join(failures)}")
    return NOT_REPRODUCED if failures else REPRODUCED


def main():
    parser = argparse.ArgumentParser(description=__doc__, formatter_class=argparse.RawDescriptionHelpFormatter)
    parser.add_argument("--cluster", help="dedicated cluster ID (use a disposable test cluster)")
    parser.add_argument("--cli", default=os.environ.get("CLI", "databricks"))
    parser.add_argument("--outdir", type=Path, help="where to write evidence (default: a temp dir)")
    parser.add_argument("--shutdown-delay", default="20s", help="tunnel idle shutdown; short keeps teardown quick")
    parser.add_argument("--keep", action="store_true", help="leave the orphaned tmux server on the driver")
    parser.add_argument("--self-test", action="store_true", help="check the verdict logic and exit")
    opts = parser.parse_args()

    if opts.self_test:
        return self_test()
    if not opts.cluster:
        parser.error("--cluster is required (or use --self-test)")

    stamp = f"{time.strftime('%Y%m%dT%H%M%SZ', time.gmtime())}-{os.getpid()}"
    opts.sock = f"repro-{stamp}"
    opts.rdir = f"/tmp/repro-wsfs-{stamp}"
    opts.outdir = opts.outdir or Path(f"/tmp/repro-wsfs-{os.getpid()}")
    opts.outdir.mkdir(parents=True, exist_ok=True)

    say("Preflight")
    info(f"cluster:        {opts.cluster}")
    info(f"tmux socket:    {opts.sock}  (private; cannot touch a real tmux server)")
    info(f"remote scratch: {opts.rdir}")
    info(f"evidence:       {opts.outdir}")
    info("! this orphans a process on the driver; use a disposable test cluster")

    try:
        say("Phase A - start a tmux server in a live session and baseline the mounts")
        baseline = remote(opts, "phase_a", "baseline")
        inner_a = baseline.get("inner", {})
        for stage in (baseline, inner_a):
            if "fatal" in stage:
                info(f"x {stage['fatal']}")
                return SETUP_FAILED
        info(f"tmux {baseline.get('tmux_version', '?')}, server pid {baseline.get('tmux_pid')}")
        show("inside tmux, notebook alive", inner_a)
        if not inner_a.get("mounts", {}).get("Workspace", "").startswith("OK"):
            info("x baseline already broken: /Workspace was not OK under a live notebook.")
            info("x The cluster is in a bad state or the wsfs latch is active. Not a valid run.")
            return SETUP_FAILED
        info("+ baseline healthy")

        say("Phase A teardown - waiting for the bootstrap run to terminate")
        if run_a := baseline.get("run_id"):
            info(f"run {run_a} ended as {wait_terminal(opts, run_a)}")
        else:
            info("! no run id recovered; sleeping past --shutdown-delay instead")
            time.sleep(45)

        say("Phase B - reconnect (new run) and probe the surviving tmux server")
        survivor = remote(opts, "phase_b", "survivor")
        inner_b = survivor.get("inner", {})
        show("control shell, new healthy session", survivor.get("control", {}))
        for stage in (survivor, inner_b):
            if "fatal" in stage:
                info(f"x {stage['fatal']}")
                return SETUP_FAILED
        show("inside the SURVIVING tmux server", inner_b)
        if diff := survivor.get("nsenter_as_notebook"):
            state = "OK" if diff["workspace_ok"] else "FAILED (unexpected)"
            info(f"/Workspace as the live notebook (pid {diff['pid']}), same mount ns: {state}")

        say("Verdict")
        code, lines = verdict(survivor.get("control", {}).get("mounts", {}), inner_b.get("mounts", {}))
        for line in lines:
            info(line)
        return code
    finally:
        if opts.keep:
            info(f"! --keep: tmux socket {opts.sock} and {opts.rdir} left on the driver")
        else:
            say("Cleanup - killing the orphaned tmux server")
            result = remote(opts, "cleanup", "cleanup", delay="10s")
            info(
                f"! cleanup failed ({result['fatal']}); check for a stray tmux -L {opts.sock}"
                if "fatal" in result
                else f"tmux killed: {result.get('tmux_killed')}, scratch removed"
            )
        info(f"Evidence: {opts.outdir}")


if __name__ == "__main__":
    sys.exit(main())
