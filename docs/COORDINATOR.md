# Coordinator

The coordinator is a component which keeps track of in-flight reports for (triggerID, upkeepID). Whenever new reports are created it stores them in memory and listens to UpkeepPerformed logs from the registry to mark them as done. Entries in coordinator also expire after a pre-configured TTL.

Coordinator is used in two places:

1. Within the transmission protocol: When a node gets its preconfigured turn within the transmission protocol, it checks the coordinator whether the report is still pending and if so attempts to transmit the report.

2. Additionally coordinator is used within preprocessing / postprocessing steps to not create a new report for the same upkeep while one is in progress. This is especially important for conditional upkeeps where a new report should be created only after the previous one has been confirmed.

3. TBD: Do we need coordinator in log trigger flow?

![Block Progression](images/conditional_coordinator_block_progression_diagram.jpg)