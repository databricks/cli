package structpath

/*
PathNode is a type to represent paths in Go structs that supports fields, map and index keys as well as wildcards for those.

It can represent map keys accurately and can be used to serialized and deserialize path without information loss as it's intended for serialized deployment plan.

The map keys are encoded in single quotes: tags['name']. Single quote can escaped by placing two single quotes: tags[''''] (map key is one single quote).

This encoding is chosen over traditional double quotes because when encoded in JSON it does not need to be escaped:

{
	"resources.jobs.foo.tags['cost-center']": {}
}

*/
