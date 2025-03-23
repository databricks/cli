import subprocess
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
        import re
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

# Main execution
if __name__ == "__main__":
    git_grep_output = run_git_grep()
    mutator_map = create_mutator_map(git_grep_output)
    for key, value in mutator_map.items():
        print(f'"{key}" -> {value}')
