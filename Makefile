.ONESHELL:
.DELETE_ON_ERROR:
MAKEFLAGS += --no-builtin-rules
NAME = "github.com/odpf/optimus"
#CTL_VERSION := `git describe --tags $(shell git rev-list --tags --max-count=1)`
CTL_VERSION := "$(shell git rev-parse --short HEAD)"
OPMS_VERSION := "$(shell git rev-parse --short HEAD)"

all: build

.PHONY: build build-optimus build-jazz smoke-test unit-test test clean generate dist init addhooks build-ui ui

build-ctl: generate
	@echo " > building opctl version ${CTL_VERSION}"
	@go build -ldflags "-X main.Version=${CTL_VERSION}" ${NAME}/cmd/opctl

# optimus uses hard-coded version strings for now
build-optimus: generate
	@echo " > building optimus version ${OPMS_VERSION}"
	@go build -ldflags "-X 'main.Version=${OPMS_VERSION}'" ${NAME}/cmd/optimus

build-ui:
	@echo " > building ui"
	@cd ./web/ui && npm i && npm run build && cp -r ./dist/ ../../resources/ui/

build: build-optimus build-ctl
	@echo " - build complete"
	
test: smoke-test unit-test

generate:
	@buf generate

ui: build-ui generate build-optimus

unit-test:
	go list ./... | grep -v extern | xargs go test -count 1 -cover -race -timeout 1m -tags=unit_test

smoke-test: build
	@bash ./scripts/smoke-test.sh

integration-test: build
	go list ./... | grep -v extern | xargs go test -count 1 -cover -race -timeout 1m

coverage:
	go test -coverprofile test_coverage.html ./... -tags=unit_test && go tool cover -html=test_coverage.html

# TODO(Aman): remove duplication between "make build" and
# ./scripts/make-distributables.sh
dist: generate
	@bash ./scripts/build-distributables.sh

clean:
	rm -rf optimus opctl dist/
	rm -rf ./resources/ui/*

addhooks:
	chmod -R +x .githooks/
	git config core.hooksPath .githooks/

init:
	@echo "> configuring git for odpf.github.io"
	go env -w GOPRIVATE=odpf.github.io
	git config --global url."git@odpf.github.io:".insteadOf "https://odpf.github.io/"

install:
	@echo "> installing dependencies"
	go get -u github.com/golang/protobuf/proto
	go get -u github.com/golang/protobuf/protoc-gen-go
	go get -u google.golang.org/grpc/cmd/protoc-gen-go-grpc
	go get -u google.golang.org/grpc
	go get -u github.com/bufbuild/buf/cmd/buf
