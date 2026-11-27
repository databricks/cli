import http.server, os

http.server.HTTPServer(
    ("", int(os.environ["DATABRICKS_APP_PORT"])), http.server.SimpleHTTPRequestHandler
).serve_forever()
