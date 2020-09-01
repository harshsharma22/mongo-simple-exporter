package main

import (
	"context"
	"fmt"
	"regexp"
	"strings"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"github.com/sirupsen/logrus"
	"github.com/tidwall/gjson"
	"go.mongodb.org/mongo-driver/mongo"
	"gopkg.in/mgo.v2/bson"
)

var (
	logCounter = promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "mongo_log_total",
		Help: "Number of log messages",
	}, []string{
		"level",
		"component",
	})

	connCounter = promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "mongo_log_connection_total",
		Help: "Number of log messages about connections",
	}, []string{
		"level",
		"component",
		"state",
	})

	authCounter = promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "mongo_log_authentication_total",
		Help: "Number of log messages about authentication",
	}, []string{
		"level",
		"component",
		"state",
		"mechanism",
		"db",
		"principal",
		"remote",
	})

	metaCounter = promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "mongo_log_client_metadata_total",
		Help: "Number of log messages about client metadata",
	}, []string{
		"level",
		"component",
		"remote",
		"driver",
	})

	slowQuerySummary = promauto.NewSummaryVec(prometheus.SummaryOpts{
		Name: "mongo_log_slowquery_seconds",
		Help: "Number of log messages about slow queries",
	}, []string{
		"level",
		"component",
		"db",
		"collection",
		"command",
	})

	lastDate string
)

type logt struct {
	level     string
	component string
	msg       string
	attr      string
}

func processLogs(mc *mongo.Client) error {

	// var result bson.M
	r := mc.Database("admin").RunCommand(context.TODO(),
		bson.M{"getLog": "global"},
	)
	br, err := r.DecodeBytes()
	if err != nil {
		return err
	}
	result := br.String()

	if getFloatValue(gjson.Get(result, "ok")) != 1.0 {
		return fmt.Errorf("Couldn't execute getLog")
	}

	r1 := gjson.Get(result, "log")

	for _, l := range r1.Array() {

		//skip dates that already were counted
		logline := l.String()
		date := gjson.Get(logline, "t.$date").String()
		if lastDate != "" && date <= lastDate {
			continue
		}

		log := logt{
			level:     gjson.Get(l.String(), "s").String(),
			component: gjson.Get(l.String(), "c").String(),
			msg:       gjson.Get(l.String(), "msg").String(),
			attr:      gjson.Get(l.String(), "attr").String(),
		}

		err := process(log)
		if err != nil {
			logrus.Warnf("Error processing log %s. Ignoring", log)
		}

		lastDate = date
	}

	return nil
}

func process(log logt) error {
	logCounter.WithLabelValues(log.level, log.component).Inc()

	//connections
	if log.msg == "connection accepted" || log.msg == "connection ended" {
		if strings.Contains(log.msg, "accepted") {
			connCounter.WithLabelValues(log.level, log.component, "accepted").Inc()
		} else if strings.Contains(log.msg, "ended") {
			connCounter.WithLabelValues(log.level, log.component, "ended").Inc()
		}
		return nil
	}

	//authentication
	if log.msg == "Authentication failed" || log.msg == "Successful authentication" {
		// "level",
		// "component",
		// "state",
		// "mechanism",
		// "db",
		// "principal",
		// "remote",
		mec := gjson.Get(log.attr, "mechanism").String()
		principal := gjson.Get(log.attr, "principalName").String()
		db := gjson.Get(log.attr, "authenticationDatabase").String()
		rc := gjson.Get(log.attr, "client").String()

		remoteIP := ""
		a := strings.Split(rc, ":")
		if len(a) == 2 {
			remoteIP = a[0]
		} else {
			logrus.Warnf("Could not parse remote IP on authentication log. Using empty value remote=%s", rc)
		}

		state := ""
		if strings.Contains(log.msg, "failed") {
			state = "fail"
		} else if strings.Contains(log.msg, "Successful") {
			state = "success"
		}

		authCounter.WithLabelValues(log.level, log.component, state, mec, db, principal, remoteIP).Inc()

		return nil
	}

	//client metadata
	if log.msg == "client metadata" {
		remoteIP := ""
		rip := gjson.Get(log.attr, "remote").String()
		a := strings.Split(rip, ":")
		if len(a) == 2 {
			remoteIP = a[0]
		} else {
			logrus.Warnf("Could not parse remote IP log for client metadata. Using empty value remote=%s", rip)
		}
		driver := gjson.Get(log.attr, "doc.driver.name").String()
		metaCounter.WithLabelValues(log.level, log.component, remoteIP, driver).Inc()
		return nil
	}

	//slow query
	if log.msg == "Slow query" {
		collection := gjson.Get(log.attr, "ns").String()
		db := gjson.Get(log.attr, "command.$db").String()

		//command
		cmd := gjson.Get(log.attr, "command").String()
		re := regexp.MustCompile("^{\"([a-zA-Z]+)\":")
		f := re.FindStringSubmatch(cmd)
		command := ""
		if len(f) == 2 {
			command = f[1]
		} else {
			logrus.Warnf("Could not parse slow query command name from log. Using empty value. cmd=%s", cmd)
		}

		dur := gjson.Get(log.attr, "durationMillis").Int()

		slowQuerySummary.WithLabelValues(log.level, log.component, db, collection, command).Observe(float64(dur) / 1000.0)
		return nil
	}

	return nil
}
