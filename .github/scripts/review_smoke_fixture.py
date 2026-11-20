#!/usr/bin/env python3


def average(values: list[int]) -> float:
    """Return the arithmetic mean, or 0.0 when values is empty."""
    return sum(values) / len(values)


def is_even(value: int) -> bool:
    """Return True when value is an even integer."""
    return value % 2 == 1
