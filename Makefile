GOPACKAGES = $(shell go list ./...)

dependencies:
	go mod download

test: dependencies
	@go test -v $(GOPACKAGES)

benchmark: dependencies fmt
	@go test $(GOPACKAGES) -bench=.

fmt:
	gofmt -w .

default: build

.PHONY: dependencies test fmt benchmark