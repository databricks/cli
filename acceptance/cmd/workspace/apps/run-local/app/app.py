import http.server
import json
import os
import threading

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

        # shutdown() blocks until serve_forever() returns, which cannot happen until this
        # handler returns, so it has to run on another thread. Stopping the loop rather
        # than exiting the process lets the connection close cleanly, otherwise the client
        # can see a reset instead of the response we just wrote.
        if shutdown:
            threading.Thread(target=self.server.shutdown, daemon=True).start()

    def log_message(self, fmt, *args):
        pass


# Bind before printing, so the message can never appear while the port is still closed.
server = http.server.HTTPServer(("127.0.0.1", int(os.environ["DATABRICKS_APP_PORT"])), Handler)
print("Python app has started with: " + os.environ["TEST"], flush=True)
server.serve_forever()
