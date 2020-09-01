package main

import (
	"context"
	"fmt"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"github.com/sirupsen/logrus"
	"github.com/tidwall/gjson"
	"go.mongodb.org/mongo-driver/mongo"
	"gopkg.in/mgo.v2/bson"
)

var (
	docCount = promauto.NewGaugeVec(prometheus.GaugeOpts{
		Name: "mongo_collection_doc_count",
		Help: "Collection document count",
	}, []string{
		"db",
		"collection",
	})

	collDocumentsSize = promauto.NewGaugeVec(prometheus.GaugeOpts{
		Name: "mongo_collection_uncompressed_bytes",
		Help: "Collection document sizes uncompressed in bytes",
	}, []string{
		"db",
		"collection",
	})

	collTotalStorage = promauto.NewGaugeVec(prometheus.GaugeOpts{
		Name: "mongo_collection_disk_bytes",
		Help: "Collection total disk storage size (documents + indexes) in bytes",
	}, []string{
		"db",
		"collection",
	})

	collIndexesStorage = promauto.NewGaugeVec(prometheus.GaugeOpts{
		Name: "mongo_collection_indexes_bytes",
		Help: "Collection total index size in bytes",
	}, []string{
		"db",
		"collection",
	})

	collShardDocCount = promauto.NewGaugeVec(prometheus.GaugeOpts{
		Name: "mongo_collection_shard_doc_count",
		Help: "Collection document count per shard",
	}, []string{
		"db",
		"collection",
		"shard",
	})

	collShardTotalStorage = promauto.NewGaugeVec(prometheus.GaugeOpts{
		Name: "mongo_collection_shard_disk_bytes",
		Help: "Collection total storage per shard",
	}, []string{
		"db",
		"collection",
		"shard",
	})
)

func processCollectionStats(mc *mongo.Client) error {

	dbs, err := mc.ListDatabaseNames(context.TODO(), bson.M{})
	if err != nil {
		return fmt.Errorf("Could not list database names. err=%s", err)
	}

	//for each database, get collection stats
	for _, dbname := range dbs {

		if dbname == "config" || dbname == "admin" {
			continue
		}

		cns, err := mc.Database(dbname).ListCollectionNames(context.TODO(), bson.M{})
		if err != nil {
			return fmt.Errorf("Could not list collection names. err=%s", err)
		}

		//get stats for each collection
		for _, collname := range cns {
			logrus.Debugf("Generating stats for collection %s.%s", dbname, collname)

			r := mc.Database(dbname).RunCommand(context.TODO(),
				bson.M{"collStats": collname},
			)
			br, err := r.DecodeBytes()
			if err != nil {
				return err
			}
			result := br.String()

			if getFloatValue(gjson.Get(result, "ok")) != 1.0 {
				return fmt.Errorf("Couldn't execute collStats for %s", collname)
			}

			docc := getFloatValue(gjson.Get(result, "count"))
			docCount.WithLabelValues(dbname, collname).Set(docc)

			uncompressedSize := getFloatValue(gjson.Get(result, "size"))
			collDocumentsSize.WithLabelValues(dbname, collname).Set(uncompressedSize)

			storageSize := getFloatValue(gjson.Get(result, "totalSize"))
			collTotalStorage.WithLabelValues(dbname, collname).Set(storageSize)

			indexSize := getFloatValue(gjson.Get(result, "totalIndexSize"))
			collIndexesStorage.WithLabelValues(dbname, collname).Set(indexSize)

			shards := gjson.Get(result, "shards").Map()
			for shardname, v := range shards {
				svs := v.String()

				scount := getFloatValue(gjson.Get(svs, "count"))
				collShardDocCount.WithLabelValues(dbname, collname, shardname).Set(scount)

				sstorage := getFloatValue(gjson.Get(svs, "totalSize"))
				collShardTotalStorage.WithLabelValues(dbname, collname, shardname).Set(sstorage)
			}

		}

	}

	return nil
}
