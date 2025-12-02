---
name: Bug report for direct deployment engine for DABs
about: Use this to report an issue with direct deployment engine in Databricks Asset Bundles.
labels: DABs engine/direct Bug
title: ''
---

### Describe the issue
A clear and concise description of what the issue is

### Configuration
Please provide a minimal reproducible configuration for the issue

### Steps to reproduce the behavior
 Please list the steps required to reproduce the issue, for example:
1. Run `databricks bundle deploy ...`
2. Run `databricks bundle run ...`
3. See error

### Expected Behavior
Clear and concise description of what should have happened

### Actual Behavior
Clear and concise description of what actually happened

### OS and CLI version
Please provide the version of the CLI (eg: v0.1.2) and the operating system (eg: windows). You can run databricks --version to get the version of your Databricks CLI

### Is this a regression?
Did this work in a previous version of the CLI? If so, which versions did you try?

### Detailed plan

If relevant, please provide redacted output of `bundle plan -o json -t <your target>`. The plan includes reasons for a given action per resource.

### Debug Logs
Output logs if you run the command with debug logs enabled. Example: databricks bundle deploy --log-level=debug. Redact if needed
