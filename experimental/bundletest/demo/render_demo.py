#!/usr/bin/env -S uv run python
"""Render the BundleTest hackathon demo with Chrome and FFmpeg."""

from __future__ import annotations

import argparse
import html
import shutil
import subprocess
import tempfile
from pathlib import Path

SCENES = (
    {
        "eyebrow": "DATABRICKS HACKATHON",
        "title": "pytest for Databricks bundles",
        "subtitle": "Catch wiring and data-contract bugs before deployment",
        "terminal": "$ databricks bundle test --local",
        "narration": (
            "Databricks bundles are easy to declare, but today we often discover wiring bugs only after a deploy "
            "and a real job run. Bundle Test brings the pie test feedback loop to Databricks bundles."
        ),
    },
    {
        "eyebrow": "THE PROBLEM",
        "title": "Unit tests stop at the function boundary",
        "subtitle": "Bundle wiring fails later, after compute and deployment time",
        "terminal": (
            "$ databricks bundle deploy\n"
            "✓ deployment complete                     2m 41s\n"
            "$ databricks bundle run transform_orders\n"
            "✗ TABLE_OR_VIEW_NOT_FOUND: raw_orders"
        ),
        "narration": (
            "A unit test can prove that a transform function works. It cannot prove that the deployed job points "
            "at the right SQL file, reads the right table, or writes the expected output. Those mistakes turn a "
            "tiny bug into a multi minute feedback loop."
        ),
    },
    {
        "eyebrow": "ONE COMMAND",
        "title": "Know what runs locally before pytest starts",
        "subtitle": "Clear fidelity boundaries instead of false confidence",
        "terminal": (
            "$ databricks bundle test --local\n\n"
            "[LOCAL ] jobs.transform_orders/transform: runs src/transform.sql\n"
            "[CLOUD ] jobs.score_model/score: notebook_task requires a workspace\n"
            "[CONFIG] 4 non-job resources: configuration assertions only\n\n"
            "summary: 1 local, 1 cloud-only, 4 config-only"
        ),
        "narration": (
            "Now I can run Databricks bundle test locally. Before pie test starts, Bundle Test shows exactly which "
            "tasks run in Duck D B, which need a workspace, and which resources support configuration assertions."
        ),
    },
    {
        "eyebrow": "REAL ARTIFACTS",
        "title": "Seed → run → assert",
        "subtitle": "The job's declared SQL runs unchanged",
        "terminal": (
            '@pytest.mark.bundle_resource("jobs.transform_orders")\n'
            "def test_transform_dedupes(env):\n"
            '    env.seed("shop.bronze.raw_orders", rows)\n'
            '    env.run_job("transform_orders")\n'
            '    assert env.table("shop.silver.orders").row_count() == 2'
        ),
        "narration": (
            "The test seeds the upstream boundary, runs the job's actual S Q L artifact, and asserts on the resulting "
            "table. The query body is never mocked or rewritten, so a wrong table name stays a real failure."
        ),
    },
    {
        "eyebrow": "ACTIONABLE FAILURES",
        "title": "Debug from the first error",
        "subtitle": "Resource context travels with the original engine message",
        "terminal": (
            "bundle job 'transform_orders' failed\n"
            "  task: transform\n"
            "  source: src/transform_orders.sql\n"
            "  backend: local\n"
            "  error: Catalog Error: Table raw_orders does not exist"
        ),
        "narration": (
            "When a run fails, the assertion includes the bundle resource, task, source file, backend, run identifier "
            "when available, and the original engine error. The next debugging step is visible immediately."
        ),
    },
    {
        "eyebrow": "FAST PULL REQUESTS",
        "title": "Run only tests affected by a change",
        "subtitle": "Bundle paths map back to resource-aware pytest markers",
        "terminal": (
            "$ databricks bundle test --local --changed --base origin/main\n\n"
            "bundletest changed selection: origin/main\n"
            "[RESOURCE] jobs.transform_orders\n\n"
            "3 passed, 36 deselected in 0.42s"
        ),
        "narration": (
            "For fast pull request checks, changed mode maps edited bundle files back to their resources and selects "
            "tests through pie test markers. Changed yam-ul safely runs the complete suite."
        ),
    },
    {
        "eyebrow": "TWO FIDELITY TIERS",
        "title": "The same test runs on a real workspace",
        "subtitle": "Choose speed locally and full fidelity when needed",
        "terminal": (
            "$ databricks bundle test --cloud \\\n"
            "    --profile hackathon \\\n"
            "    --warehouse-id abc123\n\n"
            "bundletest: running cloud tests\n"
            "✓ 6 passed in 2m 08s"
        ),
        "narration": (
            "When local fidelity is not enough, the same test runs on Databricks with an explicit profile. Local runs "
            "provide speed, cloud runs provide full fidelity, and teams keep the pie test tools they already know."
        ),
    },
    {
        "eyebrow": "BUNDLETEST",
        "title": "Fast feedback. Real bundle confidence.",
        "subtitle": "pytest-style isolation testing for Databricks bundles",
        "terminal": "LOCAL in seconds  →  CLOUD when it matters  →  SHIP with confidence",
        "narration": (
            "Bundle Test catches bundle wiring and data contract bugs before they become slow deployment failures. "
            "It is pie test for Databricks bundles."
        ),
    },
)


