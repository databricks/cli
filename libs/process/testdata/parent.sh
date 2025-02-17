#!/bin/bash
# Launch background process.
(sleep 5; echo "abc" > $2) &

# Save PID of the background process to the file specified by the first argument.
echo -n $! > $1
