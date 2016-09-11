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
		adb        *AppMetricSQLDB
		url        string
		logger     lager.Logger
		err        error
		appMetrics []*model.AppMetric
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
		})

		AfterEach(func() {
			err = adb.Close()
			Expect(err).NotTo(HaveOccurred())
		})

		JustBeforeEach(func() {
			insertAppMetric("first-app-id")
			insertAppMetric("first-app-id")
			insertAppMetric("first-app-id")
			appMetrics, err = adb.RetrieveAppMetrics("first-app-id", testMetricType, 0, time.Now().UnixNano())
		})

		Context("when retriving all the appMetrics)", func() {
			It("returns all the appMetrics", func() {
				Expect(err).NotTo(HaveOccurred())
				Expect(appMetrics).To(ConsistOf(
					&model.AppMetric{
						AppId:      "first-app-id",
						MetricType: testMetricType,
						Unit:       testUnit,
						Value:      testValue,
						Timestamp:  testTimestamp,
					},
					&model.AppMetric{
						AppId:      "first-app-id",
						MetricType: testMetricType,
						Unit:       testUnit,
						Value:      testValue,
						Timestamp:  testTimestamp,
					},
					&model.AppMetric{
						AppId:      "first-app-id",
						MetricType: testMetricType,
						Unit:       testUnit,
						Value:      testValue,
						Timestamp:  testTimestamp,
					},
				))
			})
		})
	})

})
