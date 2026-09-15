import os
from http.server import BaseHTTPRequestHandler, HTTPServer


class Handler(BaseHTTPRequestHandler):
    def do_GET(self):
        self.send_response(200)
        self.end_headers()
        self.wfile.write(b"OK")

    def log_message(self, *_):
        pass


port = int(os.environ.get("APP_PORT", "8080"))
HTTPServer(("0.0.0.0", port), Handler).serve_forever()
