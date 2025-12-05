#!/usr/bin/env python3
"""
Apps-MCP Evaluation Runner for Databricks Jobs.

Orchestrates the klaudbiusz evaluation framework to run as a scheduled Databricks job.
Results are logged to MLflow for tracking and comparison.
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
    """Clone or update appdotbuild-agent repository."""
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


def run_generation(
    klaudbiusz_dir: Path,
    output_dir: Path,
    mcp_binary: str,
    backend: str = "claude",
    model: Optional[str] = None,
    prompt_set: str = "databricks",
) -> dict:
    """Run app generation using klaudbiusz bulk_run."""
    cmd = [
        sys.executable,
        "-m",
        "cli.generation.bulk_run",
        "--mcp_binary",
        mcp_binary,
        "--output_dir",
        str(output_dir),
        "--prompts",
        prompt_set,
        "--backend",
        backend,
    ]
    if model:
        cmd.extend(["--model", model])

    env = os.environ.copy()
    env["PYTHONPATH"] = str(klaudbiusz_dir)

    result = subprocess.run(cmd, cwd=klaudbiusz_dir, env=env, capture_output=True, text=True)

    if result.returncode != 0:
        print(f"Generation failed: {result.stderr}")
        raise RuntimeError(f"Generation failed with code {result.returncode}")

    results_files = sorted(output_dir.glob("bulk_run_results_*.json"), reverse=True)
    if results_files:
        return json.loads(results_files[0].read_text())
    return {}


def run_evaluation(
    klaudbiusz_dir: Path,
    apps_dir: Path,
    parallelism: int = 4,
    fast_mode: bool = False,
) -> dict:
    """Run evaluation on generated apps."""
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

    result = subprocess.run(cmd, cwd=klaudbiusz_dir, env=env, capture_output=True, text=True)

    if result.returncode != 0:
        print(f"Evaluation output: {result.stdout}")
        print(f"Evaluation errors: {result.stderr}")

    eval_dir = klaudbiusz_dir / "cli" / "app-eval"
    report_file = eval_dir / "evaluation_report.json"
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
    catalog: str = "main",
    schema: str = "evals",
    mlflow_experiment: str = "/Shared/apps-mcp-evaluations",
    mode: str = "eval_only",
    parallelism: int = 4,
    evals_git_url: str = "https://github.com/neondatabase/appdotbuild-agent.git",
    mcp_binary: Optional[str] = None,
    fast: bool = False,
) -> None:
    """
    Run Apps-MCP evaluations.

    Args:
        catalog: Unity Catalog name
        schema: Schema for results
        mlflow_experiment: MLflow experiment path
        mode: "full" (generate + eval), "eval_only" (eval existing apps), "quick" (subset)
        parallelism: Number of parallel workers
        evals_git_url: Git URL for appdotbuild-agent eval framework
        mcp_binary: Path to MCP binary (required for full mode)
        fast: Skip slow LLM checks
    """
    print("Starting Apps-MCP Evaluation")
    print(f"  Mode: {mode}")
    print(f"  MLflow Experiment: {mlflow_experiment}")
    print(f"  Evals Repo: {evals_git_url}")
    print(f"  Parallelism: {parallelism}")
    print("=" * 60)

    setup_mlflow(mlflow_experiment)

    work_dir = Path(tempfile.mkdtemp(prefix="apps-mcp-evals-"))
    evals_dir = work_dir / "appdotbuild-agent"
    apps_dir = work_dir / "apps"
    apps_dir.mkdir(exist_ok=True)

    print(f"\nCloning evals repo to {evals_dir}...")
    clone_evals_repo(evals_git_url, evals_dir)

    generation_results = None
    if mode == "full":
        if not mcp_binary:
            raise ValueError("--mcp_binary required for full mode")
        print("\nRunning app generation...")
        generation_results = run_generation(
            klaudbiusz_dir=evals_dir,
            output_dir=apps_dir,
            mcp_binary=mcp_binary,
        )

    print("\nRunning evaluation...")
    eval_apps_dir = apps_dir if mode == "full" else evals_dir / "app"
    evaluation_report = run_evaluation(
        klaudbiusz_dir=evals_dir,
        apps_dir=eval_apps_dir,
        parallelism=parallelism,
        fast_mode=fast or mode == "quick",
    )

    if evaluation_report:
        print("\nLogging results to MLflow...")
        run_id = log_results_to_mlflow(evaluation_report, generation_results)
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
    else:
        print("No evaluation results generated")
        sys.exit(1)

    print("\nEvaluation complete!")


if __name__ == "__main__":
    fire.Fire(main)
