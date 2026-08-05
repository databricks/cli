import http.server
import json
import os

# Standard library only: acceptance tests run with the network disabled, so the
# app cannot install anything from PyPI.


class Handler(http.server.BaseHTTPRequestHandler):
    def do_GET(self):
        shutdown = self.path == "/shutdown"
        body = b"" if shutdown else json.dumps(dict(self.headers), sort_keys=True).encode() + b"\n"

        self.send_response(200)
        self.send_header("Content-Type", "application/json")
        self.send_header("Content-Length", str(len(body)))
        self.end_headers()
        self.wfile.write(body)
        self.wfile.flush()

        # Exit only once the response is on the wire, so curl does not see a dropped connection.
        if shutdown:
            os._exit(0)

    def log_message(self, fmt, *args):
        pass


server = http.server.HTTPServer(("127.0.0.1", int(os.environ["DATABRICKS_APP_PORT"])), Handler)
# Printed only once the socket is listening, so the test can use it as a readiness signal.
print("Python app has started with: " + os.environ["TEST"], flush=True)
server.serve_forever()
