#! /usr/bin/env python3
import socket
import sys

if len(sys.argv) != 2:
    print("Usage: allocate_ports.py <number_of_ports>", file=sys.stderr)
    sys.exit(1)

try:
    num_ports = int(sys.argv[1])
except ValueError:
    print("Error: number of ports must be an integer", file=sys.stderr)
    sys.exit(1)

ports = []
sockets = []
for _ in range(num_ports):
    s = socket.socket(socket.AF_INET, socket.SOCK_STREAM)
    s.bind(("127.0.0.1", 0))
    port = s.getsockname()[1]
    ports.append(str(port))
    sockets.append(s)
for s in sockets:
    s.close()
print("\n".join(ports))
