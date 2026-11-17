"""Decoder for the bundle bitmap that ``databricks bundle bitmap bitmap`` prints.

Self-contained: the schema is embedded (see ``_schema``), so decoding does not
need the CLI. :func:`decode` returns the same lines as
``databricks bundle bitmap decode`` and :func:`decode_json` the same object as
``databricks bundle bitmap decode -o json``.

Wire format (``bundle/bitmap/bitmap.go`` in the CLI): base64 of raw DEFLATE of
magic ``DBTB`` (4 bytes) | size (uint32, big-endian) | context (uint16,
big-endian) | ceil(size / 8) bitmap bytes, with bit ``i`` (MSB first within each
byte) standing for schema line ``i``.
"""

import base64
import struct
import zlib
from typing import Any

from databricks.bundles._bitmap._schema import SCHEMA

__all__ = [
    "CONTEXT_FULL_BUNDLE",
    "SCHEMA",
    "decode",
    "decode_bits",
    "decode_json",
    "text",
    "to_json",
]

CONTEXT_FULL_BUNDLE = 0

_MAGIC = b"DBTB"
_HEADER_SIZE = 10
_TRUE_SUFFIX = "=true"
_FALSE_SUFFIX = "=false"
_LIST_SUFFIX = "[*]"


def decode(encoded: str) -> list[str]:
    """Decode an encoded bitmap against the embedded schema into :func:`text` lines."""
    bits, context = _decode_bundle(encoded)
    return text(SCHEMA, bits, context)


def decode_json(encoded: str) -> dict[str, Any]:
    """Decode an encoded bitmap against the embedded schema into :func:`to_json` output."""
    bits, context = _decode_bundle(encoded)
    return to_json(SCHEMA, bits, context)


def _decode_bundle(encoded: str) -> tuple[list[bool], int]:
    bits, context = decode_bits(encoded)
    if context != CONTEXT_FULL_BUNDLE:
        raise ValueError(f"unsupported bitmap context {context}")
    # The schema is append-only, so an older bitmap decodes against a prefix of
    # it; a newer one has bits this package cannot name.
    if len(bits) > len(SCHEMA):
        raise ValueError(
            f"bitmap has {len(bits)} bits but the schema has {len(SCHEMA)} fields; "
            "decode it with a newer databricks-bundles"
        )
    return bits, context


def decode_bits(encoded: str) -> tuple[list[bool], int]:
    """Decode an encoded bitmap into its bits and context.

    Raises ``ValueError`` (including ``binascii.Error``) or ``zlib.error`` on input
    that is not an encoded bitmap.
    """
    raw = zlib.decompress(base64.b64decode(encoded.strip(), validate=True), -15)
    if len(raw) < _HEADER_SIZE:
        raise ValueError(f"payload too short: {len(raw)} bytes")
    if raw[:4] != _MAGIC:
        raise ValueError(f"bad magic {raw[:4]!r}")
    size, context = struct.unpack(">IH", raw[4:_HEADER_SIZE])
    if len(raw) - _HEADER_SIZE != (size + 7) // 8:
        raise ValueError(f"payload size mismatch: {size} bits, {len(raw)} bytes")
    bitmap = raw[_HEADER_SIZE:]
    bits = [(bitmap[i // 8] >> (7 - i % 8)) & 1 == 1 for i in range(size)]
    return bits, context


def text(schema: list[str], bits: list[bool], context: int) -> list[str]:
    """Render a ``"size=N context=C"`` header and then the set bits as
    ``"<value> <field>"`` lines in schema order.

    ``T`` or ``F`` is a bool field holding that value and ``1`` any other present
    field. A bool shown as ``1`` holds an unresolved variable reference; one shown
    as both ``T`` and ``F`` has both values under a wildcard. Unset fields are
    omitted. ``bits`` may be shorter than ``schema`` but not longer.
    """
    schema = schema[: len(bits)]
    index = {path: i for i, path in enumerate(schema)}
    lines = [f"size={len(bits)} context={context}"]
    for i, path in enumerate(schema):
        if not bits[i]:
            continue
        if path.endswith(_TRUE_SUFFIX):
            lines.append("T " + path[: -len(_TRUE_SUFFIX)])
        elif path.endswith(_FALSE_SUFFIX):
            lines.append("F " + path[: -len(_FALSE_SUFFIX)])
        elif not (
            _is_set(index, bits, path + _TRUE_SUFFIX)
            or _is_set(index, bits, path + _FALSE_SUFFIX)
        ):
            # A bool with a value bit is shown by its value line instead.
            lines.append("1 " + path)
    return lines


def _is_set(index: dict[str, int], bits: list[bool], path: str) -> bool:
    i = index.get(path)
    return i is not None and bits[i]


def to_json(schema: list[str], bits: list[bool], context: int) -> dict[str, Any]:
    """Render the set bits as ``size``, ``context`` and three trees of set fields:
    ``presence``, ``true`` and ``false``.

    The trees are keyed by path segment (``*`` for a map key, ``[*]`` for a list
    element) with ``True`` leaves. A bool field is in ``presence`` and in the tree
    of each value it holds. Keys are in the CLI's order, so ``json.dumps(...,
    indent=2)`` prints what ``databricks bundle bitmap decode -o json`` prints.
    ``bits`` may be shorter than ``schema`` but not longer.
    """
    presence: dict[str, Any] = {}
    true_tree: dict[str, Any] = {}
    false_tree: dict[str, Any] = {}
    for path, bit in zip(schema[: len(bits)], bits, strict=True):
        if not bit:
            continue
        tree = presence
        if path.endswith(_TRUE_SUFFIX):
            tree, path = true_tree, path[: -len(_TRUE_SUFFIX)]
        elif path.endswith(_FALSE_SUFFIX):
            tree, path = false_tree, path[: -len(_FALSE_SUFFIX)]
        _insert(tree, _tree_keys(path))
    return {
        "size": len(bits),
        "context": context,
        "presence": _sorted(presence),
        "true": _sorted(true_tree),
        "false": _sorted(false_tree),
    }


def _tree_keys(path: str) -> list[str]:
    keys = []
    for part in path.split("."):
        if part.endswith(_LIST_SUFFIX):
            keys += [part[: -len(_LIST_SUFFIX)], _LIST_SUFFIX]
        else:
            keys.append(part)
    return keys


def _insert(tree: dict[str, Any], keys: list[str]) -> None:
    # A field with set children is a subtree: a child replaces the field's True
    # leaf, and a leaf never overwrites a subtree.
    for key in keys[:-1]:
        child = tree.get(key)
        if not isinstance(child, dict):
            child = tree[key] = {}
        tree = child
    tree.setdefault(keys[-1], True)


def _sorted(tree: dict[str, Any]) -> dict[str, Any]:
    # The CLI marshals Go maps, which encoding/json writes with sorted keys.
    return {
        key: _sorted(value) if isinstance(value, dict) else value
        for key, value in sorted(tree.items())
    }
