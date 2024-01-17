package dynvar_test

// func TestRefsInvalid(t *testing.T) {
// 	invalidMatches := []string{
// 		"${hello_world-.world_world}",   // the first segment ending must not end with hyphen (-)
// 		"${hello_world-_.world_world}",  // the first segment ending must not end with underscore (_)
// 		"${helloworld.world-world-}",    // second segment must not end with hyphen (-)
// 		"${helloworld-.world-world}",    // first segment must not end with hyphen (-)
// 		"${helloworld.-world-world}",    // second segment must not start with hyphen (-)
// 		"${-hello-world.-world-world-}", // must not start or end with hyphen (-)
// 		"${_-_._-_.id}",                 // cannot use _- in sequence
// 		"${0helloworld.world-world}",    // interpolated first section shouldn't start with number
// 		"${helloworld.9world-world}",    // interpolated second section shouldn't start with number
// 		"${a-a.a-_a-a.id}",              // fails because of -_ in the second segment
// 		"${a-a.a--a-a.id}",              // fails because of -- in the second segment
// 	}
// 	for _, invalidMatch := range invalidMatches {
// 		match := re.FindStringSubmatch(invalidMatch)
// 		assert.True(t, len(match) == 0, "Should be invalid interpolation: %s", invalidMatch)
// 	}
// }
