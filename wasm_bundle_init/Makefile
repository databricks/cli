.PHONY: build_wasm build_cli all size run clean

all: bundle_init.wasm bundle_init

clean:
	rm bundle_init bundle_init.wasm

bundle_init.wasm: main_wasm.go ../libs/template/*
	GOOS=js GOARCH=wasm go build -ldflags="-w -s" -o bundle_init.wasm

# non-wasm build to test locally
bundle_init: main_cli.go ../libs/template/*
	go build -ldflags="-w -s" -o bundle_init

run: bundle_init
	./bundle_init default-python '{}' '{}'

size: build_cli analyze_size.py
	go tool nm -size bundle_init | python3 analyze_size.py | head -n 50

server: bundle_init.wasm
	python3 -m http.server
