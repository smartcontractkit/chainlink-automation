# Automation Plugin Design

The following document describes the architecture of Autmation plugin (`v2.1`) for `OCR3`.

## Overview

The Automation plugin is designed to provide a way to automate smart contract interaction.

As part of the transition to `OCR3`, several components were separated to allow for more
modularity in the plugin as new features are researched and added, e.g. multiple trigger types.

### Components

### Plugin

Process OCR instances in isolated thread/s.

#### Ticker

Provides a repeated sequence of ticks (block, time, etc.), where each tick triggers the associated observer.

#### Observer

Produces list of performable upkeeps from a source of unknown state, e.g. logs, blocks, etc.
Uses a Runner to parallelize the execution of upkeeps.

#### Runner

Parallelizes the execution pipeline for checking multiple upkeeps with the corresponding input data.
Maintains an internal cache to avoid duplicated checks.

#### Registry

Stores information of upkeeps, provides list of active upkeeps for upkeep type with configurations.
Provides the pipeline entry points which includes check, perform, mercury calls and maintains limited cache for mercury state and cooldowns.

Limited retry scope for mercury calls. Performs RPC batch calls by default (10 upkeeps need to be checked, they will be part of 1 batch).

#### Results Queue/Stager (TBD)

Provides a single source of performable upkeeps, with limited prioritization methods such as shuffle or FIFO (depending on observer settings).

Supports automated removal after timeout (TTL with self-cleaning).

### General Flow

The following diagram describes a simplified, general flow of upkeep data within the different components:

![Automation Block Diagram](./images/block_ocr3_base.jpg)

### Conditional Observer

The following diagram describes the flow of upkeep data in confitional triggers flow:

![Conditional Observer Diagram](./images/block_ocr3_conditional_observer.jpg)

In the case of conditional upkeeps, a block sampler and coordinator are introduced and the plugin uses samples and blocks to create coordinated ticks that feed the results queue.

Specific to conditionals, items on the results queue must be replaced when a coordinated tick is produced and shuffled when they are accessed.

The **Coordinator** ([see doc](./COORDINATOR.md)) ensures that all observations from a conditional observer are progressing in block and no duplicates are executed.

The **Sampler** provides latest block and samples active upkeeps on a cadence (every x blocks).

**Coordinated Ticker** takes quorum data from plugin on coordinated block and deduped upkeep samples.
Triggers conditional observer to do 'verification' step.

### Log Observer

The following diagram describes the flow of upkeep data in log triggers flow:

![Log Observer Diagram](./images/block_ocr3_log_observer.jpg)

In the case of log triggers, results flow is unidirectional through a FIFO queue and to the plugin.

The **Log Provider** is responsible for serving logs of active upkeeps, and act as a data source for the log observer, 
that will use the log data as an input for each upkeep pipeline.
Behind the scenes (part of `EvmRegistry`), log filters are managed (un/registered) as upkeeps are added and removed from the registry.
