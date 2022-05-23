package healthendpoint

import (
	"database/sql"

	"github.com/prometheus/client_golang/prometheus"
)

type DatabaseStatus interface {
	GetDBStatus() sql.DBStats
}

type databaseStatusCollector struct {
	MaxOpenConnectionsGauge prometheus.Gauge
	OpenConnectionsGauge    prometheus.Gauge
	InUseGauge              prometheus.Gauge
	IdleGauge               prometheus.Gauge
	WaitCountGauge          prometheus.Gauge
	WaitDurationGauge       prometheus.Gauge
	MaxIdleClosedGauge      prometheus.Gauge
	MaxLifetimeClosedGauge  prometheus.Gauge

	dbStatus DatabaseStatus
}

func NewDatabaseStatusCollector(namespace, subSystem string, dbName string, dbStatus DatabaseStatus) prometheus.Collector {
	return &databaseStatusCollector{
		MaxOpenConnectionsGauge: prometheus.NewGauge(
			prometheus.GaugeOpts{
				Namespace: namespace,
				Subsystem: subSystem,
				Name:      dbName + "_max_open_connections",
				Help:      "Maximum number of open connections to the database",
			}),
		OpenConnectionsGauge: prometheus.NewGauge(
			prometheus.GaugeOpts{
				Namespace: namespace,
				Subsystem: subSystem,
				Name:      dbName + "_open_connections",
				Help:      "The number of established connections both in use and idle",
			}),
		InUseGauge: prometheus.NewGauge(
			prometheus.GaugeOpts{
				Namespace: namespace,
				Subsystem: subSystem,
				Name:      dbName + "_in_use",
				Help:      "The number of connections currently in use",
			}),
		IdleGauge: prometheus.NewGauge(
			prometheus.GaugeOpts{
				Namespace: namespace,
				Subsystem: subSystem,
				Name:      dbName + "_idle",
				Help:      "The number of idle connections",
			}),
		WaitCountGauge: prometheus.NewGauge(
			prometheus.GaugeOpts{
				Namespace: namespace,
				Subsystem: subSystem,
				Name:      dbName + "_wait_count",
				Help:      "The total number of connections waited for",
			}),
		WaitDurationGauge: prometheus.NewGauge(
			prometheus.GaugeOpts{
				Namespace: namespace,
				Subsystem: subSystem,
				Name:      dbName + "_wait_duration",
				Help:      "The total time blocked waiting for a new connection",
			}),
		MaxIdleClosedGauge: prometheus.NewGauge(
			prometheus.GaugeOpts{
				Namespace: namespace,
				Subsystem: subSystem,
				Name:      dbName + "_max_idle_closed",
				Help:      "The total number of connections closed due to SetMaxIdleConns",
			}),
		MaxLifetimeClosedGauge: prometheus.NewGauge(
			prometheus.GaugeOpts{
				Namespace: namespace,
				Subsystem: subSystem,
				Name:      dbName + "_max_lifetime_closed",
				Help:      "The total number of connections closed due to SetConnMaxLifetime",
			}),
		dbStatus: dbStatus,
	}
}

func (c *databaseStatusCollector) Describe(ch chan<- *prometheus.Desc) {
	ch <- c.MaxOpenConnectionsGauge.Desc()
	ch <- c.OpenConnectionsGauge.Desc()
	ch <- c.InUseGauge.Desc()
	ch <- c.IdleGauge.Desc()
	ch <- c.WaitCountGauge.Desc()
	ch <- c.WaitDurationGauge.Desc()
	ch <- c.MaxIdleClosedGauge.Desc()
	ch <- c.MaxLifetimeClosedGauge.Desc()
}

func (c *databaseStatusCollector) Collect(ch chan<- prometheus.Metric) {
	dbMetrics := c.dbStatus.GetDBStatus()
	m, _ := prometheus.NewConstMetric(c.MaxOpenConnectionsGauge.Desc(), prometheus.GaugeValue, float64(dbMetrics.MaxOpenConnections))
	ch <- m
	m, _ = prometheus.NewConstMetric(c.OpenConnectionsGauge.Desc(), prometheus.GaugeValue, float64(dbMetrics.OpenConnections))
	ch <- m
	m, _ = prometheus.NewConstMetric(c.InUseGauge.Desc(), prometheus.GaugeValue, float64(dbMetrics.InUse))
	ch <- m
	m, _ = prometheus.NewConstMetric(c.IdleGauge.Desc(), prometheus.GaugeValue, float64(dbMetrics.Idle))
	ch <- m
	m, _ = prometheus.NewConstMetric(c.WaitCountGauge.Desc(), prometheus.GaugeValue, float64(dbMetrics.WaitCount))
	ch <- m
	m, _ = prometheus.NewConstMetric(c.WaitDurationGauge.Desc(), prometheus.GaugeValue, float64(dbMetrics.WaitDuration))
	ch <- m
	m, _ = prometheus.NewConstMetric(c.MaxIdleClosedGauge.Desc(), prometheus.GaugeValue, float64(dbMetrics.MaxIdleClosed))
	ch <- m
	m, _ = prometheus.NewConstMetric(c.MaxLifetimeClosedGauge.Desc(), prometheus.GaugeValue, float64(dbMetrics.MaxLifetimeClosed))
	ch <- m
}
