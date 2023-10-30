GOBASE=$(shell pwd)
GOBIN=$(GOBASE)/bin

GOPACKAGES = $(shell go list ./pkg/... && go list ./internal/... && go list ./cmd/...)

dependencies:
	go mod download

generate:
	go generate -x $(GOPACKAGES)

test: dependencies
	@go test -v $(GOPACKAGES)

race: dependencies
	@go test -race $(GOPACKAGES)

coverage: 
	@go test -coverprofile cover.out $(GOPACKAGES) && \
	go test github.com/smartcontractkit/ocr2keepers/pkg/v3/... -coverprofile coverV3.out -covermode count && \
	go tool cover -func=coverV3.out | grep total | grep -Eo '[0-9]+\.[0-9]+'

benchmark: dependencies fmt
	@go test $(GOPACKAGES) -bench=. -benchmem -run=^#

parallel: dependencies fmt
	@go test github.com/smartcontractkit/ocr2keepers/internal/keepers -bench=BenchmarkCacheParallelism -benchtime 20s -mutexprofile mutex.out -run=^#

simulator: dependencies fmt
	go build -o $(GOBIN)/simulator ./cmd/simulator/*.go || exit

fmt:
	gofmt -w .

default: build

.PHONY: dependencies test fmt benchmark simulate
