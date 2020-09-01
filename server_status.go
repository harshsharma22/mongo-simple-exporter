package main

import (
	"context"
	"fmt"

	"github.com/flaviostutz/promcollectors"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"github.com/tidwall/gjson"
	"go.mongodb.org/mongo-driver/mongo"
	"gopkg.in/mgo.v2/bson"
)

var (
	hostInfo = promcollectors.NewSettableCounterVec(prometheus.Opts{
		Name: "mongo_server_uptime_seconds",
		Help: "Basic server info and uptime in seconds",
	}, []string{
		"host",
		"version",
		"process",
	})

	connections = promauto.NewGaugeVec(prometheus.GaugeOpts{
		Name: "mongo_connections",
		Help: "Number of connections on server",
	}, []string{
		"host",
		"type",
	})

	netRequests = promcollectors.NewSettableCounterVec(prometheus.Opts{
		Name: "mongo_network_requests_total",
		Help: "Number of network requests processed",
	}, []string{
		"host",
	})

	opCounters = promcollectors.NewSettableCounterVec(prometheus.Opts{
		Name: "mongo_opcounters_total",
		Help: "Number of operations executed by op type",
	}, []string{
		"host",
		"type",
	})
)

func init() {
	prometheus.MustRegister(hostInfo)
	prometheus.MustRegister(netRequests)
	prometheus.MustRegister(opCounters)
}

func processServerInfo(mc *mongo.Client) error {

	r := mc.Database("admin").RunCommand(context.TODO(),
		bson.M{"serverStatus": 1},
	)
	br, err := r.DecodeBytes()
	if err != nil {
		return err
	}
	result := br.String()

	if getFloatValue(gjson.Get(result, "ok")) != 1.0 {
		return fmt.Errorf("Couldn't execute serverStatus")
	}

	host := gjson.Get(result, "host").String()
	version := gjson.Get(result, "version").String()
	process := gjson.Get(result, "process").String()
	uptime := getFloatValue(gjson.Get(result, "uptime"))

	hostInfo.Set(uptime, host, version, process)

	conn := gjson.Get(result, "connections")
	for typ, count := range conn.Map() {
		counter := getFloatValue(count)
		connections.WithLabelValues(host, typ).Set(counter)
	}

	netReq := getFloatValue(gjson.Get(result, "network.numRequests"))
	netRequests.Set(netReq, host)

	opc := gjson.Get(result, "opcounters").Map()
	for opname, counter := range opc {
		val := getFloatValue(counter)
		opCounters.Set(val, host, opname)
	}

	return nil
}
