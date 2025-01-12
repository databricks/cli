import sys
import requests
import os
import re
from pathlib import Path

# GitHub API base URL
GITHUB_API_URL = "https://api.github.com"

def get_headers():
    """Prepare headers for GitHub API requests, including authorization if available."""
    headers = {
        "Accept": "application/vnd.github+json",
        "User-Agent": "GitHub-Actions-SHA-Pinner"
    }
    token = os.getenv("GITHUB_TOKEN")
    if token:
        headers["Authorization"] = f"token {token}"
    return headers

def fetch_latest_tag(repo, version_prefix):
    """
    Fetch the latest tag for a repository that starts with the given version prefix.

    Args:
        repo (str): Repository in 'owner/repo' format.
        version_prefix (str): Version prefix, e.g., 'v4'.

    Returns:
        tuple: (tag_name, commit_sha) or (None, None) if not found.
    """
    url = f"{GITHUB_API_URL}/repos/{repo}/tags?per_page=100"
    response = requests.get(url, headers=get_headers())
    if response.status_code != 200:
        print(f"Error fetching tags for {repo}: {response.status_code} {response.reason}")
        return None, None

    tags = response.json()
    # Filter tags that start with the version prefix
    filtered_tags = [tag for tag in tags if tag['name'].startswith(version_prefix)]
    if not filtered_tags:
        print(f"No tags found for {repo} with prefix {version_prefix}")
        return None, None

    # Assume the first tag is the latest
    latest_tag = filtered_tags[0]['name']
    commit_sha = filtered_tags[0]['commit']['sha']
    return latest_tag, commit_sha

def update_uses_lines(file_path, repo, version_prefix, sha, tag):
    """
    Update the 'uses:' lines in a workflow file for a specific repository and version prefix.

    Args:
        file_path (Path): Path to the workflow file.
        repo (str): Repository in 'owner/repo' format.
        version_prefix (str): Version prefix, e.g., 'v4'.
        sha (str): Commit SHA to pin to.
        tag (str): Tag name for commenting.
    """
    pattern = re.compile(rf"(uses:\s*{re.escape(repo)}@){re.escape(version_prefix)}(?:\.\d+)*")
    replacement = f"uses: {repo}@{sha} # {tag}"
    updated = False

    with file_path.open('r') as file:
        lines = file.readlines()

    new_lines = []
    for line in lines:
        new_line, count = pattern.subn(replacement, line)
        if count > 0:
            updated = True
            print(f"Updated in {file_path}: {line.strip()} -> {new_line.strip()}")
        new_lines.append(new_line)

    if updated:
        with file_path.open('w') as file:
            file.writelines(new_lines)
        print(f"File updated: {file_path}")
    else:
        print(f"No matching lines found in {file_path} for {repo}@{version_prefix}")

def main():
    """
    Main function to process command-line arguments and update workflow files.

    Usage:
        python pin_actions.py owner1/repo1 vX owner2/repo2 vY ...
    """
    args = sys.argv[1:]
    if len(args) % 2 != 0 or not args:
        print("Usage: python pin_actions.py owner1/repo1 vX owner2/repo2 vY ...")
        sys.exit(1)

    # Prepare list of (repo, version_prefix) tuples
    repos = []
    for i in range(0, len(args), 2):
        repo = args[i]
        version_prefix = args[i+1]
        repos.append((repo, version_prefix))

    # Directory containing workflow files
    workflows_dir = Path(".github/workflows")
    if not workflows_dir.exists():
        print(f"Directory not found: {workflows_dir}")
        sys.exit(1)

    # Gather all YAML workflow files
    workflow_files = list(workflows_dir.rglob("*.yml")) + list(workflows_dir.rglob("*.yaml"))
    if not workflow_files:
        print(f"No workflow files found in {workflows_dir}")
        sys.exit(1)

    # Fetch latest tags and SHAs
    tag_info = {}
    for repo, version_prefix in repos:
        print(f"\nProcessing {repo} with version prefix {version_prefix}...")
        tag, sha = fetch_latest_tag(repo, version_prefix)
        if tag and sha:
            tag_info[(repo, version_prefix)] = (tag, sha)
            print(f"Latest tag: {tag}, Commit SHA: {sha}")
        else:
            print(f"Skipping {repo} due to missing tag or SHA.")

    if not tag_info:
        print("No updates to apply.")
        sys.exit(0)

    # Update workflow files
    for file_path in workflow_files:
        print(f"\nScanning file: {file_path}")
        for (repo, version_prefix), (tag, sha) in tag_info.items():
            update_uses_lines(file_path, repo, version_prefix, sha, tag)

    print("\nAll updates completed.")

if __name__ == "__main__":
    main()

