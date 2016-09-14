package sqldb_test

import (
	. "db/sqldb"
	"eventgenerator/model"
	"models"

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
					Value:      30000,
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
						Value:      10000,
					},
					&model.AppMetric{
						AppId:      "test-app-id",
						MetricType: models.MetricNameMemory,
						Unit:       models.UnitBytes,
						Timestamp:  33333333,
						Value:      30000,
					},
					&model.AppMetric{
						AppId:      "test-app-id",
						MetricType: models.MetricNameMemory,
						Unit:       models.UnitBytes,
						Timestamp:  55555555,
						Value:      50000,
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
						Value:      30000,
					},
					&model.AppMetric{
						AppId:      "test-app-id",
						MetricType: models.MetricNameMemory,
						Unit:       models.UnitBytes,
						Timestamp:  55555555,
						Value:      50000,
					}}))
			})
		})

	})

})
