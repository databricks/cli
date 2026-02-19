#!/usr/bin/env python3
"""Apps-MCP Evaluation Runner for Databricks Jobs."""

import subprocess
import sys
import tempfile
from pathlib import Path
from typing import Optional

import fire


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


def run_local_evaluation(apps_dir: Path, mlflow_experiment: str) -> dict:
    """Run local evaluation using shell scripts (no Docker/Dagger)."""
    import time
    from dataclasses import asdict

    from cli.evaluation.evaluate_app import evaluate_app
    from cli.evaluation.evaluate_all import generate_summary_report
    from cli.utils.apps_discovery import list_apps_in_dir

    app_dirs = list_apps_in_dir(apps_dir)
    if not app_dirs:
        raise ValueError(f"No apps found in: {apps_dir}")

    print(f"Evaluating {len(app_dirs)} apps locally...")

    results = []
    eval_start = time.time()

    for i, app_dir in enumerate(app_dirs, 1):
        print(f"\n[{i}/{len(app_dirs)}] {app_dir.name}")
        try:
            result = evaluate_app(app_dir, prompt=None, port=8000 + i)
            results.append(asdict(result))
        except Exception as e:
            print(f"  Error: {e}")

    eval_duration = time.time() - eval_start
    print(f"\nEvaluated {len(results)}/{len(app_dirs)} apps in {eval_duration:.1f}s")

    summary = generate_summary_report(results)
    report = {"summary": summary, "apps": results}

    if mlflow_experiment:
        from cli.evaluation.tracking import log_evaluation_to_mlflow, setup_mlflow
        if setup_mlflow(mlflow_experiment):
            run_id = log_evaluation_to_mlflow(report)
            if run_id:
                print(f"MLflow run logged: {run_id}")

    return report


def main(
    mlflow_experiment: str = "/Shared/apps-mcp-evaluations",
    parallelism: int = 4,
    apps_volume: Optional[str] = None,
    evals_git_url: str = "https://github.com/neondatabase/appdotbuild-agent.git",
) -> None:
    """Run Apps-MCP evaluations using klaudbiusz (local mode)."""
    print("=" * 60)
    print("Apps-MCP Evaluation (Local Mode)")
    print("=" * 60)
    print(f"  MLflow Experiment: {mlflow_experiment}")
    print(f"  Apps Volume: {apps_volume or 'not specified'}")

    work_dir = Path(tempfile.mkdtemp(prefix="apps-mcp-evals-"))
    clone_and_install_klaudbiusz(work_dir, evals_git_url)

    apps_dir = find_apps_dir(apps_volume) if apps_volume else None
    if apps_dir:
        print(f"  Apps Dir: {apps_dir}")
    else:
        print("  Apps Dir: not found, will use default")
        apps_dir = work_dir / "appdotbuild-agent" / "klaudbiusz" / "app"

    print("\n" + "=" * 60)
    print("Running local evaluation...")
    print("=" * 60)

    report = run_local_evaluation(
        apps_dir=apps_dir,
        mlflow_experiment=mlflow_experiment,
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
