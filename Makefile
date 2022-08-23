GOPACKAGES = $(shell go list ./...)

dependencies:
	go mod download

test: dependencies
	@go test -v $(GOPACKAGES)

coverage: 
	@go test -coverprofile cover.out $(GOPACKAGES)

benchmark: dependencies fmt
	@go test $(GOPACKAGES) -bench=. -benchmem -run=^#

fmt:
	gofmt -w .

default: build

.PHONY: dependencies test fmt benchmark