package main

import (
	"context"
	"fmt"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"github.com/tidwall/gjson"
	"go.mongodb.org/mongo-driver/mongo"
	"gopkg.in/mgo.v2/bson"
)

var (
	dbSize = promauto.NewGaugeVec(prometheus.GaugeOpts{
		Name: "mongo_database_disk_bytes",
		Help: "Size of database in disk in bytes",
	}, []string{
		"db",
	})

	dbShardSize = promauto.NewGaugeVec(prometheus.GaugeOpts{
		Name: "mongo_database_shard_disk_bytes",
		Help: "Size of database shard in disk in bytes",
	}, []string{
		"db",
		"shard",
	})
)

func processDatabases(mc *mongo.Client) error {

	r := mc.Database("admin").RunCommand(context.TODO(),
		bson.M{"listDatabases": 1},
	)
	br, err := r.DecodeBytes()
	if err != nil {
		return err
	}
	result := br.String()

	if getFloatValue(gjson.Get(result, "ok")) != 1.0 {
		return fmt.Errorf("Couldn't execute listDatabases")
	}

	dbs := gjson.Get(result, "databases")

	for _, d := range dbs.Array() {
		db := d.String()
		dbname := gjson.Get(db, "name").String()
		sizeOnDisk := getFloatValue(gjson.Get(db, "sizeOnDisk"))

		//database size on disk metrics
		dbSize.WithLabelValues(dbname).Set(sizeOnDisk)

		//shards size on disk metrics
		sr := gjson.Get(db, "shards")
		for shardName, ss := range sr.Map() {
			shardSize := getFloatValue(ss)
			dbShardSize.WithLabelValues(dbname, shardName).Set(shardSize)
		}
	}

	return nil
}
