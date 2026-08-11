#!/usr/bin/env python3
"""
Print space-separated free TCP ports on the loopback interface.
Usage: free_port.py [count]
"""

import socket
import sys


def main():
    count = int(sys.argv[1]) if len(sys.argv) > 1 else 1

    # Keep every socket bound until all ports are picked, otherwise the kernel
    # is free to hand out the same port twice.
    sockets = [socket.socket() for _ in range(count)]
    for s in sockets:
        s.bind(("127.0.0.1", 0))

    print(" ".join(str(s.getsockname()[1]) for s in sockets))

    for s in sockets:
        s.close()


if __name__ == "__main__":
    main()
