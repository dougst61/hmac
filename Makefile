APP_NAME  = featuretoken
BUILD_DIR = build
VERSION   = $(shell cat VERSION)
COMMIT    = $(shell git rev-parse --short HEAD 2>/dev/null || echo unknown)
DATE      = $(shell date -u +%Y-%m-%dT%H:%M:%SZ)
BUILT_BY  = $(shell whoami)
LDFLAGS   = -X main.Version=$(VERSION) -X main.commit=$(COMMIT) -X main.date=$(DATE) -X main.builtBy=$(BUILT_BY)
GOOS      = $(shell go env GOOS)
GOARCH    = $(shell go env GOARCH)

.PHONY: build all clean test lint fmt tidy

build:
	mkdir -p $(BUILD_DIR)
	GOOS=$(GOOS) GOARCH=$(GOARCH) go build -ldflags "$(LDFLAGS)" -o $(BUILD_DIR)/$(APP_NAME) .

all: clean
	mkdir -p $(BUILD_DIR)
	GOOS=darwin GOARCH=arm64 GOARM64=v8.4,lse,crypto go build -ldflags "$(LDFLAGS)" -o $(BUILD_DIR)/$(APP_NAME)-darwin-arm64 .
	GOOS=darwin GOARCH=amd64 GOAMD64=v3 go build -ldflags "$(LDFLAGS)" -o $(BUILD_DIR)/$(APP_NAME)-darwin-amd64 .
	GOOS=linux GOARCH=amd64 GOAMD64=v3 go build -ldflags "$(LDFLAGS)" -o $(BUILD_DIR)/$(APP_NAME)-linux-amd64 .
	GOOS=windows GOARCH=amd64 GOAMD64=v3 go build -ldflags "$(LDFLAGS)" -o $(BUILD_DIR)/$(APP_NAME)-windows-amd64.exe .

clean:
	rm -rf $(BUILD_DIR)

test:
	go test ./...

lint:
	go vet ./...

fmt:
	go fmt ./...

tidy:
	go mod tidy
