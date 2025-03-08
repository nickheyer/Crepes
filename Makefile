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
STORAGE_DIR=./storage
THUMB_DIR=./thumbnails
WEB_DIR=./internal/ui
WEB_DIST=./internal/ui/build
WEB_MODULES=./internal/ui/node_modules
DATA_DIR=./data

# FOR TESTING ONLY
PKG_LIST=$(shell $(GOCMD) list ./... | grep -v /vendor/)

.PHONY: all build clean test coverage deps tidy build-linux run dev web

all: clean deps test build

web:
	cd $(WEB_DIR) && npm install && npm run build
	

build: web 
	mkdir -p $(BUILD_DIR)
	$(GOBUILD) -o $(BUILD_DIR)/$(BINARY_NAME) $(MAIN_PATH)

clean:
	$(GOCLEAN)
	rm -rf $(BUILD_DIR) $(STORAGE_DIR) $(THUMB_DIR) $(DATA_DIR) $(WEB_DIST)

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

build-linux: web
	CGO_ENABLED=0 GOOS=linux GOARCH=amd64 $(GOBUILD) -o $(BUILD_DIR)/$(BINARY_UNIX) $(MAIN_PATH)

run: build
	$(BUILD_DIR)/$(BINARY_NAME)

dev: clean build run
