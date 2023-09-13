# Simulator V3

The goal of the simulator is to complete a full run of the automation plugin
without using a chain, p2p network, RPC providers, or chainlink node as
dependencies. What is being tested in this simulator is how the plugin 
interfaces with `libOCR` and how multiple instances interact to achieve a quorum
on tasks.

Use this tool to validate the plugin protocol **only** since the chain and
network layers are both fully simulated and do not match real instances 1:1.

The simulator uses runbooks to control some of the underlying simulations such
as p2p network latency, block production, upkeep schedules, and more.

## Profiling

Start the service in one terminal window and run the pprof tool in another. For more information on pprof, view some docs [here](https://github.com/google/pprof/blob/main/doc/README.md) to get started.

```
# terminal 1
$ ./bin/simv3 --pprof --simulate -f ./simulation_runbooks/runbook_eth_goerli_mild.json

# terminal 2
$ go tool pprof -top http://localhost:6060/debug/pprof/heap
```

## Usage

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
$ ./bin/simv3 --simulate -f ./simulation_runbooks/runbook_eth_goerli_mild.json
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