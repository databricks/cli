import subprocess
import re
import sys
from pathlib import Path


ALIASES = {
    "python": "pythonmutator",
}


def run_git_grep():
    cmd = ["git", "grep", "^func [A-Z].*Mutator {", "*.go", "(:exclude)*_test.go"]
    result = subprocess.run(cmd, stdout=subprocess.PIPE, encoding="utf-8", check=True)
    return result.stdout


def create_mutator_map(git_grep_output):
    mutator_map = {}

    # Skip the first line which is the command itself
    lines = git_grep_output.strip().split("\n")
    # if lines[0].startswith("~/"):
    #    lines = lines[1:]

    for line in lines:
        parts = line.split(":", 1)
        if len(parts) != 2:
            print(f"Cannot parse: {line!r}", file=sys.stderr)
            continue

        file_path, func_decl = parts

        match = re.match(r"func (\w+)\(.*\) bundle\.(ReadOnly)?Mutator {", func_decl)
        if not match:
            print(f"Cannot match: {line!r}", file=sys.stderr)
            continue

        func_name = match.group(1)
        path_parts = file_path.split("/")
        package_name = path_parts[-2]
        qualified_name = f"{package_name}.{func_name}"
        mutator_map[qualified_name] = file_path

    return mutator_map


def extract_mutator_calls(initialize_file, mutator_map):
    mutator_calls = {}

    with open(initialize_file, "r") as f:
        lines = f.readlines()

    for i, line in enumerate(lines):
        line_stripped = line.strip()
        if not line_stripped:
            continue

        if line.startswith("//"):
            continue

        matches_per_line = []

        for qualified_name in mutator_map:
            package_name, func_name = qualified_name.split(".")
            package_name = ALIASES.get(package_name, package_name)
            pattern = r"\b" + re.escape(qualified_name) + r"\("

            if re.search(pattern, line_stripped):
                mutator_calls.setdefault(qualified_name, []).append(i)
                matches_per_line.append(qualified_name)

        if len(matches_per_line) > 1:
            print("Warning multiple matches in {line!r}\n{matches_per_line}", file=sys.stderr)

    return mutator_calls


def run_aider(initialize_file, doc_file, mutator_file, mutator_name):
    cmd = [
        "aider",
        "--no-show-release-notes",
        "--no-check-update",
        "--map-tokens",
        "0",
        initialize_file,
        doc_file,
        mutator_file,
        "--message",
        f"Update comments for {mutator_name} in initialize.go according to the documentation in mutator_documentation.md. Only update the comments for this specific mutator call, don't change anything else.",
    ]

    cmd_str = " ".join(cmd)
    print(f"+ {cmd_str}", file=sys.stderr)
    subprocess.run(cmd)


def main():
    # Path to initialize.go
    initialize_file = "bundle/phases/initialize.go"

    # Check if the file exists
    if not Path(initialize_file).exists():
        print(f"Error: {initialize_file} does not exist.")
        print("Make sure you're running this script from the root of the repository.")
        return

    # Path to mutator_documentation.md
    doc_file = "bundle/phases/mutator_documentation.md"

    # Get mutator map
    git_grep_output = run_git_grep()
    mutator_map = create_mutator_map(git_grep_output)

    print(f"Found {len(mutator_map)} potential mutators in the codebase")
    import pprint

    pprint.pprint(mutator_map)

    mutator_calls = list(extract_mutator_calls(initialize_file, mutator_map).keys())

    print(f"Found {len(mutator_calls)} mutator calls in {initialize_file}")
    assert mutator_calls

    for qualified_name in mutator_calls:
        mutator_calls_with_lines = extract_mutator_calls(initialize_file, mutator_map)
        line_idx = mutator_calls_with_lines.get(qualified_name)
        if not line_idx:
            continue
        mutator_file = mutator_map.get(qualified_name)

        if not mutator_file:
            print(f"Could not find source file for {qualified_name}")
            continue

        with open(initialize_file, "r") as f:
            lines = f.readlines()

        context_lines = set()

        for idx in line_idx:
            for x in range(idx - 8, idx + 4):
                if x < 0 or x >= len(lines):
                    continue
                context_lines.add(x)

        print("\nContext in initialize.go:")
        print("-------------------------")
        for i in sorted(context_lines):
            prefix = ">" if i in line_idx else " "
            print(f"{prefix} {i+1:4d}: {lines[i].rstrip()}")
        print("-------------------------")

        response = input(f"Process {qualified_name} from {mutator_file}? (y/n): ")

        if response.lower() == "y":
            run_aider(initialize_file, doc_file, mutator_file, qualified_name)


if __name__ == "__main__":
    main()
