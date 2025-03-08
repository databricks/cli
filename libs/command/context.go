package command

// key is a package-local type to use for context keys.
//
// Using an unexported type for context keys prevents key collisions across
// packages since external packages cannot create values of this type.
type key int

const (
	// configUsedKey is the context key for the auth configuration used to run the
	// command.
	configUsedKey = key(2)

	// workspaceClientKey is the context key for an already configured workspace
	// client that can be used to make authenticated requests.
	workspaceClientKey = key(3)
)
