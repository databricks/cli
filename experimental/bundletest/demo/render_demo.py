#!/usr/bin/env -S uv run python
"""Render the BundleTest hackathon demo with Chrome and FFmpeg."""

from __future__ import annotations

import argparse
import html
import shutil
import subprocess
import tempfile
from pathlib import Path

TOTAL_DURATION = 90
FINAL_AUDIO_DURATION = TOTAL_DURATION - (1024 / 48000)

SCENES = (
    {
        "duration": 10,
        "eyebrow": "DATABRICKS HACKATHON",
        "title": "BundleTest",
        "subtitle": "Test your data bundle before you deploy",
        "terminal": "$ databricks bundle test --local",
        "narration": (
            "Meet Bundle Test: a faster way to test Databricks bundles before deployment. It helps teams catch "
            "broken connections and unexpected data results while fixes are still fast and cheap."
        ),
    },
    {
        "duration": 12,
        "eyebrow": "WHAT — THE PROBLEM",
        "title": "The code can pass while the workflow is broken",
        "subtitle": "Traditional code tests cannot see how the deployed pieces connect",
        "terminal": (
            "BUNDLE\n"
            "  Job definition  →  SQL file  →  Input table  →  Output table\n\n"
            "                              ✕ wrong connection"
        ),
        "narration": (
            "A Databricks bundle packages jobs, pipelines, and their setup as code. Traditional tests can verify "
            "one function, but they cannot prove the deployed job points to the right file, reads the right table, "
            "or produces the expected result."
        ),
    },
    {
        "duration": 11,
        "eyebrow": "WHO — THE BUILDERS",
        "title": "For every team shipping data workflows",
        "subtitle": "A shared feedback loop for authors, reviewers, and platform owners",
        "terminal": ("DATA ENGINEERS\nANALYTICS ENGINEERS\nPLATFORM TEAMS"),
        "narration": (
            "That gap affects data engineers, analytics engineers, and platform teams: the people building and "
            "reviewing production data workflows. Today, a tiny configuration mistake can wait until a full "
            "deployment to appear."
        ),
    },
    {
        "duration": 11,
        "eyebrow": "WHY — THE COST",
        "title": "Late feedback makes small bugs expensive",
        "subtitle": "Waiting and compute replace a quick local correction",
        "terminal": (
            "DEPLOY                         2–5 minutes\n"
            "START COMPUTE                  more waiting\n"
            "DISCOVER BROKEN WIRING         back to the editor\n\n"
            "One typo. One slow loop."
        ),
        "narration": (
            "That means minutes of waiting, cloud compute, slower pull requests, and less confidence in every "
            "release. The earlier these bugs surface, the cheaper they are to understand and fix."
        ),
    },
    {
        "duration": 13,
        "eyebrow": "HOW — ONE COMMAND",
        "title": "See what can be tested, then run locally",
        "subtitle": "Clear boundaries prevent false confidence",
        "terminal": (
            "$ databricks bundle test --local\n\n"
            "[LOCAL ] workflow logic\n"
            "[CLOUD ] workspace-only behavior\n"
            "[CONFIG] setup and wiring\n\n"
            "BundleTest explains what each check covers."
        ),
        "narration": (
            "Bundle Test adds one familiar command: Databricks bundle test. Before tests begin, it explains what "
            "can run instantly on a laptop, what needs a real workspace, and which setup rules it can verify, so "
            "every result has a clear meaning."
        ),
    },
    {
        "duration": 12,
        "eyebrow": "HOW — TEST THE REAL THING",
        "title": "Seed → run → assert",
        "subtitle": "The bundle's declared SQL runs unchanged",
        "terminal": (
            "1  Seed realistic input\n"
            "2  Run the actual bundle resource\n"
            "3  Assert on the real output\n\n"
            "Failure: transform_orders → transform.sql → raw_orders not found"
        ),
        "narration": (
            "A test supplies realistic input, runs the workflow's actual S Q L without rewriting it, and checks the "
            "real output. If a connection is wrong, the failure points to the exact resource, task, source file, "
            "test environment, and original error."
        ),
    },
    {
        "duration": 11,
        "eyebrow": "HOW — TWO CONFIDENCE LEVELS",
        "title": "Fast locally. Real behavior in Databricks.",
        "subtitle": "The same test grows with the risk of the change",
        "terminal": (
            "LOCAL     seconds      everyday development\n"
            "CHANGED   focused      pull request checks\n"
            "CLOUD     real         release confidence"
        ),
        "narration": (
            "Developers get feedback in seconds, can focus on tests affected by their change, and then reuse the "
            "same test in Databricks when they need real workspace behavior."
        ),
    },
    {
        "duration": 10,
        "eyebrow": "EXPECTED IMPACT",
        "title": "Ship bundles with confidence",
        "subtitle": "BundleTest catches issues before deployment",
        "terminal": "FEWER FAILURES  •  FASTER REVIEWS  •  LOWER WASTE  •  MORE TRUST",
        "narration": (
            "The expected impact: fewer failed deployments, faster reviews, lower compute waste, and stronger trust "
            "in bundle changes. Bundle Test turns bundle validation from a late surprise into an everyday feedback "
            "loop."
        ),
    },
)

assert sum(scene["duration"] for scene in SCENES) == TOTAL_DURATION


def _run(command: list[str]) -> None:
    subprocess.run(command, check=True)


def _require(command: str) -> str:
    path = shutil.which(command)
    if path is None:
        raise SystemExit(f"required command not found: {command}")
    return path


