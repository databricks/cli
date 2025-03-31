import sys
import os

info = " ".join(sys.argv[1:])
env = os.getenv("SOME_ENV_VAR", "default")
sys.stdout.write(f"from myscript.py {info}: env: {env}\n")

exitcode = int(sys.argv[1])
sys.exit(exitcode)
