import base64
import struct
import zlib

import pytest

from databricks.bundles._bitmap import (
    CONTEXT_FULL_BUNDLE,
    SCHEMA,
    decode,
    decode_bits,
    decode_json,
    text,
    to_json,
)


def _encode(raw: bytes) -> str:
    compressor = zlib.compressobj(wbits=-15)
    return base64.b64encode(compressor.compress(raw) + compressor.flush()).decode()


def _payload(bits: list[bool], context: int = 0) -> bytes:
    bitmap = bytearray((len(bits) + 7) // 8)
    for i, bit in enumerate(bits):
        if bit:
            bitmap[i // 8] |= 1 << (7 - i % 8)
    return b"DBTB" + struct.pack(">IH", len(bits), context) + bytes(bitmap)


def test_embedded_schema():
    assert "bundle.name" in SCHEMA
    assert "telemetry.SelectUsed=true" in SCHEMA
    assert len(SCHEMA) == len(set(SCHEMA))


@pytest.mark.parametrize("n", [0, 1, 7, 8, 9, 16, 17])
def test_decode_bits_round_trip(n: int):
    bits = [i % 3 == 0 for i in range(n)]
    assert decode_bits(_encode(_payload(bits, context=0xBEEF))) == (bits, 0xBEEF)


def test_decode_bits_reads_cli_payload():
    # The raw payload that TestEncodeBitOrder pins for the CLI's Encode of bits
    # 1,0,0,0,0,0,0,1,1, so both implementations read the same bit order.
    encoded = _encode(
        bytes([0x44, 0x42, 0x54, 0x42, 0, 0, 0, 9, 0, 0, 0b1000_0001, 0b1000_0000])
    )
    assert decode_bits(encoded) == (
        [True, False, False, False, False, False, False, True, True],
        0,
    )


@pytest.mark.parametrize(
    ("encoded", "error"),
    [
        ("not-valid-base64!!!", "base64"),
        (base64.b64encode(b"\xff\xff").decode(), "decompressing"),
        (_encode(b"DBTB"), "payload too short"),
        (_encode(b"XXXX" + struct.pack(">IH", 1, 0) + b"\x80"), "bad magic"),
        (_encode(b"DBTB" + struct.pack(">IH", 9, 0)), "payload size mismatch"),
        (_encode(b"DBTB" + struct.pack(">IH", 0xFFFFFFFF, 0)), "payload size mismatch"),
    ],
)
def test_decode_bits_errors(encoded: str, error: str):
    with pytest.raises((ValueError, zlib.error), match=error):
        decode_bits(encoded)


SCHEMA_FOR_TEXT = [
    "bundle",
    "bundle.name",
    "flag",
    "flag=true",
    "flag=false",
    "jobs.*.enabled",
    "jobs.*.enabled=true",
    "jobs.*.enabled=false",
    "ref",
    "ref=true",
    "ref=false",
]


@pytest.mark.parametrize(
    ("set_paths", "lines"),
    [
        ([], []),
        (["bundle", "bundle.name"], ["P   bundle", "P   bundle.name"]),
        (["flag", "flag=true"], ["PT  flag"]),
        (["flag", "flag=false"], ["P F flag"]),
        (
            ["jobs.*.enabled", "jobs.*.enabled=true", "jobs.*.enabled=false"],
            ["PTF jobs.*.enabled"],
        ),
        (["ref"], ["P   ref"]),
        (["flag=true"], [" T  flag"]),
    ],
)
def test_text(set_paths: list[str], lines: list[str]):
    bits = [path in set_paths for path in SCHEMA_FOR_TEXT]
    header = f"size={len(SCHEMA_FOR_TEXT)} context=0"
    assert text(SCHEMA_FOR_TEXT, bits, CONTEXT_FULL_BUNDLE) == [header, *lines]


def test_text_older_bitmap():
    assert text(SCHEMA_FOR_TEXT, [True], 7) == ["size=1 context=7", "P   bundle"]


def test_decode():
    bits = [path in ("bundle", "bundle.name") for path in SCHEMA]
    assert decode(_encode(_payload(bits))) == [
        f"size={len(SCHEMA)} context=0",
        "P   bundle",
        "P   bundle.name",
    ]


@pytest.mark.parametrize("decoder", [decode, decode_json])
def test_decode_rejects_unknown_context(decoder):
    with pytest.raises(ValueError, match="unsupported bitmap context 1"):
        decoder(_encode(_payload([], context=1)))


@pytest.mark.parametrize("decoder", [decode, decode_json])
def test_decode_rejects_newer_bitmap(decoder):
    with pytest.raises(ValueError, match="decode it with a newer databricks-bundles"):
        decoder(_encode(_payload([False] * (len(SCHEMA) + 1))))


SCHEMA_FOR_JSON = [
    "bundle",
    "bundle.name",
    "jobs",
    "jobs.*",
    "jobs.*.tasks",
    "jobs.*.tasks[*]",
    "jobs.*.tasks[*].enabled",
    "jobs.*.tasks[*].enabled=true",
    "jobs.*.tasks[*].enabled=false",
    "later",
]
TASKS = ["jobs", "jobs.*", "jobs.*.tasks"]
ENABLED = [*TASKS, "jobs.*.tasks[*]", "jobs.*.tasks[*].enabled"]
ENABLED_TREE = {"jobs": {"*": {"tasks[*]": {"enabled": True}}}}


@pytest.mark.parametrize(
    ("set_paths", "presence", "true", "false"),
    [
        ([], {}, {}, {}),
        # A parent with set children is a subtree.
        (["bundle", "bundle.name"], {"bundle": {"name": True}}, {}, {}),
        # A set field without set children is a leaf.
        (TASKS, {"jobs": {"*": {"tasks": True}}}, {}, {}),
        # A list's elements replace its leaf.
        ([*TASKS, "jobs.*.tasks[*]"], {"jobs": {"*": {"tasks[*]": True}}}, {}, {}),
        ([*ENABLED, "jobs.*.tasks[*].enabled=false"], ENABLED_TREE, {}, ENABLED_TREE),
        # Both values under a wildcard.
        (
            [*ENABLED, "jobs.*.tasks[*].enabled=true", "jobs.*.tasks[*].enabled=false"],
            ENABLED_TREE,
            ENABLED_TREE,
            ENABLED_TREE,
        ),
    ],
)
def test_to_json(set_paths: list[str], presence: dict, true: dict, false: dict):
    bits = [path in set_paths for path in SCHEMA_FOR_JSON]
    assert to_json(SCHEMA_FOR_JSON, bits, CONTEXT_FULL_BUNDLE) == {
        "size": len(SCHEMA_FOR_JSON),
        "context": 0,
        "presence": presence,
        "true": true,
        "false": false,
    }


def test_to_json_older_bitmap():
    assert to_json(SCHEMA_FOR_JSON, [True], 7) == {
        "size": 1,
        "context": 7,
        "presence": {"bundle": True},
        "true": {},
        "false": {},
    }


def test_to_json_list_leaf_in_any_order():
    out = to_json(["x[*]", "x"], [True, True], CONTEXT_FULL_BUNDLE)
    assert out["presence"] == {"x[*]": True}


def test_to_json_key_order():
    # Same order as the CLI, which sorts map keys, so the printed JSON matches.
    out = to_json(["b", "a", "a.z", "a.y"], [True] * 4, CONTEXT_FULL_BUNDLE)
    assert list(out) == ["size", "context", "presence", "true", "false"]
    assert list(out["presence"]) == ["a", "b"]
    assert list(out["presence"]["a"]) == ["y", "z"]


def test_decode_json():
    bits = [path in ("bundle", "bundle.name") for path in SCHEMA]
    assert decode_json(_encode(_payload(bits))) == {
        "size": len(SCHEMA),
        "context": 0,
        "presence": {"bundle": {"name": True}},
        "true": {},
        "false": {},
    }
