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

var _ = Describe("ScheduleSqldb", func() {
	var (
		sdb            *ScheduleSQLDB
		url            string
		logger         lager.Logger
		err            error
		activeSchedule *models.ActiveSchedule
	)

	BeforeEach(func() {
		logger = lager.NewLogger("schedule-sqldb-test")
		url = os.Getenv("DBURL")
	})

	Describe("NewScheduleSQLDB", func() {
		JustBeforeEach(func() {
			sdb, err = NewScheduleSQLDB(url, logger)
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

		Context("when db url is correct", func() {
			It("should not error", func() {
				Expect(err).NotTo(HaveOccurred())
				Expect(sdb).NotTo(BeNil())
			})
		})
	})

	Describe("GetActiveSchedule", func() {
		BeforeEach(func() {
			sdb, err = NewScheduleSQLDB(url, logger)
			Expect(err).NotTo(HaveOccurred())
			err = cleanActiveScheduleTable()
			Expect(err).NotTo(HaveOccurred())
		})

		AfterEach(func() {
			err = sdb.Close()
			Expect(err).NotTo(HaveOccurred())
		})

		JustBeforeEach(func() {
			activeSchedule, err = sdb.GetActiveSchedule("an-app-id")
		})

		Context("when there is no active schedule for the given app", func() {
			It("should not error and return nil", func() {
				Expect(err).NotTo(HaveOccurred())
				Expect(activeSchedule).To(BeNil())
			})
		})

		Context("when there is active schedule with an InstanceMinInitial", func() {
			BeforeEach(func() {
				err = insertActiveSchedule("an-app-id", 2, 10, 5, "starting")
				Expect(err).NotTo(HaveOccurred())
			})
			It("return the active schedule", func() {
				Expect(err).NotTo(HaveOccurred())
				Expect(activeSchedule.InstanceMin).To(Equal(2))
				Expect(activeSchedule.InstanceMax).To(Equal(10))
				Expect(activeSchedule.InstanceMinInitial).To(Equal(5))
			})
		})

		Context("when there is active schedule with no InstanceMinInitial", func() {
			BeforeEach(func() {
				err = insertActiveSchedule("an-app-id", 2, 10, -1, "starting")
				Expect(err).NotTo(HaveOccurred())
			})
			It("return the active schedule with InstanceMinInitial set to be zero ", func() {
				Expect(activeSchedule.InstanceMin).To(Equal(2))
				Expect(activeSchedule.InstanceMax).To(Equal(10))
				Expect(activeSchedule.InstanceMinInitial).To(Equal(0))
			})
		})

		Context("when there is multiple active schedules for the given app", func() {
			BeforeEach(func() {
				err = insertActiveSchedule("an-app-id", 2, 10, -1, "starting")
				Expect(err).NotTo(HaveOccurred())
				err = insertActiveSchedule("an-app-id", 3, 9, 5, "starting")
				Expect(err).NotTo(HaveOccurred())

			})
			It("return the latest active schedule", func() {
				Expect(err).NotTo(HaveOccurred())
				Expect(activeSchedule.InstanceMin).To(Equal(3))
				Expect(activeSchedule.InstanceMax).To(Equal(9))
				Expect(activeSchedule.InstanceMinInitial).To(Equal(5))
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
