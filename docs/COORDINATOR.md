# Coordinator

The coordinator is currently for conditional upkeeps to ensure that upkeep 
performs are not duplicated, perform status is maintained between OCR rounds
and new block state, and upkeep performs are always progressing in block number
regardless of reverts or performs.

![Block Progression](images/conditional_coordinator_block_progression_diagram.jpg)