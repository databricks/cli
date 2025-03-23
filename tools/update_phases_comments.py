import subprocess
import re
import os
import sys
from pathlib import Path


def run_git_grep():
    cmd = ["git", "grep", "^func [A-Z].*Mutator {", "*.go", "(:exclude)*_test.go"]
    result = subprocess.run(cmd, stdout=subprocess.PIPE, encoding="utf-8", check=True)
    return result.stdout


def create_mutator_map(git_grep_output):
    mutator_map = {}

    # Skip the first line which is the command itself
    lines = git_grep_output.strip().split("\n")
    #if lines[0].startswith("~/"):
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
    """
    Extract mutator calls from initialize.go with their line numbers
    
    Args:
        initialize_file (str): Path to initialize.go
        mutator_map (dict): Map of qualified mutator names to file paths
        
    Returns:
        list: List of tuples (mutator_call, line_number)
    """
    mutator_calls = []
    
    with open(initialize_file, "r") as f:
        lines = f.readlines()
    
    for i, line in enumerate(lines):
        line_stripped = line.strip()
        if not line_stripped:
            continue
            
        # Check for matches against mutators in the map
        for qualified_name in mutator_map:
            package_name, func_name = qualified_name.split(".")
            pattern = rf"\b{package_name}\.{func_name}\b"
            
            if re.search(pattern, line_stripped):
                # Handle special case for PythonMutator
                if package_name == "pythonmutator" and func_name == "PythonMutator":
                    phase_match = re.search(r"PythonMutator\(pythonmutator\.(\w+)\)", line_stripped)
                    if phase_match:
                        phase = phase_match.group(1)
                        mutator_calls.append((f"{package_name}.{func_name}({phase})", i))
                    else:
                        mutator_calls.append((f"{package_name}.{func_name}", i))
                else:
                    mutator_calls.append((f"{package_name}.{func_name}", i))
                break  # Use first match only
    
    print(f"Debug: Found these mutator calls: {[call for call, _ in mutator_calls]}")
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


def debug_apply_seq_block(initialize_file):
    """Debug function to print the ApplySeq block content"""
    with open(initialize_file, "r") as f:
        content = f.read()

    apply_seq_match = re.search(r"bundle\.ApplySeq\(ctx, b,(.*?)\)", content, re.DOTALL)
    if not apply_seq_match:
        print("Could not find ApplySeq block in initialize.go")
        return

    apply_seq_block = apply_seq_match.group(1)
    print("\nApplySeq block content:")
    print("----------------------")
    print(apply_seq_block)
    print("----------------------")


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

    # Extract mutator calls from initialize.go with line numbers
    mutator_calls_with_lines = extract_mutator_calls(initialize_file, mutator_map)

    print(f"Found {len(mutator_calls_with_lines)} mutator calls in {initialize_file}")

    # If no mutator calls were found, run debug function
    if not mutator_calls_with_lines:
        debug_apply_seq_block(initialize_file)
        print("\nNo mutator calls were found. This could be because:")
        print("1. The initialize.go file doesn't contain any mutator calls")
        print("2. The pattern used to detect mutator calls doesn't match the format in the file")
        print("\nTry running this command to see the content of initialize.go:")
        print(f"cat {initialize_file}")
        return

    # Process each mutator call
    for mutator_call, line_idx in mutator_calls_with_lines:
        # Extract package and function name
        parts = mutator_call.split("(")[0].split(".")
        package_name = parts[0]
        func_name = parts[1]

        # Construct qualified name
        qualified_name = f"{package_name}.{func_name}"

        # Find the mutator file
        mutator_file = mutator_map.get(qualified_name)

        if not mutator_file:
            print(f"Could not find source file for {qualified_name}")
            continue

        # Use the line number we already found
        with open(initialize_file, "r") as f:
            lines = f.readlines()

        # Get context: 8 lines before and 3 lines after
        start_idx = max(0, line_idx - 8)
        end_idx = min(len(lines), line_idx + 4)

        print("\nContext in initialize.go:")
        print("-------------------------")
        for i in range(start_idx, end_idx):
            prefix = ">" if i == line_idx else " "
            print(f"{prefix} {i+1:4d}: {lines[i].rstrip()}")
        print("-------------------------\n")

        response = input(f"Process {qualified_name} from {mutator_file}? (y/n): ")

        if response.lower() == "y":
            run_aider(initialize_file, doc_file, mutator_file, qualified_name)


if __name__ == "__main__":
    main()
