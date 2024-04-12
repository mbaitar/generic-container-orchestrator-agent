BINARY := out/agent
PACKAGE_NAME := github.com/mabaitar/gco/agent

VERSION := v0.1.0
BUILD := $(shell date +%Y%m%d)
COMMIT=$(shell git rev-parse HEAD)

BUILD_FLAG := ${PACKAGE_NAME}/build.Build=${BUILD}
VERSION_FLAG := ${PACKAGE_NAME}/build.Version=${VERSION}
COMMIT_FLAG := ${PACKAGE_NAME}/build.Commit=${COMMIT}

LDFLAGS := "-w -s -X ${VERSION_FLAG} -X ${BUILD_FLAG} -X ${COMMIT_FLAG}"

build:
	GOARCH=arm64 GOOS=linux go build --ldflags ${LDFLAGS} -o out/agent_${VERSION}_arm64_linux main.go
	GOARCH=amd64 GOOS=linux go build --ldflags ${LDFLAGS} -o out/agent_${VERSION}_amd64_linux main.go
	GOARCH=arm64 GOOS=darwin go build --ldflags ${LDFLAGS} -o out/agent_${VERSION}_arm64_darwin main.go
	GOARCH=amd64 GOOS=darwin go build --ldflags ${LDFLAGS} -o out/agent_${VERSION}_amd64_darwin main.go

test:
	go test -cover ./...

generate:
	cd proto && buf generate

build-local:
	@go build -o bin/app

run: build-local
	@./bin/app


.PHONY: test build generate