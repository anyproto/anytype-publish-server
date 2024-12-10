.PHONY: build test deps protos
SHELL=/usr/bin/env bash
export GOPRIVATE=github.com/anyproto
export PATH:=deps:$(PATH)
BUILD_GOOS:=$(shell go env GOOS)
BUILD_GOARCH:=$(shell go env GOARCH)


build:
	@$(eval FLAGS := $$(shell PATH=$(PATH) govvv -flags -pkg github.com/anyproto/any-sync/app))
	GOOS=$(BUILD_GOOS) GOARCH=$(BUILD_GOARCH) go build $(TAGS) -v -o bin/anytype-publish-server -ldflags "$(FLAGS) -X github.com/anyproto/any-sync/app.AppName=anytype-publish-server" github.com/anyproto/anytype-publish-server/cmd/server

test:
	go test ./... --cover

proto:
	protoc --gogofaster_out=:. --go-drpc_out=protolib=github.com/gogo/protobuf:. publishclient/publishapi/protos/*.proto


deps:
	go mod download
	go build -o deps/ storj.io/drpc/cmd/protoc-gen-go-drpc
	go build -o deps/ github.com/gogo/protobuf/protoc-gen-gogofaster
	go build -o deps/ github.com/ahmetb/govvv