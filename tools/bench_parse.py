#!/usr/bin/env python3
"""
Parses output of benchmark runs (e.g. "./task bench-100") and prints a summary table.
"""

import re
import statistics
import sys
from collections import defaultdict


def parse_key_values(text):
    """Parse key=value pairs from a string.

    >>> parse_key_values("wall=10.316  ru_utime=19.207  ru_stime=0.505  ru_maxrss=573079552")
    {'wall': 10.316, 'ru_utime': 19.207, 'ru_stime': 0.505, 'ru_maxrss': 573079552.0}

    Empty string returns empty dict:

    >>> parse_key_values("")
    {}

    Single pair:

    >>> parse_key_values("key=42")
    {'key': 42.0}

    Non-numeric values are kept as strings:

    >>> parse_key_values("name=test  count=5")
    {'name': 'test', 'count': 5.0}

    Pairs without equals sign are skipped:

    >>> parse_key_values("valid=1  invalid  also_valid=2")
    {'valid': 1.0, 'also_valid': 2.0}
    """
    result = {}
    for kv_pair in text.split():
        if "=" in kv_pair:
            key, value = kv_pair.split("=", 1)
            try:
                result[key] = float(value)
            except ValueError:
                result[key] = value
    return result


def parse_bench_output(file_path):
    """Parse benchmark output and extract test results."""
    results = defaultdict(lambda: defaultdict(list))

    current_test = None

    with open(file_path) as f:
        for line in f:
            test_match = re.match(r"=== RUN\s+(.+)", line)
            if test_match:
                current_test = test_match.group(1)
                current_test = current_test.removeprefix("TestAccept/bundle/benchmarks/")
                continue

            if "TESTLOG: Run #" in line and "(count)" in line:
                if current_test:
                    # Extract everything after the run label
                    parts = line.split("(count):")
                    if len(parts) == 2:
                        kv_data = parse_key_values(parts[1].strip())
                        for key, value in kv_data.items():
                            results[current_test][key].append(value)

    return results


def calculate_means(results):
    """Calculate mean values for each metric.

    >>> calculate_means({"test1": {"wall": [10, 20]}})
    {'test1': {'wall': 15}}

    >>> calculate_means({"test1": {"wall": [1.5, 2.5, 3.0]}})  # doctest: +ELLIPSIS
    {'test1': {'wall': 2.33...}}

    Multiple tests and metrics:

    >>> sorted(calculate_means({"t1": {"m1": [2, 4], "m2": [100]}, "t2": {"m1": [3]}}).items())
    [('t1', {'m1': 3, 'm2': 100}), ('t2', {'m1': 3})]

    Empty metrics list returns zero:

    >>> calculate_means({"test": {"metric": []}})
    {'test': {'metric': 0}}
    """
    means = {}
    for test_name, metrics in results.items():
        means[test_name] = {metric: statistics.mean(values) if values else 0 for metric, values in metrics.items()}
    return means


def print_results(results):
    """Output table for single file."""
    means = calculate_means(results)

    all_metrics = {}
    for metrics in means.values():
        for key in metrics:
            all_metrics.setdefault(key, None)
    all_metrics = list(all_metrics.keys())

    testname_width = max(len("testname"), max((len(name) for name in means.keys()), default=0))
    metric_width = 12

    header = f"{'testname':<{testname_width}}"
    for metric in all_metrics:
        header += f"  {metric:>{metric_width}}"
    print(header)
    print("-" * len(header))

    for test_name in sorted(means.keys()):
        m = means[test_name]
        row = f"{test_name:<{testname_width}}"
        for metric in all_metrics:
            value = m.get(metric, 0)
            if isinstance(value, float) and value > 1000000:
                row += f"  {value:>{metric_width}.0f}"
            else:
                row += f"  {value:>{metric_width}.3f}"
        print(row)


def main():
    for filename in sys.argv[1:]:
        results = parse_bench_output(filename)
        print_results(results)


if __name__ == "__main__":
    main()
