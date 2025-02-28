#!/usr/bin/env python3
import sys
import os
import time
import platform
import subprocess


def wait_pid(pid):
    max_attempts = 600  # 600 * 0.1 seconds = 1 minute
    sleep_time = 0.1

    for i in range(max_attempts):
        # Check if we are on Windows or a Unix system.
        if platform.system().lower() == "windows":
            # Windows approach: use tasklist to check for the existence of the process.
            try:
                # Get the output of 'tasklist'
                output = subprocess.check_output(["tasklist"], text=True)
            except subprocess.CalledProcessError:
                print("[wait_pid] Error retrieving tasklist. Assuming process has ended.")
                return 0

            # if the PID is not found in the list then assume the process ended.
            if str(pid) not in output:
                print("[wait_pid] process has ended")
                return 0
        else:
            # Linux/macOS approach: using os.kill with signal 0 to check if the process exists.
            try:
                os.kill(pid, 0)
            except OSError:
                print("[wait_pid] process has ended")
                return 0

        time.sleep(sleep_time)

    print(f"Timeout: Process {pid} did not end within 1 minute")
    return 1


try:
    pid = int(sys.argv[1])
except ValueError:
    print("Error: PID must be an integer.")
    sys.exit(1)

exit_code = wait_pid(pid)
sys.exit(exit_code)
