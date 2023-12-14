# Simulator

The goal of the simulator is to complete a full run of the automation plugin without using a chain, p2p network, RPC
providers, or chainlink node as dependencies. What is being tested in this simulator is how the plugin interfaces
with `libOCR` and how multiple instances interact to achieve a quorum on tasks.

Use this tool to validate the plugin protocol **only** since the chain and network layers are both fully simulated and
do not match real instances 1:1.

The simulator uses runbooks to control some of the underlying simulations such as p2p network latency, block
production, upkeep schedules, and more.

## Usage

The current iteration of the simulator requires a full build before a run since the simulator doesn't run binaries of
the plugin. Instead the plugin is built within the simulator binary. The current limitation is that multiple custom
builds cannot be run as part of a combined network. All instances in the simulated network will be identical.

Outputs can be directed to a specific directory, which is advised since each instance produces its own log files. With
more than 4 instances running for long periods of time, these log files can become large. Logging is full debug
by default.

Charts are useful to visualize RPC failures and overall simulated latency, both p2p and RPC. The charts are provided
by an HTTP endpoint on localhost.

*Example*
```
$ ./bin/simulator --simulate -f ./tools/simulator/plans/simplan_fast_check.json
```

*Options*
- `--simulation-file | -f [string]`: default ./simulation_plan.json, path to JSON file defining simulation parameters
- `--output-directory | -o [string]`: default ./simulation_logs, path to output directory where run logs are written
- `--verbose [bool]`: default false, make output verbose (prints logs to output directory)
- `--simulate [bool]`: default false, run simulation and output results
- `--charts [bool]`: default false, start and run charts service to display results
- `--pprof [bool]`: default false, run pprof server on simulation startup
- `--pprof-port [int]`: default 6060, port to serve pprof profiler on

### Assertions and Exit Status

Exit status of any simulation is dictated by assertions on whether or not eligible upkeeps were performed. Both positive
and negative assertions are possible by configuring upkeep perform expectations. A negative assertion would be one where
it is expected that no upkeeps are performed, which can be accomplished by altering network variables such as the
simulated RPC or not sending an OCR config event.

All simulations will return an exit status, which is used in CI runs to indicate success or failure of a simulation
plan.

## Profiling

