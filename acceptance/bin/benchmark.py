#!/usr/bin/env python3
import argparse
import subprocess
import time
import statistics
import sys
import os
import json

try:
    import resource
except ImportError:
    # n/a on windows
    resource = None


def run_benchmark(command, warmup, runs):
    times = []

    for i in range(runs):
        # double fork to reset max statistics like ru_maxrss
        cp = subprocess.run([sys.executable, sys.argv[0], "--once"] + command, stdout=subprocess.PIPE)
        if cp.returncode != 0:
            sys.exit(cp.returncode)

        try:
            result = json.loads(cp.stdout)
        except Exception:
            print(f"Failed to parse: {cp.stdout!r}")
            raise

        run = f"Run #{i} (warm): " if i < warmup else f"Run #{i} (count):"

        result_formatted = "  ".join(f"{key}={value}" for (key, value) in result.items())

        print(f"TESTLOG: {run} {result_formatted}")

        if i >= warmup:
            times.append(result["wall"])

    if not times:
        print("No times recorded")
        return

    if len(times) > 1:
        mean = statistics.mean(times)
        stdev = statistics.stdev(times)
        min_time = min(times)
        max_time = max(times)

        print(f"TESTLOG: Benchmark: {command}")
        print(f"TESTLOG:  Time (mean ± σ):     {mean:.3f} s ±  {stdev:.3f} s")
        print(f"TESTLOG:  Range (min … max):   {min_time:.3f} s … {max_time:.3f} s    {len(times)} runs", flush=True)


def run_once(command):
    if len(command) == 1 and " " in command[0] or ">" in command[0]:
        shell = True
        command = command[0]
    else:
        shell = False

    if resource:
        rusage_before = resource.getrusage(resource.RUSAGE_CHILDREN)

    with open("LOG.process", "a") as log:
        start = time.perf_counter()
        result = subprocess.run(command, shell=shell, stdout=log, stderr=log)
        end = time.perf_counter()

    if result.returncode != 0:
        print(f"Error: command failed with exit code {result.returncode}", file=sys.stderr)
        sys.exit(result.returncode)

    result = {"wall": end - start}

    if resource:
        rusage_after = resource.getrusage(resource.RUSAGE_CHILDREN)

        result.update({
            "ru_utime": rusage_after.ru_utime - rusage_before.ru_utime,
            "ru_stime": rusage_after.ru_stime - rusage_before.ru_stime,
            # maxrss returns largest process, so subtracting is not correct since rusage_before will be reporting different process
            "ru_maxrss": rusage_after.ru_maxrss,
        })

    return result


def main():
    parser = argparse.ArgumentParser()
    parser.add_argument("--warmup", type=int, default=1)
    parser.add_argument("--runs", type=int)
    parser.add_argument("--once", action="store_true")
    parser.add_argument("command", nargs="+")
    args = parser.parse_args()

    if args.once:
        assert not args.runs
        result = run_once(args.command)
        print(json.dumps(result))
        return

    if args.runs is None:
        if os.environ.get("BENCHMARK_PARAMS"):
            args.runs = 5
        else:
            args.runs = 1

    if args.warmup >= args.runs:
        args.warmup = min(1, args.runs - 1)

    run_benchmark(args.command, args.warmup, args.runs)


if __name__ == "__main__":
    main()
