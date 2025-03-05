.PHONY: all build clean test linux darwin windows

BINARY=rtop
VERSION=$(shell git describe --tags --always --dirty)
BUILD_TIME=$(shell date +%FT%T%z)
LDFLAGS=-ldflags "-s -w -X main.VERSION=${VERSION} -X main.BUILD_TIME=${BUILD_TIME}"

all: clean linux darwin windows

build:
	go build ${LDFLAGS} -o ${BINARY}

clean:
	rm -f ${BINARY}
	rm -f ${BINARY}-linux-*
	rm -f ${BINARY}-darwin-*
	rm -f ${BINARY}-windows-*

test:
	go test -v ./...

# Cross compilation
linux:
	GOOS=linux GOARCH=amd64 go build ${LDFLAGS} -o ${BINARY}-linux-amd64
	GOOS=linux GOARCH=arm64 go build ${LDFLAGS} -o ${BINARY}-linux-arm64

darwin:
	GOOS=darwin GOARCH=amd64 go build ${LDFLAGS} -o ${BINARY}-darwin-amd64
	GOOS=darwin GOARCH=arm64 go build ${LDFLAGS} -o ${BINARY}-darwin-arm64

windows:
	GOOS=windows GOARCH=amd64 go build ${LDFLAGS} -o ${BINARY}-windows-amd64.exe

# Create release archives
release: all
	mkdir -p release
	# Linux
	tar -czvf release/${BINARY}-linux-amd64.tar.gz ${BINARY}-linux-amd64
	tar -czvf release/${BINARY}-linux-arm64.tar.gz ${BINARY}-linux-arm64
	# MacOS
	tar -czvf release/${BINARY}-darwin-amd64.tar.gz ${BINARY}-darwin-amd64
	tar -czvf release/${BINARY}-darwin-arm64.tar.gz ${BINARY}-darwin-arm64
	# Windows
	zip -j release/${BINARY}-windows-amd64.zip ${BINARY}-windows-amd64.exe

# Install locally
install: build
	mv ${BINARY} /usr/local/bin/