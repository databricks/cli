#!/usr/bin/env python3
"""
Cross-platform script to kill processes using specific ports.
Usage: kill_port.py <port1> [port2] [port3] ...
"""
import sys
import subprocess
import platform
import re


def kill_processes_on_port(port):
    """Kill all processes using the specified port."""
    system = platform.system().lower()
    
    try:
        if system in ['linux', 'darwin']:  # Linux or macOS
            # Try lsof first (most reliable)
            try:
                result = subprocess.run(['lsof', '-ti', f':{port}'], 
                                      capture_output=True, text=True, check=False)
                if result.returncode == 0 and result.stdout.strip():
                    pids = result.stdout.strip().split('\n')
                    for pid in pids:
                        if pid.strip():
                            subprocess.run(['kill', '-9', pid.strip()], check=False)
                            print(f"Killed process {pid.strip()} on port {port}")
            except FileNotFoundError:
                # Fall back to netstat if lsof not available
                result = subprocess.run(['netstat', '-tlnp'], 
                                      capture_output=True, text=True, check=False)
                if result.returncode == 0:
                    for line in result.stdout.split('\n'):
                        if f':{port} ' in line:
                            # Extract PID from output like "tcp 0 0 :::8080 :::* LISTEN 12345/python"
                            match = re.search(r'(\d+)/', line)
                            if match:
                                pid = match.group(1)
                                subprocess.run(['kill', '-9', pid], check=False)
                                print(f"Killed process {pid} on port {port}")
                                
        elif system == 'windows':
            # Windows using netstat and taskkill
            result = subprocess.run(['netstat', '-ano'], 
                                  capture_output=True, text=True, check=False)
            if result.returncode == 0:
                for line in result.stdout.split('\n'):
                    if f':{port} ' in line and 'LISTENING' in line:
                        # Extract PID from last column
                        parts = line.split()
                        if len(parts) >= 5:
                            pid = parts[-1]
                            if pid.isdigit():
                                subprocess.run(['taskkill', '/PID', pid, '/F'], 
                                             check=False, capture_output=True)
                                print(f"Killed process {pid} on port {port}")
        else:
            print(f"Warning: Unsupported platform {system}")
            
    except Exception as e:
        # Silently continue - port cleanup is best effort
        pass


def main():
    if len(sys.argv) < 2:
        print("Usage: kill_port.py <port1> [port2] [port3] ...")
        sys.exit(1)
    
    for port_str in sys.argv[1:]:
        try:
            port = int(port_str)
            kill_processes_on_port(port)
        except ValueError:
            print(f"Warning: Invalid port number '{port_str}'")


if __name__ == '__main__':
    main()