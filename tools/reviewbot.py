#!/usr/bin/env python3
from __future__ import annotations

import json
import shutil
import subprocess
import sys
from pathlib import Path

REVIEW_OUTPUT_FILE = "/tmp/reviewbot_output.json"

PUBLISH_PROMPT = """
IMPORTANT: After completing your review, you MUST call the publish_review tool to save your review.
The review will be published to GitHub after user confirmation.

The publish_review tool takes a single JSON argument with:
- event: One of "APPROVE", "REQUEST_CHANGES", or "COMMENT"
- body: Overall review summary (will be prefixed with a reviewbot notice)
- comments: Array of inline comments, each with:
  - path: File path relative to repo root (MUST be a file in the PR diff)
  - line: Line number within a diff hunk (MUST be within the @@ range shown in the diff)
  - body: Comment text

CRITICAL: Inline comments can ONLY be placed on:
1. Files that appear in the PR diff
2. Line numbers within the diff hunks (the @@ -X,Y +A,B @@ ranges)

If you want to comment on code NOT in the diff (e.g., suggesting changes to other files),
include those comments in the main "body" field instead.

Example:
```json
{
  "event": "REQUEST_CHANGES",
  "body": "Good progress but a few issues need addressing.\n\n**Additional suggestion:** Consider also updating cmd/other.go to handle this case.",
  "comments": [
    {"path": "cmd/root.go", "line": 15, "body": "Consider adding error handling here."}
  ]
}
```

Guidelines:
- Inline comments: ONLY for lines visible in the diff
- Body: General observations AND suggestions for files/lines not in the diff
- Be specific and actionable
"""


