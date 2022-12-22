# IPEHR Smart-Contracts
Here located all IPEHR contracts. At now we have contracts: 

- Direct delivery - is a task in a chainlink that accepts requests from outside, for a small fee in link tokens, listens to the operator's contract and returns the result from the statistics server.
    - [Config](chainlink-job-configs/direct-statistics.toml)
    - [Example of consumer contract](contracts/DirectConsumer.sol)
- Scheduled delivery - a task in the chainlink that updates the storage contract with fresh data according to the schedule. Other contracts can make shareware requests to contract data. (An example of a consumer contract is also available).
    - [Config](chainlink-job-configs/cron-statistics.toml)
    - [Storage contract](contracts/StatisticsContract.sol)
    - [Example of consumer contract](contracts/StatisticsConsumer.sol)

### Build
```
npm install
npm run build
```

After that you can found all json in `artifacts/contracts` folder.

### Publish/Usage
You can check [helper scripts](scripts/README.md) with guide how to deploy all contracts & config jobs in chainlink.
