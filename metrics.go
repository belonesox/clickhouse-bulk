package main

import (
	"github.com/prometheus/client_golang/prometheus"
)

var pushCounter = prometheus.NewCounter(
	prometheus.CounterOpts{
		Name: "ch_received_count",
		Help: "Received requests count from launch",
	})
var sentCounter = prometheus.NewCounter(
	prometheus.CounterOpts{
		Name: "ch_sent_count",
		Help: "Sent request count from launch",
	})
var dumpCounter = prometheus.NewCounter(
	prometheus.CounterOpts{
		Name: "ch_dump_count",
		Help: "Dumps saved from launch",
	})
var goodServers = prometheus.NewGauge(
	prometheus.GaugeOpts{
		Name: "ch_good_servers",
		Help: "Actual good servers count",
	})
var badServers = prometheus.NewGauge(
	prometheus.GaugeOpts{
		Name: "ch_bad_servers",
		Help: "Actual count of bad servers",
	})

var queuedDumps = prometheus.NewGauge(
	prometheus.GaugeOpts{
		Name: "ch_queued_dumps",
		Help: "Actual dump files id directory",
	})

var departmentsBlocked = prometheus.NewGauge(
	prometheus.GaugeOpts{
		Name: "departments_blocked_counter",
		Help: "Amount of blocked users",
	})

var rowsInserted = prometheus.NewGauge(
	prometheus.GaugeOpts{
		Name: "rows_inserted",
		Help: "Rows inserted",
	})

var activeDeparts = prometheus.NewGauge(
	prometheus.GaugeOpts{
		Name: "active_departs",
		Help: "If users from current departmet try to send query and this deparment !blocked it is active",
	})

var flushIntervals = prometheus.NewHistogram(
	prometheus.HistogramOpts{
		Name:    "flush_intervals",
		Help:    "Accumulats info about how many seconds left since each insert to CH",
		Buckets: prometheus.LinearBuckets(1, 0.5, 10),
	})

var flushCounts = prometheus.NewHistogram(
	prometheus.HistogramOpts{
		Name:    "flush_counts",
		Help:    "Accumulats info about how many rows were send in each insert to CH",
		Buckets: prometheus.LinearBuckets(10, 1, 10),
	})

// InitMetrics - init prometheus metrics
func InitMetrics() {

	prometheus.MustRegister(pushCounter)
	prometheus.MustRegister(sentCounter)
	prometheus.MustRegister(dumpCounter)
	prometheus.MustRegister(queuedDumps)
	prometheus.MustRegister(goodServers)
	prometheus.MustRegister(badServers)
	prometheus.MustRegister(departmentsBlocked)
	prometheus.MustRegister(rowsInserted)
	prometheus.MustRegister(activeDeparts)
	prometheus.MustRegister(flushIntervals)
	prometheus.MustRegister(flushCounts)

}
