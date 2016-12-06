.PHONY: build doc fmt lint run test vet precommit build_targets clean
.DEFAULT_GOAL: test

BUILD_TIME ?= $(shell date +%FT%T%z)
GODOC_PORT ?= 6060
PACKAGE_NAME := $(shell expr `pwd` : '.*/\([^/]*/[^/]*/[^/]*\)$\')
PROJECT_FILES := $(shell find . -name \*.go | grep -v vendor)
PROJECT_NAME := $(shell basename `pwd`)
TARGET ?= $(shell ls -1 ./cmd/)
VERSION ?= $(shell git rev-parse HEAD)

ccdefault="\033[1;97m"
ccend="\033[0m"

test: lint vet
	govendor test +local -test.race -cover

check: test

build_targets:
	@ls -1 ./cmd/

build: test
	@for file in ${TARGET}; do \
		printf '%b %b\n' $(ccdefault)[BUILD $${file}]$(ccend); \
		go build -o bin/$${file} cmd/$${file}/*.go; \
	done

lint:
	@for file in ${PROJECT_FILES}; do \
		golint $${file}; \
	done

vet:
	@for file in ${PROJECT_FILES}; do \
		go vet $${file}; \
	done

fmt:
	@for file in ${PROJECT_FILES}; do \
		go fmt $${file}; \
	done

doc:
	@echo "Serving godocs on http://localhost:${GODOC_PORT}..."
	@echo "This Package: http://localhost:${GODOC_PORT}/pkg/${PACKAGE_NAME}"
	@godoc -http=":${GODOC_PORT}"

init:
	go get -u github.com/kardianos/govendor
	go get -u github.com/golang/lint/golint
	go get -u golang.org/x/tools/cmd/goimports
	go get -u golang.org/x/tools/cmd/cover
	@printf "Installing git precommit hook... "
	@echo "#!/bin/sh\nmake precommit" > .git/hooks/pre-commit
	@chmod +x .git/hooks/pre-commit
	@printf "Done.\n"
	govendor install +local

clean:
	@for file in ${TARGET}; do \
		rm -f ./bin/$${file}; \
	done

precommit: fmt vet lint test

print-%:
	@echo $* = $($*)