def _run(command: list[str]) -> None:
    subprocess.run(command, check=True)


def _require(command: str) -> str:
    path = shutil.which(command)
    if path is None:
        raise SystemExit(f"required command not found: {command}")
    return path


def _slide(scene: dict[str, str], position: int) -> str:
    terminal = html.escape(scene["terminal"])
    return f"""<!doctype html>
<html><head><meta charset="utf-8"><style>
* {{ box-sizing: border-box; }}
body {{ margin: 0; width: 1920px; height: 1080px; overflow: hidden; color: #f8fafc;
  font-family: Arial, sans-serif; background: radial-gradient(circle at 85% 10%, #312e81 0, #111827 36%, #070b14 78%); }}
.orb {{ position: absolute; width: 520px; height: 520px; right: -130px; top: -180px; border-radius: 50%;
  background: linear-gradient(135deg, #ff3621, #8b5cf6); filter: blur(4px); opacity: .72; }}
.wrap {{ position: relative; padding: 92px 128px; height: 100%; }}
.eyebrow {{ color: #ff8a76; font-size: 25px; font-weight: 800; letter-spacing: 5px; margin-bottom: 24px; }}
h1 {{ font-size: 72px; line-height: 1.04; max-width: 1350px; margin: 0 0 22px; letter-spacing: -2px; }}
.subtitle {{ color: #b8c1d9; font-size: 32px; margin-bottom: 50px; }}
.terminal {{ width: 100%; min-height: 350px; padding: 34px 40px; border: 1px solid #344057; border-radius: 18px;
  background: rgba(3, 7, 18, .86); box-shadow: 0 30px 80px rgba(0, 0, 0, .42); color: #d7f9e9;
  font: 27px/1.48 'DejaVu Sans Mono', monospace; white-space: pre-wrap; }}
.dots {{ color: #fb7185; margin-bottom: 17px; font-size: 23px; letter-spacing: 8px; }}
.footer {{ position: absolute; left: 128px; right: 128px; bottom: 42px; display: flex; justify-content: space-between;
  color: #7f8aa3; font-size: 19px; letter-spacing: 2px; }}
</style></head><body><div class="orb"></div><main class="wrap">
<div class="eyebrow">{html.escape(scene["eyebrow"])}</div><h1>{html.escape(scene["title"])}</h1>
<div class="subtitle">{html.escape(scene["subtitle"])}</div>
<div class="terminal"><div class="dots">● ● ●</div>{terminal}</div>
<div class="footer"><span>BUNDLETEST</span><span>{position:02d} / {len(SCENES):02d}</span></div>
</main></body></html>"""


