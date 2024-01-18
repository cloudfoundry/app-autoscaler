package sqldb_test

import (
	"context"
	"os"
	"strings"
	"sync"
	"time"

	. "code.cloudfoundry.org/app-autoscaler/src/autoscaler/testhelpers"

	"code.cloudfoundry.org/app-autoscaler/src/autoscaler/db"
	. "code.cloudfoundry.org/app-autoscaler/src/autoscaler/db/sqldb"
	"code.cloudfoundry.org/app-autoscaler/src/autoscaler/models"

	"code.cloudfoundry.org/lager/v3"
	"github.com/go-sql-driver/mysql"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("AppMetricSQLDB", func() {
	var (
		adb            *AppMetricSQLDB
		dbConfig       db.DatabaseConfig
		dbHost         = os.Getenv("DB_HOST")
		logger         lager.Logger
		err            error
		appMetrics     []*models.AppMetric
		start, end     int64
		before         int64
		metricName     string
		testMetricName string
		testMetricUnit string
		appId          string
		orderType      db.OrderType
	)

	dbUrl := GetDbUrl()
	BeforeEach(func() {
		logger = lager.NewLogger("appmetric-sqldb-test")
		logger.RegisterSink(lager.NewWriterSink(GinkgoWriter, lager.DEBUG))
		dbConfig = db.DatabaseConfig{
			URL:                   dbUrl,
			MaxOpenConnections:    10,
			MaxIdleConnections:    5,
			ConnectionMaxLifetime: 5 * time.Second,
			ConnectionMaxIdleTime: 10 * time.Millisecond,
		}
		testMetricName = addProcessIdTo("Test-Metric-Name")
		testMetricUnit = addProcessIdTo("Test-Metric-Unit")

		adb, err = NewAppMetricSQLDB(dbConfig, logger)
		FailOnError("NewAppMetricSQLDB", err)
		DeferCleanup(func() error {
			if adb != nil {
				return adb.Close()
			}
			return nil
		})

		appId = addProcessIdTo("an-app-id")
		cleanAppMetricTable(appId)
		DeferCleanup(func() { cleanAppMetricTable(appId) })
	})

	Context("NewAppMetricSQLDB", func() {
		JustBeforeEach(func() {
			if adb != nil {
				_ = adb.Close()
			}
			adb, err = NewAppMetricSQLDB(dbConfig, logger)
		})

		Context("when db url is not correct", func() {
			BeforeEach(func() {
				if !strings.Contains(dbUrl, "postgres") {
					Skip("Postgres only test")
				}
				dbConfig.URL = "postgres://not-exist-user:not-exist-password@localhost/autoscaler?sslmode=disable"
			})
			It("should throw an error", func() {
				Expect(err).To(HaveOccurred())
			})
		})

		Context("when mysql db url is not correct", func() {
			BeforeEach(func() {
				if strings.Contains(dbUrl, "postgres") {
					Skip("Not configured for postgres")
				}
				dbConfig.URL = "not-exist-user:not-exist-password@tcp(" + dbHost + ")/autoscaler?tls=false"
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

		Context("When inserting an app_metric", func() {
			BeforeEach(func() {
				appMetric := &models.AppMetric{
					AppId:      appId,
					MetricType: testMetricName,
					Unit:       testMetricUnit,
					Timestamp:  11111111,
					Value:      "300",
				}
				err = adb.SaveAppMetric(appMetric)
				Expect(err).NotTo(HaveOccurred())
			})

			It("has the appMetric in database", func() {
				Expect(hasAppMetric(appId, testMetricName, 11111111, "300")).To(BeTrue())
			})
		})

	})
	Context("SaveAppMetricsInBulk", func() {

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
						AppId:      appId,
						MetricType: testMetricName,
						Unit:       testMetricUnit,
						Timestamp:  11111111,
						Value:      "300",
					},
					{
						AppId:      appId,
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
				Expect(hasAppMetric(appId, testMetricName, 11111111, "300")).To(BeTrue())
				Expect(hasAppMetric(appId, testMetricName, 22222222, "400")).To(BeTrue())
			})
		})
		Context("When there are errors in transaction", func() {
			var wg = &sync.WaitGroup{}
			BeforeEach(func() {
				testMetricName = "Test-Metric-Name-this-is-a-too-long-metric-name-too-looooooooooooooooooooooooooooooooooooooooooooooooooooooooooooooooooooooooooooooooooooooooooooooooooooooooong"
				appMetrics := []*models.AppMetric{
					{
						AppId:      appId,
						MetricType: testMetricName,
						Unit:       testMetricUnit,
						Timestamp:  11111111,
						Value:      "300",
					},
					{
						AppId:      appId,
						MetricType: testMetricName,
						Unit:       testMetricUnit,
						Timestamp:  22222222,
						Value:      "400",
					},
				}
				wg.Add(100)
				for i := 0; i < 100; i++ {
					go func() {
						err := adb.SaveAppMetricsInBulk(appMetrics)
						Expect(err).To(HaveOccurred())
						defer wg.Done()
					}()
				}
			})
			It("all connections should be released after transactions' rolling back", func() {
				wg.Wait()
				Eventually(func() int {
					return adb.GetDBStatus().OpenConnections
				}, 120*time.Second, 10*time.Millisecond).Should(BeZero())
			})
		})
	})
	Context("RetrieveAppMetrics", func() {
		BeforeEach(func() {
			orderType = db.ASC

			appMetric := &models.AppMetric{
				AppId:      appId,
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

			metricName = testMetricName
			start = 0
			end = -1

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
						AppId:      appId,
						MetricType: testMetricName,
						Unit:       testMetricUnit,
						Timestamp:  11111111,
						Value:      "100",
					},
					{
						AppId:      appId,
						MetricType: testMetricName,
						Unit:       testMetricUnit,
						Timestamp:  33333333,
						Value:      "200",
					},
					{
						AppId:      appId,
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
						AppId:      appId,
						MetricType: testMetricName,
						Unit:       testMetricUnit,
						Timestamp:  33333333,
						Value:      "200",
					},
					{
						AppId:      appId,
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
						AppId:      appId,
						MetricType: testMetricName,
						Unit:       testMetricUnit,
						Timestamp:  55555555,
						Value:      "300",
					},
					{
						AppId:      appId,
						MetricType: testMetricName,
						Unit:       testMetricUnit,
						Timestamp:  33333333,
						Value:      "200",
					},
					{
						AppId:      appId,
						MetricType: testMetricName,
						Unit:       testMetricUnit,
						Timestamp:  11111111,
						Value:      "100",
					},
				}))
			})
		})
	})

	Context("PruneAppMetrics", Serial, func() {
		BeforeEach(func() {
			appMetric := &models.AppMetric{
				AppId:      appId,
				MetricType: testMetricName,
				Unit:       testMetricUnit,
				Timestamp:  11111111,
				Value:      "100",
			}

			err = adb.SaveAppMetric(appMetric)
			FailOnError("SaveAppMetric", err)

			appMetric.Timestamp = 55555555
			appMetric.Value = "200"
			err = adb.SaveAppMetric(appMetric)
			FailOnError("SaveAppMetric", err)

			appMetric.Timestamp = 33333333
			appMetric.Value = "300"
			err = adb.SaveAppMetric(appMetric)
			FailOnError("SaveAppMetric", err)
		})

		JustBeforeEach(func() {
			err = adb.PruneAppMetrics(context.TODO(), before)
		})

		Context("when pruning app metrics before all the timestamps of metrics", func() {
			BeforeEach(func() {
				before = 0
			})

			It("does not remove any metrics", func() {
				Expect(err).NotTo(HaveOccurred())
				Expect(getNumberOfMetricsForApp(appId)).To(Equal(3))
			})
		})

		Context("when pruning all the metrics", func() {
			BeforeEach(func() {
				before = time.Now().UnixNano()
			})

			It("empties the app metrics table", func() {
				Expect(err).NotTo(HaveOccurred())
				Expect(getNumberOfMetricsForApp(appId)).To(Equal(0))
			})
		})

		Context("when pruning part of the metrics", func() {
			BeforeEach(func() {
				before = 33333333
			})

			It("removes metrics before the time specified", func() {
				Expect(err).NotTo(HaveOccurred())
				Expect(getNumberOfMetricsForApp(appId)).To(Equal(1))
				Expect(hasAppMetric(appId, testMetricName, 55555555, "200")).To(BeTrue())
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
