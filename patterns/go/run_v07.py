#!/usr/bin/env python3
import subprocess
import sys
import os

env = os.environ.copy()
env["GOCACHE"] = "/tmp/gocache"
env["GOPATH"] = "/tmp/gopath"

result = subprocess.run(
    ["go", "run", ".", "v07"],
    cwd="/home/hermes/projects/AI/DOF/patterns/go",
    capture_output=True, text=True,
    env=env,
    timeout=300,
)
print(result.stdout)
if result.stderr:
    print(result.stderr, file=sys.stderr)
sys.exit(result.returncode)
