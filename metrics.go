package main

import (
	"github.com/prometheus/client_golang/prometheus"
)

var pushCounter prometheus.Counter
var sentCounter prometheus.Counter
var dumpCounter prometheus.Counter
var goodServers prometheus.Gauge
var badServers prometheus.Gauge
var queuedDumps prometheus.Gauge
var departmentsBlocked prometheus.Gauge
var rowsInserted prometheus.Gauge
var activeDeparts prometheus.Gauge
var flushIntervals prometheus.Histogram
var flushCounts prometheus.Histogram
var userButch prometheus.Histogram

func Width(flush_count int) (width float64) {
	return float64(flush_count) / 5
}

func Start(width float64) (start float64) {
	return width * 1.05
}

// InitMetrics - init prometheus metrics
func InitMetrics(cnf Config) {
	pushCounter = prometheus.NewCounter(
		prometheus.CounterOpts{
			Name: "ch_received_count",
			Help: "Received requests count from launch",
		})

	sentCounter = prometheus.NewCounter(
		prometheus.CounterOpts{
			Name: "ch_sent_count",
			Help: "Sent request count from launch",
		})

	dumpCounter = prometheus.NewCounter(
		prometheus.CounterOpts{
			Name: "ch_dump_count",
			Help: "Dumps saved from launch",
		})

	goodServers = prometheus.NewGauge(
		prometheus.GaugeOpts{
			Name: "ch_good_servers",
			Help: "Actual good servers count",
		})

	badServers = prometheus.NewGauge(
		prometheus.GaugeOpts{
			Name: "ch_bad_servers",
			Help: "Actual count of bad servers",
		})

	queuedDumps = prometheus.NewGauge(
		prometheus.GaugeOpts{
			Name: "ch_queued_dumps",
			Help: "Actual dump files id directory",
		})

	departmentsBlocked = prometheus.NewGauge(
		prometheus.GaugeOpts{
			Name: "ch_departments_blocked_counter",
			Help: "Amount of blocked users",
		})

	rowsInserted = prometheus.NewGauge(
		prometheus.GaugeOpts{
			Name: "ch_rows_inserted",
			Help: "Rows inserted",
		})

	activeDeparts = prometheus.NewGauge(
		prometheus.GaugeOpts{
			Name: "ch_active_departs",
			Help: "If users from current departmet try to send query and this deparment !blocked it is active",
		})

	flushIntervals = prometheus.NewHistogram(
		prometheus.HistogramOpts{
			Name:    "ch_flush_intervals",
			Help:    "Accumulats info about how many seconds left since each insert to CH",
			Buckets: prometheus.ExponentialBucketsRange(1, (float64(cnf.FlushInterval)/1000)*1.2, 5),
		})

	width := Width(cnf.FlushCount)
	start := Start(width)
	flushCounts = prometheus.NewHistogram(
		prometheus.HistogramOpts{
			Name:    "ch_flush_counts",
			Help:    "Accumulats info about how many rows were send in each insert to CH",
			Buckets: prometheus.LinearBuckets(start, width, 5),
		})

	userButch = prometheus.NewHistogram(
		prometheus.HistogramOpts{
			Name:    "ch_user_butch",
			Help:    "Accumulats info about how many rows were send in each insert from user",
			Buckets: prometheus.ExponentialBucketsRange(float64(cnf.SpectatorRows), float64(cnf.SpectatorButch), 2),
		})

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
	prometheus.MustRegister(userButch)
}
