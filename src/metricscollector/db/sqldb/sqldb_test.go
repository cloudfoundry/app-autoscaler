package sqldb_test

import (
	"metricscollector/config"
	. "metricscollector/db/sqldb"
	"metricscollector/metrics"

	"code.cloudfoundry.org/lager"
	"github.com/lib/pq"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"os"
	"time"
)

var _ = Describe("Sqldb", func() {
	var (
		conf       *config.DbConfig
		db         *SQLDB
		logger     lager.Logger
		err        error
		metric     *metrics.Metric
		mtrcs      []*metrics.Metric
		start      int64
		end        int64
		before     int64
		appId      string
		metricName string
		appIds     map[string]bool
	)

	BeforeEach(func() {
		logger = lager.NewLogger("sqldb-test")
		dbUrl := os.Getenv("DBURL")
		conf = &config.DbConfig{MetricsDbUrl: dbUrl, PolicyDbUrl: dbUrl}
	})

	Describe("NewSQLDB", func() {
		JustBeforeEach(func() {
			db, err = NewSQLDB(conf, logger)
		})

		AfterEach(func() {
			if db != nil {
				err = db.Close()
				Expect(err).NotTo(HaveOccurred())
			}
		})

		Context("when db config is not correct", func() {
			Context("when metrics db url is not correct", func() {
				BeforeEach(func() {
					conf.MetricsDbUrl = "postgres://non-exist-user:non-exist-password@localhost/autoscaler?sslmode=disable"
				})
				It("should error", func() {
					Expect(err).To(BeAssignableToTypeOf(&pq.Error{}))
				})

			})

			Context("when policy db url is not correct", func() {
				BeforeEach(func() {
					conf.PolicyDbUrl = "postgres://non-exist-user:non-exist-password@localhost/autoscaler?sslmode=disable"
				})
				It("should error", func() {
					Expect(err).To(BeAssignableToTypeOf(&pq.Error{}))
				})
			})

		})

		Context("when db config is correct", func() {
			It("should not error", func() {
				Expect(err).NotTo(HaveOccurred())
				Expect(db).NotTo(BeNil())
			})
		})
	})

	Describe("SaveMetric", func() {
		BeforeEach(func() {
			db, err = NewSQLDB(conf, logger)
			Expect(err).NotTo(HaveOccurred())
			cleanMetricsTable()
		})

		AfterEach(func() {
			err = db.Close()
			Expect(err).NotTo(HaveOccurred())
		})

		Context("When inserting a metric of an app", func() {
			BeforeEach(func() {
				metric = &metrics.Metric{
					AppId:     "test-app-id",
					Name:      metrics.MetricNameMemory,
					Unit:      metrics.UnitBytes,
					TimeStamp: 11111111,
					Instances: []metrics.InstanceMetric{{23456312, 0, "3333"}, {23556312, 1, "6666"}},
				}
				err = db.SaveMetric(metric)
			})

			It("has the metric in database", func() {
				Expect(err).NotTo(HaveOccurred())
				Expect(hasMetric("test-app-id", metrics.MetricNameMemory, 11111111)).To(BeTrue())
			})
		})

		Context("When inserting multiple metrics of an app", func() {
			BeforeEach(func() {
				metric = &metrics.Metric{
					AppId: "test-app-id",
					Name:  metrics.MetricNameMemory,
					Unit:  metrics.UnitBytes,
				}
			})

			It("has all the metrics in database", func() {
				metric.TimeStamp = 111111
				metric.Instances = []metrics.InstanceMetric{}
				err = db.SaveMetric(metric)
				Expect(err).NotTo(HaveOccurred())

				metric.TimeStamp = 222222
				metric.Instances = []metrics.InstanceMetric{{23456312, 0, "3333"}}
				db.SaveMetric(metric)
				Expect(err).NotTo(HaveOccurred())

				metric.TimeStamp = 333333
				metric.Instances = []metrics.InstanceMetric{{23456312, 0, "3333"}, {23556312, 1, "6666"}}
				db.SaveMetric(metric)
				Expect(err).NotTo(HaveOccurred())

				Expect(hasMetric("test-app-id", metrics.MetricNameMemory, 111111)).To(BeTrue())
				Expect(hasMetric("test-app-id", metrics.MetricNameMemory, 222222)).To(BeTrue())
				Expect(hasMetric("test-app-id", metrics.MetricNameMemory, 333333)).To(BeTrue())
			})
		})

	})

	Describe("RetrieveMetrics", func() {
		BeforeEach(func() {
			db, err = NewSQLDB(conf, logger)
			Expect(err).NotTo(HaveOccurred())
			cleanMetricsTable()

			start = 0
			end = -1
			appId = "test-app-id"
			metricName = metrics.MetricNameMemory
		})

		AfterEach(func() {
			err = db.Close()
			Expect(err).NotTo(HaveOccurred())
		})

		JustBeforeEach(func() {
			metric = &metrics.Metric{
				AppId: appId,
				Name:  metricName,
				Unit:  metrics.UnitBytes,
			}

			metric.TimeStamp = 666666
			metric.Instances = []metrics.InstanceMetric{{654321, 0, "6666"}, {764321, 1, "8888"}}
			err = db.SaveMetric(metric)
			Expect(err).NotTo(HaveOccurred())

			metric.TimeStamp = 222222
			metric.Instances = []metrics.InstanceMetric{}
			err = db.SaveMetric(metric)
			Expect(err).NotTo(HaveOccurred())

			metric.TimeStamp = 333333
			metric.Instances = []metrics.InstanceMetric{{123456, 0, "3333"}}
			err = db.SaveMetric(metric)
			Expect(err).NotTo(HaveOccurred())

			mtrcs, err = db.RetrieveMetrics("test-app-id", metrics.MetricNameMemory, start, end)
		})

		Context("The app has no metrics", func() {
			BeforeEach(func() {
				appId = "other-app-id"
			})

			It("returns empty metrics", func() {
				Expect(err).NotTo(HaveOccurred())
				Expect(mtrcs).To(BeEmpty())
			})

		})

		Context("when the app has no memory metrics", func() {
			BeforeEach(func() {
				metricName = "other-metric"
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
				Expect(mtrcs).To(HaveLen(3))
				Expect(*mtrcs[0]).To(Equal(
					metrics.Metric{
						AppId:     "test-app-id",
						Name:      metrics.MetricNameMemory,
						Unit:      metrics.UnitBytes,
						TimeStamp: 222222,
						Instances: []metrics.InstanceMetric{},
					}))
				Expect(*mtrcs[1]).To(Equal(
					metrics.Metric{
						AppId:     "test-app-id",
						Name:      metrics.MetricNameMemory,
						Unit:      metrics.UnitBytes,
						TimeStamp: 333333,
						Instances: []metrics.InstanceMetric{{123456, 0, "3333"}},
					}))
				Expect(*mtrcs[2]).To(Equal(
					metrics.Metric{
						AppId:     "test-app-id",
						Name:      metrics.MetricNameMemory,
						Unit:      metrics.UnitBytes,
						TimeStamp: 666666,
						Instances: []metrics.InstanceMetric{{654321, 0, "6666"}, {764321, 1, "8888"}},
					}))
			})
		})

		Context("When retriving part of the metrics", func() {
			BeforeEach(func() {
				start = 234567
				end = 555555
			})

			It("returns correct metrics", func() {
				Expect(err).NotTo(HaveOccurred())
				Expect(mtrcs).To(ConsistOf(&metrics.Metric{
					AppId:     "test-app-id",
					Name:      metrics.MetricNameMemory,
					Unit:      metrics.UnitBytes,
					TimeStamp: 333333,
					Instances: []metrics.InstanceMetric{{123456, 0, "3333"}},
				}))
			})
		})
	})

	Describe("PruneMetrics", func() {
		BeforeEach(func() {
			db, err = NewSQLDB(conf, logger)
			Expect(err).NotTo(HaveOccurred())

			cleanMetricsTable()

			instances := []metrics.InstanceMetric{{123456, 0, "3333"}, {123476, 1, "6666"}}
			metric := &metrics.Metric{
				AppId:     "test-app-id",
				Name:      metrics.MetricNameMemory,
				Unit:      metrics.UnitBytes,
				Instances: instances,
			}

			metric.TimeStamp = 666666
			err = db.SaveMetric(metric)
			Expect(err).NotTo(HaveOccurred())

			metric.TimeStamp = 222222
			err = db.SaveMetric(metric)
			Expect(err).NotTo(HaveOccurred())

			metric.TimeStamp = 333333
			err = db.SaveMetric(metric)
			Expect(err).NotTo(HaveOccurred())

		})

		AfterEach(func() {
			err = db.Close()
			Expect(err).NotTo(HaveOccurred())
		})

		JustBeforeEach(func() {
			err = db.PruneMetrics(before)
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
				Expect(hasMetric("test-app-id", metrics.MetricNameMemory, 666666)).To(BeTrue())
			})
		})

	})

	Describe("GetAppIds", func() {
		BeforeEach(func() {
			db, err = NewSQLDB(conf, logger)
			Expect(err).NotTo(HaveOccurred())

			cleanPolicyTable()
		})

		AfterEach(func() {
			err = db.Close()
			Expect(err).NotTo(HaveOccurred())
		})

		JustBeforeEach(func() {
			appIds, err = db.GetAppIds()
		})

		Context("when policy table is empty", func() {
			It("returns no app ids", func() {
				Expect(err).NotTo(HaveOccurred())
				Expect(appIds).To(BeEmpty())
			})

		})

		Context("when policy table is not empty", func() {
			BeforeEach(func() {
				insertPolicy("first-app-id")
				insertPolicy("second-app-id")
				insertPolicy("third-app-id")
			})

			It("returns all app ids", func() {
				Expect(err).NotTo(HaveOccurred())
				Expect(appIds).To(HaveKey("first-app-id"))
				Expect(appIds).To(HaveKey("second-app-id"))
				Expect(appIds).To(HaveKey("third-app-id"))
			})

		})

	})

})
