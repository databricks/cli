.PHONY: all clean release debug run size server export

all: release debug nativecli

clean:
	rm -rf build

build:
	mkdir -p build

build/bundle_init.release.wasm: main_wasm.go ../libs/template/* | build
	GOOS=js GOARCH=wasm go build -ldflags="-w -s" -gcflags=all=-l -o $@
	ls -lh $@

build/bundle_init.optimized.wasm: build/bundle_init.release.wasm
	wasm-opt --enable-bulk-memory -Oz -o $@ $<
	ls -lh $@

build/bundle_init.optimized.wasm.gz: build/bundle_init.optimized.wasm
	gzip -9 -k -f $<
	ls -lh $@

bundle_init.optimized.wasm.gz: build/bundle_init.optimized.wasm.gz
	cp $< $@

build/bundle_init.optimized.wasm.br: build/bundle_init.optimized.wasm
	brotli -f $<
	ls -lh $@

release: build/bundle_init.optimized.wasm.br build/bundle_init.optimized.wasm.gz

debug: build/bundle_init.debug.wasm

build/bundle_init.debug.wasm: main_wasm.go ../libs/template/* | build
	GOOS=js GOARCH=wasm go build -gcflags=all=-l -o $@
	ls -lh $@

# non-wasm build to test locally
build/nativecli: main_cli.go ../libs/template/* | build
	#go build -ldflags="-w -s" -gcflags=all=-l -o bundle_init
	# -s prevents 'make size' from working
	go build -ldflags="-w" -gcflags=all=-l -o $@

run: build/nativecli
	./build/nativecli default-python '{}' '{}'

size: build/nativecli analyze_size.py
	go tool nm -size build/nativecli | python3 analyze_size.py | head -n 50

server: bundle_init.optimized.wasm.gz
	python3 -m http.server

export: build/bundle_init.optimized.wasm.gz
	cp index.html wasm_exec.js build/
	git checkout gh-pages
	git pull origin gh-pages
	mkdir -p ../experimental/render
	cp build/bundle_init.optimized.wasm.gz build/index.html build/wasm_exec.js ../experimental/render
