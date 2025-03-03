GOCMD=go
GOBUILD=$(GOCMD) build
GOCLEAN=$(GOCMD) clean
GOTEST=$(GOCMD) test
GOGET=$(GOCMD) get
GOMOD=$(GOCMD) mod
BINARY_NAME=crepes
BINARY_UNIX=$(BINARY_NAME)_unix
MAIN_PATH=./cmd/Crepes
BUILD_DIR=./build

# FOR TESTING ONLY
PKG_LIST=$(shell $(GOCMD) list ./... | grep -v /vendor/)

.PHONY: all build clean test coverage deps tidy build-linux run dev

all: clean deps test build

build: 
	mkdir -p $(BUILD_DIR)
	$(GOBUILD) -o $(BUILD_DIR)/$(BINARY_NAME) $(MAIN_PATH)

clean:
	$(GOCLEAN)
	rm -rf $(BUILD_DIR)

test:
	$(GOTEST) -v $(PKG_LIST)

coverage:
	$(GOTEST) -coverprofile=coverage.out $(PKG_LIST)
	$(GOCMD) tool cover -html=coverage.out

deps:
	$(GOGET) -u
	$(GOMOD) tidy

tidy:
	$(GOMOD) tidy

build-linux:
	CGO_ENABLED=0 GOOS=linux GOARCH=amd64 $(GOBUILD) -o $(BUILD_DIR)/$(BINARY_UNIX) $(MAIN_PATH)

run: build
	$(BUILD_DIR)/$(BINARY_NAME)

dev: clean build run
