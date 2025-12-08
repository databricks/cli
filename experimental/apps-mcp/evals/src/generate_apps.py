#!/usr/bin/env python3
"""Generate apps using klaudbiusz with CLI-built MCP server."""

import os
import shutil
import subprocess
import sys
from datetime import datetime
from pathlib import Path

import fire


def clone_klaudbiusz(work_dir: Path) -> Path:
    """Clone the klaudbiusz generation framework."""
    repo_dir = work_dir / "appdotbuild-agent"
    if repo_dir.exists():
        shutil.rmtree(repo_dir)

    print("Cloning appdotbuild-agent repository...")
    subprocess.run(
        [
            "git",
            "clone",
            "--depth",
            "1",
            "https://github.com/neondatabase/appdotbuild-agent.git",
            str(repo_dir),
        ],
        check=True,
    )
    return repo_dir


def install_klaudbiusz_deps(klaudbiusz_dir: Path) -> None:
    """Install klaudbiusz Python dependencies."""
    print("Installing klaudbiusz dependencies...")
    result = subprocess.run(
        [sys.executable, "-m", "pip", "install", "-e", str(klaudbiusz_dir)],
        capture_output=True,
        text=True,
    )
    if result.returncode != 0:
        print(f"Warning: pip install had issues: {result.stderr[:500]}")


def run_generation(
    klaudbiusz_dir: Path,
    mcp_binary: str,
    output_dir: Path,
    prompts: str,
    max_concurrency: int,
) -> None:
    """Run bulk app generation using klaudbiusz."""
    print(f"\nStarting app generation...")
    print(f"  MCP binary: {mcp_binary}")
    print(f"  Prompts: {prompts}")
    print(f"  Max concurrency: {max_concurrency}")
    print(f"  Output dir: {output_dir}")

    env = os.environ.copy()
    env["PYTHONPATH"] = str(klaudbiusz_dir)

    cmd = [
        sys.executable,
        "-m",
        "cli.generation.bulk_run",
        f"--prompts={prompts}",
        f"--mcp_binary={mcp_binary}",
        '--mcp_args=["experimental", "apps-mcp"]',
        f"--max_concurrency={max_concurrency}",
        f"--output_dir={output_dir}",
    ]

    print(f"\nRunning: {' '.join(cmd)}")
    result = subprocess.run(cmd, cwd=klaudbiusz_dir, env=env)

    if result.returncode != 0:
        print(f"Generation completed with return code: {result.returncode}")


def upload_to_volume(local_dir: Path, volume_path: str) -> int:
    """Upload generated apps to UC Volume."""
    if not local_dir.exists():
        print(f"No apps directory found at {local_dir}")
        return 0

    apps = list(local_dir.iterdir())
    if not apps:
        print("No apps generated")
        return 0

    print(f"\nUploading {len(apps)} apps to {volume_path}...")

    volume_dir = Path(volume_path)
    volume_dir.mkdir(parents=True, exist_ok=True)

    timestamp = datetime.now().strftime("%Y%m%d_%H%M%S")
    dest_dir = volume_dir / f"run_{timestamp}"

    shutil.copytree(local_dir, dest_dir)
    print(f"Uploaded to {dest_dir}")

    latest_link = volume_dir / "latest"
    if latest_link.exists():
        latest_link.unlink()
    latest_link.symlink_to(dest_dir.name)

    return len(apps)


def main(
    mcp_binary: str,
    output_volume: str,
    prompts: str = "databricks",
    max_concurrency: int = 4,
) -> None:
    """
    Generate apps using klaudbiusz with the Databricks CLI as MCP server.

    Args:
        mcp_binary: Path to databricks-linux binary in UC Volume
        output_volume: UC Volume path for generated apps
        prompts: Prompt set (databricks, databricks_v2, test)
        max_concurrency: Number of parallel generations
    """
    print("=" * 60)
    print("Apps-MCP Generation")
    print("=" * 60)
    print(f"  MCP Binary: {mcp_binary}")
    print(f"  Output Volume: {output_volume}")
    print(f"  Prompts: {prompts}")
    print(f"  Max Concurrency: {max_concurrency}")

    if not Path(mcp_binary).exists():
        print(f"\nError: MCP binary not found at {mcp_binary}")
        print("Please upload the databricks-linux binary to the UC Volume first.")
        sys.exit(1)

    subprocess.run(["chmod", "+x", mcp_binary], check=True)

    work_dir = Path("/tmp/apps-generation")
    work_dir.mkdir(exist_ok=True)

    repo_dir = clone_klaudbiusz(work_dir)
    klaudbiusz_dir = repo_dir / "klaudbiusz"

    install_klaudbiusz_deps(klaudbiusz_dir)

    local_output = work_dir / "generated_apps"
    local_output.mkdir(exist_ok=True)

    run_generation(
        klaudbiusz_dir=klaudbiusz_dir,
        mcp_binary=mcp_binary,
        output_dir=local_output,
        prompts=prompts,
        max_concurrency=max_concurrency,
    )

    app_count = upload_to_volume(local_output, output_volume)

    print("\n" + "=" * 60)
    print("Generation Complete")
    print("=" * 60)
    print(f"  Apps generated: {app_count}")
    print(f"  Output location: {output_volume}")


if __name__ == "__main__":
    fire.Fire(main)
