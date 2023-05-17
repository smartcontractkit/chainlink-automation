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

The goal of the simulator is to complete a full run of the automation plugin
without using a chain, p2p network, RPC providers, or chainlink node as
dependencies. What is being tested in this simulator is how the plugin 
interfaces with `libOCR` and how multiple instances interact to acheive a quorum
on tasks.

Use this tool to validate the plugin protocol **only** since the chain and
network layers are both fully simulated and do not match real instances 1:1.

The simulator uses runbooks to control some of the underlying simulations such
as p2p network latency, block production, upkeep schedules, and more.

### Usage

The current iteration of the simulator requires a full build before a run as the
simulator doesn't run binaries of the plugin, but instead the plugin is built
within the simulator binary. The current limitation is that multiple custom
builds cannot be run as part of a combined network. All instances in the 
simulated network will be identical.

Outputs can be directed to a specific directory, which is advised since each
instance produces its own log files. With more than 4 instances running for
long periods of time, these log files can become large. Logging is full debug
by default.

Charts are useful to visualize RPC failures and overall simulated latency, both
p2p and RPC. The charts are provided by an HTTP endpoint on localhost.

*Example*
```
$ ./bin/simv2 --simulate -f ./simulation_runbooks/runbook_eth_goerli_mild.json
```

*Options*
- `--simulation-file | -f [string]`: default ./runbook.json, path to JSON file defining simulation parameters
- `--output-directory | -o [string]`: default ./runbook_logs, path to output directory where run logs are written
- `--simulate [bool]`: default false, run simulation and output results
- `--charts [bool]`: default false, start and run charts service to display results
- `--pprof [bool]`: default false, run pprof server on simulation startup
- `--pprof-port [int]`: default 6060, port to serve pprof profiler on

### Runbook Options

A runbook is a set of configurations for the simulator defined in a JSON file.
Each property is described below.

*nodes*
[int]

Total number of nodes to run in the simulation. Each node is connected via a
simulated p2p network and provided an isolated contract/RPC simulation.

*maxNodeServiceWorkers*
[int]

Service workers are used to parallelize RPC calls to be able to process more in
a short time. This number is to set the upper limit on the number of service
workers per simulated node.

*maxNodeServiceQueueSize*
[int]

Max queue size for sending work to service workers. This should be deprecated
soon.

*avgNetworkLatency*
[int]

The total amount of time a message should take to be sent in the simulated p2p
network. This is an average and is calculated by taking a random number between
0 and the defined latency.

*rpcDetail*
[object]

This object is a container for RPC related configurations. There is currently a
limit of a single RPC simulation configuration and applies to all instances.

*rpcDetail.maxBlockDelay*
[int]

The maximum delay in in milliseconds that an RPC would deliver a new block.

*rpcDetail.averageLatency*
[int]

The average response latency of a simulated RPC call. All latency calculations
have a baseline of 50 milliseconds with an added latency calculated as a
binomial distribution of the configuration where `N = conf * 2` and `P = 0.4`. 

*rpcDetail.errorRate*
[float]

The probability that an RPC call will return an error. `0.02` is `2%`

*rpcDetail.rateLimitThreshold*
[int]

Total number of calls per second before returning a rate limit response from the
simulated RPC provider.

*blockDetail*
[object]

Configuration object for simulated chain. The chain is a coordinated block
producer that feeds each simulated RPC by a dedicated channel.

*blockDetail.genesisBlock*
[int]

The block number for the first simulated block. Formatted as time.

*blockDetail.blockCadence*
[string]

The rate at which new blocks are created. Formatted as time.

*blockDetail.blockCadenceJitter*
[string]

Some chains produce blocks on a well defined cadence. Most do not. This
parameter allows some jitter to be applied to the block cadence.

*blockDetail.durationInBlocks*
[int]

A simulation only runs for this defined number of blocks. The configured upkeeps
are applied within this range.

*blockDetail.endPadding*
[int]

The simulated chain continues to broadcast blocks for the end padding duration
to allow all performs to have time to be completed. The configured upkeeps do
not apply to this block set.

*configEvents*
[array[object]]

Config events change the state of the network and at least 1 is required to
start the network configuration. Each event is broadcast by the simulated chain
at the block defined.

*configEvents.triggerBlockNumber*
[int]

The block to broadcast the event on. This block should be after the genesis
block and before the final simulation block.

*configEvents.offchainConfigJSON*
[string]

Stringified JSON for off-chain configuration.
