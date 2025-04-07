#!/usr/bin/env python3
"""
Script to find open PRs by non-team members that are approved by team members,
and start integration tests for them if not already running.
"""

import json
import subprocess
import sys
from pathlib import Path
#import requests
import time
import re

def get_team_members():
    codeowners_path = Path(".github/CODEOWNERS")
    with open(codeowners_path, "r") as f:
        first_line = f.readline().strip()
    
    # Extract GitHub usernames from the first line
    team_members = re.findall(r'@([a-zA-Z0-9-]+)', first_line)
    return team_members

def get_approved_prs_by_non_team():
    team_members = get_team_members()
    
    # Get open PRs
    result = subprocess.run(
        ["gh", "pr", "list", "--json", "number,author,reviews,headRefOid"],
        capture_output=True, text=True
    )
    
    prs = json.loads(result.stdout)
    approved_prs = []
    
    for pr in prs:
        # Skip PRs by team members
        if pr["author"]["login"] in team_members:
            continue
        
        # Check if approved by a team member
        is_approved = False
        for review in pr.get("reviews", []):
            if review["state"] == "APPROVED" and review["author"]["login"] in team_members:
                is_approved = True
                break
        
        if is_approved:
            approved_prs.append({
                "number": pr["number"],
                "commit": pr["headRefOid"],
                "author": pr["author"]["login"]
            })
    
    return approved_prs

def check_if_job_running(pr_number, commit_sha):
    # Check if job is already running for this PR and commit
    url = f"https://github.com/databricks-eng/eng-dev-ecosystem/actions/workflows/cli-isolated-pr.yml"
    
    result = subprocess.run(
        ["gh", "api", f"/repos/databricks-eng/eng-dev-ecosystem/actions/workflows/cli-isolated-pr.yml/runs?status=in_progress"],
        capture_output=True, text=True
    )
    
    runs = json.loads(result.stdout)
    
    for run in runs.get("workflow_runs", []):
        if f"refs/pull/{pr_number}/head" == run["head_branch"] and commit_sha == run["head_sha"]:
            return True
    
    return False

def start_job(pr_number, commit_sha, author):
    # Get PR details including title
    result = subprocess.run(
        ["gh", "pr", "view", str(pr_number), "--json", "title,url"],
        capture_output=True, text=True
    )
    pr_details = json.loads(result.stdout)
    
    pr_title = pr_details.get("title", "")
    pr_url = pr_details.get("url", "")
    
    print(f"PR #{pr_number}: \"{pr_title}\" by {author} (commit {commit_sha[:7]})")
    print(f"URL: {pr_url}")
    
    # Get approver information
    result = subprocess.run(
        ["gh", "pr", "view", str(pr_number), "--json", "reviews"],
        capture_output=True, text=True
    )
    reviews = json.loads(result.stdout).get("reviews", [])
    approvers = [review["author"]["login"] for review in reviews if review["state"] == "APPROVED"]
    
    print(f"This PR is approved by {', '.join(approvers)} but has no running tests.")
    response = input("Start integration tests? (y/n): ")
    
    if response.lower() == "y":
        # Use the commit SHA directly instead of the PR ref
        result = subprocess.run([
            "gh", "workflow", "run", "cli-isolated-pr.yml",
            "-R", "databricks-eng/eng-dev-ecosystem",
            "-f", f"pr_number={pr_number}",
            "-f", f"sha={commit_sha}"
        ], capture_output=True, text=True)
        
        if result.returncode != 0:
            print(f"Error starting workflow: {result.stderr}")
            return
        print(f"Started integration tests for PR #{pr_number}")
    else:
        print("Skipped starting tests")

def main():
    approved_prs = get_approved_prs_by_non_team()
    
    if not approved_prs:
        print("No approved PRs from non-team members found.")
        return
    
    for pr in approved_prs:
        if not check_if_job_running(pr["number"], pr["commit"]):
            start_job(pr["number"], pr["commit"], pr["author"])
        else:
            print(f"Tests already running for PR #{pr['number']} (commit {pr['commit'][:7]})")

if __name__ == "__main__":
    main()
