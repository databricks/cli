#!/usr/bin/env python3
import json
import sys


def main():
    if len(sys.argv) < 2:
        print("Usage: publish_review.py '<json>'")
        sys.exit(1)

    review = json.loads(sys.argv[1])
    with open("/tmp/reviewbot_output.json", "w") as f:
        json.dump(review, f, indent=2)
    print("Review saved. Awaiting user confirmation to publish.")


if __name__ == "__main__":
    main()
