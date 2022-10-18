GOBASE=$(shell pwd)
GOBIN=$(GOBASE)/bin

GOPACKAGES = $(shell go list ./...)

dependencies:
	go mod download

test: dependencies
	@go test -v $(GOPACKAGES)

race: dependencies
	@go test -race $(GOPACKAGES)

coverage: 
	@go test -coverprofile cover.out $(GOPACKAGES)

benchmark: dependencies fmt
	@go test $(GOPACKAGES) -bench=. -benchmem -run=^#

parallel: dependencies fmt
	@go test github.com/smartcontractkit/ocr2keepers/internal/keepers -bench=BenchmarkCacheParallelism -benchtime 20s -mutexprofile mutex.out -run=^#

simulator: dependencies fmt
	go build -o $(GOBIN)/simulator ./cmd/simulator/*.go || exit

# This builds the simulator and runs it using the test contract we submitted for audit and a goerli rpc node
run-sim: simulator
	./bin/simulator --contract 0xB1cD4FC0B3Ad15CC5E6811637a36C635E722a6f9 --rpc-node https://BA5WGSISNDP2HSTTGDI7:V2BBPTFE5GNKEMBGKDS3JTZY3KGKSI7UTS6QDNHD@b268a16e-231d-4b17-98aa-5873a48e25e7.ethereum.bison.run

fmt:
	gofmt -w .

default: build

.PHONY: dependencies test fmt benchmark simulate
