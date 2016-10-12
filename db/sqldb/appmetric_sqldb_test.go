package sqldb_test

import (
	. "autoscaler/db/sqldb"
	"autoscaler/eventgenerator/model"
	"autoscaler/models"

	"code.cloudfoundry.org/lager"
	"github.com/lib/pq"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"os"
	"time"
)

var _ = Describe("AppMetricSQLDB", func() {
	var (
		adb               *AppMetricSQLDB
		url               string
		logger            lager.Logger
		err               error
		appMetrics        []*model.AppMetric
		start, end        int64
		before            int64
		appId, metricName string
	)

	BeforeEach(func() {
		logger = lager.NewLogger("appmetric-sqldb-test")
		url = os.Getenv("DBURL")
	})

	Describe("NewAppMetricSQLDB", func() {
		JustBeforeEach(func() {
			adb, err = NewAppMetricSQLDB(url, logger)
		})

		AfterEach(func() {
			if adb != nil {
				err = adb.Close()
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

		Context("when db url is correct", func() {
			It("should not error", func() {
				Expect(err).NotTo(HaveOccurred())
				Expect(adb).NotTo(BeNil())
			})
		})
	})

	Describe("SaveAppMetric", func() {
		BeforeEach(func() {
			adb, err = NewAppMetricSQLDB(url, logger)
			Expect(err).NotTo(HaveOccurred())
			cleanAppMetricTable()
		})

		AfterEach(func() {
			err = adb.Close()
			Expect(err).NotTo(HaveOccurred())
		})

		Context("When inserting a metric of an app", func() {
			BeforeEach(func() {
				appMetric := &model.AppMetric{
					AppId:      "test-app-id",
					MetricType: models.MetricNameMemory,
					Unit:       models.UnitBytes,
					Timestamp:  11111111,
					Value:      GetInt64Pointer(30000),
				}
				err = adb.SaveAppMetric(appMetric)
			})

			It("has the appMetric in database", func() {
				Expect(err).NotTo(HaveOccurred())
				Expect(hasAppMetric("test-app-id", models.MetricNameMemory, 11111111)).To(BeTrue())
			})
		})

	})
	Describe("RetrieveAppMetrics", func() {
		value1 := GetInt64Pointer(10000)
		value2 := GetInt64Pointer(50000)
		value3 := GetInt64Pointer(30000)
		BeforeEach(func() {
			adb, err = NewAppMetricSQLDB(url, logger)
			Expect(err).NotTo(HaveOccurred())
			cleanAppMetricTable()

			appMetric := &model.AppMetric{
				AppId:      "test-app-id",
				MetricType: models.MetricNameMemory,
				Unit:       models.UnitBytes,
				Timestamp:  11111111,
				Value:      value1,
			}
			err = adb.SaveAppMetric(appMetric)
			Expect(err).NotTo(HaveOccurred())

			appMetric.Timestamp = 33333333
			appMetric.Value = value2
			err = adb.SaveAppMetric(appMetric)
			Expect(err).NotTo(HaveOccurred())

			appMetric.Timestamp = 55555555
			appMetric.Value = value3
			err = adb.SaveAppMetric(appMetric)
			Expect(err).NotTo(HaveOccurred())

			appId = "test-app-id"
			metricName = models.MetricNameMemory
			start = 0
			end = time.Now().UnixNano()

		})

		AfterEach(func() {
			err = adb.Close()
			Expect(err).NotTo(HaveOccurred())
		})

		JustBeforeEach(func() {
			appMetrics, err = adb.RetrieveAppMetrics(appId, metricName, start, end)
		})

		Context("The app has no metrics", func() {
			BeforeEach(func() {
				appId = "app-id-no-metrics"
			})

			It("returns empty metrics", func() {
				Expect(err).NotTo(HaveOccurred())
				Expect(appMetrics).To(BeEmpty())
			})

		})

		Context("when the app has no metrics with the given metric name", func() {
			BeforeEach(func() {
				metricName = "metric-name-no-metrics"
			})

			It("returns empty metrics", func() {
				Expect(err).NotTo(HaveOccurred())
				Expect(appMetrics).To(BeEmpty())
			})
		})

		Context("when end time is before all the metrics timestamps", func() {
			BeforeEach(func() {
				end = 11111110
			})

			It("returns empty metrics", func() {
				Expect(err).NotTo(HaveOccurred())
				Expect(appMetrics).To(BeEmpty())
			})

		})

		Context("when start time is after all the metrics timestamps", func() {
			BeforeEach(func() {
				start = 55555556
			})

			It("returns empty metrics", func() {
				Expect(err).NotTo(HaveOccurred())
				Expect(appMetrics).To(BeEmpty())
			})

		})

		Context("when start time > end time", func() {
			BeforeEach(func() {
				start = 33333333
				end = 22222222
			})

			It("returns empty metrics", func() {
				Expect(err).NotTo(HaveOccurred())
				Expect(appMetrics).To(BeEmpty())
			})
		})

		Context("when retriving all the appMetrics)", func() {
			It("returns all the appMetrics ordered by timestamp", func() {
				Expect(err).NotTo(HaveOccurred())
				Expect(appMetrics).To(Equal([]*model.AppMetric{
					&model.AppMetric{
						AppId:      "test-app-id",
						MetricType: models.MetricNameMemory,
						Unit:       models.UnitBytes,
						Timestamp:  11111111,
						Value:      value1,
					},
					&model.AppMetric{
						AppId:      "test-app-id",
						MetricType: models.MetricNameMemory,
						Unit:       models.UnitBytes,
						Timestamp:  33333333,
						Value:      value2,
					},
					&model.AppMetric{
						AppId:      "test-app-id",
						MetricType: models.MetricNameMemory,
						Unit:       models.UnitBytes,
						Timestamp:  55555555,
						Value:      value3,
					}}))
			})
		})

		Context("when retriving part of the appMetrics)", func() {
			BeforeEach(func() {
				start = 22222222
				end = 66666666
			})
			It("returns correct appMetrics ordered by timestamp", func() {
				Expect(err).NotTo(HaveOccurred())
				Expect(appMetrics).To(Equal([]*model.AppMetric{
					&model.AppMetric{
						AppId:      "test-app-id",
						MetricType: models.MetricNameMemory,
						Unit:       models.UnitBytes,
						Timestamp:  33333333,
						Value:      value2,
					},
					&model.AppMetric{
						AppId:      "test-app-id",
						MetricType: models.MetricNameMemory,
						Unit:       models.UnitBytes,
						Timestamp:  55555555,
						Value:      value3,
					}}))
			})
		})

	})

	Describe("PruneAppMetrics", func() {
		BeforeEach(func() {
			adb, err = NewAppMetricSQLDB(url, logger)
			Expect(err).NotTo(HaveOccurred())

			cleanAppMetricTable()

			appMetric := &model.AppMetric{
				AppId:      "test-app-id",
				MetricType: models.MetricNameMemory,
				Unit:       models.UnitBytes,
				Timestamp:  11111111,
				Value:      10000,
			}

			err = adb.SaveAppMetric(appMetric)
			Expect(err).NotTo(HaveOccurred())

			appMetric.Timestamp = 55555555
			appMetric.Value = 50000
			err = adb.SaveAppMetric(appMetric)
			Expect(err).NotTo(HaveOccurred())

			appMetric.Timestamp = 33333333
			appMetric.Value = 30000
			err = adb.SaveAppMetric(appMetric)
			Expect(err).NotTo(HaveOccurred())

		})

		AfterEach(func() {
			err = adb.Close()
			Expect(err).NotTo(HaveOccurred())
		})

		JustBeforeEach(func() {
			err = adb.PruneAppMetrics(before)
		})

		Context("when pruning app metrics before all the timestamps of metrics", func() {
			BeforeEach(func() {
				before = 0
			})

			It("does not remove any metrics", func() {
				Expect(err).NotTo(HaveOccurred())
				Expect(getNumberOfAppMetrics()).To(Equal(3))
			})
		})

		Context("when pruning all the metrics", func() {
			BeforeEach(func() {
				before = time.Now().UnixNano()
			})

			It("empties the app metrics table", func() {
				Expect(err).NotTo(HaveOccurred())
				Expect(getNumberOfAppMetrics()).To(Equal(0))
			})
		})

		Context("when pruning part of the metrics", func() {
			BeforeEach(func() {
				before = 33333333
			})

			It("removes metrics before the time specified", func() {
				Expect(err).NotTo(HaveOccurred())
				Expect(getNumberOfAppMetrics()).To(Equal(1))
				Expect(hasAppMetric("test-app-id", models.MetricNameMemory, 55555555)).To(BeTrue())
			})
		})

		Context("When not connected to the database", func() {
			BeforeEach(func() {
				before = 0
				err = adb.Close()
				Expect(err).NotTo(HaveOccurred())
			})

			It("should error", func() {
				Expect(err).To(MatchError(MatchRegexp("sql: database is closed")))
			})
		})

	})
})
