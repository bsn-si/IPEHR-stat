### IPEHR Oracle
A subproject that contains all contracts and scripts for publishing and interacting with the oracle.

### About
We writen a set of contracts for providing access to statistics on the blockchain, based on ether and chainlink, along with storage contracts, and example contracts.

We have several types of data delivery about statistics - this is direct delivery on request, and request from storage - which receives data on a schedule.

- Direct delivery - is a task in a chainlink that accepts requests from outside, for a small fee in link tokens, listens to the operator's contract and returns the result from the statistics server.
- Scheduled delivery - a task in the chainlink that updates the storage contract with fresh data according to the schedule. Other contracts can make shareware requests to contract data. (An example of a consumer contract is also available).

Also, a set of scripts was written for the provided contracts to simplify interaction and testing, with which you can publish and call contracts, as well as view some of the chainlink statuses and replenish its balance.

### How To
Several manuals are available for working with contracts and oracle.

- [Setup chainlink node](chainlink/README.md)
- [Contracts with examples](contracts/README.md)
- [How to use scripts for interaction](scripts/README.md)
