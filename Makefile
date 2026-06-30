VERSION ?= 1.0.0
GO_MAIN_PACKAGE ?= ./cmd/md2x
CLI_NAME ?= md2x
GOFLAGS ?= -buildvcs=false
TMPDIR ?= /tmp
export GOCACHE ?= $(TMPDIR)/md2x-go-build
LDFLAGS := -s -w -X github.com/geekjourneyx/md2x/internal/cli.version=$(VERSION)

.PHONY: build test vet coverage quality-gates release-check

build:
	go build $(GOFLAGS) -trimpath -ldflags="$(LDFLAGS)" -o bin/$(CLI_NAME) $(GO_MAIN_PACKAGE)

test:
	go test $(GOFLAGS) ./...

vet:
	go vet $(GOFLAGS) ./...

coverage:
	go test $(GOFLAGS) -coverprofile=coverage.out ./...

quality-gates:
	bash scripts/quality-gates.sh

release-check:
	bash scripts/release-check.sh $(VERSION)
