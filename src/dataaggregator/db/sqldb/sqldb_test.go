package sqldb_test

import (
	"dataaggregator/config"
	. "dataaggregator/db/sqldb"

	"code.cloudfoundry.org/lager"
	"github.com/lib/pq"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"dataaggregator/appmetric"
	"dataaggregator/policy"
	"os"
)

var _ = Describe("Sqldb", func() {
	var (
		conf     *config.Config
		db       *SQLDB
		logger   lager.Logger
		err      error
		policies []*policy.PolicyJson
	)

	BeforeEach(func() {
		logger = lager.NewLogger("sqldb-test")
		dbUrl := os.Getenv("DBURL")
		conf = &config.Config{PolicyDbUrl: dbUrl, AppMetricDbUrl: dbUrl}
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
			Context("when policy db url is not correct", func() {
				BeforeEach(func() {
					conf.PolicyDbUrl = "postgres://non-exist-user:non-exist-password@localhost/autoscaler?sslmode=disable"
				})
				It("should error", func() {
					Expect(err).To(BeAssignableToTypeOf(&pq.Error{}))
				})
			})
			Context("when appmetric db url is not correct", func() {
				BeforeEach(func() {
					conf.AppMetricDbUrl = "postgres://non-exist-user:non-exist-password@localhost/autoscaler?sslmode=disable"
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

	Describe("RetrievePolicies", func() {
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
			insertPolicy("first-app-id")
			insertPolicy("second-app-id")
			insertPolicy("third-app-id")
			policies, err = db.RetrievePolicies()
		})

		Context("when retriving all the policies)", func() {
			It("returns all the policies", func() {
				Expect(err).NotTo(HaveOccurred())
				Expect(policies).To(ConsistOf(
					&policy.PolicyJson{
						AppId:     "first-app-id",
						PolicyStr: `{"instance_min_count": 1,"instance_max_count": 5}`,
					},
					&policy.PolicyJson{
						AppId:     "second-app-id",
						PolicyStr: `{"instance_min_count": 1,"instance_max_count": 5}`,
					},
					&policy.PolicyJson{
						AppId:     "third-app-id",
						PolicyStr: `{"instance_min_count": 1,"instance_max_count": 5}`,
					},
				))
			})
		})
	})
	Describe("SaveAppMetric", func() {
		BeforeEach(func() {
			db, err = NewSQLDB(conf, logger)
			Expect(err).NotTo(HaveOccurred())
			cleanAppMetricTable()
		})

		AfterEach(func() {
			err = db.Close()
			Expect(err).NotTo(HaveOccurred())
		})

		Context("When inserting a metric of an app", func() {
			BeforeEach(func() {
				appMetric := &appmetric.AppMetric{
					AppId:      "test-app-id",
					MetricType: "MemoryUsage",
					Unit:       "bytes",
					Timestamp:  11111111,
					Value:      30000,
				}
				err = db.SaveAppMetric(appMetric)
			})

			It("has the appMetric in database", func() {
				Expect(err).NotTo(HaveOccurred())
				Expect(hasAppMetric("test-app-id", "MemoryUsage", 11111111)).To(BeTrue())
			})
		})

	})
})
