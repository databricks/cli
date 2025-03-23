Each mutator invocation in this directory should be prepended with a short description and information on what bundle fields it reads and writes.
If there already is a description, consider that it is being reviewed and vetted, so only propose changes if you believe it is not factual anymore.

The format is like this:

<ensure there is empty line here>
// Reads (dynamic): list of fields that are being read dynamically (short description of why they are read)
// Reads (static): list of fields that are being read statically (short description of why they are read)
// Updates (dynamic): list of fields that are being written to dynamically (short description of the update logic)
// Updates (static): list of fields that are being written to statically (short description of the update logic)
// <free form short summary, 1 or 2 lines, precise, does not need to repeat the above information, could be a summary of explaining the overral purpose>

Omit empty Reads and Updates lines.

Example:
	// Reads (dynamic): sync.paths (checks that it is really absent)
	// Updates (static): b.Config.Sync.Path (set to ["."] if not set already)
	// Configure the default sync path to equal the bundle root if not explicitly configured.
	// By default, this means all files in the bundle root directory are synchronized.
	mutator.SyncDefaultPath(),

The static reads and writes use standard go syntax and access Bundle pointer (typically named 'b'), e.g b.Config.Sync.Path
The dynamic reads and writes use libs/dyn library and often operate in patterns, e.g. resource.*.*.

Examples of dynamic updates pattern syntax:

code: dyn.NewPattern(dyn.Key("resources"), dyn.AnyKey(), dyn.AnyKey())
pattern in documentation: resource.*.*

code: p := dyn.NewPath(dyn.Key("workspace"), dyn.Key(fieldName))
pattern in documentation: workspace.{<list of fields there if you can figure out them, ... if you cannot or there is too many}

code: 	err := b.Config.Mutate(func(root dyn.Value) (dyn.Value, error) {
		// Walk through the bundle configuration, check all the string leafs and
		// see if any of the prefixes are used in the remote path.
		return dyn.Walk(root, func(p dyn.Path, v dyn.Value) (dyn.Value, error) {
			vv, ok := v.AsString()
			if !ok {
				return v, nil
			}
pattern in documentation: * (strings)
