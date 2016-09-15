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

var _ = Describe("HistorySqldb", func() {
	var (
		url    string
		logger lager.Logger
		hdb    *HistorySQLDB
		err    error

		history   *models.AppScalingHistory
		start     int64
		end       int64
		appId     string
		histories []*models.AppScalingHistory
	)

	BeforeEach(func() {
		logger = lager.NewLogger("history-sqldb-test")
		url = os.Getenv("DBURL")
	})

	Describe("NewHistorySQLDB", func() {
		JustBeforeEach(func() {
			hdb, err = NewHistorySQLDB(url, logger)
		})

		AfterEach(func() {
			if hdb != nil {
				err = hdb.Close()
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
				Expect(hdb).NotTo(BeNil())
			})
		})
	})

	Describe("SaveScalingHistory", func() {
		BeforeEach(func() {
			hdb, err = NewHistorySQLDB(url, logger)
			Expect(err).NotTo(HaveOccurred())
			cleanScalingHistoryTable()
		})

		AfterEach(func() {
			err = hdb.Close()
			Expect(err).NotTo(HaveOccurred())
		})

		Context("When inserting a scaling history record of an app", func() {
			BeforeEach(func() {
				history = &models.AppScalingHistory{
					AppId:        "an-app-id",
					Timestamp:    111111,
					ScalingType:  models.ScalingTypeDynamic,
					Status:       models.ScalingStatusSucceeded,
					OldInstances: 2,
					NewInstances: 4,
					Reason:       "a reason",
					Message:      "a message",
				}
				err = hdb.SaveScalingHistory(history)
			})

			It("has the scaling history record in database", func() {
				Expect(err).NotTo(HaveOccurred())
				Expect(hasScalingHistory("an-app-id", 111111)).To(BeTrue())
			})
		})

		Context("When inserting multiple scaling history records of an app", func() {
			BeforeEach(func() {
				history = &models.AppScalingHistory{
					ScalingType:  models.ScalingTypeSchedule,
					Status:       models.ScalingStatusFailed,
					OldInstances: 2,
					NewInstances: 4,
					Reason:       "a reason",
					Message:      "a message",
					Error:        "an error",
				}
				history.AppId = "an-app-id"
				history.Timestamp = 111111
				err = hdb.SaveScalingHistory(history)
				Expect(err).NotTo(HaveOccurred())

				history.AppId = "an-app-id"
				history.Timestamp = 222222
				err = hdb.SaveScalingHistory(history)
				Expect(err).NotTo(HaveOccurred())

				history.AppId = "another-app-id"
				history.Timestamp = 333333
				err = hdb.SaveScalingHistory(history)
				Expect(err).NotTo(HaveOccurred())

			})

			It("has all the histories in database", func() {

				Expect(hasScalingHistory("an-app-id", 111111)).To(BeTrue())
				Expect(hasScalingHistory("an-app-id", 222222)).To(BeTrue())
				Expect(hasScalingHistory("another-app-id", 333333)).To(BeTrue())

			})
		})

	})

	Describe("RetrieveScalingHistories", func() {

		BeforeEach(func() {
			hdb, err = NewHistorySQLDB(url, logger)
			Expect(err).NotTo(HaveOccurred())
			cleanScalingHistoryTable()

			start = 0
			end = -1
			appId = "an-app-id"
		})

		AfterEach(func() {
			err = hdb.Close()
			Expect(err).NotTo(HaveOccurred())
		})

		JustBeforeEach(func() {
			history = &models.AppScalingHistory{
				AppId:        "an-app-id",
				OldInstances: 2,
				NewInstances: 4,
				Reason:       "a reason",
				Message:      "a message",
			}

			history.Timestamp = 666666
			history.ScalingType = models.ScalingTypeDynamic
			history.Status = models.ScalingStatusSucceeded
			history.Error = ""
			err = hdb.SaveScalingHistory(history)
			Expect(err).NotTo(HaveOccurred())

			history.Timestamp = 222222
			history.ScalingType = models.ScalingTypeDynamic
			history.Status = models.ScalingStatusFailed
			history.Error = "an error"
			err = hdb.SaveScalingHistory(history)
			Expect(err).NotTo(HaveOccurred())

			history.Timestamp = 555555
			history.ScalingType = models.ScalingTypeSchedule
			history.Status = models.ScalingStatusFailed
			history.Error = "an error"
			err = hdb.SaveScalingHistory(history)
			Expect(err).NotTo(HaveOccurred())

			history.Timestamp = 333333
			history.ScalingType = models.ScalingTypeSchedule
			history.Status = models.ScalingStatusIgnored
			history.Error = ""
			err = hdb.SaveScalingHistory(history)
			Expect(err).NotTo(HaveOccurred())

			histories, err = hdb.RetrieveScalingHistories(appId, start, end)
		})

		Context("When the app has no hisotry", func() {
			BeforeEach(func() {
				appId = "app-id-no-history"
			})

			It("returns empty metrics", func() {
				Expect(err).NotTo(HaveOccurred())
				Expect(histories).To(BeEmpty())
			})

		})

		Context("when end time is now (end = -1)", func() {
			BeforeEach(func() {
				start = 333333
				end = -1
			})

			It("returns histories from start time to now", func() {
				Expect(err).NotTo(HaveOccurred())
				Expect(histories).To(HaveLen(3))
			})

		})

		Context("when end time is before all the history timestamps", func() {
			BeforeEach(func() {
				start = 111111
				end = 222221
			})

			It("returns empty histories", func() {
				Expect(err).NotTo(HaveOccurred())
				Expect(histories).To(BeEmpty())
			})

		})

		Context("when start time is after all the history timestamps", func() {
			BeforeEach(func() {
				start = 777777
				end = 888888
			})

			It("returns empty histories", func() {
				Expect(err).NotTo(HaveOccurred())
				Expect(histories).To(BeEmpty())
			})

		})

		Context("when start time > end time", func() {
			BeforeEach(func() {
				start = 555555
				end = 555533
			})

			It("returns empty histories", func() {
				Expect(err).NotTo(HaveOccurred())
				Expect(histories).To(BeEmpty())
			})
		})

		Context("when retrieving all the histories( start = 0, end = -1) ", func() {
			It("returns all the histories of the app ordered by timestamp", func() {
				Expect(err).NotTo(HaveOccurred())
				Expect(histories).To(Equal([]*models.AppScalingHistory{
					&models.AppScalingHistory{
						AppId:        "an-app-id",
						Timestamp:    222222,
						ScalingType:  models.ScalingTypeDynamic,
						Status:       models.ScalingStatusFailed,
						OldInstances: 2,
						NewInstances: 4,
						Reason:       "a reason",
						Message:      "a message",
						Error:        "an error",
					},
					&models.AppScalingHistory{
						AppId:        "an-app-id",
						Timestamp:    333333,
						ScalingType:  models.ScalingTypeSchedule,
						Status:       models.ScalingStatusIgnored,
						OldInstances: 2,
						NewInstances: 4,
						Reason:       "a reason",
						Message:      "a message",
					},
					&models.AppScalingHistory{
						AppId:        "an-app-id",
						Timestamp:    555555,
						ScalingType:  models.ScalingTypeSchedule,
						Status:       models.ScalingStatusFailed,
						OldInstances: 2,
						NewInstances: 4,
						Reason:       "a reason",
						Message:      "a message",
						Error:        "an error",
					},
					&models.AppScalingHistory{
						AppId:        "an-app-id",
						Timestamp:    666666,
						ScalingType:  models.ScalingTypeDynamic,
						Status:       models.ScalingStatusSucceeded,
						OldInstances: 2,
						NewInstances: 4,
						Reason:       "a reason",
						Message:      "a message",
					}}))
			})
		})

		Context("When retrieving part of the histories", func() {
			BeforeEach(func() {
				start = 333333
				end = 555566
			})

			It("return correct histories", func() {
				Expect(err).NotTo(HaveOccurred())
				Expect(histories).To(Equal([]*models.AppScalingHistory{
					&models.AppScalingHistory{
						AppId:        "an-app-id",
						Timestamp:    333333,
						ScalingType:  models.ScalingTypeSchedule,
						Status:       models.ScalingStatusIgnored,
						OldInstances: 2,
						NewInstances: 4,
						Reason:       "a reason",
						Message:      "a message",
					},
					&models.AppScalingHistory{
						AppId:        "an-app-id",
						Timestamp:    555555,
						ScalingType:  models.ScalingTypeSchedule,
						Status:       models.ScalingStatusFailed,
						OldInstances: 2,
						NewInstances: 4,
						Reason:       "a reason",
						Message:      "a message",
						Error:        "an error",
					}}))

			})

		})
	})

})
