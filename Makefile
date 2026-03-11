TARGET := stoat
VERSION := 0.2.0
GO := go
GOFMT := gofmt
LINTER := golangci-lint

.PHONY: build test fmt lint clean release

build:
	$(GO) build -o bin/$(TARGET) cmd/$(TARGET)/main.go

test:
	$(GO) test ./...

fmt:
	$(GOFMT) -s -w .

lint:
	$(LINTER) run ./...

clean:
	rm -rf bin
	rm -f $(TARGET)-$(VERSION).tar.gz

release: clean
	$(GO) build -o bin/$(TARGET) cmd/$(TARGET)/main.go
	tar -czvf $(TARGET)-$(VERSION).tar.gz bin/$(TARGET)