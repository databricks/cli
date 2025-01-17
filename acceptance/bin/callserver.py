#!/usr/bin/env python3
import sys
import os
import urllib.parse



args = "/".join(sys.argv[1:])
args = urllib.parse.quote_plus(args)

url = os.environ["CMD_SERVER_URL"] + "/?args=" + args
os.execvp("curl", ["curl", "-s", url])
