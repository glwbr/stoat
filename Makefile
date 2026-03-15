TARGET := stoat
VERSION := 0.5.2
GO := go
GOFMT := gofmt
LINTER := golangci-lint
PREFIX ?= /usr/local

.PHONY: build test test-integration fmt lint clean release install install-prefix

LDFLAGS := -ldflags "-s -w -X main.version=v$(VERSION)"

build:
	$(GO) build $(LDFLAGS) -o bin/$(TARGET) cmd/$(TARGET)/main.go

test:
	$(GO) test ./...

test-integration:
	TESTCONTAINERS_RYUK_DISABLED=true $(GO) test -count=1 -tags integration ./internal/database/integration/...

fmt:
	$(GOFMT) -s -w .

lint:
	$(LINTER) run ./...

clean:
	rm -rf bin dist

release: clean
	mkdir -p dist
	GOOS=darwin  GOARCH=amd64 $(GO) build $(LDFLAGS) -o dist/$(TARGET)-darwin-amd64   ./cmd/$(TARGET)
	GOOS=darwin  GOARCH=arm64 $(GO) build $(LDFLAGS) -o dist/$(TARGET)-darwin-arm64   ./cmd/$(TARGET)
	GOOS=linux   GOARCH=amd64 $(GO) build $(LDFLAGS) -o dist/$(TARGET)-linux-amd64    ./cmd/$(TARGET)
	GOOS=linux   GOARCH=arm64 $(GO) build $(LDFLAGS) -o dist/$(TARGET)-linux-arm64    ./cmd/$(TARGET)

# Install to $GOBIN. Ensure $GOBIN is in your PATH.
install:
	$(GO) install $(LDFLAGS) ./cmd/$(TARGET)

install-prefix: build
	install -d $(DESTDIR)$(PREFIX)/bin
	install -m 755 bin/$(TARGET) $(DESTDIR)$(PREFIX)/bin/$(TARGET)
