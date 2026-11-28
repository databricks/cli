import http.server, os

print("version one")

http.server.HTTPServer(
    ("", int(os.environ["DATABRICKS_APP_PORT"])), http.server.SimpleHTTPRequestHandler
).serve_forever()
