package patchwheel

/*

Patching whl file with a dynamic version suffix.

When developing a DAB, users want to redeploy a wheel without updating a version in pyproject.toml / setup.py manually.

However, installing the same version with pip causes pip to skip the install. Databricks envs follow this behaviour.

For this reason, we've modified default-python template to auto-update the version https://github.com/databricks/cli/pull/1034

However, that makes it tied to setup.py / setuptools and puts onus on users to keep this behaviour.

This package removes the constraint on how the wheel is built and allows adding dynamic part as a post-build step.

PatchWheel(ctx, path, outputDir) takes existing whl file and creates a new patched one with a version that includes
   mtime of the original wheel as a suffix.
   METADATA, directory names, RECORD are all updated to ensure the correct format.

ParseWheelFilename(filename) extracts version from the filename, according to WHL format rules.
*/
