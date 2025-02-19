#!/usr/bin/env python3
import sys
import time
import threading
from http.server import BaseHTTPRequestHandler, HTTPServer

if len(sys.argv) < 2:
    print("Usage: python script.py <port_file_path>")
    sys.exit(1)

port_file_path = sys.argv[1]

class SimpleHandler(BaseHTTPRequestHandler):
    def do_GET(self):
        # Send HTTP 200 response with plain text content
        self.send_response(200)
        self.send_header("Content-type", "text/plain")
        self.end_headers()
        self.wfile.write(b"child says hi")

# Bind to localhost on port 0 to let the OS pick an available port.
server_address = ('localhost', 0)
httpd = HTTPServer(server_address, SimpleHandler)

# Retrieve the assigned port.
assigned_port = httpd.server_address[1]

# Write the port number to the provided file path.
with open(port_file_path, "w") as f:
    f.write(str(assigned_port))

def shutdown_server(httpd):
    time.sleep(120)
    print("2 minutes elapsed. Shutting down the server.")
    httpd.shutdown()

# Start a background thread that will shut down the server after 120 seconds).
# This is a precautions to prevent tests servers from running indefinitely and consuming resources.
shutdown_thread = threading.Thread(target=shutdown_server, args=(httpd,), daemon=True)
shutdown_thread.start()

try:
    httpd.serve_forever()
except KeyboardInterrupt:
    print("\nServer is shutting down.")
    httpd.server_close()
finally:
    httpd.server_close()
