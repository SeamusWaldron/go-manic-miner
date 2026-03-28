.PHONY: run build test clean

# Suppress macOS 15 CVDisplayLink deprecation warnings from Ebitengine's Metal driver.
export CGO_CFLAGS = -Wno-deprecated-declarations

run:
	go run .

build:
	go build -o manicminer .

test:
	go test ./engine/ -v -count=1

clean:
	rm -f manicminer
