import sys

exitcode, name = sys.argv[1], sys.argv[2]
print(f"from {name}: hello")
sys.exit(int(exitcode))
