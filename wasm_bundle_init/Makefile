.PHONY: all clean release debug run size server export

all: release debug nativecli

clean:
	rm -f build

bundle_init.release.wasm: main_wasm.go ../libs/template/*
	GOOS=js GOARCH=wasm go build -ldflags="-w -s" -gcflags=all=-l -o $@
	ls -lh $@

bundle_init.optimized.wasm: bundle_init.release.wasm
	wasm-opt --enable-bulk-memory -Oz -o $@ bundle_init.release.wasm
	ls -lh $@

bundle_init.optimized.wasm.gz: bundle_init.optimized.wasm
	gzip -9 -k -f $<
	ls -lh $@

bundle_init.optimized.wasm.br: bundle_init.optimized.wasm
	brotli -f $<
	ls -lh $@

release: bundle_init.optimized.wasm.br bundle_init.optimized.wasm.gz

debug: bundle_init.debug.wasm

bundle_init.debug.wasm: main_wasm.go ../libs/template/*
	GOOS=js GOARCH=wasm go build -gcflags=all=-l -o $@
	ls -lh $@

# non-wasm build to test locally
nativecli: main_cli.go ../libs/template/*
	#go build -ldflags="-w -s" -gcflags=all=-l -o bundle_init
	# -s prevents 'make size' from working
	go build -ldflags="-w" -gcflags=all=-l -o $@

run: nativecli
	./nativecli default-python '{}' '{}'

size: nativecli analyze_size.py
	go tool nm -size nativecli | python3 analyze_size.py | head -n 50

server: bundle_init.optimized.wasm.gz
	python3 -m http.server

export: bundle_init.optimized.wasm.gz
	mkdir -p output
	cp bundle_init.optimized.wasm.gz index.html wasm_exec.js output/
	git checkout gh-pages
	git pull origin gh-pages
	mkdir -p ../experimental/render
	cp output/bundle_init.optimized.wasm.gz output/index.html output/wasm_exec.js ../experimental/render
