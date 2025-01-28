import sys

info = " ".join(sys.argv[1:])
sys.stderr.write(f"from myscript.py {info}: hello stderr!\n")
sys.stdout.write(f"from myscript.py {info}: hello stdout!\n")

exitcode = int(sys.argv[1])
sys.exit(exitcode)
