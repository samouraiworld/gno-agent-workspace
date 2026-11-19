# Serves page.html over TLS so chromium loads it as an https page.
# Usage: python3 -I serve.py <dir> <port> <cert.pem> <key.pem>
import functools
import http.server
import ssl
import sys

directory, port, cert, key = sys.argv[1], int(sys.argv[2]), sys.argv[3], sys.argv[4]
ctx = ssl.SSLContext(ssl.PROTOCOL_TLS_SERVER)
ctx.load_cert_chain(cert, key)
handler = functools.partial(http.server.SimpleHTTPRequestHandler, directory=directory)
srv = http.server.HTTPServer(("127.0.0.1", port), handler)
srv.socket = ctx.wrap_socket(srv.socket, server_side=True)
srv.serve_forever()
