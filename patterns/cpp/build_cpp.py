#!/usr/bin/env python3
import subprocess
import sys
import os

env = os.environ.copy()
result = subprocess.run(
    ["g++", "-std=c++17", "-O2", "-I.", "-o", "/tmp/dof_cpp", "main.cpp"],
    cwd="/home/hermes/projects/AI/DOF/patterns/cpp",
    capture_output=True, text=True,
    env=env,
    timeout=300,
)
print("STDOUT:", result.stdout)
print("STDERR:", result.stderr)
print("Return code:", result.returncode)
