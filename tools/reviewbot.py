#!/usr/bin/env python3
from __future__ import annotations

import json
import subprocess
import sys
from pathlib import Path


class ReviewBot:
    def __init__(self):
        self.prompts_dir = Path(__file__).parent.parent / "prompts"
        self.review_prompt_file = self.prompts_dir / "review.md"

    def list_review_requests(self) -> list[dict]:
        """List PRs with open review requests for the current user."""
        result = subprocess.run(
            ["gh", "pr", "list", "--search", "review-requested:@me", "--json", "number,title,url,headRefName,body"],
            capture_output=True,
            text=True,
            check=True,
        )
        return json.loads(result.stdout)

    def get_current_branch(self) -> str:
        """Get the current git branch name."""
        result = subprocess.run(
            ["git", "rev-parse", "--abbrev-ref", "HEAD"],
            capture_output=True,
            text=True,
            check=True,
        )
        return result.stdout.strip()

    def checkout_branch(self, branch: str) -> None:
        """Checkout a git branch."""
        subprocess.run(["git", "checkout", branch], check=True)

    def checkout_pr(self, pr_number: int) -> None:
        """Checkout a PR locally."""
        subprocess.run(["gh", "pr", "checkout", str(pr_number)], check=True)

    def get_pr_diff(self, pr_number: int) -> str:
        """Get the diff for a PR."""
        result = subprocess.run(
            ["gh", "pr", "diff", str(pr_number)],
            capture_output=True,
            text=True,
            check=True,
        )
        return result.stdout

    def read_review_prompt(self) -> str:
        """Read the review prompt template."""
        if not self.review_prompt_file.exists():
            raise FileNotFoundError(f"Review prompt not found at {self.review_prompt_file}")
        return self.review_prompt_file.read_text()

    def review_with_claude(self, pr_info: dict) -> None:
        """Run Claude CLI to review the PR."""
        review_guidelines = self.read_review_prompt()
        diff = self.get_pr_diff(pr_info["number"])

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

Please review this PR according to the guidelines. You can also explore the codebase
to understand context better. Provide your review in the format specified in the guidelines.
"""
        subprocess.run(["claude", "-p", prompt, "--allowedTools", "Read,Glob,Grep,Task"], check=True)

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
            # Show list and prompt for selection
            print(f"Found {len(prs)} PR(s) with review requests:\n")
            for i, pr in enumerate(prs, 1):
                print(f"  {i}. PR #{pr['number']}: {pr['title']}")
                print(f"     {pr['url']}\n")

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
        print(f"{'=' * 80}")
        if pr.get("body"):
            print(f"\n{pr['body']}\n")
        print()

        original_branch = self.get_current_branch()
        self.checkout_pr(pr["number"])
        print(f"Checked out PR #{pr['number']}\n")

        try:
            self.review_with_claude(pr)
            print(f"\nCompleted review of PR #{pr['number']}")
        finally:
            self.checkout_branch(original_branch)
            print(f"Restored original branch: {original_branch}")


def main():
    bot = ReviewBot()

    # Optional: specific PR number as argument
    pr_number = int(sys.argv[1]) if len(sys.argv) > 1 else None
    bot.run(pr_number)


if __name__ == "__main__":
    main()
