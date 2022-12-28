# IPEHR-stat

This repository contains a service for pushing statistical data available in the IPEHR system to public space using Chainlink network.
The service implements an open API with specified metrics.
The data is collected and processed by accessing to [IPEHR-blockchain-indexes](https://github.com/bsn-si/IPEHR-blockchain-indexes) smart contracts.

Medical statistics can be collected by the service through:
- transaction analysis of contracts [IPEHR-blockchain-indexes](https://github.com/bsn-si/IPEHR-blockchain-indexes)
- periodic direct invocation of contract methods [IPEHR-blockchain-indexes](https://github.com/bsn-si/IPEHR-blockchain-indexes)
- making AQL queries to IPEHR-gateway

For demonstration purposes, the following metrics are implemented:
- number of patients registered in the system over all time;
- number of patients logged in the system for a specified period;
- number of EHR documents registered in the system;
- number of EHR documents registered in the system for a given period;

[API Documentation](https://stat.ipehr.org/swagger/index.html)

## Local deployment

### Go 
Please follow installation instructions provided [here](https://go.dev/doc/install).

### Clone this repo

```
git clone https://github.com/bsn-si/IPEHR-stat
```

### Build IPEHR-gateway

```
cd ./IPEHR-stat
go build -o ./bin/ipehr-stat cmd/main.go
```

### Change config parameters

Write your settings in `config.json`. It can be based on `config.json.example`.  
The actual addresses of deployed contracts can be found [here](https://github.com/bsn-si/IPEHR-blockchain-indexes/blob/develop/deploys.md).

### Run IPEHR-stat

```
./bin/ipehr-stat -config=./config.json
```

## Usage examples

Direct consumer delivery:
[![video-m5-1](https://user-images.githubusercontent.com/98888366/209851585-3ecf965f-0f71-49fe-a35e-25b4e3641c8b.png)](https://media.bsn.si/ipehr/video-m5-1.mp4)

Scheduled delivery:
[![video-m5-2](https://user-images.githubusercontent.com/98888366/209851873-ffe97a94-bc75-43fe-baa2-eba73a36744c.png)](https://media.bsn.si/ipehr/video-m5-2.mp4)
