# Rearchitecture Motivations

## Decouple Network Calls from OCR

The OCR protocol is very sensitive to cadence. Grace periods and progress 
timeouts can force the entire network to start over on new rounds if the cadence
isn't progressing as expected. By consequence, network calls synchronous with
the pipeline are also subject to the sensitive cadence.

Issues with network calls in OCR:

1. they add unnecessary delays to the pipeline
2. they introduce the possibility of errors or delays that slow throughput
3. network calls are constrained to the OCR cadence which won't play well with some APIs

The network calls to this point have been isolated to RPC calls, but Offchain 
Lookup will bring more variance. Each upkeep can have 1 or more external network
calls that will need to be run synchronously with checking the upkeep. While
RPC providers sometimes attempt to focus on performance, many external data APIs
don't. This will add significant time overhead to checking upkeeps.

API latencies in offchain lookup could even be considered a potential attack
vector for halting the pipeline. By registering enough upkeeps that 
intentionally use the maximum allowable network timeout, an attacker could block
the pipeline by flooding the network with long running upkeep checks.

### Proposed Solution

Consolidating all network calls into a separate process that feeds observations
to the OCR pipeline insulates it from network timeouts and errors. This can free
up configurations to be more and more complex over time and not be constrained
by the OCR cadence.

With the concept of an `active window` of observations and consolidating network
calls to a separate process, the protocol is afforded an insulation to chain
state variance and RPC failures. Within an `active window` is a collection of
multiple reads made by an RPC+API combo. If a single window is able to contain
5 instances of an observed upkeep, that's 5 tries for that node within the given
time frame to produce a successful observation.

## Code Organization

Organization in this case falls to two categories: separation of responsibility
and modularity. In short, the first helps reduce the complexity of each layer,
the second improves the ability to add onto the product over time, and finally 
both reduce scope of future changes.

- Separation of responsibility
  - need to improve maintainability by isolating scope of changes. With clear boundaries, future additions or changes to specific layers are simplified and reduced in scope
  - Defining clear boundaries in functional responsibility enhances testing by reducing complexity and scope of tests.
  - This separation also reduces mental load when working on specific layers.
  - The current design is tightly coupled. Additions will require significant refactors
  - contract changes can have a very direct impact on the plugin structure. The concept of reading logs to determine pending state can very much change from one contract iteration to the next. this is high risk for future improvements or additions.
- A refactor is already necessary to clean up externally consumed types and interfaces. This was not part of the initial design and was discovered well into production.

## General Scope

The proposal involves 3 things: rearchitecting the protocol layer, refactoring
the external interface of the plugin, and reusing existing tools in the current
plugin to match the new interface.

### something old

The method of polling at every block doesn't change. Currently we use a worker
group and batching to do these checks as quickly as possible. Everything within
checking upkeeps stays the same.

Most of the new process will rely on caches with timeouts on specific values.
This is already a utility component extensively used by the current 
implementation.

Telemetry already doesn't unpack observations or reports due to the current
issues of external concrete types and interfaces. Nothing changes here, however
the changes will bring stabilized types and interfaces.

### something new

The automation protocol changes to produce a different observation structure
which impacts report generation. A new way of generating reports directly from
observations without further network calls would need to be created.

### something borrowed

The general method of creating observations from checked upkeeps will need to be
modified to match the new observation type. It's a borrowing in the sense that
most of the logic doesn't change. Only the output types change.

The registry implementation that keeps track of active upkeeps and runs upkeep
checks larely won't need changes. Return types might need changes along with 
interface implementations, but should be very minor.

The concept of the filter will be shifted to the responsibility of an observer.
Much of the logic for tracking upkeeps in a pending state won't change.

Encoding/decoding of upkeep identifiers, reports, etc. will be consolidated into
versioned encoders that will be paired with interfaces expected by the plugin.
These will be dependencies injected into the plugin instead of being integrated
into the plugin itself.

### something blue

Not sure what to put here, but it's required for the rhyme. 

***a sixpence in your shoe...***

# Tradeoffs

- larger observations passed over the network
  - since the perform data must be sent along with the upkeep identifier, and multiple copies of the same upkeep are also included, a single observation is much larger than the previous design
  - the impact of this can be reduced by using data compression on observations before broadcasting to the network
  - a cost analysis is included to give more insight into this tradeoff
- state observation becomes a registry implementation responsibility
  - with the tight coupling in the current design, state observation and comparison is shared by both the registry and the OCR interface implementation
  - this change would move all state observation and tracking into the scope of the contract registry implementation itself. each contract would need to implement its own upkeep state tracking.