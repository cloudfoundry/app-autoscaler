package sqldb_test

import (
	"context"
	"os"
	"strconv"
	"strings"
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

var _ = Describe("ScalingEngineSqldb", func() {
	var (
		dbConfig          db.DatabaseConfig
		dbHost            = os.Getenv("DB_HOST")
		logger            lager.Logger
		sdb               *ScalingEngineSQLDB
		err               error
		history           *models.AppScalingHistory
		start             int64
		end               int64
		orderType         db.OrderType
		appId             string
		appId2            string
		appId3            string
		scheduleId        string
		scheduleId2       string
		scheduleId3       string
		histories         []*models.AppScalingHistory
		canScale          bool
		cooldownExpiredAt int64
		activeSchedule    *models.ActiveSchedule
		schedules         map[string]string
		before            int64
		includeAll        bool
	)

	dbUrl := GetDbUrl()
	BeforeEach(func() {
		logger = lager.NewLogger("history-sqldb-test")
		logger.RegisterSink(lager.NewWriterSink(GinkgoWriter, lager.DEBUG))
		dbConfig = db.DatabaseConfig{
			URL:                   dbUrl,
			MaxOpenConnections:    10,
			MaxIdleConnections:    5,
			ConnectionMaxLifetime: 10 * time.Second,
			ConnectionMaxIdleTime: 10 * time.Second,
		}
		sdb, err = NewScalingEngineSQLDB(dbConfig, logger)
		FailOnError("Could not open db connection", err)
		DeferCleanup(func() error {
			if sdb != nil {
				return sdb.Close()
			}
			return nil
		})
		appId = addProcessIdTo("an-app-id")
		appId2 = addProcessIdTo("second-app-id")
		appId3 = addProcessIdTo("third-app-id")
		scheduleId = addProcessIdTo("schedule-id-1")
		scheduleId2 = addProcessIdTo("schedule-id-2")
		scheduleId3 = addProcessIdTo("schedule-id-3")
		cleanupForApp(appId)
		cleanupForApp(appId2)
		cleanupForApp(appId3)
		DeferCleanup(func() {
			cleanupForApp(appId)
			cleanupForApp(appId2)
			cleanupForApp(appId3)
		})
	})

	Describe("NewHistorySQLDB", func() {
		JustBeforeEach(func() {
			_ = sdb.Close()
			sdb, err = NewScalingEngineSQLDB(dbConfig, logger)
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
				dbConfig.URL = "postgres://not-exist-user:not-exist-password@localhost/autoscaler?sslmode=disable"
			})
			It("should throw an error", func() {
				Expect(err).To(HaveOccurred())
			})
		})

		Context("when mysql db url is not correct", func() {
			BeforeEach(func() {
				if strings.Contains(dbUrl, "postgres") {
					Skip("Not configured for mysql")
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

	Describe("SaveScalingHistory", func() {
		Context("When inserting a scaling history record of an app", func() {
			BeforeEach(func() {
				history = &models.AppScalingHistory{
					AppId:        addProcessIdTo("an-app-id"),
					Timestamp:    111111,
					ScalingType:  models.ScalingTypeDynamic,
					Status:       models.ScalingStatusSucceeded,
					OldInstances: 2,
					NewInstances: 4,
					Reason:       "a reason",
					Message:      "a message",
				}
				err = sdb.SaveScalingHistory(history)
			})

			It("has the scaling history record in database", func() {
				Expect(err).NotTo(HaveOccurred())
				Expect(hasScalingHistory(addProcessIdTo("an-app-id"), 111111)).To(BeTrue())
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
				history.AppId = appId
				history.Timestamp = 111111
				err = sdb.SaveScalingHistory(history)
				Expect(err).NotTo(HaveOccurred())

				history.AppId = appId
				history.Timestamp = 222222
				err = sdb.SaveScalingHistory(history)
				Expect(err).NotTo(HaveOccurred())

				history.AppId = appId2
				history.Timestamp = 333333
				err = sdb.SaveScalingHistory(history)
				Expect(err).NotTo(HaveOccurred())

			})

			It("has all the histories in database", func() {

				Expect(hasScalingHistory(appId, 111111)).To(BeTrue())
				Expect(hasScalingHistory(appId, 222222)).To(BeTrue())
				Expect(hasScalingHistory(appId2, 333333)).To(BeTrue())

			})
		})

	})

	Describe("RetrieveScalingHistories", func() {
		BeforeEach(func() {
			start = 0
			end = -1
			orderType = db.DESC
			includeAll = true

			history = &models.AppScalingHistory{
				AppId:        appId,
				OldInstances: 2,
				NewInstances: 4,
				Reason:       "a reason",
				Message:      "a message",
			}
			history.Timestamp = 666666
			history.ScalingType = models.ScalingTypeDynamic
			history.Status = models.ScalingStatusSucceeded
			history.Error = ""
			err = sdb.SaveScalingHistory(history)
			FailOnError("Failed to add scaling history", err)

			history.Timestamp = 222222
			history.ScalingType = models.ScalingTypeDynamic
			history.Status = models.ScalingStatusFailed
			history.Error = "an error"
			err = sdb.SaveScalingHistory(history)
			FailOnError("Failed to add scaling history", err)

			history.Timestamp = 555555
			history.ScalingType = models.ScalingTypeSchedule
			history.Status = models.ScalingStatusFailed
			history.Error = "an error"
			err = sdb.SaveScalingHistory(history)
			FailOnError("Failed to add scaling history", err)

			history.Timestamp = 333333
			history.ScalingType = models.ScalingTypeSchedule
			history.Status = models.ScalingStatusIgnored
			history.Error = ""
			err = sdb.SaveScalingHistory(history)
			FailOnError("Failed to add scaling history", err)

		})

		JustBeforeEach(func() {
			histories, err = sdb.RetrieveScalingHistories(context.TODO(), appId, start, end, orderType, includeAll, 1, 50)
		})

		Context("When the app has no history", func() {
			It("returns empty metrics", func() {
				histories, err = sdb.RetrieveScalingHistories(context.TODO(), "app-id-no-history", start, end, orderType, includeAll, 1, 50)
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

		Context("when retrieving all the histories( start = 0, end = -1, orderType = ASC) ", func() {
			BeforeEach(func() {
				orderType = db.ASC
			})
			It("returns all the histories of the app ordered by timestamp asc", func() {
				Expect(err).NotTo(HaveOccurred())
				Expect(histories).To(Equal([]*models.AppScalingHistory{
					{
						AppId:        appId,
						Timestamp:    222222,
						ScalingType:  models.ScalingTypeDynamic,
						Status:       models.ScalingStatusFailed,
						OldInstances: 2,
						NewInstances: 4,
						Reason:       "a reason",
						Message:      "a message",
						Error:        "an error",
					},
					{
						AppId:        appId,
						Timestamp:    333333,
						ScalingType:  models.ScalingTypeSchedule,
						Status:       models.ScalingStatusIgnored,
						OldInstances: 2,
						NewInstances: 4,
						Reason:       "a reason",
						Message:      "a message",
					},
					{
						AppId:        appId,
						Timestamp:    555555,
						ScalingType:  models.ScalingTypeSchedule,
						Status:       models.ScalingStatusFailed,
						OldInstances: 2,
						NewInstances: 4,
						Reason:       "a reason",
						Message:      "a message",
						Error:        "an error",
					},
					{
						AppId:        appId,
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

		Context("when retrieving all the histories( start = 0, end = -1, orderType = DESC) ", func() {
			BeforeEach(func() {
				orderType = db.DESC
			})
			It("returns all the histories of the app ordered by timestamp desc", func() {
				Expect(err).NotTo(HaveOccurred())
				Expect(histories).To(Equal([]*models.AppScalingHistory{
					{
						AppId:        appId,
						Timestamp:    666666,
						ScalingType:  models.ScalingTypeDynamic,
						Status:       models.ScalingStatusSucceeded,
						OldInstances: 2,
						NewInstances: 4,
						Reason:       "a reason",
						Message:      "a message",
					},
					{
						AppId:        appId,
						Timestamp:    555555,
						ScalingType:  models.ScalingTypeSchedule,
						Status:       models.ScalingStatusFailed,
						OldInstances: 2,
						NewInstances: 4,
						Reason:       "a reason",
						Message:      "a message",
						Error:        "an error",
					},
					{
						AppId:        appId,
						Timestamp:    333333,
						ScalingType:  models.ScalingTypeSchedule,
						Status:       models.ScalingStatusIgnored,
						OldInstances: 2,
						NewInstances: 4,
						Reason:       "a reason",
						Message:      "a message",
					},
					{
						AppId:        appId,
						Timestamp:    222222,
						ScalingType:  models.ScalingTypeDynamic,
						Status:       models.ScalingStatusFailed,
						OldInstances: 2,
						NewInstances: 4,
						Reason:       "a reason",
						Message:      "a message",
						Error:        "an error",
					},
				}))
			})
		})

		Context("When retrieving part of the histories", func() {
			BeforeEach(func() {
				start = 333333
				end = 555566
				orderType = db.DESC

			})

			It("return correct histories", func() {
				Expect(err).NotTo(HaveOccurred())
				Expect(histories).To(Equal([]*models.AppScalingHistory{
					{
						AppId:        appId,
						Timestamp:    555555,
						ScalingType:  models.ScalingTypeSchedule,
						Status:       models.ScalingStatusFailed,
						OldInstances: 2,
						NewInstances: 4,
						Reason:       "a reason",
						Message:      "a message",
						Error:        "an error",
					}, {
						AppId:        appId,
						Timestamp:    333333,
						ScalingType:  models.ScalingTypeSchedule,
						Status:       models.ScalingStatusIgnored,
						OldInstances: 2,
						NewInstances: 4,
						Reason:       "a reason",
						Message:      "a message",
					}}))

			})

		})

		Context("when only retrieving succeeded and failed history", func() {
			BeforeEach(func() {
				includeAll = false
			})

			It("skips ignored scaling history", func() {
				Expect(err).NotTo(HaveOccurred())
				Expect(histories).To(Equal([]*models.AppScalingHistory{
					{
						AppId:        appId,
						Timestamp:    666666,
						ScalingType:  models.ScalingTypeDynamic,
						Status:       models.ScalingStatusSucceeded,
						OldInstances: 2,
						NewInstances: 4,
						Reason:       "a reason",
						Message:      "a message",
					},
					{
						AppId:        appId,
						Timestamp:    555555,
						ScalingType:  models.ScalingTypeSchedule,
						Status:       models.ScalingStatusFailed,
						OldInstances: 2,
						NewInstances: 4,
						Reason:       "a reason",
						Message:      "a message",
						Error:        "an error",
					},
					{
						AppId:        appId,
						Timestamp:    222222,
						ScalingType:  models.ScalingTypeDynamic,
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

	Describe("PruneScalingHistories", Serial, func() {
		BeforeEach(func() {
			history = &models.AppScalingHistory{AppId: appId}
			history.Timestamp = 666666
			err = sdb.SaveScalingHistory(history)
			Expect(err).NotTo(HaveOccurred())

			history.Timestamp = 222222
			err = sdb.SaveScalingHistory(history)
			Expect(err).NotTo(HaveOccurred())

			history.Timestamp = 555555
			err = sdb.SaveScalingHistory(history)
			Expect(err).NotTo(HaveOccurred())

			history.Timestamp = 333333
			err = sdb.SaveScalingHistory(history)
			Expect(err).NotTo(HaveOccurred())
		})

		JustBeforeEach(func() {
			err = sdb.PruneScalingHistories(context.TODO(), before)
		})

		Context("when pruning histories before all the timestamps", func() {
			BeforeEach(func() {
				before = 111111
			})

			It("does not remove any histories", func() {
				Expect(err).NotTo(HaveOccurred())
				Expect(getScalingHistoryForApp(appId)).To(Equal(4))
			})
		})

		Context("when pruning all the histories", func() {
			BeforeEach(func() {
				before = time.Now().UnixNano()
			})

			It("empties the scalinghistory table", func() {
				Expect(err).NotTo(HaveOccurred())
				Expect(getScalingHistoryForApp(appId)).To(Equal(0))
			})
		})

		Context("when pruning part of the histories", func() {
			BeforeEach(func() {
				before = 333333
			})

			It("removes histories before the time specified", func() {
				Expect(err).NotTo(HaveOccurred())
				Expect(getScalingHistoryForApp(appId)).To(Equal(2))
			})
		})

		Context("when db fails", func() {
			BeforeEach(func() {
				_ = sdb.Close()
			})
			It("should error", func() {
				Expect(err).To(MatchError(MatchRegexp("sql: .*")))
			})
		})
	})

	Describe("PruneCooldowns", Serial, func() {
		var appIds []string

		BeforeEach(func() {

			appIds = make([]string, 10)
			for i := 0; i < 10; i++ {
				appIds[i] = addProcessIdTo("an-app-id-" + strconv.Itoa(i))
				err := sdb.UpdateScalingCooldownExpireTime(appIds[i], 111111*int64(i+1))
				Expect(err).NotTo(HaveOccurred())
			}

		})

		JustBeforeEach(func() {
			err = sdb.PruneCooldowns(context.TODO(), before)
		})

		Context("when pruning cooldowns before all the timestamps", func() {
			BeforeEach(func() {
				before = 111111
			})

			It("does not remove any cooldowns", func() {
				Expect(err).NotTo(HaveOccurred())
				Expect(getNumberOfCooldownEntries()).To(Equal(10))
			})
		})

		Context("when pruning all the cooldowns", func() {
			BeforeEach(func() {
				before = time.Now().UnixNano()
			})

			It("empties the scalingcooldowns table", func() {
				Expect(err).NotTo(HaveOccurred())
				Expect(getNumberOfCooldownEntries()).To(Equal(0))
			})
		})

		Context("when pruning part of the cooldowns", func() {
			BeforeEach(func() {
				before = 333333
			})

			It("removes cooldowns before the time specified", func() {
				Expect(err).NotTo(HaveOccurred())
				Expect(getNumberOfCooldownEntries()).To(Equal(8))
			})
		})

		Context("when db fails", func() {
			BeforeEach(func() {
				_ = sdb.Close()
			})
			It("should error", func() {
				Expect(err).To(MatchError(MatchRegexp("sql: .*")))
			})
		})
	})

	Describe("UpdateScalingCooldownExpireTime", func() {

		JustBeforeEach(func() {
			err = sdb.UpdateScalingCooldownExpireTime(appId, 222222)
		})

		Context("when there is no previous app cooldown record", func() {
			It("creates the record", func() {
				Expect(err).NotTo(HaveOccurred())
				Expect(hasScalingCooldownRecord(appId, 222222)).To(BeTrue())
			})
		})

		Context("when there is previous app cooldown record", func() {
			BeforeEach(func() {
				err = sdb.UpdateScalingCooldownExpireTime(appId, 111111)
				Expect(err).NotTo(HaveOccurred())
			})

			It("removes the previous record and inserts a new record", func() {
				Expect(err).NotTo(HaveOccurred())
				Expect(hasScalingCooldownRecord(appId, 111111)).To(BeFalse())
				Expect(hasScalingCooldownRecord(appId, 222222)).To(BeTrue())
			})
		})
	})

	Describe("CanScaleApp", func() {
		JustBeforeEach(func() {
			canScale, cooldownExpiredAt, err = sdb.CanScaleApp(appId)
		})

		Context("when there is no cooldown record before", func() {
			It("returns true", func() {
				Expect(err).NotTo(HaveOccurred())
				Expect(canScale).To(BeTrue())
				Expect(cooldownExpiredAt).To(Equal(int64(0)), "Expected there to be no entries for: "+appId)
			})
		})

		Context("when the app is still in cooldown period", func() {
			fakeCoolDownExpiredTime := time.Now().Add(100 * time.Second).UnixNano()
			BeforeEach(func() {
				err = sdb.UpdateScalingCooldownExpireTime(appId, fakeCoolDownExpiredTime)
				Expect(err).NotTo(HaveOccurred())
			})
			It("returns false", func() {
				Expect(err).NotTo(HaveOccurred())
				Expect(canScale).To(BeFalse())
				Expect(cooldownExpiredAt).To(Equal(fakeCoolDownExpiredTime))
			})
		})

		Context("when the app passes cooldown period", func() {
			fakeCoolDownExpiredTime := time.Now().Add(0 - 100*time.Second).UnixNano()
			BeforeEach(func() {
				err = sdb.UpdateScalingCooldownExpireTime(appId, fakeCoolDownExpiredTime)
				Expect(err).NotTo(HaveOccurred())
			})
			It("returns true", func() {
				Expect(err).NotTo(HaveOccurred())
				Expect(canScale).To(BeTrue())
				Expect(cooldownExpiredAt).To(Equal(fakeCoolDownExpiredTime))
			})
		})
	})

	Describe("GetActiveSchedule", func() {
		JustBeforeEach(func() {
			activeSchedule, err = sdb.GetActiveSchedule(appId)
		})

		Context("when there is no active schedule for the given app", func() {
			It("should not error and return nil", func() {
				Expect(err).NotTo(HaveOccurred())
				Expect(activeSchedule).To(BeNil())
			})
		})

		Context("when there is active schedule ", func() {
			BeforeEach(func() {
				err = insertActiveSchedule(appId, "an-schedule-id", 2, 10, 5)
				Expect(err).NotTo(HaveOccurred())
			})
			It("return the active schedule", func() {
				Expect(err).NotTo(HaveOccurred())
				Expect(activeSchedule).To(Equal(&models.ActiveSchedule{
					ScheduleId:         "an-schedule-id",
					InstanceMin:        2,
					InstanceMax:        10,
					InstanceMinInitial: 5,
				}))
			})
		})
		Context("when there is database error", func() {
			BeforeEach(func() {
				_ = sdb.Close()
			})
			It("should error", func() {
				Expect(err).To(HaveOccurred())
			})
		})
	})

	Describe("GetActiveSchedules", Serial, func() {

		JustBeforeEach(func() {
			schedules, err = sdb.GetActiveSchedules()
		})

		Context("when the table is empty", func() {
			It("returns an empty active schedules", func() {
				Expect(err).NotTo(HaveOccurred())
				Expect(schedules).To(BeEmpty())
			})
		})

		Context("when the table is not empty", func() {
			BeforeEach(func() {
				err = insertActiveSchedule(appId, scheduleId, 2, 10, 5)
				Expect(err).NotTo(HaveOccurred())
				err = insertActiveSchedule(appId2, scheduleId2, 5, 9, -1)
				Expect(err).NotTo(HaveOccurred())
				err = insertActiveSchedule(appId3, scheduleId3, 3, 9, 6)
				Expect(err).NotTo(HaveOccurred())
			})
			It("return all active schedules", func() {
				Expect(err).NotTo(HaveOccurred())
				Expect(schedules).To(HaveLen(3))
				Expect(schedules).To(HaveKeyWithValue(appId, scheduleId))
				Expect(schedules).To(HaveKeyWithValue(appId2, scheduleId2))
				Expect(schedules).To(HaveKeyWithValue(appId3, scheduleId3))
			})

		})
		Context("when there is database error", func() {
			BeforeEach(func() {
				_ = sdb.Close()
			})
			It("should error", func() {
				Expect(err).To(HaveOccurred())
			})
		})
	})

	Describe("RemoveActiveSchedule", func() {

		JustBeforeEach(func() {
			err = sdb.RemoveActiveSchedule(appId)
		})

		Context("when there is no active schedule in table", func() {
			It("should not error", func() {
				Expect(err).NotTo(HaveOccurred())
			})
		})

		Context("when there is active schedule in table", func() {
			BeforeEach(func() {
				err = insertActiveSchedule(appId, "existing-schedule-id", 3, 6, 0)
			})

			It("should remove the active schedule from table", func() {
				Expect(err).NotTo(HaveOccurred())
				schedule, err := sdb.GetActiveSchedule(appId)
				Expect(err).NotTo(HaveOccurred())
				Expect(schedule).To(BeNil())
			})
		})

		Context("when there is database error", func() {
			BeforeEach(func() {
				_ = sdb.Close()
			})
			It("should error", func() {
				Expect(err).To(HaveOccurred())
			})
		})
	})

	Describe("SetActiveSchedule", func() {
		BeforeEach(func() {
			activeSchedule = &models.ActiveSchedule{
				ScheduleId:         "a-schedule-id",
				InstanceMin:        2,
				InstanceMax:        8,
				InstanceMinInitial: 5,
			}
		})

		JustBeforeEach(func() {
			err = sdb.SetActiveSchedule(appId, activeSchedule)
		})

		Context("when there is no active schedule in table", func() {
			It("should insert the active schedule", func() {
				Expect(err).NotTo(HaveOccurred())
				schedule, err := sdb.GetActiveSchedule(appId)
				Expect(err).NotTo(HaveOccurred())
				Expect(schedule).To(Equal(activeSchedule))
			})
		})

		Context("when there is existing active schedule in table", func() {
			BeforeEach(func() {
				err = insertActiveSchedule(appId, "existing-schedule-id", 3, 6, 0)
			})

			It("should remove the existing one and insert the new active schedule", func() {
				Expect(err).NotTo(HaveOccurred())
				schedule, err := sdb.GetActiveSchedule(appId)
				Expect(err).NotTo(HaveOccurred())
				Expect(schedule).To(Equal(activeSchedule))
			})
		})

		Context("when there is database error", func() {
			BeforeEach(func() {
				_ = sdb.Close()
			})

			It("should error", func() {
				Expect(err).To(HaveOccurred())
			})
		})
	})
})

func cleanupForApp(appId string) {
	removeScalingHistoryForApp(appId)
	removeCooldownForApp(appId)
	removeActiveScheduleForApp(appId)
}
