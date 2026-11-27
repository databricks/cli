import sys

# A progress line that overwrites itself and erases to the end of the line.
sys.stdout.write("progress 1/2\rprogress 2/2\x1b[K\n")
sys.stdout.write("\x1b[32mdone\x1b[0m\n")
