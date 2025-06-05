#!/bin/bash
set -ex
exec go test -bench=. -benchmem -run ^x
