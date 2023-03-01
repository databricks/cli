package schema

// func TestSchemaToDocs(t *testing.T) {
// 	type Bar struct {
// 		A int    `json:"a"`
// 		B string `json:"b,string"`
// 	}

// 	type Foo struct {
// 		Bar Bar `json:"bar"`
// 	}

// 	type MyStruct struct {
// 		Foo map[string]Bar `json:"foo"`
// 	}

// 	elem := MyStruct{}

// 	schema, err := New(reflect.TypeOf(elem), nil)
// 	require.NoError(t, err)

// 	docs, err := newDocs(reflect.TypeOf(elem))
// 	require.NoError(t, err)

// 	jsonSchema, err := json.MarshalIndent(schema, "		", "	")
// 	assert.NoError(t, err)

// 	jsonDocs, err := json.MarshalIndent(docs, "		", "	")
// 	assert.NoError(t, err)

// 	t.Log("[DEBUG] schema: ", string(jsonSchema))
// 	t.Log("[DEBUG] docs: ", string(jsonDocs))
// 	assert.False(t, true)
// }
