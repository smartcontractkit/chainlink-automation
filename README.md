# ocr2keepers Oracle Plugin
Initialize the plugin by creating a new Delegate

```go
delegate, err := ocr2keepers.NewDelegate(delegateConfig)
```

## Unit Testing
Unit testing is used extensively and the primary goal is to keep test coverage above 70%.

Test coverage should be rerun with every commit either by running a git hook or `make coverage` to help maintain a high level of test coverage.

It is recommended that you install the git hooks so that the automated tooling is part of your workflow. Simply run:

```
cp -r .githooks/ .git/hooks
```

Explore test coverage per file with
```
$ go tool cover -html=cover.out
```

## Benchmarking
Benchmarking helps identify general function inefficiencies both with memory and processor time. Only benchmark functions that are likely to run multiple times, asynchronously, or be processor/memory intensive.

Using benchmarking consistently requires that os, arch, and cpu be kept consistent. Do not overwrite the `benchmark.txt` file unless your specs are identical.

To run benchmarking:
```
$ make benchmark | new.txt
```

To view a diff in benchmarks:
```
$ go install golang.org/x/perf/cmd/benchstat@latest
$ benchstat benchmarks.txt new.txt
```

## Simulator
The simulator exists to create 1 or more 'nodes' and run an integration test where the simulator steps through basic OCR rounds calling functions on the plugin interface (Query, Observation, Report, ShouldAcceptFinalizedReport) and logs the reports collected from each round.

The simulator does not sign reports and many other features of OCR such as report selection or slow node handling are heavily simplified.

Each 'node' is configured using the given contract address (registry) and RPC. Limit rounds, round time, and number of nodes to run a simple integration test between the plugin, contract, and OCR interface.

Use this simulator to test how the plugin responds to the time limits enforced by OCR and to verify the pipeline of data from upkeeps to reports.

### Usage

Start with building the simulator by running `make simulator`. The output will be `bin/simulator`. This can be added to your path or you can call the program directly.

To run the simulator, you can use the defaults for most inputs. The two most important are the contract address and the RPC client for calling the contract.

Example:
```
$ ./bin/simulator -contract 0x02777053d6764996e594c3E88AF1D58D5363a2e6 -rpc https://rinkeby.infura.io/v3/[your key]
```

Other options include:
- `-nodes`: integer, default 3, the number of parallel nodes to simulate. minimum of 1
- `-round-time`: integer, default 5, the time in seconds a round should take. this is a hard limit and the round context will be cancelled at the end of the round
- `-rounds`: integer, default 2, defines the number of rounds the simulator should run. 0 is unlimited
- `max-run-time`: integer, default 0, the number of seconds to run the simulation. use this in place of number of rounds to do a time based simulation instead of a round based simulation. use 0 for no limit

The simulation will end when one of the following occurs:
1. Defined number of rounds occurs
2. Defined simulation time limit is reached
3. SIGTERM syscall is encountered