#!/usr/bin/env python3
"""Minimal test server for the text widget URL feature. GET / returns 5 lines of plain text."""

from http.server import BaseHTTPRequestHandler, HTTPServer

TEXT = (
    "# Mir IoT Hub\n"
    "Status: **operational**  \n"
    "Devices online: 42  \n"
    "Last sync: 2026-04-08 12:00 UTC  \n"
    "All systems nominal.  \n"
)

PORT = 8765


class Handler(BaseHTTPRequestHandler):
    def do_GET(self):
        body = TEXT.encode()
        self.send_response(200)
        self.send_header("Content-Type", "text/plain; charset=utf-8")
        self.send_header("Content-Length", str(len(body)))
        self.send_header("Access-Control-Allow-Origin", "*")
        self.end_headers()
        self.wfile.write(body)

    def log_message(self, fmt, *args):
        print(f"{self.address_string()} - {fmt % args}")


if __name__ == "__main__":
    server = HTTPServer(("", PORT), Handler)
    print(f"Serving on http://localhost:{PORT}")
    server.serve_forever()
