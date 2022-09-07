GOBASE=$(shell pwd)
GOBIN=$(GOBASE)/bin

GOPACKAGES = $(shell go list ./...)

dependencies:
	go mod download

test: dependencies
	@go test -v $(GOPACKAGES)

coverage: 
	@go test -coverprofile cover.out $(GOPACKAGES)

benchmark: dependencies fmt
	@go test $(GOPACKAGES) -bench=. -benchmem -run=^#

simulator: dependencies fmt
	go build -o $(GOBIN)/simulator ./cmd/simulator/*.go || exit

fmt:
	gofmt -w .

default: build

.PHONY: dependencies test fmt benchmark simulate