Start the service in one terminal window and run the pprof tool in another. For more information on pprof, view some
docs [here](https://github.com/google/pprof/blob/main/doc/README.md) to get started.

```
# terminal 1
$ ./bin/simulator --pprof --simulate -f ./tools/simulator/plans/simplan_fast_check.json

# terminal 2
$ go tool pprof -top http://localhost:6060/debug/pprof/heap
```

## Simulation Plan

A simulation plan is a set of configurations for the simulator defined in a JSON file and is designed to produce a
consistent simulation between runs. Be aware that there is some variance in simulation outcomes from the same simulation
plan due to randomness in the `rpc`, `blocks`, or `p2pNetwork` configurations. Events are precise relative to block
production.

### Node

The instantiation of a node involves creating simulated dependencies and providing them to the delegate constructor for
the plugin.

*node*
[object]

Configuration values that apply to setting up nodes in the network.

*node.totalNodeCount*
[int]

Total number of nodes to run in the simulation. Each node is connected via a simulated p2p network and provided an
isolated contract/RPC simulation.

*node.maxNodeServiceWorkers*
[int]

Service workers are used to parallelize RPC calls to be able to process more in a short time. This number is to set the
upper limit on the number of service workers per simulated node.

*node.maxNodeServiceQueueSize*
[int]

Max queue size for sending work to service workers. This should be deprecated soon.

### P2PNetwork

The p2p network simulation does not include any tcp/udp networking layers and only serves the perpose of inter-node
communication. The simulated network can be configured to simulate nodes operating on the same hardware or in close
proximity or configured to simulate nodes spread across a large physical distance. A `maxLatency` of `300ms` might
simulate the physical distance of nodes operating in Paris and Singapore, for example.

*p2pNetwork*
[object]

Configure the simulated p2p network.

*p2pNetwork.maxLatency*
[int]

The maximum amount of time a message should take to be sent in the simulated p2p network. This is calculated by taking
a random number between 0 and the provided latency.

### RPC

A simulated RPC is the connection layer between the block producer and the node. In a real-world environment, the block
production can be imagined as a singular source and RPCs independently read the state of block production. In this way,
each RPC can have a different 'view' of the singular block source.

Each node gets an isolated instance of a simulated RPC. The role the RPC plays is to surface changes made to the
singular block source such as new upkeeps being created, ocr configs being committed, or logs being emitted.

*rpc*
[object]

Configure the behavior of the simulated RPC. There is currently a limit of a single RPC simulation configuration and
applies to all instances.

*rpc.maxBlockDelay*
[int]

The maximum delay in in milliseconds that an RPC would deliver a new block.

*rpc.averageLatency*
[int]

The average response latency of a simulated RPC call. All latency calculations have a baseline of 50 milliseconds with
an added latency calculated as a binomial distribution of the configuration where `N = conf * 2` and `P = 0.4`. 

*rpc.errorRate*
[float]

The probability that an RPC call will return an error. `0.02` is `2%`. RPC providers are essentially cloud services that
have potential failures. Use this configuration to simulate a flaky RPC provider.

*rpc.rateLimitThreshold*
[int]

Total number of calls per second before returning a rate limit response from the simulated RPC provider.

### Blocks

Simulated blocks in the context of the simulator are only containers of events that apply to the network of nodes and
that are provided to the network on a defined cadence. The concept of signatures, and hashes doesn't apply. Where the
term `hash` is used, the value is likely either a randomly generated value or a value derived from some block data.

*blocks*
[object]

Configuration object for simulated chain. The chain is a coordinated block producer that feeds each simulated RPC by a
dedicated channel. Each simulated RPC can receive blocks at different times.

*blocks.genesisBlock*
[int]

The block number for the first simulated block.

*blocks.blockCadence*
[string]

The rate at which new blocks are created. Formatted as time.

*blocks.blockCadenceJitter*
[string]

Block cadenece jitter is applied to block production such that each block is not produced exactly on the cadence.

*blocks.durationInBlocks*
[int]

A simulation only runs for this defined number of blocks. The configured upkeeps are applied within this range.

*blocks.endPadding*
[int]

The simulated chain continues to broadcast blocks for the end padding duration to allow all performs to have time to be
completed. The configured upkeeps do not apply to this block set.

### Events

All events are applied to blocks as they are produced. Each event contains at least a `type` that describes how to 
process the event and a `eventBlockNumber` which defines the block in which to apply the event. Many events are
singular. Some events are generative in that multiple events are generated from a single configuration.

Generative events, such as `generateUpkeeps`, allows a more collapsed JSON config. The specific case of generating
upkeeps will create `count` upkeep create events for the same `eventBlockNumber`.

*events*
[array[object]]

Config events change the state of the network and at least 1 is required to start the network configuration. Each event
is broadcast by the simulated chain at the block defined.

Every event has some common properties:

*type*
[string:required]

Determines the event type. Options include `ocr3config`, `generateUpkeeps`, `logTrigger`

*eventBlockNumber*
[int:required]

Block number to commit this event to block history.

*comment*
[string:optional]

Optional reference value. Not output on logs.


#### OCR 3 Config

- type: `ocr3config`

An event with the type `ocr3config` indicates that a new network configuration was committed to the block history. In a
real-world scenario, this would be an on-chain transaction that emits a log. In the simulation, the event is a specific
type and is recognized by the simulated RPC.

TODO: describe OCR config values and how they are provided

*encodedOffchainConfig*
[string]

The encoded config does not currently enforce an encoding type. An OCR off-chain config is an array of bytes allowing
the encoding to be anything. This configuration property should be the value already encoded as the plugin expects. In
the case of JSON, include character escaping such as `{\"version\":\"v3\"}`.

#### Generate Upkeeps

- type: `generateUpkeeps`

Multiple upkeep events can be generated by using this event type. An upkeep event simulates an upkeep being added to a
registry and made active. The only states relevant to the simulator regarding registered upkeeps are: is it active, and
is it eligible?

An upkeep becomes active on the `eventBlockNumber` where it is committed to the block history. From that point,
eligibility begins to apply, which is defined by the `eligibilityFunc`. No other events will apply to an upkeep until
after the upkeep becomes active in the block history, which includes `logTrigger` type events. 

The `eligibilityFunc` allows a basic linear function to be supplied indicating when, relative to the trigger block, an
upkeep should be eligible.

Example:

func: `2x + 1`
active at block: `100`
final block: `500`

```
let start = 100;
let end = 500;
let i = 0;
let next = 0;

let eligible = [];

while next < end {
    if next > start {
        eligible.push(next);
    }

    let y = (2 * i) + 1;

    next = start + Math.round(y);

    i++;
}

// the eligibility function makes the upkeep eligible every 2 blocks with a relative
// offset of 1 to the start block
// eligible: [101, 103, 105, 107, ...]
```

The `offsetFunc` advances the `start` point for each generated upkeep to ensure eligibility doesn't overlap for each
generated upkeep.

Special options such as `always` and `never` are also available.

*count*
[int]

Total number of upkeeps to generate for the event configuration.

*startID*
[int]

ID to reference the upkeep in the config. The UpkeepID output will be different.

*eligibilityFunc*
[string]

Simple linear equation for generating eligibility. Also allowed are `always` and `never`.

*offsetFunc*
[string]

Simple linear equation for eligibility start offset. Allowed to be empty when `eligibilityFunc` is `always` or `never`.

*upkeepType*
[string]

Options are `conditional` and `logTrigger`.

*logTriggeredBy*
[string]

This value applies to a log trigger type upkeep and is the reference point for a `logTrigger` event. If this value
matches the `triggerValue` of a `logTrigger` event, this upkeep is 'triggered' by the log trigger event.

*expected*
[string]

An upkeep may or may not be expected to perform even though the upkeep is eligible. This may be due to other configured
components of the simulation and provides a way to do negative assertions. That is, an assertion that upkeeps DID NOT
perform. Available options include `all` and `none`. Default is `all`.

#### Log Events

- type: `logTrigger`

A simulated log event does only simulates the existence of a real-world log and the value for matching to an upkeep
trigger. Once a log event is active in the block history, it can 'trigger' any active and eligible `logTrigger` type
upkeeps with a matching `triggerValue`.

*triggerValue*
[string]

Value used to 'trigger' upkeeps.