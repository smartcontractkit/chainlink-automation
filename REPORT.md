# What is a Report?
In the context of OCR Automation, a report contains all data required to perform
one or more upkeeps per user specifications.

## Pending Transmission
Once a report is generated and agreed on by an OCR Automation network of nodes,
it should be considered as pending along with all its contents. This means that
any upkeep contained in a pending report should not be included in any
observations until one of the following occurs:

- either a report log is received
- an upkeep perform log is received
- a per upkeep timeout elapses

Pending status is state that individual nodes should track to assist in making
accurate observations that other nodes will agree with. Report generation should
never interact with this state.

## Accepting a Report
A report should be accepted if all upkeeps it identifies in the report are not
pending. Once the report passes this test, all upkeeps should be marked as
pending to all observers.

## Transmitting a Report
A report should be transmitted if all the upkeeps it identifies in the report
are not pending or the report identifier is unlocked. Different version will
need to be supported. For the former, 