package sqldb_test

import (
	"os"
	"strings"
	"sync"
	"time"

	"code.cloudfoundry.org/app-autoscaler/src/autoscaler/db"
	. "code.cloudfoundry.org/app-autoscaler/src/autoscaler/db/sqldb"
	"code.cloudfoundry.org/app-autoscaler/src/autoscaler/models"

	"code.cloudfoundry.org/lager"
	"github.com/go-sql-driver/mysql"
	"github.com/lib/pq"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("AppMetricSQLDB", func() {
	var (
		adb               *AppMetricSQLDB
		dbConfig          db.DatabaseConfig
		logger            lager.Logger
		err               error
		appMetrics        []*models.AppMetric
		start, end        int64
		before            int64
		appId, metricName string
		testMetricName    string
		testMetricUnit    = "Test-Metric-Unit"
		testAppId         = "Test-App-ID"
		orderType         db.OrderType
	)

	BeforeEach(func() {
		logger = lager.NewLogger("appmetric-sqldb-test")
		dbConfig = db.DatabaseConfig{
			URL:                   os.Getenv("DBURL"),
			MaxOpenConnections:    10,
			MaxIdleConnections:    5,
			ConnectionMaxLifetime: 10 * time.Second,
			ConnectionMaxIdleTime: 10 * time.Second,
		}
		testMetricName = "Test-Metric-Name"

	})

	Context("NewAppMetricSQLDB", func() {
		JustBeforeEach(func() {
			adb, err = NewAppMetricSQLDB(dbConfig, logger)
		})

		AfterEach(func() {
			if adb != nil {
				err = adb.Close()
				Expect(err).NotTo(HaveOccurred())
			}
		})

		Context("when db url is not correct", func() {
			BeforeEach(func() {
				if !strings.Contains(os.Getenv("DBURL"), "postgres") {
					Skip("Not configured for postgres")
				}
				dbConfig.URL = "postgres://not-exist-user:not-exist-password@localhost/autoscaler?sslmode=disable"
			})
			It("should throw an error", func() {
				Expect(err).To(BeAssignableToTypeOf(&pq.Error{}))
			})
		})

		Context("when mysql db url is not correct", func() {
			BeforeEach(func() {
				if strings.Contains(os.Getenv("DBURL"), "postgres") {
					Skip("Not configured for mysql")
				}
				dbConfig.URL = "not-exist-user:not-exist-password@tcp(localhost)/autoscaler?tls=false"
			})
			It("should throw an error", func() {
				Expect(err).To(BeAssignableToTypeOf(&mysql.MySQLError{}))
			})
		})

		Context("when db url is correct", func() {
			It("should not error", func() {
				Expect(err).NotTo(HaveOccurred())
				Expect(adb).NotTo(BeNil())
			})
		})
	})

	Context("SaveAppMetric", func() {
		BeforeEach(func() {
			adb, err = NewAppMetricSQLDB(dbConfig, logger)
			Expect(err).NotTo(HaveOccurred())
			cleanAppMetricTable()
		})

		AfterEach(func() {
			err = adb.Close()
			Expect(err).NotTo(HaveOccurred())
		})

		Context("When inserting an app_metric", func() {
			BeforeEach(func() {
				appMetric := &models.AppMetric{
					AppId:      testAppId,
					MetricType: testMetricName,
					Unit:       testMetricUnit,
					Timestamp:  11111111,
					Value:      "300",
				}
				err = adb.SaveAppMetric(appMetric)
				Expect(err).NotTo(HaveOccurred())
			})

			It("has the appMetric in database", func() {
				Expect(hasAppMetric(testAppId, testMetricName, 11111111, "300")).To(BeTrue())
			})
		})

	})
	Context("SaveAppMetricsInBulk", func() {
		BeforeEach(func() {
			adb, err = NewAppMetricSQLDB(dbConfig, logger)
			Expect(err).NotTo(HaveOccurred())
			cleanAppMetricTable()
		})

		AfterEach(func() {
			err = adb.Close()
			Expect(err).NotTo(HaveOccurred())
		})
		Context("When inserting an empty array of app_metric", func() {
			BeforeEach(func() {
				appMetrics := []*models.AppMetric{}
				err = adb.SaveAppMetricsInBulk(appMetrics)
			})
			It("Should return nil", func() {
				Expect(err).To(BeNil())
			})
		})

		Context("When inserting an array of app_metric", func() {
			BeforeEach(func() {
				appMetrics := []*models.AppMetric{
					{
						AppId:      testAppId,
						MetricType: testMetricName,
						Unit:       testMetricUnit,
						Timestamp:  11111111,
						Value:      "300",
					},
					{
						AppId:      testAppId,
						MetricType: testMetricName,
						Unit:       testMetricUnit,
						Timestamp:  22222222,
						Value:      "400",
					},
				}

				err = adb.SaveAppMetricsInBulk(appMetrics)
				Expect(err).NotTo(HaveOccurred())
			})
			It("has the array of app_metric in database", func() {
				Expect(hasAppMetric(testAppId, testMetricName, 11111111, "300")).To(BeTrue())
				Expect(hasAppMetric(testAppId, testMetricName, 22222222, "400")).To(BeTrue())
			})
		})
		Context("When there are errors in transaction", func() {
			var lock = &sync.Mutex{}
			var count = 0
			BeforeEach(func() {
				testMetricName = "Test-Metric-Name-this-is-a-too-long-metric-name-too-looooooooooooooooooooooooooooooooooooooooooooooooooooooooooooooooooooooooooooooooooooooooooooooooooooooooong"
				appMetrics := []*models.AppMetric{
					{
						AppId:      testAppId,
						MetricType: testMetricName,
						Unit:       testMetricUnit,
						Timestamp:  11111111,
						Value:      "300",
					},
					{
						AppId:      testAppId,
						MetricType: testMetricName,
						Unit:       testMetricUnit,
						Timestamp:  22222222,
						Value:      "400",
					},
				}
				for i := 0; i < 100; i++ {
					go func(count *int) {
						err := adb.SaveAppMetricsInBulk(appMetrics)
						Expect(err).To(HaveOccurred())
						lock.Lock()
						*count = *count + 1
						lock.Unlock()
					}(&count)

				}

			})
			It("all connections should be released after transactions' rolling back", func() {
				Eventually(func() int {
					lock.Lock()
					defer lock.Unlock()
					return count
				}, 120*time.Second, 1*time.Second).Should(Equal(100))
				Eventually(func() int {
					return adb.GetDBStatus().OpenConnections
				}, 120*time.Second, 10*time.Millisecond).Should(BeZero())
			})
		})
	})
	Context("RetrieveAppMetrics", func() {
		BeforeEach(func() {
			adb, err = NewAppMetricSQLDB(dbConfig, logger)
			Expect(err).NotTo(HaveOccurred())
			cleanAppMetricTable()
			orderType = db.ASC

			appMetric := &models.AppMetric{
				AppId:      testAppId,
				MetricType: testMetricName,
				Unit:       testMetricUnit,
				Timestamp:  11111111,
				Value:      "100",
			}
			err = adb.SaveAppMetric(appMetric)
			Expect(err).NotTo(HaveOccurred())

			appMetric.Timestamp = 33333333
			appMetric.Value = "200"
			err = adb.SaveAppMetric(appMetric)
			Expect(err).NotTo(HaveOccurred())

			appMetric.Timestamp = 55555555
			appMetric.Value = "300"
			err = adb.SaveAppMetric(appMetric)
			Expect(err).NotTo(HaveOccurred())

			appId = testAppId
			metricName = testMetricName
			start = 0
			end = -1

		})

		AfterEach(func() {
			err = adb.Close()
			Expect(err).NotTo(HaveOccurred())
		})

		JustBeforeEach(func() {
			appMetrics, err = adb.RetrieveAppMetrics(appId, metricName, start, end, orderType)
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

		Context("when retrieving all the appMetrics)", func() {
			It("returns all the appMetrics ordered by timestamp", func() {
				Expect(err).NotTo(HaveOccurred())
				Expect(appMetrics).To(Equal([]*models.AppMetric{
					{
						AppId:      testAppId,
						MetricType: testMetricName,
						Unit:       testMetricUnit,
						Timestamp:  11111111,
						Value:      "100",
					},
					{
						AppId:      testAppId,
						MetricType: testMetricName,
						Unit:       testMetricUnit,
						Timestamp:  33333333,
						Value:      "200",
					},
					{
						AppId:      testAppId,
						MetricType: testMetricName,
						Unit:       testMetricUnit,
						Timestamp:  55555555,
						Value:      "300",
					}}))
			})
		})

		Context("when retrieving part of the appMetrics", func() {
			BeforeEach(func() {
				start = 22222222
				end = 66666666
			})
			It("returns correct appMetrics ordered by timestamp", func() {
				Expect(err).NotTo(HaveOccurred())
				Expect(appMetrics).To(Equal([]*models.AppMetric{
					{
						AppId:      testAppId,
						MetricType: testMetricName,
						Unit:       testMetricUnit,
						Timestamp:  33333333,
						Value:      "200",
					},
					{
						AppId:      testAppId,
						MetricType: testMetricName,
						Unit:       testMetricUnit,
						Timestamp:  55555555,
						Value:      "300",
					}}))
			})
		})

		Context("when retrieving the appMetrics with descending order)", func() {
			BeforeEach(func() {
				orderType = db.DESC
			})
			It("returns all the appMetrics ordered by timestamp with descending order", func() {
				Expect(err).NotTo(HaveOccurred())
				Expect(appMetrics).To(Equal([]*models.AppMetric{
					{
						AppId:      testAppId,
						MetricType: testMetricName,
						Unit:       testMetricUnit,
						Timestamp:  55555555,
						Value:      "300",
					},
					{
						AppId:      testAppId,
						MetricType: testMetricName,
						Unit:       testMetricUnit,
						Timestamp:  33333333,
						Value:      "200",
					},
					{
						AppId:      testAppId,
						MetricType: testMetricName,
						Unit:       testMetricUnit,
						Timestamp:  11111111,
						Value:      "100",
					},
				}))
			})
		})
	})

	Context("PruneAppMetrics", func() {
		BeforeEach(func() {
			adb, err = NewAppMetricSQLDB(dbConfig, logger)
			Expect(err).NotTo(HaveOccurred())

			cleanAppMetricTable()

			appMetric := &models.AppMetric{
				AppId:      testAppId,
				MetricType: testMetricName,
				Unit:       testMetricUnit,
				Timestamp:  11111111,
				Value:      "100",
			}

			err = adb.SaveAppMetric(appMetric)
			Expect(err).NotTo(HaveOccurred())

			appMetric.Timestamp = 55555555
			appMetric.Value = "200"
			err = adb.SaveAppMetric(appMetric)
			Expect(err).NotTo(HaveOccurred())

			appMetric.Timestamp = 33333333
			appMetric.Value = "300"
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
				Expect(hasAppMetric(testAppId, testMetricName, 55555555, "200")).To(BeTrue())
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
