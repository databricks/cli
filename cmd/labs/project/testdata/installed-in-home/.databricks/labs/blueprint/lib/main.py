import os, sys, json

payload = json.loads(sys.argv[1])

if "echo" == payload["command"]:
    json.dump(
        {
            "command": payload["command"],
            "flags": payload["flags"],
            "env": {k: v for k, v in os.environ.items()},
        },
        sys.stdout,
    )
    sys.exit(0)

if "table" == payload["command"]:
    sys.stderr.write("some intermediate info\n")
    json.dump(
        {
            "records": [
                {"key": "First", "value": "Second"},
                {"key": "Third", "value": "Fourth"},
            ]
        },
        sys.stdout,
    )
    sys.exit(0)

print(f"host is {os.environ['DATABRICKS_HOST']}")

print(f"[{payload['command']}] command flags are {payload['flags']}")

answer = input("What is your name? ")

print(f"Hello, {answer}!")
