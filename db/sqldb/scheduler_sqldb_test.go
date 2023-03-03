package sqldb_test

import (
	"os"
	"strings"
	"time"

	"code.cloudfoundry.org/app-autoscaler/src/autoscaler/testhelpers"

	"code.cloudfoundry.org/app-autoscaler/src/autoscaler/db"
	. "code.cloudfoundry.org/app-autoscaler/src/autoscaler/db/sqldb"
	"code.cloudfoundry.org/app-autoscaler/src/autoscaler/models"

	"code.cloudfoundry.org/lager/v3"
	"github.com/go-sql-driver/mysql"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("SchedulerSqldb", func() {
	var (
		dbConfig  db.DatabaseConfig
		logger    lager.Logger
		sdb       *SchedulerSQLDB
		err       error
		schedules map[string]*models.ActiveSchedule
		dbHost    = os.Getenv("DB_HOST")
	)

	dbUrl := testhelpers.GetDbUrl()
	BeforeEach(func() {
		logger = lager.NewLogger("scheduler-sqldb-test")
		logger.RegisterSink(lager.NewWriterSink(GinkgoWriter, lager.DEBUG))
		dbConfig = db.DatabaseConfig{
			URL:                   dbUrl,
			MaxOpenConnections:    10,
			MaxIdleConnections:    5,
			ConnectionMaxLifetime: 10 * time.Second,
			ConnectionMaxIdleTime: 10 * time.Second,
		}
	})

	Describe("NewSchedulerSQLDB", func() {
		JustBeforeEach(func() {
			sdb, err = NewSchedulerSQLDB(dbConfig, logger)
		})

		AfterEach(func() {
			if sdb != nil {
				err = sdb.Close()
				Expect(err).NotTo(HaveOccurred())
			}
		})

		Context("when db url is not correct", func() {
			BeforeEach(func() {
				if !strings.Contains(dbUrl, "postgres") {
					Skip("Not configured for postgres")
				}
				dbConfig.URL = "postgres://not-exist-user:not-exist-password@" + dbHost + "/autoscaler?sslmode=disable"
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

		Context("when url is correct", func() {
			It("should not error", func() {
				Expect(err).NotTo(HaveOccurred())
				Expect(sdb).NotTo(BeNil())
			})
		})
	})

	Describe("GetActiveSchedules", func() {
		BeforeEach(func() {
			sdb, err = NewSchedulerSQLDB(dbConfig, logger)
			Expect(err).NotTo(HaveOccurred())
			err = cleanSchedulerActiveScheduleTable()
			Expect(err).NotTo(HaveOccurred())
		})

		AfterEach(func() {
			err = sdb.Close()
			Expect(err).NotTo(HaveOccurred())
		})

		JustBeforeEach(func() {
			schedules, err = sdb.GetActiveSchedules()
		})

		Context("when the table is empty", func() {
			It("returns empty active schedules", func() {
				Expect(err).NotTo(HaveOccurred())
				Expect(schedules).To(HaveLen(0))
			})
		})

		Context("when the table is not empty", func() {
			BeforeEach(func() {
				err = insertSchedulerActiveSchedule(111111, "app-id-1", 1, 2, 10, 5)
				Expect(err).NotTo(HaveOccurred())
				err = insertSchedulerActiveSchedule(222222, "app-id-2", 2, 3, 7, 5)
				Expect(err).NotTo(HaveOccurred())
				err = insertSchedulerActiveSchedule(333333, "app-id-3", 3, 5, 12, -1)
				Expect(err).NotTo(HaveOccurred())
			})

			It("returns all the active schedules", func() {
				Expect(err).NotTo(HaveOccurred())
				Expect(schedules).To(HaveKeyWithValue("app-id-1", &models.ActiveSchedule{
					ScheduleId:         "111111",
					InstanceMin:        2,
					InstanceMax:        10,
					InstanceMinInitial: 5,
				}))
				Expect(schedules).To(HaveKeyWithValue("app-id-2", &models.ActiveSchedule{
					ScheduleId:         "222222",
					InstanceMin:        3,
					InstanceMax:        7,
					InstanceMinInitial: 5,
				}))
				Expect(schedules).To(HaveKeyWithValue("app-id-3", &models.ActiveSchedule{
					ScheduleId:         "333333",
					InstanceMin:        5,
					InstanceMax:        12,
					InstanceMinInitial: 0,
				}))
			})
		})
		Context("when there is database error", func() {
			BeforeEach(func() {
				sdb.Close()
			})
			It("should error", func() {
				Expect(err).To(HaveOccurred())
			})
		})

	})

})
