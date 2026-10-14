package structdiff

/*

Recursively calculate difference between two structs of the same type.

Zero values of omitempty fields are treated as absent; optional values that must be
distinguished from absent ones are pointers.

*/
