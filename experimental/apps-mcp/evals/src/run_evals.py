#!/usr/bin/env python3
"""
Apps-MCP Evaluation Runner for Databricks Jobs.

Runs bundle deploy/run to generate apps, then evaluates and logs to MLflow.
"""

import json
import os
import subprocess
import sys
import tempfile
from datetime import datetime
from pathlib import Path
from typing import Optional

import fire
import mlflow


def setup_mlflow(experiment_name: str) -> None:
    """Configure MLflow to use Databricks tracking."""
    mlflow.set_tracking_uri("databricks")
    mlflow.set_experiment(experiment_name)


def clone_evals_repo(git_url: str, target_dir: Path) -> Path:
    """Clone appdotbuild-agent repository."""
    if target_dir.exists():
        subprocess.run(["git", "-C", str(target_dir), "pull"], check=True, capture_output=True)
    else:
        result = subprocess.run(
            ["git", "clone", "--depth", "1", git_url, str(target_dir)],
            check=False,
            capture_output=True,
            text=True,
        )
        if result.returncode != 0:
            print(f"Git clone stderr: {result.stderr}")
            raise RuntimeError(f"Failed to clone {git_url}: {result.stderr}")
    return target_dir


def install_klaudbiusz_deps(evals_dir: Path) -> None:
    """Install klaudbiusz dependencies using pip."""
    klaudbiusz_dir = evals_dir / "klaudbiusz"
    if not klaudbiusz_dir.exists():
        print(f"klaudbiusz directory not found at {klaudbiusz_dir}")
        return

    print("Installing klaudbiusz dependencies...")
    result = subprocess.run(
        [sys.executable, "-m", "pip", "install", "-e", str(klaudbiusz_dir)],
        capture_output=True,
        text=True,
    )
    if result.returncode != 0:
        print(f"pip install output: {result.stdout}")
        print(f"pip install errors: {result.stderr}")


def run_evaluation(
    evals_dir: Path,
    apps_dir: Path,
    parallelism: int = 4,
    fast_mode: bool = True,
) -> dict:
    """Run evaluation on generated apps using klaudbiusz."""
    klaudbiusz_dir = evals_dir / "klaudbiusz"

    cmd = [
        sys.executable,
        "-m",
        "cli.evaluation.evaluate_all",
        "--dir",
        str(apps_dir),
        "--parallel",
        str(parallelism),
    ]
    if fast_mode:
        cmd.append("--fast")

    env = os.environ.copy()
    env["PYTHONPATH"] = str(klaudbiusz_dir)

    print(f"Running: {' '.join(cmd)}")
    print(f"Working dir: {klaudbiusz_dir}")
    print(f"Apps dir: {apps_dir}")

    result = subprocess.run(cmd, cwd=klaudbiusz_dir, env=env, capture_output=True, text=True)

    print(f"Evaluation stdout: {result.stdout[:2000] if result.stdout else 'empty'}")
    if result.returncode != 0:
        print(f"Evaluation errors: {result.stderr[:2000] if result.stderr else 'empty'}")

    eval_output_dir = klaudbiusz_dir / "cli" / "app-eval"
    report_file = eval_output_dir / "evaluation_report.json"
    if report_file.exists():
        return json.loads(report_file.read_text())
    return {}


def log_results_to_mlflow(
    evaluation_report: dict,
    generation_results: Optional[dict] = None,
    run_name: Optional[str] = None,
) -> str:
    """Log evaluation results to MLflow."""
    if not run_name:
        run_name = f"eval-{datetime.utcnow().strftime('%Y%m%d_%H%M%S')}"

    with mlflow.start_run(run_name=run_name) as run:
        mlflow.set_tag("framework", "apps-mcp-evals")
        mlflow.set_tag("run_type", "scheduled")

        summary = evaluation_report.get("summary", {})
        mlflow.log_param("total_apps", summary.get("total_apps", 0))
        mlflow.log_param("timestamp", summary.get("evaluated_at", ""))

        metrics = summary.get("metrics_summary", {})
        if metrics:
            mlflow.log_metric("avg_appeval_100", metrics.get("avg_appeval_100", 0))
            if metrics.get("avg_eff_units") is not None:
                mlflow.log_metric("avg_eff_units", metrics["avg_eff_units"])
            mlflow.log_metric(
                "build_success_rate", metrics.get("build_success", 0) / max(summary.get("total_apps", 1), 1)
            )
            mlflow.log_metric(
                "runtime_success_rate", metrics.get("runtime_success", 0) / max(summary.get("total_apps", 1), 1)
            )
            mlflow.log_metric("local_runability_avg", metrics.get("local_runability_avg", 0))
            mlflow.log_metric("deployability_avg", metrics.get("deployability_avg", 0))

        if generation_results:
            gen_metrics = generation_results.get("generation_metrics", {})
            if gen_metrics.get("total_cost_usd"):
                mlflow.log_metric("generation_cost_usd", gen_metrics["total_cost_usd"])
            if gen_metrics.get("avg_turns"):
                mlflow.log_metric("avg_turns_per_app", gen_metrics["avg_turns"])

        with tempfile.NamedTemporaryFile(mode="w", suffix=".json", delete=False) as f:
            json.dump(evaluation_report, f, indent=2)
            mlflow.log_artifact(f.name, "reports")

        return run.info.run_id


