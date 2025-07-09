# Sync Acceptance Tests

This directory contains comprehensive acceptance tests for the Databricks CLI sync functionality. These tests are designed to run against real cloud environments and verify that sync operations work correctly end-to-end.

## Test Overview

The following acceptance tests are included:

### 1. Full File Sync (`full-file-sync/`)
Tests the complete file synchronization workflow with the `--full` flag:
- Creates, modifies, and deletes files
- Verifies remote content matches local content
- Tests immediate synchronization without snapshots

### 2. Incremental Sync (`incremental-sync/`)
Tests incremental synchronization functionality:
- Creates and maintains snapshot files
- Verifies incremental updates work correctly
- Tests file creation, modification, and deletion with snapshots

### 3. Nested Folder Sync (`nested-folder-sync/`)
Tests handling of nested directory structures:
- Creates deeply nested directory structures
- Verifies proper cleanup of empty directories
- Tests directory listing and file placement

### 4. Notebook Conversion (`notebook-conversion/`)
Tests automatic conversion between notebooks and regular files:
- Python notebooks (`# Databricks notebook source`)
- SQL notebooks (`-- Databricks notebook source`)
- Scala notebooks (`// Databricks notebook source`)
- Conversion between notebook and file formats based on content

### 5. Special Characters (`special-characters/`)
Tests handling of files and directories with special characters:
- Spaces in names
- Plus signs (`+`) in names
- Hash symbols (`#`) in names
- Proper escaping and encoding

### 6. File Overwrites Folder (`file-overwrites-folder/`)
Tests edge cases where files and folders replace each other:
- Folder deleted and replaced by file with same name
- File deleted and replaced by folder with same name
- Nested structure replacements

## Running the Tests

### Prerequisites
- Databricks CLI built and available
- Cloud environment configured (Azure, AWS, or GCP)
- Appropriate permissions to create workspace files and directories

### Configuration
Set `Cloud = true` in the `test.toml` file for each test you want to run against the cloud environment.

### Running Individual Tests

To run a specific sync test:

```bash
# Using the provided helper functions
cloudrun() {
    deco env run -i -n azure-prod-ucws -- go test -timeout 30s -run ^TestAccept/$1$ github.com/databricks/cli/acceptance
}

# Run full file sync test
cloudrun full-file-sync

# Run incremental sync test
cloudrun incremental-sync

# Run nested folder sync test
cloudrun nested-folder-sync

# Run notebook conversion test
cloudrun notebook-conversion

# Run special characters test
cloudrun special-characters

# Run file overwrites folder test
cloudrun file-overwrites-folder
```

### Running All Sync Tests

```bash
# Run all sync tests
cloudrun "sync/*"
```

### Updating Test Expectations

If you need to update test expectations:

```bash
cloudupdate() {
    deco env run -i -n azure-prod-ucws -- go test -timeout 30s -run ^TestAccept/$1$ github.com/databricks/cli/acceptance -update
}

# Update a specific test
cloudupdate full-file-sync
```

## Test Structure

Each test follows this structure:

```
acceptance/sync/test-name/
├── test.toml          # Test configuration
├── script             # Test script (executable)
└── README.md          # Test-specific documentation (optional)
```

### Test Configuration (`test.toml`)
- `Local = false` - Don't run locally
- `Cloud = true` - Run against cloud environment
- `[EnvMatrix]` - Environment matrix for testing different configurations

### Test Script (`script`)
- Bash script that performs the actual test
- Uses helper functions for common operations
- Starts sync in background and tests various scenarios
- Includes cleanup to remove temporary files/directories

## Helper Functions

Common helper functions used across tests:

- `check_remote_file_exists()` - Check if a file exists in the remote workspace
- `check_remote_object_type()` - Verify object type (FILE, DIRECTORY, NOTEBOOK)
- `check_remote_language()` - Check notebook language
- `list_remote_dir()` - List contents of remote directory
- `check_remote_file_content()` - Verify file content matches expected

## Important Notes

1. **Cloud Environment Required**: These tests require a real cloud environment and cannot be run locally.

2. **Cleanup**: All tests include cleanup procedures to remove temporary files and directories.

3. **Timing**: Tests include sleep statements to account for sync delays. Adjust timing if needed for different environments.

4. **Permissions**: Tests assume the user has permissions to create/delete files in their workspace user directory.

5. **Unique Paths**: Each test uses unique remote paths to avoid conflicts when running multiple tests simultaneously.

## Troubleshooting

### Common Issues

1. **Permission Errors**: Ensure the test user has workspace file permissions
2. **Timeout Issues**: Increase sleep durations if sync operations are slow
3. **Path Conflicts**: Ensure unique remote paths are used
4. **Environment Issues**: Verify cloud environment is properly configured

### Debugging

To debug test failures:

1. Check the sync output log (`sync_output.log`) created by each test
2. Verify the remote workspace state manually
3. Check CLI permissions and configuration
4. Run tests individually to isolate issues

## Converting from Integration Tests

These acceptance tests were converted from the integration tests in `integration/cmd/sync/sync_test.go`. The main differences:

- Use shell scripts instead of Go code
- Run against real cloud environments
- Use CLI commands instead of SDK calls
- Include proper cleanup procedures

## Contributing

When adding new sync tests:

1. Create a new directory under `acceptance/sync/`
2. Follow the existing structure and naming conventions
3. Include proper cleanup procedures
4. Add documentation for the new test
5. Test against multiple cloud environments if possible
