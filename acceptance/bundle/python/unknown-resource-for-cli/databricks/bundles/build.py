import argparse
import json
import sys
import pathlib


def main():
    parser = argparse.ArgumentParser()
    parser.add_argument("--input", default=None)
    parser.add_argument("--output", default=None)
    parser.add_argument("--diagnostics", default=None)

    args = sys.argv[1:]
    parsed, _ = parser.parse_known_args(args)

    with open(parsed.input, "r") as f:
        input = json.load(f)

    input["resources"] = input.get("resources", {})
    input["resources"]["unknown_resource"] = {"my_resource": {"name": "My Resource"}}

    pathlib.Path(parsed.diagnostics).touch()

    with open(parsed.output, "w") as f:
        json.dump(input, f, ensure_ascii=False)


if __name__ == "__main__":
    main()
