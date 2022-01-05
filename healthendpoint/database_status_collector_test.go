package healthendpoint_test

import (
	"time"

	"code.cloudfoundry.org/app-autoscaler/src/autoscaler/fakes"
	. "code.cloudfoundry.org/app-autoscaler/src/autoscaler/healthendpoint"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/prometheus/client_golang/prometheus"

	"database/sql"
)

var _ = Describe("DatabaseStatusCollector", func() {
	var (
		databaseStatusCollector prometheus.Collector
		namespace               = "test_name_space"
		subSystem               = "test_sub_system"
		dbName                  = "test_db_name"
		fakeDB                  *fakes.FakeDatabaseStatus
		descChan                chan *prometheus.Desc
		metricChan              chan prometheus.Metric
		dbStatusResult          = sql.DBStats{
			MaxOpenConnections: 100,
			OpenConnections:    50,
			InUse:              25,
			Idle:               25,
			WaitCount:          20,
			WaitDuration:       10 * time.Second,
			MaxIdleClosed:      10,
			MaxLifetimeClosed:  15,
		}
		//describe
		maxOpenConnectionsDesc = prometheus.NewDesc(
			prometheus.BuildFQName(namespace, subSystem, dbName+"_max_open_connections"),
			"Maximum number of open connections to the database",
			nil,
			nil,
		)
		openConnectionsDesc = prometheus.NewDesc(
			prometheus.BuildFQName(namespace, subSystem, dbName+"_open_connections"),
			"The number of established connections both in use and idle",
			nil,
			nil,
		)
		inUseDesc = prometheus.NewDesc(
			prometheus.BuildFQName(namespace, subSystem, dbName+"_in_use"),
			"The number of connections currently in use",
			nil,
			nil,
		)
		idleDesc = prometheus.NewDesc(
			prometheus.BuildFQName(namespace, subSystem, dbName+"_idle"),
			"The number of idle connections",
			nil,
			nil,
		)
		waitCountDesc = prometheus.NewDesc(
			prometheus.BuildFQName(namespace, subSystem, dbName+"_wait_count"),
			"The total number of connections waited for",
			nil,
			nil,
		)
		waitDurationDesc = prometheus.NewDesc(
			prometheus.BuildFQName(namespace, subSystem, dbName+"_wait_duration"),
			"The total time blocked waiting for a new connection",
			nil,
			nil,
		)
		maxIdleClosedDesc = prometheus.NewDesc(
			prometheus.BuildFQName(namespace, subSystem, dbName+"_max_idle_closed"),
			"The total number of connections closed due to SetMaxIdleConns",
			nil,
			nil,
		)
		maxLifetimeClosedDesc = prometheus.NewDesc(
			prometheus.BuildFQName(namespace, subSystem, dbName+"_max_lifetime_closed"),
			"The total number of connections closed due to SetConnMaxLifetime",
			nil,
			nil,
		)
		// metrics
		maxOpenConnectionsMetric, _ = prometheus.NewConstMetric(maxOpenConnectionsDesc, prometheus.GaugeValue, float64(dbStatusResult.MaxOpenConnections))
		openConnectionsMetric, _    = prometheus.NewConstMetric(openConnectionsDesc, prometheus.GaugeValue, float64(dbStatusResult.OpenConnections))
		inUseMetric, _              = prometheus.NewConstMetric(inUseDesc, prometheus.GaugeValue, float64(dbStatusResult.InUse))
		idleMetric, _               = prometheus.NewConstMetric(idleDesc, prometheus.GaugeValue, float64(dbStatusResult.Idle))
		waitCountMetric, _          = prometheus.NewConstMetric(waitCountDesc, prometheus.GaugeValue, float64(dbStatusResult.WaitCount))
		waitDurationMetric, _       = prometheus.NewConstMetric(waitDurationDesc, prometheus.GaugeValue, float64(dbStatusResult.WaitDuration))
		maxIdleClosedMetric, _      = prometheus.NewConstMetric(maxIdleClosedDesc, prometheus.GaugeValue, float64(dbStatusResult.MaxIdleClosed))
		maxLifetimeClosedMetric, _  = prometheus.NewConstMetric(maxLifetimeClosedDesc, prometheus.GaugeValue, float64(dbStatusResult.MaxLifetimeClosed))
	)
	BeforeEach(func() {
		fakeDB = &fakes.FakeDatabaseStatus{}
		databaseStatusCollector = NewDatabaseStatusCollector(namespace, subSystem, dbName, fakeDB)
		fakeDB.GetDBStatusReturns(dbStatusResult)
		descChan = make(chan *prometheus.Desc, 10)
		metricChan = make(chan prometheus.Metric, 100)

	})
	Context("Describe", func() {
		BeforeEach(func() {
			databaseStatusCollector.Describe(descChan)
		})
		It("Receive descs", func() {
			Eventually(descChan).Should(Receive(Equal(maxOpenConnectionsDesc)))
			Eventually(descChan).Should(Receive(Equal(openConnectionsDesc)))
			Eventually(descChan).Should(Receive(Equal(inUseDesc)))
			Eventually(descChan).Should(Receive(Equal(idleDesc)))
			Eventually(descChan).Should(Receive(Equal(waitCountDesc)))
			Eventually(descChan).Should(Receive(Equal(waitDurationDesc)))
			Eventually(descChan).Should(Receive(Equal(maxIdleClosedDesc)))
			Eventually(descChan).Should(Receive(Equal(maxLifetimeClosedDesc)))
		})
	})

	Context("Collect", func() {
		BeforeEach(func() {
			databaseStatusCollector.Collect(metricChan)
		})
		It("Receive metrics", func() {
			Eventually(metricChan).Should(Receive(Equal(maxOpenConnectionsMetric)))
			Eventually(metricChan).Should(Receive(Equal(openConnectionsMetric)))
			Eventually(metricChan).Should(Receive(Equal(inUseMetric)))
			Eventually(metricChan).Should(Receive(Equal(idleMetric)))
			Eventually(metricChan).Should(Receive(Equal(waitCountMetric)))
			Eventually(metricChan).Should(Receive(Equal(waitDurationMetric)))
			Eventually(metricChan).Should(Receive(Equal(maxIdleClosedMetric)))
			Eventually(metricChan).Should(Receive(Equal(maxLifetimeClosedMetric)))
		})
	})
})
