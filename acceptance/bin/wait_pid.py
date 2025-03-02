#!/usr/bin/env python3
import sys
import os
import time
import platform
import subprocess


def is_finished_windows(pid):
    assert int(pid) > 0

    # Run task list but filter the list by pid..
    output = subprocess.check_output(f'tasklist /fi "PID eq {pid}"', text=True)

    # If tasklist does not find any process, that means the process we are
    # waiting for has terminated.
    unaliveMsg = "No tasks are running which match the specified criteria."
    if unaliveMsg in output:
        return True

    return False


def is_finished_unix(pid):
    assert int(pid) > 0

    # Send signal 0 to the process to check if it exists.
    # From the docs:
    #    If sig is 0, then no signal is sent, but existence and permission
    #    checks are still performed; this can be used to check for the
    #    existence of a process ID or process group ID that the caller is
    #    permitted to signal.
    # ref: https://man7.org/linux/man-pages/man2/kill.2.html
    try:
        os.kill(pid, 0)
    except OSError:
        return True

    return False


def wait_pid(pid):
    max_attempts = 600  # 600 * 0.1 seconds = 1 minute
    sleep_time = 0.1

    for i in range(max_attempts):
        if platform.system().lower() == "windows":
            if is_finished_windows(pid):
                print("[wait_pid] process has ended")
                return 0

        else:
            if is_finished_unix(pid):
                print("[wait_pid] process has ended")
                return 0

        time.sleep(sleep_time)

    print(f"Timeout: Process {pid} did not end within 1 minute")
    return 1


exit_code = wait_pid(int(sys.argv[1]))
sys.exit(exit_code)
