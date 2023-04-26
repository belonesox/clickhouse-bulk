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
		Name: "ch_departments_blocked_counter",
		Help: "Amount of blocked users",
	})

var rowsInserted = prometheus.NewGauge(
	prometheus.GaugeOpts{
		Name: "ch_rows_inserted",
		Help: "Rows inserted",
	})

var activeDeparts = prometheus.NewGauge(
	prometheus.GaugeOpts{
		Name: "ch_active_departs",
		Help: "If users from current departmet try to send query and this deparment !blocked it is active",
	})

var flushIntervals prometheus.Histogram

var flushCounts prometheus.Histogram

var tcpConnectionsBulk = prometheus.NewGauge(
	prometheus.GaugeOpts{
		Name: "ch_tcp_connections_bulk",
		Help: "Count TCP connections to proxy",
	})

var userButch = prometheus.NewHistogram(
	prometheus.HistogramOpts{
		Name:    "ch_user_butch",
		Help:    "Accumulats info about how many rows were send in each insert from user",
		Buckets: prometheus.ExponentialBucketsRange(6, 100, 2),
	})

func Width(flush_count int) (width float64) {
	return float64(flush_count) / 5
}

func Start(width float64) (start float64) {
	return width * 1.05
}

// InitMetrics - init prometheus metrics
func InitMetrics(cnf Config) {

	prometheus.MustRegister(pushCounter)
	prometheus.MustRegister(sentCounter)
	prometheus.MustRegister(dumpCounter)
	prometheus.MustRegister(queuedDumps)
	prometheus.MustRegister(goodServers)
	prometheus.MustRegister(badServers)
	prometheus.MustRegister(departmentsBlocked)
	prometheus.MustRegister(rowsInserted)
	prometheus.MustRegister(activeDeparts)
	prometheus.MustRegister(userButch)

	flushIntervals = prometheus.NewHistogram(
		prometheus.HistogramOpts{
			Name:    "ch_flush_intervals",
			Help:    "Accumulats info about how many seconds left since each insert to CH",
			Buckets: prometheus.ExponentialBucketsRange(1, (float64(cnf.FlushInterval)/1000)*1.2, 5),
		})
	prometheus.MustRegister(flushIntervals)

	width := Width(cnf.FlushCount)
	start := Start(width)
	flushCounts = prometheus.NewHistogram(
		prometheus.HistogramOpts{
			Name:    "ch_flush_counts",
			Help:    "Accumulats info about how many rows were send in each insert to CH",
			Buckets: prometheus.LinearBuckets(start, width, 5),
		})
	prometheus.MustRegister(flushCounts)
	prometheus.MustRegister(tcpConnectionsBulk)
}
