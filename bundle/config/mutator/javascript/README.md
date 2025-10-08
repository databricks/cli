# JavaScript Mutator

The JavaScript mutator allows you to define bundle resources using JavaScript files.

## Configuration

To use the JavaScript mutator, add the following to your `databricks.yml`:

```yaml
experimental:
  javascript:
    resources:
      - resources.js
```

## JavaScript File Format

The JavaScript file should accept command-line arguments and output resources in JSON format.

### Required Arguments

- `--phase`: The phase of the mutator (currently only `load_resources` is supported)
- `--input`: Path to the input JSON file containing the current bundle configuration
- `--output`: Path where the output JSON should be written
- `--diagnostics`: Path where diagnostics (errors/warnings) should be written
- `--locations`: (optional) Path where source locations should be written

### Expected Output

The JavaScript file should:

1. Read the input JSON from the `--input` file
2. Generate resources
3. Write the output JSON to the `--output` file
4. Write diagnostics to the `--diagnostics` file (as newline-delimited JSON)
5. (Optional) Write source locations to the `--locations` file (as newline-delimited JSON)

### Output Format

The output JSON should have the following structure:

```json
{
  "resources": {
    "jobs": {
      "my_job": {
        "name": "My Job",
        "tasks": [...]
      }
    }
  }
}
```

### Diagnostics Format

Each diagnostic should be a JSON object on its own line:

```json
{"severity": "error", "summary": "Something went wrong", "detail": "More details..."}
{"severity": "warning", "summary": "Potential issue", "detail": "Consider fixing..."}
```

### Locations Format

Each location should be a JSON object on its own line:

```json
{"path": "resources.jobs.my_job", "file": "resources.js", "line": 10, "column": 5}
{"path": "resources.jobs.my_job.tasks[0]", "file": "resources.js", "line": 15, "column": 7}
```

## Example

Here's a simple example of a JavaScript file that generates a job:

```javascript
#!/usr/bin/env node

const fs = require('fs');
const path = require('path');

// Parse command-line arguments
const args = {};
for (let i = 2; i < process.argv.length; i += 2) {
  const key = process.argv[i].replace('--', '');
  args[key] = process.argv[i + 1];
}

// Read input
const input = JSON.parse(fs.readFileSync(args.input, 'utf8'));

// Generate resources
const output = {
  ...input,
  resources: {
    ...input.resources,
    jobs: {
      ...input.resources?.jobs,
      my_javascript_job: {
        name: "My JavaScript Job",
        tasks: [
          {
            task_key: "main",
            notebook_task: {
              notebook_path: "${workspace.file_path}/notebook.py"
            },
            new_cluster: {
              num_workers: 1,
              spark_version: "13.3.x-scala2.12",
              node_type_id: "i3.xlarge"
            }
          }
        ]
      }
    }
  }
};

// Write output
fs.writeFileSync(args.output, JSON.stringify(output, null, 2));

// Write empty diagnostics (no errors or warnings)
fs.writeFileSync(args.diagnostics, '');

// Write locations if requested
if (args.locations) {
  const locations = [
    {
      path: "resources.jobs.my_javascript_job",
      file: path.join(process.cwd(), "resources.js"),
      line: 10,
      column: 5
    }
  ];
  fs.writeFileSync(args.locations, locations.map(l => JSON.stringify(l)).join('\n') + '\n');
}
```

## Features

- **Load Resources**: Add new resources to the bundle using JavaScript
- **Variable References**: JavaScript output can include variable references (e.g., `${workspace.file_path}`)
- **Source Locations**: Track the source location of generated resources for better error messages
- **Diagnostics**: Report errors and warnings from JavaScript code

## Limitations

- Only the `resources` phase is currently supported (adding new resources)
- Mutators phase (modifying existing resources) is not yet implemented
- JavaScript files must be executable by Node.js
