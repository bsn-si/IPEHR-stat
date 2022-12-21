# IPEHR-stat

This repository contains a service for providing public statistical data collected and processed from the IPEHR system. 
The service implements an open API with specified metrics.
The data is collected and processed by accessing [IPEHR-blockchain-indexes] smart contracts (https://github.com/bsn-si/IPEHR-blockchain-indexes).

Medical statistics can be collected by the service through:
- transaction analysis of contracts [IPEHR-blockchain-indexes](https://github.com/bsn-si/IPEHR-blockchain-indexes)
- periodic direct invocation of contract methods [IPEHR-blockchain-indexes](https://github.com/bsn-si/IPEHR-blockchain-indexes)
- making AQL queries to IPEHR-gateway

For demonstration purposes, the following metrics are implemented:
- number of patients registered in the system over all time
- number of patients logged in the system for a specified month
- nb of EHR documents logged in the system for all time
- EHR sign-in count for a given month

API Documentation: [https://stat.ipehr.org/swagger/index.html](https://stat.ipehr.org/swagger/index.html)

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
The actual addresses of the downloaded contracts can be found [here](https://github.com/bsn-si/IPEHR-blockchain-indexes/blob/develop/deploys.md).

### Run IPEHR-stat

```
./bin/ipehr-stat -config=./config.json
```
