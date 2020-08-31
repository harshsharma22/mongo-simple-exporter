package main

import (
	"context"
	"flag"
	"fmt"
	"net/http"
	"time"

	"github.com/flaviostutz/signalutils"
	"github.com/gorilla/mux"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/sirupsen/logrus"
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
	logrus.Infof("Connected to MongoDB server")

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	err = client.Connect(ctx)
	if err != nil {
		panic(fmt.Sprintf("Error connecting to MongoDB server. err=%s", err))
	}
	logrus.Infof("Connected to MongoDB server")

	logrus.Debugf("Setup prometheus metrics...")
	// metricActiveConnections := promauto.NewGaugeVec(prometheus.GaugeOpts{
	// 	Name: "mongos_connections_active_total",
	// 	Help: "Number of connections with current activity",
	// }, []string{
	// 	"label",
	// })

	router := mux.NewRouter()
	router.Handle("/metrics", promhttp.Handler())

	logrus.Debugf("Starting to report mongo metrics...")
	signalutils.StartWorker(context.Background(), "test1", func() error {
		//do some real work here
		time.Sleep(15 * time.Millisecond)
		return nil
	}, 0.0001, 0.1, true)

	logrus.Infof("Mongos exporter listening on 0.0.0.0:8880/metrics")
	http.ListenAndServe("0.0.0.0:8880", router)
}
