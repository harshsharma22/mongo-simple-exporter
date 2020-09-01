# mongos-exporter

MongoDB metrics Prometheus exporter for metrics regarding to slow queries, number of connections, log counters, total storage, operations count etc.

This works with standalone or clusters by pointing this exporter to the mongos instance.

For creating a whole MongoDB cluster, see https://github.com/stutzlab/mongo-cluster

The metrics are updated every ~10s (collection stats are updated each 30s).

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

* Run `docker-compose up`

```sh
DEBU[0000] Connecting to Mongo server...
INFO[0000] Connected to MongoDB server
DEBU[0000] Setup prometheus metrics...
DEBU[0000] Starting report workers...
INFO[0003] Mongo exporter listening on 0.0.0.0:8880/metrics
```

* Call `curl localhost:8880/metrics` and see some metrics about the running mongo instance

  * Some lines from a real execution (to see all generated metrics, run a real example)

```sh
...
# HELP mongo_collection_disk_bytes Collection total disk storage size (documents + indexes) in bytes
mongo_collection_disk_bytes{collection="aaaaa",db="test3"} 8192
# HELP mongo_collection_doc_count Collection document count
mongo_collection_doc_count{collection="aaaaa",db="test3"} 42
# HELP mongo_collection_indexes_bytes Collection total index size in bytes
mongo_collection_indexes_bytes{collection="aaaaa",db="test3"} 4096
# HELP mongo_log_slowquery_seconds Number of log messages about slow queries
mongo_log_slowquery_seconds_sum{collection="admin.$cmd",command="isMaster",component="COMMAND",db="admin",level="I"} 60.038
mongo_log_slowquery_seconds_count{collection="admin.$cmd",command="isMaster",component="COMMAND",db="admin",level="I"} 6
# HELP mongo_opcounters_total Number of operations executed by op type
mongo_opcounters_total{host="dfc1af64c51a",type="query"} 22252
mongo_opcounters_total{host="dfc1af64c51a",type="update"} 281
# HELP mongo_database_disk_bytes Size of database in disk in bytes
mongo_database_disk_bytes{db="admin"} 229376
mongo_database_disk_bytes{db="config"} 5.890048e+06
...
```

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

* During local development, use either

  * `docker-compose up --build` (docker only run with no need to install Golang in your machine) or

```sh
docker-compose up mongo -d
go run . --mongodb-url=mongodb://root:example@localhost
```
