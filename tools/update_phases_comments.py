import subprocess
import re
import os
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
    
    # Find the ApplySeq block
    apply_seq_match = re.search(r'bundle\.ApplySeq\(ctx, b,(.*?)\)', content, re.DOTALL)
    if not apply_seq_match:
        print("Could not find ApplySeq block in initialize.go")
        return []
    
    apply_seq_block = apply_seq_match.group(1)
    
    # Extract mutator calls
    mutator_calls = []
    
    # Pattern to match mutator calls like mutator.SomeFunction(), or pythonmutator.PythonMutator(...)
    pattern = r'(\w+)\.(\w+)\(.*?\)'
    
    for match in re.finditer(pattern, apply_seq_block):
        package_name = match.group(1)
        func_name = match.group(2)
        
        # Skip non-mutator functions like bundle.ApplySeq
        if func_name == "ApplySeq":
            continue
            
        # Handle special cases
        if package_name == "pythonmutator" and func_name == "PythonMutator":
            # Extract the phase parameter
            phase_match = re.search(r'PythonMutator\(pythonmutator\.(\w+)\)', match.group(0))
            if phase_match:
                phase = phase_match.group(1)
                mutator_calls.append(f"{package_name}.{func_name}({phase})")
            else:
                mutator_calls.append(f"{package_name}.{func_name}")
        else:
            mutator_calls.append(f"{package_name}.{func_name}")
    
    return mutator_calls

def run_aider(initialize_file, doc_file, mutator_file, mutator_name):
    """
    Run aider to update comments for a mutator
    
    Args:
        initialize_file (str): Path to initialize.go
        doc_file (str): Path to mutator_documentation.md
        mutator_file (str): Path to the mutator source file
        mutator_name (str): Name of the mutator
    """
    cmd = [
        "aider", "-m",
        initialize_file, doc_file, mutator_file,
        "--message", f"Update comments for {mutator_name} in initialize.go according to the documentation in mutator_documentation.md. Only update the comments for this specific mutator call, don't change anything else."
    ]
    
    print(f"Running: {' '.join(cmd)}")
    subprocess.run(cmd)

def main():
    # Path to initialize.go
    initialize_file = "bundle/phases/initialize.go"
    
    # Path to mutator_documentation.md
    doc_file = "bundle/phases/mutator_documentation.md"
    
    # Get mutator map
    git_grep_output = run_git_grep()
    mutator_map = create_mutator_map(git_grep_output)
    
    # Extract mutator calls from initialize.go
    mutator_calls = extract_mutator_calls(initialize_file)
    
    print(f"Found {len(mutator_calls)} mutator calls in {initialize_file}")
    
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
        response = input(f"Process {qualified_name} from {mutator_file}? (y/n): ")
        
        if response.lower() == 'y':
            run_aider(initialize_file, doc_file, mutator_file, qualified_name)

if __name__ == "__main__":
    main()