def _render_audio(ffmpeg: str, build: Path) -> list[Path]:
    audio: list[Path] = []
    for index, scene in enumerate(SCENES, 1):
        narration = build / f"{index:02d}.txt"
        narration.write_text(scene["narration"])
        output = build / f"{index:02d}.wav"
        _run(
            [
                ffmpeg,
                "-loglevel",
                "error",
                "-y",
                "-f",
                "lavfi",
                "-i",
                f"flite=textfile={narration}:voice=slt",
                "-af",
                "atempo=1.35,apad=pad_dur=0.6",
                "-ar",
                "48000",
                str(output),
            ]
        )
        audio.append(output)
    return audio


def _concat_audio(ffmpeg: str, audio: list[Path], output: Path, build: Path) -> None:
    manifest = build / "audio.txt"
    manifest.write_text("".join(f"file '{path}'\n" for path in audio))
    _run([ffmpeg, "-loglevel", "error", "-y", "-f", "concat", "-safe", "0", "-i", str(manifest), str(output)])


def render(output: Path, audio_only: bool) -> None:
    ffmpeg = _require("ffmpeg")
    _require("ffprobe")
    chrome = _require("google-chrome")
    with tempfile.TemporaryDirectory(prefix="bundletest-demo-") as directory:
        build = Path(directory)
        audio = _render_audio(ffmpeg, build)
        output.parent.mkdir(parents=True, exist_ok=True)
        if audio_only:
            _concat_audio(ffmpeg, audio, output, build)
            return

        segments: list[Path] = []
        for index, (scene, narration) in enumerate(zip(SCENES, audio, strict=True), 1):
            page = build / f"{index:02d}.html"
            image = build / f"{index:02d}.png"
            segment = build / f"{index:02d}.mp4"
            page.write_text(_slide(scene, index))
            _run(
                [
                    chrome,
                    "--headless=new",
                    "--no-sandbox",
                    "--disable-gpu",
                    "--disable-dev-shm-usage",
                    f"--user-data-dir={build / 'chrome'}",
                    "--hide-scrollbars",
                    "--window-size=1920,1080",
                    f"--screenshot={image}",
                    page.as_uri(),
                ]
            )
            _run(
                [
                    ffmpeg,
                    "-loglevel",
                    "error",
                    "-y",
                    "-loop",
                    "1",
                    "-framerate",
                    "30",
                    "-i",
                    str(image),
                    "-i",
                    str(narration),
                    "-c:v",
                    "libx264",
                    "-preset",
                    "veryfast",
                    "-tune",
                    "stillimage",
                    "-pix_fmt",
                    "yuv420p",
                    "-c:a",
                    "aac",
                    "-b:a",
                    "160k",
                    "-shortest",
                    str(segment),
                ]
            )
            segments.append(segment)

        manifest = build / "video.txt"
        manifest.write_text("".join(f"file '{path}'\n" for path in segments))
        _run(
            [
                ffmpeg,
                "-loglevel",
                "error",
                "-y",
                "-f",
                "concat",
                "-safe",
                "0",
                "-i",
                str(manifest),
                "-c",
                "copy",
                "-movflags",
                "+faststart",
                str(output),
            ]
        )


def main() -> None:
    parser = argparse.ArgumentParser(description=__doc__)
    mode = parser.add_mutually_exclusive_group(required=True)
    mode.add_argument("--output", type=Path, help="render a narrated MP4")
    mode.add_argument("--audio-only", type=Path, help="render only the WAV voiceover")
    options = parser.parse_args()
    target = options.output or options.audio_only
    render(target.resolve(), audio_only=options.audio_only is not None)
    print(target.resolve())


if __name__ == "__main__":
    main()
