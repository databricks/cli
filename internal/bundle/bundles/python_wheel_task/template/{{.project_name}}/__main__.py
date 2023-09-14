"""
The entry point of the Python Wheel
"""

import sys
import os


def main():
    # This method will print the provided arguments
    print("Hello from my func")
    print("Got arguments:")
    print(sys.argv)

    retval = os.getcwd()
    print("Directory changed successfully %s" % retval)


if __name__ == "__main__":
    main()
