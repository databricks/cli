#!/usr/bin/env python3
"""
Parses output of benchmark runs (e.g. "make bench100") and prints a summary table.
"""

import sys
import re
import statistics
from collections import defaultdict


def parse_key_values(text):
    """Parse key=value pairs from a string.

    >>> parse_key_values("wall=10.316  ru_utime=19.207  ru_stime=0.505  ru_maxrss=573079552")
    {'wall': 10.316, 'ru_utime': 19.207, 'ru_stime': 0.505, 'ru_maxrss': 573079552.0}
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
            # Match test name
            test_match = re.match(r"=== RUN\s+(.+)", line)
            if test_match:
                current_test = test_match.group(1)
                current_test = current_test.removeprefix("TestAccept/bundle/benchmarks/")
                continue

            # Match benchmark run data (only count runs, skip warm)
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
    """Calculate mean values for each metric."""
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

    # Calculate column widths
    testname_width = max(len("testname"), max((len(name) for name in means.keys()), default=0))
    metric_width = 12

    # Print header
    header = f"{'testname':<{testname_width}}"
    for metric in all_metrics:
        header += f"  {metric:>{metric_width}}"
    print(header)
    print("-" * len(header))

    # Print rows
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
