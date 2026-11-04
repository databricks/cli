import base64
import struct
import zlib

import pytest

from databricks.bundles._bitmap import SCHEMA, decode, decode_bits, text


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
        (["bundle", "bundle.name"], ["1 bundle", "1 bundle.name"]),
        (["flag", "flag=true"], ["T flag"]),
        (["flag", "flag=false"], ["F flag"]),
        (
            ["jobs.*.enabled", "jobs.*.enabled=true", "jobs.*.enabled=false"],
            ["T jobs.*.enabled", "F jobs.*.enabled"],
        ),
        (["ref"], ["1 ref"]),
    ],
)
def test_text(set_paths: list[str], lines: list[str]):
    bits = [path in set_paths for path in SCHEMA_FOR_TEXT]
    assert text(SCHEMA_FOR_TEXT, bits) == lines


def test_text_older_bitmap():
    assert text(SCHEMA_FOR_TEXT, [True]) == ["1 bundle"]


def test_decode():
    bits = [path in ("bundle", "bundle.name") for path in SCHEMA]
    assert decode(_encode(_payload(bits))) == ["1 bundle", "1 bundle.name"]


def test_decode_rejects_unknown_context():
    with pytest.raises(ValueError, match="unsupported bitmap context 1"):
        decode(_encode(_payload([], context=1)))


def test_decode_rejects_newer_bitmap():
    with pytest.raises(ValueError, match="decode it with a newer databricks-bundles"):
        decode(_encode(_payload([False] * (len(SCHEMA) + 1))))
