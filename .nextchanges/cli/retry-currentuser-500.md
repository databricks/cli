* Retry the current-user (SCIM `Me`) lookup on transient HTTP 500 responses so a temporarily-unavailable backend no longer fails bundle commands outright.
