import subprocess
import re
import os
import sys
from pathlib import Path

def run_git_grep():
    """
    Run git grep command to find mutator functions and return the output.
    
    Returns:
        str: Output from git grep command
    """
    cmd = ["git", "grep", "^func [A-Z].*Mutator {", "*.go", "(:exclude)*_test.go"]
    result = subprocess.run(cmd, capture_output=True, text=True, check=True)
    return result.stdout

def create_mutator_map(git_grep_output):
    """
    Create a mapping from mutator name to file path based on git grep output.
    
    Args:
        git_grep_output (str): Output from git grep command
        
    Returns:
        dict: Mapping from qualified mutator name to file path
    """
    mutator_map = {}
    
    # Skip the first line which is the command itself
    lines = git_grep_output.strip().split('\n')
    if lines[0].startswith('~/'):
        lines = lines[1:]
    
    for line in lines:
        # Split the line into file path and function declaration
        parts = line.split(':', 1)
        if len(parts) != 2:
            continue
            
        file_path, func_decl = parts
        
        # Extract the function name from the declaration
        match = re.match(r'func (\w+)\(.*\) bundle\.Mutator {', func_decl)
        if not match:
            continue
            
        func_name = match.group(1)
        
        # Determine the package name based on directory structure
        path_parts = file_path.split('/')
        
        # Handle different directory structures
        if len(path_parts) == 1:
            # File in root directory
            package_name = path_parts[0].split('.')[0]  # Remove file extension
        elif "config/loader" in file_path:
            package_name = "loader"
        elif "config/mutator" in file_path:
            package_name = "mutator"
        else:
            # Use the first directory as the package name
            package_name = path_parts[0]
        
        # Construct the fully qualified name
        qualified_name = f"{package_name}.{func_name}"
        
        # Add to the map
        mutator_map[qualified_name] = file_path
    
    return mutator_map

def extract_mutator_calls(initialize_file):
    """
    Extract mutator calls from initialize.go
    
    Args:
        initialize_file (str): Path to initialize.go
        
    Returns:
        list: List of mutator calls
    """
    with open(initialize_file, 'r') as f:
        content = f.read()
    
    # Extract mutator calls directly from the file
    mutator_calls = []
    
    # Get all lines in the file
    lines = content.split('\n')
    
    # Find all lines with mutator calls
    for i, line in enumerate(lines):
        line = line.strip()
        if not line:
            continue
            
        # Look for package.FunctionName pattern
        match = re.search(r'(\w+)\.([A-Z]\w+)\(', line)
        if match:
            package_name = match.group(1)
            func_name = match.group(2)
            
            # Skip non-mutator functions and common package prefixes
            if package_name == "bundle" and func_name in ["ApplySeq", "Apply"]:
                continue
            
            # Skip validation functions
            if package_name == "validate" and not func_name.endswith("Mutator"):
                continue
                
            # Handle special cases
            if package_name == "pythonmutator" and func_name == "PythonMutator":
                # Extract the phase parameter if present
                phase_match = re.search(r'PythonMutator\(pythonmutator\.(\w+)\)', line)
                if phase_match:
                    phase = phase_match.group(1)
                    mutator_calls.append(f"{package_name}.{func_name}({phase})")
                else:
                    mutator_calls.append(f"{package_name}.{func_name}")
            else:
                mutator_calls.append(f"{package_name}.{func_name}")
    
    print(f"Debug: Found these mutator calls: {mutator_calls}")
    return mutator_calls

def run_aider(initialize_file, doc_file, mutator_file, mutator_name):
    cmd = [
        "aider", "--no-show-release-notes", "--no-check-update", "--no-repo-map",
        initialize_file, doc_file, mutator_file,
        "--message", f"Update comments for {mutator_name} in initialize.go according to the documentation in mutator_documentation.md. Only update the comments for this specific mutator call, don't change anything else."
    ]
    
    cmd_str = ' '.join(cmd)
    print(f"+ {cmd_str}", file=sys.stderr)
    subprocess.run(cmd)

def debug_apply_seq_block(initialize_file):
    """Debug function to print the ApplySeq block content"""
    with open(initialize_file, 'r') as f:
        content = f.read()
    
    apply_seq_match = re.search(r'bundle\.ApplySeq\(ctx, b,(.*?)\)', content, re.DOTALL)
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
    if not os.path.exists(initialize_file):
        print(f"Error: {initialize_file} does not exist.")
        print("Make sure you're running this script from the root of the repository.")
        return
    
    # Path to mutator_documentation.md
    doc_file = "bundle/phases/mutator_documentation.md"
    
    # Get mutator map
    git_grep_output = run_git_grep()
    mutator_map = create_mutator_map(git_grep_output)
    
    print(f"Found {len(mutator_map)} potential mutators in the codebase")
    
    # Extract mutator calls from initialize.go
    mutator_calls = extract_mutator_calls(initialize_file)
    
    print(f"Found {len(mutator_calls)} mutator calls in {initialize_file}")
    
    # If no mutator calls were found, run debug function
    if not mutator_calls:
        debug_apply_seq_block(initialize_file)
        print("\nNo mutator calls were found. This could be because:")
        print("1. The initialize.go file doesn't contain any mutator calls")
        print("2. The pattern used to detect mutator calls doesn't match the format in the file")
        print("\nTry running this command to see the content of initialize.go:")
        print(f"cat {initialize_file}")
        return
    
    # Process each mutator call
    for mutator_call in mutator_calls:
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
        
        # Ask user if they want to process this mutator
        # Print relevant source code context
        with open(initialize_file, 'r') as f:
            lines = f.readlines()
        
        # Find the line with the mutator call
        mutator_line_idx = None
        for i, line in enumerate(lines):
            if qualified_name in line:
                mutator_line_idx = i
                break
        
        if mutator_line_idx is not None:
            # Get context: 8 lines before and 3 lines after
            start_idx = max(0, mutator_line_idx - 8)
            end_idx = min(len(lines), mutator_line_idx + 4)
            
            print("\nContext in initialize.go:")
            print("-------------------------")
            for i in range(start_idx, end_idx):
                prefix = ">" if i == mutator_line_idx else " "
                print(f"{prefix} {i+1:4d}: {lines[i].rstrip()}")
            print("-------------------------\n")
        
        response = input(f"Process {qualified_name} from {mutator_file}? (y/n): ")
        
        if response.lower() == 'y':
            run_aider(initialize_file, doc_file, mutator_file, qualified_name)

if __name__ == "__main__":
    main()
