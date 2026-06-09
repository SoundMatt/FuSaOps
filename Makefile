# FuSaOps developer Makefile

BINARY   := fusaops
GO_FLAGS := -race -count=1

.PHONY: all build test vet lint cover scan check report serve selfcheck clean

all: build

build:
	go build -o $(BINARY) ./cmd/fusaops

test:
	go test $(GO_FLAGS) ./...

vet:
	go vet ./...

lint:
	golangci-lint run ./...

cover:
	go test -coverprofile=coverage.out ./...
	go tool cover -func=coverage.out | tail -1

scan: build
	./$(BINARY) scan

check: build
	./$(BINARY) check

report: build
	./$(BINARY) report --format html --output fusaops-report.html

serve: build
	./$(BINARY) serve

# Dogfooding: gate FuSaOps's own Go source with go-FuSa.
selfcheck:
	gofusa check

clean:
	rm -f $(BINARY) fusaops-report.* coverage.out
