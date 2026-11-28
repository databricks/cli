import sys

sys.stdout.write("line 1\n")
sys.stdout.write("line 2\n")
# A progress line that overwrites itself and erases to the end of the line.
sys.stdout.write("progress 1/2\rprogress 2/2\x1b[K\n")
sys.stdout.write("line without newline")
