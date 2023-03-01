# OCR2 Automation Oracle Plugin
The intention of this package is to be a library that a joining layer between the OCR2 protocol, the chainlink node, and the automation pipeline. The primary functions involve converting chain state into OCR2 observations and utility functions for producing reports.

## Interface Implementation Requirements
This package provides interfaces instead of distict types to allow for future extensibility and to isolate relevant functionality.

## Observation Detail
An observation of upkeep state can be simplified to which upkeeps are eligible and which are processing. Those which are ineligible are not necessary to broadcast in an observation. The upkeeps that are in progress can be agreed on by a quorum by using a simple identifier. This can be an integer or string id. The upkeeps that are eligible require a deep comparison to reach agreement. For example, if one node checks an upkeep at block 1 and another node checks the same upkeep at block 2, the results could not be determined to be equal.

```go
delegate, err := ocr2keepers.NewDelegate(delegateConfig)
```

# Development
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

## Logging
To reduce dependencies on the main chainlink repo, all loggers are based on the default go log.Logger. When using the NewDelegate function, a new logger is created with `[keepers-plugin] ` prepending all logs and includes short file names/numbers. This logger writes its output to the ocr logger provided to the delegate as `Debug` logs.

The strategy of logging in this repo is to have two types of outcomes from logs:
1. actionable - errors and panics (which should be handled by the chainlink node itself)
2. debug info - extra log info about inner workings of the plugin (optional based on provided ocr logger settings)

If an error cannot be handled, it should be bubbled up. If it cannot be bubbled up, it should panic. The plugin shouldn't be concerned with managing runtime errors, log severity, or panic recovery unless it cannot be handled by the chainlink node process. An example might be a background service that is created a plugin startup but not managed by the chainlink node. If there is such a service, it should handle its own recovery within the context of a Start/Stop service.

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
$ ./bin/simulator --contract 0x02777053d6764996e594c3E88AF1D58D5363a2e6 --rpc-node https://rinkeby.infura.io/v3/[your key]
```

Other options include:
- `--nodes | -n [int]`: default 3, the number of parallel nodes to simulate. minimum of 1
- `--round-time | -t [int]`: default 5, the time in seconds a round should take. this is a hard limit and the round context will be cancelled at the end of the round
- `--query-time | -qt [int]`: default 0, the max time in seconds to run an OCR Query operation. use this for a more fine-grained simulation to OCR
- `--observation-time | -ot [int]`: default 0, the max time in seconds to run an OCR Observation operation. use this for a more fine-grained simulation to OCR
- `--report-time | -rt [int]`: default 0, the max time in seconds to run an OCR Report operation. use this for a more fine-grained simulation to OCR
- `--rounds | -r [int]`: default 2, defines the number of rounds the simulator should run. 0 is unlimited
- `--max-run-time | -m [int]`: default 0, the number of seconds to run the simulation. use this in place of number of rounds to do a time based simulation instead of a round based simulation. use 0 for no limit

The simulation will end when one of the following occurs:
1. Defined number of rounds occurs
2. Defined simulation time limit is reached
3. SIGTERM syscall is encountered

### Profiling

Execute profiling using `pprof` to see things like heap allocation, goroutines, and more.
- `--pprof`: default false, add to turn on profiling
- `--pprof-port [int]`: default 6060, the port on localhost to listen for pprof requests

**Example:**
Start the service in one terminal window and run the pprof tool in another. For more information on pprof, view some docs [here](https://github.com/google/pprof/blob/main/doc/README.md) to get started.

```
# terminal 1
$ ./bin/simulator --pprof --contract 0x02777053d6764996e594c3E88AF1D58D5363a2e6 --rpc-node https://rinkeby.infura.io/v3/[your key]

# terminal 2
$ go tool pprof -top http://localhost:6060/debug/pprof/heap
```

## Simulator V2

This simulator uses OCR directly to immitate an automation network. The version uses runbooks as seen in `simulation_runbooks` to configure simulations.

### Usage

The executable takes a runbook json file to define configuration parameters.

```
$ ./bin/simv2 --simulation-file ./simulation_runbooks/runbook_eth_goerli_mild.json
```