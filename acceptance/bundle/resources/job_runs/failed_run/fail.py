import sys

print("intentional failure", file=sys.stderr)
sys.exit(1)
