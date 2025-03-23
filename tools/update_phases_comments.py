
// AI TODO: run this, capture output,. pass to create_mutator_map
// git grep '^func [A-Z].*Mutator {' '*.go' '(:exclude)*_test.go'

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

# Example usage:
if __name__ == "__main__":
    git_grep_output = """~/work/cli-main/bundle % git grep '^func [A-Z].*Mutator {' '*.go' '(:exclude)*_test.go' | head -n 10
apps/interpolate_variables.go:func InterpolateVariables() bundle.Mutator {
apps/upload_config.go:func UploadConfig() bundle.Mutator {
apps/validate.go:func Validate() bundle.Mutator {
artifacts/build.go:func Build() bundle.Mutator {
artifacts/prepare.go:func Prepare() bundle.Mutator {
artifacts/upload.go:func CleanUp() bundle.Mutator {
config/loader/entry_point.go:func EntryPoint() bundle.Mutator {
config/loader/process_include.go:func ProcessInclude(fullPath, relPath string) bundle.Mutator {
config/loader/process_root_includes.go:func ProcessRootIncludes() bundle.Mutator {
config/mutator/capture_schema_dependency.go:func CaptureSchemaDependency() bundle.Mutator {"""

    mutator_map = create_mutator_map(git_grep_output)
    for key, value in mutator_map.items():
        print(f'"{key}" -> {value}')
