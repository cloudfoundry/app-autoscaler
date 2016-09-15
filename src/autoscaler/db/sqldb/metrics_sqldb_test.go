package sqldb_test

import (
	. "autoscaler/db/sqldb"
	"autoscaler/models"

	"code.cloudfoundry.org/lager"
	"github.com/lib/pq"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"os"
	"time"
)

var _ = Describe("MetricsSQLDB", func() {
	var (
		url        string
		mdb        *MetricsSQLDB
		logger     lager.Logger
		err        error
		metric     *models.Metric
		mtrcs      []*models.Metric
		start      int64
		end        int64
		before     int64
		appId      string
		metricName string
	)

	BeforeEach(func() {
		logger = lager.NewLogger("metrics-sqldb-test")
		url = os.Getenv("DBURL")
	})

	Describe("NewMetricsSQLDB", func() {
		JustBeforeEach(func() {
			mdb, err = NewMetricsSQLDB(url, logger)
		})

		AfterEach(func() {
			if mdb != nil {
				err = mdb.Close()
				Expect(err).NotTo(HaveOccurred())
			}
		})

		Context("when db url is not correct", func() {
			BeforeEach(func() {
				url = "postgres://not-exist-user:not-exist-password@localhost/autoscaler?sslmode=disable"
			})
			It("should error", func() {
				Expect(err).To(BeAssignableToTypeOf(&pq.Error{}))
			})

		})

		Context("when url is correct", func() {
			It("should not error", func() {
				Expect(err).NotTo(HaveOccurred())
				Expect(mdb).NotTo(BeNil())
			})
		})
	})

	Describe("SaveMetric", func() {
		BeforeEach(func() {
			mdb, err = NewMetricsSQLDB(url, logger)
			Expect(err).NotTo(HaveOccurred())
			cleanMetricsTable()
		})

		AfterEach(func() {
			err = mdb.Close()
			Expect(err).NotTo(HaveOccurred())
		})

		Context("When inserting a metric of an app", func() {
			BeforeEach(func() {
				metric = &models.Metric{
					AppId:     "test-app-id",
					Name:      models.MetricNameMemory,
					Unit:      models.UnitBytes,
					TimeStamp: 11111111,
					Instances: []models.InstanceMetric{{23456312, 0, "3333"}, {23556312, 1, "6666"}},
				}
				err = mdb.SaveMetric(metric)
			})

			It("has the metric in database", func() {
				Expect(err).NotTo(HaveOccurred())
				Expect(hasMetric("test-app-id", models.MetricNameMemory, 11111111)).To(BeTrue())
			})
		})

		Context("When inserting multiple metrics of an app", func() {
			BeforeEach(func() {
				metric = &models.Metric{
					AppId: "test-app-id",
					Name:  models.MetricNameMemory,
					Unit:  models.UnitBytes,
				}
			})

			It("has all the metrics in database", func() {
				metric.TimeStamp = 111111
				metric.Instances = []models.InstanceMetric{}
				err = mdb.SaveMetric(metric)
				Expect(err).NotTo(HaveOccurred())

				metric.TimeStamp = 222222
				metric.Instances = []models.InstanceMetric{{23456312, 0, "3333"}}
				mdb.SaveMetric(metric)
				Expect(err).NotTo(HaveOccurred())

				metric.TimeStamp = 333333
				metric.Instances = []models.InstanceMetric{{23456312, 0, "3333"}, {23556312, 1, "6666"}}
				mdb.SaveMetric(metric)
				Expect(err).NotTo(HaveOccurred())

				Expect(hasMetric("test-app-id", models.MetricNameMemory, 111111)).To(BeTrue())
				Expect(hasMetric("test-app-id", models.MetricNameMemory, 222222)).To(BeTrue())
				Expect(hasMetric("test-app-id", models.MetricNameMemory, 333333)).To(BeTrue())
			})
		})

	})

	Describe("RetrieveMetrics", func() {
		BeforeEach(func() {
			mdb, err = NewMetricsSQLDB(url, logger)
			Expect(err).NotTo(HaveOccurred())
			cleanMetricsTable()

			metric = &models.Metric{
				AppId: "test-app-id",
				Name:  models.MetricNameMemory,
				Unit:  models.UnitBytes,
			}

			metric.TimeStamp = 666666
			metric.Instances = []models.InstanceMetric{{654321, 0, "6666"}, {764321, 1, "8888"}}
			err = mdb.SaveMetric(metric)
			Expect(err).NotTo(HaveOccurred())

			metric.TimeStamp = 222222
			metric.Instances = []models.InstanceMetric{}
			err = mdb.SaveMetric(metric)
			Expect(err).NotTo(HaveOccurred())

			metric.TimeStamp = 333333
			metric.Instances = []models.InstanceMetric{{123456, 0, "3333"}}
			err = mdb.SaveMetric(metric)
			Expect(err).NotTo(HaveOccurred())

			start = 0
			end = -1
			appId = "test-app-id"
			metricName = models.MetricNameMemory
		})

		AfterEach(func() {
			err = mdb.Close()
			Expect(err).NotTo(HaveOccurred())
		})

		JustBeforeEach(func() {
			mtrcs, err = mdb.RetrieveMetrics(appId, metricName, start, end)
		})

		Context("The app has no metrics", func() {
			BeforeEach(func() {
				appId = "app-id-no-metric"
			})

			It("returns empty metrics", func() {
				Expect(err).NotTo(HaveOccurred())
				Expect(mtrcs).To(BeEmpty())
			})

		})

		Context("when the app has no metrics with the given metric name", func() {
			BeforeEach(func() {
				metricName = "metric-name-no-metrics"
			})

			It("returns empty metrics", func() {
				Expect(err).NotTo(HaveOccurred())
				Expect(mtrcs).To(BeEmpty())
			})
		})

		Context("when end time is now (end = -1)", func() {
			BeforeEach(func() {
				start = 333333
				end = -1
			})

			It("returns metrics from start time to now", func() {
				Expect(err).NotTo(HaveOccurred())
				Expect(mtrcs).To(HaveLen(2))
			})

		})

		Context("when end time is before all the metrics timestamps", func() {
			BeforeEach(func() {
				start = 111111
				end = 211111
			})

			It("returns empty metrics", func() {
				Expect(err).NotTo(HaveOccurred())
				Expect(mtrcs).To(BeEmpty())
			})

		})

		Context("when start time is after all the metrics timestamps", func() {
			BeforeEach(func() {
				start = 777777
				end = 888888
			})

			It("returns empty metrics", func() {
				Expect(err).NotTo(HaveOccurred())
				Expect(mtrcs).To(BeEmpty())
			})

		})

		Context("when start time > end time", func() {
			BeforeEach(func() {
				start = 555555
				end = 111111
			})

			It("returns empty metrics", func() {
				Expect(err).NotTo(HaveOccurred())
				Expect(mtrcs).To(BeEmpty())
			})
		})

		Context("when retriving all the metrics( start = 0, end = -1) ", func() {
			It("returns all the metrics of the app ordered by timestamp", func() {
				Expect(err).NotTo(HaveOccurred())
				Expect(mtrcs).To(Equal([]*models.Metric{
					&models.Metric{
						AppId:     "test-app-id",
						Name:      models.MetricNameMemory,
						Unit:      models.UnitBytes,
						TimeStamp: 222222,
						Instances: []models.InstanceMetric{},
					},
					&models.Metric{
						AppId:     "test-app-id",
						Name:      models.MetricNameMemory,
						Unit:      models.UnitBytes,
						TimeStamp: 333333,
						Instances: []models.InstanceMetric{{123456, 0, "3333"}},
					},
					&models.Metric{
						AppId:     "test-app-id",
						Name:      models.MetricNameMemory,
						Unit:      models.UnitBytes,
						TimeStamp: 666666,
						Instances: []models.InstanceMetric{{654321, 0, "6666"}, {764321, 1, "8888"}},
					}}))
			})
		})

		Context("When retriving part of the metrics", func() {
			BeforeEach(func() {
				start = 234567
				end = 555555
			})

			It("returns correct metrics", func() {
				Expect(err).NotTo(HaveOccurred())
				Expect(mtrcs).To(Equal([]*models.Metric{
					&models.Metric{
						AppId:     "test-app-id",
						Name:      models.MetricNameMemory,
						Unit:      models.UnitBytes,
						TimeStamp: 333333,
						Instances: []models.InstanceMetric{{123456, 0, "3333"}},
					}}))
			})
		})
	})

	Describe("PruneMetrics", func() {
		BeforeEach(func() {
			mdb, err = NewMetricsSQLDB(url, logger)
			Expect(err).NotTo(HaveOccurred())

			cleanMetricsTable()

			instances := []models.InstanceMetric{{123456, 0, "3333"}, {123476, 1, "6666"}}
			metric := &models.Metric{
				AppId:     "test-app-id",
				Name:      models.MetricNameMemory,
				Unit:      models.UnitBytes,
				Instances: instances,
			}

			metric.TimeStamp = 666666
			err = mdb.SaveMetric(metric)
			Expect(err).NotTo(HaveOccurred())

			metric.TimeStamp = 222222
			err = mdb.SaveMetric(metric)
			Expect(err).NotTo(HaveOccurred())

			metric.TimeStamp = 333333
			err = mdb.SaveMetric(metric)
			Expect(err).NotTo(HaveOccurred())

		})

		AfterEach(func() {
			err = mdb.Close()
			Expect(err).NotTo(HaveOccurred())
		})

		JustBeforeEach(func() {
			err = mdb.PruneMetrics(before)
		})

		Context("when pruning metrics before all the timestamps of metrics", func() {
			BeforeEach(func() {
				before = 0
			})

			It("does not remove any metrics", func() {
				Expect(err).NotTo(HaveOccurred())
				Expect(getNumberOfMetrics()).To(Equal(3))
			})
		})

		Context("when pruning all the metrics", func() {
			BeforeEach(func() {
				before = time.Now().UnixNano()
			})

			It("empties the metrics table", func() {
				Expect(err).NotTo(HaveOccurred())
				Expect(getNumberOfMetrics()).To(Equal(0))
			})
		})

		Context("when pruning part of the metrics", func() {
			BeforeEach(func() {
				before = 333333
			})

			It("removes metrics before the time specified", func() {
				Expect(err).NotTo(HaveOccurred())
				Expect(getNumberOfMetrics()).To(Equal(1))
				Expect(hasMetric("test-app-id", models.MetricNameMemory, 666666)).To(BeTrue())
			})
		})

	})

})
