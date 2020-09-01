package main

import (
	"context"
	"flag"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/flaviostutz/signalutils"
	"github.com/gorilla/mux"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/sirupsen/logrus"
	"github.com/tidwall/gjson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

func main() {

	logrus.SetLevel(logrus.DebugLevel)

	mongodbURL := ""
	flag.StringVar(&mongodbURL, "mongodb-url", "", "MongoDB URL in format 'mongodb://[user]:[pass]@[host]:[port]/[db]'. required")
	flag.Parse()

	if mongodbURL == "" {
		panic("mongodb-url is required")
	}

	logrus.Debugf("Connecting to Mongo server...")
	client, err := mongo.NewClient(options.Client().ApplyURI(mongodbURL))
	if err != nil {
		panic(fmt.Sprintf("Error connecting to MongoDB server. err=%s", err))
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	err = client.Connect(ctx)
	if err != nil {
		panic(fmt.Sprintf("Error connecting to MongoDB server. err=%s", err))
	}
	logrus.Infof("Connected to MongoDB server")

	logrus.Debugf("Setup prometheus metrics...")

	// processLogs(client)

	router := mux.NewRouter()
	router.Handle("/metrics", promhttp.Handler())

	logrus.Debugf("Starting report workers...")

	signalutils.StartWorker(context.Background(), "getLog", func() error {
		return processLogs(client)
	}, 0.0001, 0.1, false)
	time.Sleep(1 * time.Second)

	signalutils.StartWorker(context.Background(), "listDatabases", func() error {
		return processDatabases(client)
	}, 0.0001, 0.1, false)
	time.Sleep(1 * time.Second)

	signalutils.StartWorker(context.Background(), "serverStatus", func() error {
		return processServerInfo(client)
	}, 0.0001, 0.1, false)
	time.Sleep(1 * time.Second)

	signalutils.StartWorker(context.Background(), "serverStatus", func() error {
		return processCollectionStats(client)
	}, 0.0001, 0.03, false)

	logrus.Infof("Mongo exporter listening on 0.0.0.0:8880/metrics")
	http.ListenAndServe("0.0.0.0:8880", router)
}

func getFloatValue(v gjson.Result) float64 {
	for _, v2 := range v.Map() {
		rv, err := strconv.ParseFloat(v2.String(), 64)
		if err != nil {
			logrus.Warnf("Could not parse %s", v2.String())
		}
		return rv
		// return gjson.Get(rs, k).Float()
	}
	return -1
}
