# mongos-exporter

MongoDB metrics Prometheus exporter based on mongos daemon for more getting general metrics regarding to slow queries, number of connections, log counters etc.

It was created mainly for working with mongo clusters by pointing to mongos router daemons, but some metrics works on standalone mongo instances too.

For creating a whole MongoDB cluster, see https://github.com/stutzlab/mongo-cluster

The metrics are updated every 10s.

## Usage

* Create a docker-compose.yml

```yml
version: '3.5'
services:
  mongos-exporter:
    build: .
    image: stutzlab/mongos-exporter
    environment:
      - MONGODB_URL=mongodb://root:example@mongo
    ports:
      - 8880:8880

  mongo:
    image: mongo
    restart: always
    ports:
      - 27017:27017
    environment:
      - MONGO_INITDB_ROOT_USERNAME=root
      - MONGO_INITDB_ROOT_PASSWORD=example

  prometheus:
    image: flaviostutz/prometheus
    ports:
      - 9090:9090
    environment:
      - SCRAPE_INTERVAL=15s
      - SCRAPE_TIMEOUT=10s
      - DNS_SCRAPE_TARGETS=mongos-exporter@mongos-exporter:8880/metrics
```

* Call `curl localhost:8880/metrics` and see some metrics about the running mongo instance

## ENVs

* MONGODB_URL - mongodb url in format "mongodb://[user]:[pass]@[host]/[db]". Ex: "mongodb://user1:pass1@mymongo.org/test". Required.

## Developer tips

* Internally, we are issuing queries to mongo server and transforming them into Prometheus metrics:

```js
//get logs in order to look at slow queries and other important clues
db.adminCommand( { getLog: "global" } )

//get various metrics regarding to issued queries, connections state etc
db.serverStatus()
```

* Those commands works both on sharded and standalone clusters

