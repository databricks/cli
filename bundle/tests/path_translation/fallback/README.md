# Test path translation (with fallback to previous behavior)

As of v0.214.0, all paths in a resource definition were resolved relative to the path
where that resource was first defined. If those paths were specified in the same file,
or in a different file in the same directory, this would be intuitive.

If those paths were specified in a different file in a different directory, they would
still be resolved relative to the original file.

For example, a job defined in `./resources/my_job.yml` with an override
in `./override.yml` would have to use paths relative to `./resources`.
This is counter-intuitive and error-prone, and we changed this behavior
in https://github.com/databricks/cli/pull/1273.

## Appendix

Q: Why did this behavior apply as of v0.214.0?

A: With the introduction of dynamic configuration loading, we keep track
   of the location (file, line, column) where a resource is defined.
   This location information is used to perform path translation, but upon
   introduction in v0.214.0, the code still used only a single path per resource.
   Due to the semantics of merging two `dyn.Value` objects, the location
   information of the first existing value is used for the merged value.
   This meant that all paths for a resource were resolved relative to the
   location where the resource was first defined.

Q: What was the behavior before v0.214.0?

A: Before we relied on dynamic configuration loading, all configuration was
   maintained in a typed struct. The path for a resource was an unexported field on the
   resource and was set right after loading the configuration file that contains it.
   Target overrides contained the same path field, and applying a target override
   would set the path for the resource to the path of the target override.
   This meant that all paths for a resource were resolved relative to the
   location where the resource was last defined.

Q: Why are we maintaining compatibility with the old behavior?

A: We want to avoid breaking existing configurations that depend on this behavior.
   Use of the old behavior should trigger warnings with a call to action to update.
   We can include a deprecation timeline to remove the old behavior in the future.
