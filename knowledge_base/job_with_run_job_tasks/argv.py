import sys

# Display the task's arguments to confirm parameter propagation.
print(f"argv[0]: {sys.argv[0]}")
print(f"argv[1:]: {', '.join(sys.argv[1:])}")
