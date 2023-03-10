# Cost Analysis of Running a Node

This cost analysis is intended to attach cost estimations to the design
decisions contained in this proposal. It does not cover all costs of running a 
chainlink automation node, nor is it highly accurate for estimating actual 
costs. The objective of this analysis is to make an informed design decision
based on a cost/revenue model.

## Observations

Since an observation contains multiple upkeep states replicated over multiple
blocks with all data necessary to perform any of the observed upkeeps, the 
size in bytes of an observation can be in the range of 12 MB under the following
assumptions. The cost of broadcasting observations as both a leader and a
follower is estimated below.

**Assumptions**
- the report gas limit is 15 million
- a single upkeep gas limit is 5 million
- the evm estimates 16 gas per non-zero byte of data
- an identifier is a uint64 value (8 bytes)
- an active window is 12s
- throughput is 3 upkeeps per second average
- report throughput is 1 report per second average
- network is a 12 node cluster
- each node has a 1/12 chance of being leader (leader is required to broadcast)

---

**Egress Calculation per Node**

Values:
- `o`: size of an observation per round => `12 MB`
- `n`: number of rounds per month => `86,400 * 30 = 2,592,000`

```
Follower egress formula in GB / month (every 12th round a node is a leader):
((o * n) - (o * n/12)) / 1000

Leader egress formula in GB / month (a leader broadcasts to all nodes):
o * n / 1000

Together
(((o * n) - (o * n/12)) + (o * n)) / 1000 = 59,616 GB / month
```

---

**Egress Cost by Data Center Provider**

Egress costs for data centers (ingress costs are typically free, and highest 
cost regions were chosen for the example calculation):
- GCP: $0.15 / GB (from Indonesia or Oceania2 to any other Google Cloud region) [Google Pricing](https://cloud.google.com/vpc/network-pricing)
- AWS: $0.12 / GB (highest estimated) [AWS Pricing](https://aws.amazon.com/blogs/apn/aws-data-transfer-charges-for-server-and-serverless-architectures/)
- Azure: $0.18 / GB (From South America to any destination) [Azure Pricing](https://azure.microsoft.com/en-us/pricing/details/bandwidth/)

Therefore a single node running in one of the above data center providers might
incur a maximum egress cost of:

```
59,616 * 0.18 = $10,730.88
```

| $ / GB | Total $ / Month |
| --- | ---: |
| $0.01 | $596.16 |
| $0.02 | $1,192.32 |
| $0.05 | $2,980.80 |
| $0.10 | $5,961.60 |
| $0.15 | $8,942.40 |
| $0.20 | $11,923.20 |

---

**Optimal Results**

Some of the egress costs can be as low as $0.01 / GB depending on the origin 
and destination regions. Therefore, the above is intended to be a maximum 
estimate on the cost to run an instance of the automation plugin with the 
intended design within the context of data egress on ***observations only***.

With data compression on observations using [zstd](https://pkg.go.dev/github.com/klauspost/compress/zstd#section-readme), an optimal compression ratio 
could be upwards of 1/10, which reduces egress by an order of magnitude. If 
egress costs are assumed to be the lower of the estimates of $0.01 / GB, that
lowers the cost by an order of magnitude. Therefore, a cost estimate under 
optimal conditions might be in the range of $120 / month or less to support the
observation design mentioned.

Values:
- `o`: size of an observation per round => `1.2 MB`
- `n`: number of rounds per month => `86,400 * 30 = 2,592,000`

```
Together
(((o * n) - (o * n/12)) + (o * n)) / 1000 = 5,961.6 GB / month
```

| $ / GB | Total $ / Month |
| --- | ---: |
| $0.01 | $59.62 |
| $0.02 | $119.23 |
| $0.05 | $298.08 |
| $0.10 | $596.16 |
| $0.15 | $894.24 |
| $0.20 | $1,192.32 |

## Reports

A single report can contain up to the maxumim number of bytes that would fall
under the max report gas limit. This is a configurable value, but the assumptions
are provided below for calculations. 

Note that reports are first sent to the leader from each follower and finally
the leader could broadcast the same report back to all followers. This incurs
an egress cost in both directions.

**Assumptions**
- the report gas limit is 15 million
- all reports are full
- the evm estimates 16 gas per non-zero byte of data
- report throughput is 1 report per second average
- network is a 12 node cluster
- each node has a 1/12 chance of being leader (leader is required to broadcast)

---

**Egress Calculation per Node**

Values:
- `o`: size of a report per round => `940 KB`
- `n`: number of rounds per month => `86,400 * 30 = 2,592,000`

```
Follower egress formula in GB / month (every 12th round a node is a leader):
((o * n) - (o * n/12)) / 1000

Leader egress formula in GB / month (a leader broadcasts to all nodes):
o * n / 1000

Together
(((o * n) - (o * n/12)) + (o * n)) / 1000 = 4,670 GB / month
```

Therefore a single node running in one of the above data center providers might
incur a maximum egress cost of:

```
4,670 * 0.18 = $840.60
```

| $ / GB | Total $ / Month |
| --- | ---: |
| $0.01 | $46.70 |
| $0.02 | $93.40 |
| $0.05 | $233.50 |
| $0.10 | $466.99 |
| $0.15 | $700.49 |
| $0.20 | $933.98 |

**Optimal Results**

Reports can also be compressed when sending between nodes, however a report
cannot be compressed when sending to chain. The latter is not within scope of 
the plugin, though. The same compression potential could apply similarly in
this case with a high end ratio of 1/10 and a low end of 1/3.

## Other Messages and Egress

Other messages exist as part of the libOCR protocol, but are not significant 
enough in size to alter the estimates much. If these values need to be added,
consider a 5% padding to cover them.

Communication between the node and an RPC is also not included. Most RPC used
by the chainlink node use the wss protocol and are relatively lightweight. The
primary egress to consider is sending a report to chain, which should follow
similar egress calculations as `Report`.

## Summary

For a network producing a report that will cost 15 million gas every second will
very roughly cost the following in egress:

| $ / GB | Observation $ / Month | Report $ / Month | Total $ / Month |
| --- | ---: | ---: | ---: |
| $0.01 | $596.16 | $46.70 | $642.86 |
| $0.02 | $1,192.32 | $93.40 | $1,285.72 |
| $0.05 | $2,980.80 | $233.50 | $3,214.30 |
| $0.10 | $5,961.60 | $466.99 | $6,428.59 |
| $0.15 | $8,942.40 | $700.49 | $9,642.89 |
| $0.20 | $11,923.20 | $933.98 | $12,857.18 |

Estimates with data compression (1/10 on observations; 1/3 on reports):

| $ / GB | Observation $ / Month | Report $ / Month | Total $ / Month |
| --- | ---: | ---: | ---: |
| $0.01 | $59.62 | $15.57 | $75.19 |
| $0.02 | $119.23 | $31.13 | $150.36 |
| $0.05 | $298.08 | $77.83 | $375.91 |
| $0.10 | $596.16 | $155.66 | $751.82 |
| $0.15 | $894.24 | $233.50 | $1,127.74 |
| $0.20 | $1,192.32 | $311.33 | $1,503.65 |
