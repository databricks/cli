package schema

import _ "embed"

//go:embed jsonschema.json
var Bytes []byte

//go:embed jsonschema_ref_only.json
var BytesRefOnly []byte