def main(
    mlflow_experiment: str = "/Shared/apps-mcp-evaluations",
    parallelism: int = 4,
    evals_git_url: str = "https://github.com/neondatabase/appdotbuild-agent.git",
    apps_volume: Optional[str] = None,
) -> None:
    """
    Run Apps-MCP evaluations.

    Args:
        mlflow_experiment: MLflow experiment path
        parallelism: Number of parallel workers
        evals_git_url: Git URL for appdotbuild-agent eval framework
        apps_volume: UC Volume path containing generated apps (optional)
    """
    print("Starting Apps-MCP Evaluation")
    print(f"  MLflow Experiment: {mlflow_experiment}")
    print(f"  Evals Repo: {evals_git_url}")
    print(f"  Parallelism: {parallelism}")
    print(f"  Apps Volume: {apps_volume or 'not specified'}")
    print("=" * 60)

    setup_mlflow(mlflow_experiment)

    work_dir = Path(tempfile.mkdtemp(prefix="apps-mcp-evals-"))
    evals_dir = work_dir / "appdotbuild-agent"

    print(f"\nCloning evals repo to {evals_dir}...")
    clone_evals_repo(evals_git_url, evals_dir)

    print("\nInstalling dependencies...")
    install_klaudbiusz_deps(evals_dir)

    print("\n" + "=" * 60)
    print("RUNNING EVALUATION")
    print("=" * 60)

    klaudbiusz_dir = evals_dir / "klaudbiusz"

    if apps_volume:
        volume_path = Path(apps_volume)
        latest_file = volume_path / "latest.txt"
        if latest_file.exists():
            apps_dir = Path(latest_file.read_text().strip())
            print(f"Using apps from UC Volume (via latest.txt): {apps_dir}")
        elif volume_path.exists():
            subdirs = [d for d in volume_path.iterdir() if d.is_dir()]
            if subdirs:
                apps_dir = max(subdirs, key=lambda d: d.name)
                print(f"Using most recent apps dir: {apps_dir}")
            else:
                apps_dir = volume_path
        else:
            print(f"Warning: Apps volume not found at {apps_volume}")
            apps_dir = klaudbiusz_dir / "app"
    else:
        apps_dir = klaudbiusz_dir / "app"

    if not apps_dir.exists():
        print(f"Apps directory not found at {apps_dir}")
        print("Creating empty apps dir for sample run...")
        apps_dir.mkdir(parents=True, exist_ok=True)

    evaluation_report = run_evaluation(
        evals_dir=evals_dir,
        apps_dir=apps_dir,
        parallelism=parallelism,
    )

    if not evaluation_report:
        print("No apps found - creating sample report for infrastructure validation")
        evaluation_report = {
            "summary": {
                "total_apps": 0,
                "evaluated_at": datetime.utcnow().strftime("%Y-%m-%dT%H:%M:%SZ"),
                "metrics_summary": {
                    "avg_appeval_100": 0,
                    "build_success": 0,
                    "runtime_success": 0,
                    "local_runability_avg": 0,
                    "deployability_avg": 0,
                },
            },
            "apps": [],
        }

    print("\nLogging results to MLflow...")
    run_id = log_results_to_mlflow(evaluation_report)
    print(f"MLflow Run ID: {run_id}")

    summary = evaluation_report.get("summary", {})
    metrics = summary.get("metrics_summary", {})
    print("\n" + "=" * 60)
    print("EVALUATION SUMMARY")
    print("=" * 60)
    print(f"Total Apps: {summary.get('total_apps', 0)}")
    print(f"Avg AppEval Score: {metrics.get('avg_appeval_100', 0):.1f}/100")
    print(f"Build Success: {metrics.get('build_success', 0)}")
    print(f"Runtime Success: {metrics.get('runtime_success', 0)}")
    print(f"Local Runability: {metrics.get('local_runability_avg', 0):.1f}/5")
    print(f"Deployability: {metrics.get('deployability_avg', 0):.1f}/5")

    print("\nEvaluation complete!")


if __name__ == "__main__":
    fire.Fire(main)
