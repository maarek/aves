# Go parameters
GOCMD=go
GOBUILD=$(GOCMD) build
GOCLEAN=$(GOCMD) clean
GOTEST=$(GOCMD) test
GOGET=$(GOCMD) get
BUILD_DIR=generated
PROTO_DIR=api


ifndef $(GOPATH)
    GOPATH=$(shell go env GOPATH)
    export GOPATH
endif

all: build

build: 
	$(GOBUILD) -o ./bin/aves ./cmd/server/main.go
	$(GOBUILD) -o ./bin/avcli ./cmd/cli/main.go

test:
	echo "Tests Not Implemented"
