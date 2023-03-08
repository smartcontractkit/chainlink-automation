# Security and Safety Guarantees
Security relates to the effects of malicious inputs causing negative impact on
the Automation network. Safety guarantees relate to assurances that upkeeps will
only be performed when they are expected to be.

## Secure Inputs
All data coming from other nodes should be considered malicious by default and
undergo thorough inspection and validation.

## User configurations
The design goal of OCR Automation is that the protocol should provide all 
required safety guarantees that user configurations affecting upkeep
performability are noticed and acted on collectively by the network in a 
confined time frame.

- upkeep cancellation
- upkeep check data changes
- upkeep perform gas changes

## State changes
Perform executions will change the state of upkeeps as time progresses. The 
network should provide all necessary assurances that observed states are 
accurate and fall under quorum.

- last perform
- check data
- perform data
- block number checked at
- block hash at check time