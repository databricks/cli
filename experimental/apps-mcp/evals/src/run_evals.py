#!/usr/bin/env python3
"""Apps-MCP Evaluation Runner for Databricks Jobs."""

import os
import subprocess
import sys
import tempfile
import time
from pathlib import Path
from typing import Optional

import fire


def start_docker_daemon() -> bool:
    """Start Docker daemon with vfs storage driver (works without privileges)."""
    print("Checking Docker installation...")

    # Check if Docker CLI is available
    result = subprocess.run(["which", "docker"], capture_output=True, text=True)
    if result.returncode != 0:
        print("Docker CLI not found, attempting to install...")
        subprocess.run(
            ["sudo", "bash", "-c", "curl -fsSL https://get.docker.com | sh"],
            check=False,
        )

    # Check if Docker is already running
    result = subprocess.run(
        ["docker", "info"], capture_output=True, text=True, timeout=10
    )
    if result.returncode == 0:
        print("Docker daemon already running")
        return True

    print("Starting Docker daemon...")

    # Start dockerd in background (config already set by init script)
    proc = subprocess.Popen(
        ["sudo", "dockerd"],
        stdout=subprocess.PIPE,
        stderr=subprocess.PIPE,
    )

    # Wait for Docker to start
    for i in range(60):
        time.sleep(1)
        result = subprocess.run(
            ["sudo", "docker", "info"], capture_output=True, text=True, timeout=10
        )
        if result.returncode == 0:
            print(f"Docker daemon started after {i+1}s")
            # Fix socket permissions
            subprocess.run(["sudo", "chmod", "666", "/var/run/docker.sock"], check=False)
            return True
        if proc.poll() is not None:
            stdout, stderr = proc.communicate()
            print(f"dockerd exited with code {proc.returncode}")
            print(f"stderr: {stderr.decode()[:500]}")
            break

    print("Failed to start Docker daemon")
    return False


def clone_and_install_klaudbiusz(work_dir: Path, git_url: str) -> Path:
    """Clone klaudbiusz and install dependencies."""
    print(f"Cloning {git_url}...")
    repo_dir = work_dir / "appdotbuild-agent"
    subprocess.run(["git", "clone", "--depth", "1", git_url, str(repo_dir)], check=True)
    klaudbiusz_dir = repo_dir / "klaudbiusz"
    print("Installing klaudbiusz...")
    subprocess.run([sys.executable, "-m", "pip", "install", "-e", str(klaudbiusz_dir)], check=True)
    sys.path.insert(0, str(klaudbiusz_dir))
    return klaudbiusz_dir


def find_apps_dir(apps_volume: str) -> Optional[Path]:
    """Find apps directory from UC Volume."""
    volume_path = Path(apps_volume)
    latest_file = volume_path / "latest.txt"
    if latest_file.exists():
        return Path(latest_file.read_text().strip())
    if volume_path.exists():
        run_dirs = [d for d in volume_path.iterdir() if d.is_dir() and d.name.startswith("run_")]
        if run_dirs:
            return max(run_dirs, key=lambda d: d.name)
    return None


def main(
    mlflow_experiment: str = "/Shared/apps-mcp-evaluations",
    parallelism: int = 4,
    apps_volume: Optional[str] = None,
    evals_git_url: str = "https://github.com/neondatabase/appdotbuild-agent.git",
) -> None:
    """Run Apps-MCP evaluations using klaudbiusz."""
    print("=" * 60)
    print("Apps-MCP Evaluation")
    print("=" * 60)
    print(f"  MLflow Experiment: {mlflow_experiment}")
    print(f"  Parallelism: {parallelism}")
    print(f"  Apps Volume: {apps_volume or 'not specified'}")

    # Try to start Docker daemon
    docker_available = start_docker_daemon()
    if not docker_available:
        print("Warning: Docker not available, container-based checks will fail")

    work_dir = Path(tempfile.mkdtemp(prefix="apps-mcp-evals-"))
    clone_and_install_klaudbiusz(work_dir, evals_git_url)

    from cli.evaluation import run_evaluation_simple

    apps_dir = find_apps_dir(apps_volume) if apps_volume else None
    if apps_dir:
        print(f"  Apps Dir: {apps_dir}")
    else:
        print("  Apps Dir: not found, will use default")
        apps_dir = work_dir / "appdotbuild-agent" / "klaudbiusz" / "app"

    print("\n" + "=" * 60)
    print("Running evaluation...")
    print("=" * 60)

    report = run_evaluation_simple(
        apps_dir=str(apps_dir),
        mlflow_experiment=mlflow_experiment,
        parallelism=parallelism,
        fast_mode=True,
    )

    summary = report.get("summary", {})
    metrics = summary.get("metrics_summary", {})

    print("\n" + "=" * 60)
    print("EVALUATION SUMMARY")
    print("=" * 60)
    print(f"Total Apps: {summary.get('total_apps', 0)}")
    print(f"Avg AppEval Score: {metrics.get('avg_appeval_100', 0):.1f}/100")
    print(f"Build Success: {metrics.get('build_success', 0)}")
    print(f"Runtime Success: {metrics.get('runtime_success', 0)}")
    print(f"Type Safety: {metrics.get('type_safety_pass', 0)}")
    print(f"Tests Pass: {metrics.get('tests_pass', 0)}")
    print("\nEvaluation complete!")


def cli():
    """CLI entry point."""
    fire.Fire(main)


if __name__ == "__main__":
    cli()