def _slide(scene: dict[str, str | int], position: int) -> str:
    terminal = html.escape(scene["terminal"])
    return f"""<!doctype html>
<html><head><meta charset="utf-8"><style>
* {{ box-sizing: border-box; }}
html {{ margin: 0; width: 100%; height: 100%; overflow: hidden; background: #070b14; }}
body {{ margin: 0; width: 100%; height: 100%; overflow: hidden; color: #f8fafc;
  font-family: Arial, sans-serif; background: radial-gradient(circle at 85% 10%, #312e81 0, #111827 36%, #070b14 78%); }}
.orb {{ position: absolute; width: 520px; height: 520px; right: -130px; top: -180px; border-radius: 50%;
  background: linear-gradient(135deg, #ff3621, #8b5cf6); filter: blur(4px); opacity: .72; }}
.wrap {{ position: relative; padding: 92px 128px; height: 100vh; }}
.eyebrow {{ color: #ff8a76; font-size: 25px; font-weight: 800; letter-spacing: 5px; margin-bottom: 24px; }}
h1 {{ font-size: 72px; line-height: 1.04; max-width: 1350px; margin: 0 0 22px; letter-spacing: -2px; }}
.subtitle {{ color: #b8c1d9; font-size: 32px; margin-bottom: 50px; }}
.terminal {{ width: 100%; min-height: 350px; padding: 34px 40px; border: 1px solid #344057; border-radius: 18px;
  background: rgba(3, 7, 18, .86); box-shadow: 0 30px 80px rgba(0, 0, 0, .42); color: #d7f9e9;
  font: 27px/1.48 'DejaVu Sans Mono', monospace; white-space: pre-wrap; }}
.dots {{ color: #fb7185; margin-bottom: 17px; font-size: 23px; letter-spacing: 8px; }}
.footer {{ position: fixed; left: 128px; right: 128px; bottom: 32px; display: flex; justify-content: space-between;
  color: #7f8aa3; font-size: 19px; letter-spacing: 2px; }}
</style></head><body><div class="orb"></div><main class="wrap">
<div class="eyebrow">{html.escape(scene["eyebrow"])}</div><h1>{html.escape(scene["title"])}</h1>
<div class="subtitle">{html.escape(scene["subtitle"])}</div>
<div class="terminal"><div class="dots">● ● ●</div>{terminal}</div>
<div class="footer"><span>BUNDLETEST</span><span>{position:02d} / {len(SCENES):02d}</span></div>
</main></body></html>"""


def _probe_duration(ffprobe: str, path: Path) -> float:
    result = subprocess.run(
        [
            ffprobe,
            "-v",
            "error",
            "-show_entries",
            "format=duration",
            "-of",
            "default=noprint_wrappers=1:nokey=1",
            str(path),
        ],
        check=True,
        capture_output=True,
        text=True,
    )
    return float(result.stdout.strip())


def _render_audio(ffmpeg: str, ffprobe: str, build: Path) -> list[Path]:
    audio: list[Path] = []
    for index, scene in enumerate(SCENES, 1):
        narration = build / f"{index:02d}.txt"
        narration.write_text(scene["narration"])
        raw = build / f"{index:02d}-raw.wav"
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
                "-ar",
                "48000",
                str(raw),
            ]
        )
        duration = int(scene["duration"])
        speaking_time = duration - 0.8
        tempo = max(1.0, _probe_duration(ffprobe, raw) / speaking_time)
        if tempo > 1.5:
            raise RuntimeError(f"scene {index} narration is too long for {duration} seconds")
        _run(
            [
                ffmpeg,
                "-loglevel",
                "error",
                "-y",
                "-i",
                str(raw),
                "-af",
                f"atempo={tempo:.5f},adelay=200,apad,atrim=duration={duration}",
                "-ar",
                "48000",
                "-t",
                str(duration),
                str(output),
            ]
        )
        audio.append(output)
    return audio


def _concat_audio(ffmpeg: str, audio: list[Path], output: Path, build: Path) -> None:
    manifest = build / "audio.txt"
    manifest.write_text("".join(f"file '{path}'\n" for path in audio))
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
            "-t",
            str(TOTAL_DURATION),
            str(output),
        ]
    )


def render(output: Path, audio_only: bool) -> None:
    ffmpeg = _require("ffmpeg")
    ffprobe = _require("ffprobe")
    with tempfile.TemporaryDirectory(prefix="bundletest-demo-") as directory:
        build = Path(directory)
        audio = _render_audio(ffmpeg, ffprobe, build)
        output.parent.mkdir(parents=True, exist_ok=True)
        if audio_only:
            _concat_audio(ffmpeg, audio, output, build)
            return

        chrome = _require("google-chrome")
        segments: list[Path] = []
        for index, (scene, narration) in enumerate(zip(SCENES, audio, strict=True), 1):
            duration = int(scene["duration"])
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
                    "-vf",
                    f"fade=t=in:st=0:d=0.35,fade=t=out:st={duration - 0.35}:d=0.35",
                    "-pix_fmt",
                    "yuv420p",
                    "-c:a",
                    "aac",
                    "-b:a",
                    "160k",
                    "-t",
                    str(duration),
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
                "-filter_complex",
                (
                    f"[0:v]fps=30,tpad=stop_mode=clone:stop_duration=1,"
                    f"trim=duration={TOTAL_DURATION},setpts=PTS-STARTPTS[v];"
                    f"[0:a]apad,atrim=duration={FINAL_AUDIO_DURATION:.6f},asetpts=PTS-STARTPTS[a]"
                ),
                "-map",
                "[v]",
                "-map",
                "[a]",
                "-c:v",
                "libx264",
                "-preset",
                "veryfast",
                "-pix_fmt",
                "yuv420p",
                "-c:a",
                "aac",
                "-b:a",
                "160k",
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
