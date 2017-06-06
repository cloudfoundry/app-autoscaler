package sqldb_test

import (
	. "autoscaler/db/sqldb"
	"autoscaler/models"

	"code.cloudfoundry.org/lager"
	"github.com/lib/pq"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"os"
)

var _ = Describe("SchedulerSqldb", func() {
	var (
		url       string
		logger    lager.Logger
		sdb       *SchedulerSQLDB
		err       error
		schedules map[string]*models.ActiveSchedule
		appIdMap  map[string]struct{}
	)

	BeforeEach(func() {
		logger = lager.NewLogger("scheduler-sqldb-test")
		url = os.Getenv("DBURL")
	})

	Describe("NewSchedulerSQLDB", func() {
		JustBeforeEach(func() {
			sdb, err = NewSchedulerSQLDB(url, logger)
		})

		AfterEach(func() {
			if sdb != nil {
				err = sdb.Close()
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
				Expect(sdb).NotTo(BeNil())
			})
		})
	})

	Describe("GetActiveSchedules", func() {
		BeforeEach(func() {
			sdb, err = NewSchedulerSQLDB(url, logger)
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

	Describe("SyncActivesSchedules", func() {
		BeforeEach(func() {
			sdb, err = NewSchedulerSQLDB(url, logger)
			Expect(err).NotTo(HaveOccurred())
			err = cleanSchedulerActiveScheduleTable()
			Expect(err).NotTo(HaveOccurred())

			err = insertSchedulerActiveSchedule(111111, "app-id-1", 1, 2, 10, 5)
			Expect(err).NotTo(HaveOccurred())
			err = insertSchedulerActiveSchedule(222222, "app-id-2", 2, 3, 7, 5)
			Expect(err).NotTo(HaveOccurred())
			err = insertSchedulerActiveSchedule(333333, "app-id-3", 3, 5, 12, -1)
			Expect(err).NotTo(HaveOccurred())
		})

		AfterEach(func() {
			err = sdb.Close()
			Expect(err).NotTo(HaveOccurred())
		})

		JustBeforeEach(func() {
			err = sdb.SynchronizeActiveSchedules(appIdMap)
			schedules, err = sdb.GetActiveSchedules()
		})
		Context("when appIdMap is empty", func() {
			BeforeEach(func() {
				appIdMap = map[string]struct{}{}
			})
			It("no active schedule is removed", func() {
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
		Context("when appIdMap is not empty", func() {
			BeforeEach(func() {
				appIdMap = map[string]struct{}{
					"app-id-2": struct{}{},
				}
			})
			It("active schedules of app which appId is not in appIdMap are removed", func() {
				Expect(schedules["app-id-1"]).To(BeNil())
				Expect(schedules).To(HaveKeyWithValue("app-id-2", &models.ActiveSchedule{
					ScheduleId:         "222222",
					InstanceMin:        3,
					InstanceMax:        7,
					InstanceMinInitial: 5,
				}))
				Expect(schedules["app-id-3"]).To(BeNil())
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