class ReviewBot:
    def __init__(self):
        self.repo_root = Path(
            subprocess.run(
                ["git", "rev-parse", "--show-toplevel"],
                capture_output=True,
                text=True,
                check=True,
            ).stdout.strip()
        )
        self.prompts_dir = Path(__file__).parent.parent / "prompts"
        self.review_prompt_file = self.prompts_dir / "review.md"
        self.tool_script = Path(__file__).parent / "publish_review.py"
        self.worktrees_dir = self.repo_root / ".worktrees"
        self.reviews_dir = self.repo_root / ".reviews"

    def list_review_requests(self) -> list[dict]:
        """List PRs with open review requests for the current user."""
        result = subprocess.run(
            [
                "gh",
                "pr",
                "list",
                "--search",
                "review-requested:@me draft:false",
                "--json",
                "number,title,url,headRefName,body,author,createdAt",
            ],
            capture_output=True,
            text=True,
            check=True,
        )
        return json.loads(result.stdout)

    def get_pr_diff(self, pr_number: int) -> str:
        """Get the diff for a PR."""
        result = subprocess.run(
            ["gh", "pr", "diff", str(pr_number)],
            capture_output=True,
            text=True,
            check=True,
        )
        return result.stdout

    def parse_diff_ranges(self, diff: str) -> dict[str, list[tuple[int, int]]]:
        """Parse diff to get valid line ranges for each file.

        Returns a dict mapping file paths to lists of (start, end) line ranges
        where inline comments can be placed.
        """
        import re

        valid_ranges: dict[str, list[tuple[int, int]]] = {}
        current_file = None

        for line in diff.split("\n"):
            # Match file header: +++ b/path/to/file
            if line.startswith("+++ b/"):
                current_file = line[6:]
                valid_ranges[current_file] = []
            # Match hunk header: @@ -X,Y +A,B @@
            elif line.startswith("@@") and current_file:
                match = re.search(r"\+(\d+)(?:,(\d+))?", line)
                if match:
                    start = int(match.group(1))
                    count = int(match.group(2)) if match.group(2) else 1
                    valid_ranges[current_file].append((start, start + count - 1))

        return valid_ranges

    def is_valid_comment_location(self, path: str, line: int, valid_ranges: dict[str, list[tuple[int, int]]]) -> bool:
        """Check if a comment can be placed at the given path and line."""
        if path not in valid_ranges:
            return False
        for start, end in valid_ranges[path]:
            if start <= line <= end:
                return True
        return False

    def get_head_sha(self, pr_number: int) -> str:
        """Get the HEAD commit SHA of the PR."""
        result = subprocess.run(
            ["gh", "pr", "view", str(pr_number), "--json", "headRefOid", "-q", ".headRefOid"],
            capture_output=True,
            text=True,
            check=True,
        )
        return result.stdout.strip()

    def create_worktree(self, pr_number: int, branch: str) -> Path:
        """Create a git worktree for the PR."""
        self.worktrees_dir.mkdir(exist_ok=True)
        worktree_path = self.worktrees_dir / f"pr-{pr_number}"

        # Remove existing worktree if present
        if worktree_path.exists():
            self.remove_worktree(worktree_path)

        # Fetch the PR branch
        subprocess.run(
            ["gh", "pr", "checkout", str(pr_number), "--detach"],
            capture_output=True,
            check=False,
        )
        # Get back to original state - we just wanted to fetch
        subprocess.run(["git", "checkout", "-"], capture_output=True, check=False)

        # Create worktree with the PR branch
        subprocess.run(
            ["git", "worktree", "add", str(worktree_path), branch],
            check=True,
        )
        return worktree_path

    def remove_worktree(self, worktree_path: Path) -> None:
        """Remove a git worktree."""
        subprocess.run(
            ["git", "worktree", "remove", str(worktree_path), "--force"],
            capture_output=True,
            check=False,
        )
        # Clean up if still exists
        if worktree_path.exists():
            shutil.rmtree(worktree_path)

    def read_review_prompt(self) -> str:
        """Read the review prompt template."""
        if not self.review_prompt_file.exists():
            raise FileNotFoundError(f"Review prompt not found at {self.review_prompt_file}")
        return self.review_prompt_file.read_text()

    def review_with_claude(self, pr_info: dict, worktree_path: Path) -> dict | None:
        """Run Claude CLI to review the PR and return structured output."""
        review_guidelines = self.read_review_prompt()
        diff = self.get_pr_diff(pr_info["number"])

        # Clean up any previous output
        output_path = Path(REVIEW_OUTPUT_FILE)
        if output_path.exists():
            output_path.unlink()

        prompt = f"""Review this Pull Request following the guidelines below.

# PR Information
- **PR #{pr_info["number"]}**: {pr_info["title"]}
- **URL**: {pr_info["url"]}
- **Branch**: {pr_info["headRefName"]}

## PR Description
{pr_info.get("body", "No description provided.")}

## Review Guidelines
{review_guidelines}

## PR Diff
```diff
{diff}
```

{PUBLISH_PROMPT}

Please review this PR according to the guidelines. Explore the codebase to understand context.
When done, call publish_review with your structured review.
"""
        subprocess.run(
            [
                "claude",
                "-p",
                prompt,
                "--allowedTools",
                f"Read,Glob,Grep,Task,Bash({self.tool_script}:*)",
            ],
            cwd=worktree_path,
            check=True,
        )

        # Read the output
        if output_path.exists():
            return json.loads(output_path.read_text())
        return None

    def display_review(self, review: dict, pr_info: dict) -> None:
        """Display the review for user confirmation."""
        print(f"\n{'=' * 80}")
        print("REVIEW TO BE PUBLISHED")
        print(f"{'=' * 80}\n")

        event_colors = {
            "APPROVE": "\033[32m",
            "REQUEST_CHANGES": "\033[31m",
            "COMMENT": "\033[33m",
        }
        reset = "\033[0m"
        event = review.get("event", "COMMENT")
        print(f"Verdict: {event_colors.get(event, '')}{event}{reset}")
        print(f"PR: #{pr_info['number']} - {pr_info['title']}\n")

        print("--- Overall Comment ---")
        print(review.get("body", "(no body)"))
        print()

        comments = review.get("comments", [])
        if comments:
            print(f"--- Inline Comments ({len(comments)}) ---")
            for i, c in enumerate(comments, 1):
                print(f"\n[{i}] {c.get('path', '?')}:{c.get('line', '?')}")
                print(f"    {c.get('body', '(no comment)')}")
        else:
            print("--- No inline comments ---")

        print(f"\n{'=' * 80}")

    def save_review_payload(self, payload: dict, pr_number: int) -> Path:
        """Save the review payload to .reviews/pr-<num>.json."""
        self.reviews_dir.mkdir(exist_ok=True)
        review_file = self.reviews_dir / f"pr-{pr_number}.json"
        review_file.write_text(json.dumps(payload, indent=2))
        return review_file

    def publish_to_github(self, review: dict, pr_info: dict) -> None:
        """Publish the review to GitHub."""
        head_sha = self.get_head_sha(pr_info["number"])
        diff = self.get_pr_diff(pr_info["number"])
        valid_ranges = self.parse_diff_ranges(diff)

        body = review.get("body", "")

        # Separate valid and invalid comments
        valid_comments = []
        invalid_comments = []

        for comment in review.get("comments", []):
            path = comment.get("path", "")
            line = comment.get("line", 0)
            if self.is_valid_comment_location(path, line, valid_ranges):
                valid_comments.append(comment)
            else:
                invalid_comments.append(comment)

        # Add invalid comments to the body
        if invalid_comments:
            body += "\n\n---\n**Additional comments** (on lines not in diff):\n"
            for c in invalid_comments:
                body += f"\n**{c.get('path', '?')}:{c.get('line', '?')}**\n{c.get('body', '')}\n"

        full_body = f"{body}\n\n---\n*Review generated by [reviewbot](https://github.com/databricks/cli/tree/main/tools/reviewbot.py)*"

        # Determine event - never use REQUEST_CHANGES, convert to COMMENT
        event = review.get("event", "COMMENT")
        if event == "REQUEST_CHANGES":
            event = "COMMENT"

        # Build the review payload
        payload = {
            "commit_id": head_sha,
            "body": full_body,
            "event": event,
            "comments": [],
        }

        for comment in valid_comments:
            payload["comments"].append(
                {
                    "path": comment["path"],
                    "line": comment["line"],
                    "body": comment["body"],
                }
            )

        # Save the payload for debugging
        review_file = self.save_review_payload(payload, pr_info["number"])
        print(f"Full review payload saved to: {review_file}")

        # Submit via gh api
        result = subprocess.run(
            [
                "gh",
                "api",
                "-X",
                "POST",
                f"/repos/databricks/cli/pulls/{pr_info['number']}/reviews",
                "--input",
                "-",
            ],
            input=json.dumps(payload),
            capture_output=True,
            text=True,
        )

        if result.returncode != 0:
            print(f"\nError publishing review!")
            print(f"GitHub API error: {result.stderr}")
            if result.stdout:
                print(f"Response: {result.stdout}")
            print(f"\nSee full payload at: {review_file}")
            raise RuntimeError("Failed to publish review")

        print(f"\nReview published to {pr_info['url']}")

    def run(self, pr_number: int | None = None) -> None:
        """Main entry point for the review bot."""
        prs = self.list_review_requests()

        if not prs:
            print("No PRs with review requests found.")
            return

        # If a specific PR number was provided, use it directly
        if pr_number:
            pr = next((p for p in prs if p["number"] == pr_number), None)
            if not pr:
                print(f"PR #{pr_number} does not have a review request for you.")
                return
        else:
            # Sort by creation date (newest first)
            prs.sort(key=lambda p: p.get("createdAt", ""), reverse=True)

            # Show list and prompt for selection
            print(f"Found {len(prs)} PR(s) with review requests:\n")
            for i, pr in enumerate(prs, 1):
                author = pr.get("author", {}).get("login", "unknown")
                created_at = pr.get("createdAt", "")[:10]
                print(f"  {i}. PR #{pr['number']}: {pr['title']}")
                print(f"     @{author} | {created_at} | {pr['url']}\n")

            while True:
                try:
                    choice = input("Select a PR to review (number): ").strip()
                    idx = int(choice) - 1
                    if 0 <= idx < len(prs):
                        pr = prs[idx]
                        break
                    print(f"Please enter a number between 1 and {len(prs)}")
                except ValueError:
                    print("Please enter a valid number")
                except (EOFError, KeyboardInterrupt):
                    print("\nCancelled.")
                    return

        print(f"\n{'=' * 80}")
        print(f"Reviewing PR #{pr['number']}: {pr['title']}")
        author = pr.get("author", {}).get("login", "unknown")
        created_at = pr.get("createdAt", "")[:10]  # Just the date part
        print(f"Author: @{author} | Created: {created_at}")
        print(f"{'=' * 80}")
        if pr.get("body"):
            print(f"\n{pr['body']}\n")
        print()

        # Create worktree for the PR
        print(f"Creating worktree for PR #{pr['number']}...")
        worktree_path = self.create_worktree(pr["number"], pr["headRefName"])
        print(f"Worktree created at {worktree_path}\n")

        try:
            review = self.review_with_claude(pr, worktree_path)

            if not review:
                print("\nNo review output generated.")
                return

            self.display_review(review, pr)

            # Save review for inspection
            self.reviews_dir.mkdir(exist_ok=True)
            review_file = self.reviews_dir / f"pr-{pr['number']}.json"
            review_file.write_text(json.dumps(review, indent=2))
            print(f"Full review saved to: {review_file}")

            # Ask for confirmation
            while True:
                try:
                    confirm = input("\nPublish this review to GitHub? [y/N]: ").strip().lower()
                    if confirm in ("y", "yes"):
                        self.publish_to_github(review, pr)
                        break
                    elif confirm in ("n", "no", ""):
                        print("Review not published.")
                        break
                    else:
                        print("Please enter 'y' or 'n'")
                except (EOFError, KeyboardInterrupt):
                    print("\nReview not published.")
                    break

            print(f"\nCompleted review of PR #{pr['number']}")
        finally:
            print(f"Cleaning up worktree...")
            self.remove_worktree(worktree_path)


def main():
    bot = ReviewBot()

    # Optional: specific PR number as argument
    pr_number = int(sys.argv[1]) if len(sys.argv) > 1 else None
    bot.run(pr_number)


if __name__ == "__main__":
    main()
