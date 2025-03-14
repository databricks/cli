import sys
from collections import Counter


def format_size(n):
    if n >= 1000_000:
        return f"{n / 1024 / 1024.:.1f}MB"
    if n >= 100_000:
        return f"{n / 1024 / 1024.:.2f}MB"
    if n >= 10_000:
        return f"{n / 1024 / 1024.:.3f}MB"
    return str(n)


def process(input_lines):
    agg = Counter()
    counts = Counter()
    # symbols_by_start = {}

    for line in input_lines:
        parts = line.strip().split(maxsplit=3)
        if len(parts) != 4 or parts[2] in ("U", "B"):
            continue
        address = int(parts[0], 16)
        size = int(parts[1])
        symbol = parts[3]
        if address in symbols:
            # print(f"WARNING: overlap found:\n  prev: {symbols[address][-1]}\n  new:  {line.strip()}", file=sys.stderr)
            continue
        symbols[address] = (address + size, size, symbol, line.strip())

    total_size = 0
    agg = Counter()
    counts = Counter()

    for start in sorted(symbols):
        _, size, symbol, _ = symbols[start]
        total_size += size
        agg[symbol] += size

        separators = [idx for idx, c in enumerate(symbol) if c in "./"]
        for idx in separators:
            prefix = symbol[: idx + 1]
            key = prefix + "→"
            agg[key] += size
            counts[key] += 1

        agg["[total]"] = total_size
        counts["[total]"] += 1

    for k, v in agg.most_common():
        cc = counts[k]
        if cc == 1 and k.endswith("→"):
            continue
        try:
            if cc:
                print(f"{format_size(v)} {k} ({cc} items)")
            else:
                print(f"{format_size(v)} {k}")
        except BrokenPipeError:
            break


if __name__ == "__main__":
    symbols = {}
    process(sys.stdin)
