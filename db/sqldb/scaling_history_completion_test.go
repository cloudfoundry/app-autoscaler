package sqldb_test

import (
	"context"
	"database/sql"
	"fmt"
	"math"
	"strings"
	"sync"
	"time"

	"code.cloudfoundry.org/app-autoscaler/src/autoscaler/db"
	. "code.cloudfoundry.org/app-autoscaler/src/autoscaler/db/sqldb"
	"code.cloudfoundry.org/app-autoscaler/src/autoscaler/models"
	. "code.cloudfoundry.org/app-autoscaler/src/autoscaler/testhelpers"
	"code.cloudfoundry.org/lager/v3"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

// These tests share the migrated database with the existing suite. Serial also
// isolates the pruning and deliberately failing database-constraint tests.
var _ = Describe("Vertical scaling history persistence", Serial, func() {
	var (
		ctx       context.Context
		sdb       *ScalingEngineSQLDB
		config    db.DatabaseConfig
		logger    lager.Logger
		history   *models.AppScalingHistory
		appID     string
		secondApp string
	)

	BeforeEach(func() {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(context.Background(), 20*time.Second)
		DeferCleanup(cancel)
		logger = lager.NewLogger("vertical-history-sqldb-test")
		logger.RegisterSink(lager.NewWriterSink(GinkgoWriter, lager.DEBUG))
		config = db.DatabaseConfig{URL: GetDbUrl(), MaxOpenConnections: 10, MaxIdleConnections: 5}
		var err error
		sdb, err = NewScalingEngineSQLDB(config, logger)
		Expect(err).NotTo(HaveOccurred())
		DeferCleanup(func() error { return sdb.Close() })
		appID = addProcessIdTo("vertical-history-app")
		secondApp = addProcessIdTo("vertical-history-second-app")
		cleanupForApp(appID)
		cleanupForApp(secondApp)
		DeferCleanup(func() {
			cleanupForApp(appID)
			cleanupForApp(secondApp)
		})
		history = &models.AppScalingHistory{
			AppId: appID, Timestamp: 100, ScalingType: models.ScalingTypeDynamic,
			Status: models.ScalingStatusPending, OldInstances: 2, NewInstances: 2,
			Reason: "entitlementutil >= 80", Message: "deployment requested",
			ScalingDimension: models.ScalingDimensionVertical,
			CorrelationID:    fmt.Sprintf("vertical-correlation-%d", GinkgoParallelProcess()),
			OldMemoryMB:      1024, NewMemoryMB: 2048, CoolDownSeconds: 120,
		}
	})

	retrieve := func() *models.AppScalingHistory {
		histories, err := sdb.RetrieveScalingHistories(ctx, appID, 0, -1, db.ASC, true, 1, 50)
		ExpectWithOffset(1, err).NotTo(HaveOccurred())
		ExpectWithOffset(1, histories).To(HaveLen(1))
		return histories[0]
	}

	It("round-trips a pending request with the effective original cooldown", func() {
		Expect(sdb.SaveScalingHistory(history)).To(Succeed())
		Expect(retrieve()).To(Equal(history))
		open, err := sdb.GetOpenScalingHistories(ctx)
		Expect(err).NotTo(HaveOccurred())
		Expect(open).To(ContainElement(Equal(history)))
		Expect(sdb.HasOpenScalingHistory(ctx, appID)).To(BeTrue())
		Expect(sdb.HasOpenScalingHistory(ctx, secondApp)).To(BeFalse())
		canScale, expires, err := sdb.CanScaleApp(appID)
		Expect(err).NotTo(HaveOccurred())
		Expect(canScale).To(BeTrue())
		Expect(expires).To(BeZero(), "accepting a request must not start cooldown")
	})

	It("reads nullable fields from pre-migration horizontal histories", func() {
		_, err := dbHelper.ExecContext(ctx, dbHelper.Rebind("INSERT INTO scalinghistory(appid, timestamp, scalingtype, status, oldinstances, newinstances, reason, message, error) VALUES(?, ?, ?, ?, ?, ?, ?, NULL, NULL)"),
			appID, 100, models.ScalingTypeDynamic, models.ScalingStatusSucceeded, 2, 4, "legacy reason")
		Expect(err).NotTo(HaveOccurred())
		Expect(retrieve()).To(Equal(&models.AppScalingHistory{
			AppId: appID, Timestamp: 100, ScalingType: models.ScalingTypeDynamic,
			Status: models.ScalingStatusSucceeded, OldInstances: 2, NewInstances: 4, Reason: "legacy reason",
		}))
		Expect(sdb.HasOpenScalingHistory(ctx, appID)).To(BeFalse())
	})

	It("keeps horizontal records and ignored filtering unchanged alongside an open request", func() {
		Expect(sdb.SaveScalingHistory(history)).To(Succeed())
		for index, status := range []models.ScalingStatus{models.ScalingStatusSucceeded, models.ScalingStatusFailed, models.ScalingStatusIgnored} {
			Expect(sdb.SaveScalingHistory(&models.AppScalingHistory{
				AppId: appID, Timestamp: int64(101 + index), Status: status,
				ScalingType: models.ScalingTypeSchedule, OldInstances: 2, NewInstances: 4,
			})).To(Succeed())
		}
		Expect(sdb.CountScalingHistories(ctx, appID, 0, -1, true)).To(Equal(4))
		Expect(sdb.CountScalingHistories(ctx, appID, 0, -1, false)).To(Equal(3))
		histories, err := sdb.RetrieveScalingHistories(ctx, appID, 0, -1, db.ASC, false, 1, 50)
		Expect(err).NotTo(HaveOccurred())
		Expect(histories).To(HaveLen(3))
		Expect(histories[0]).To(Equal(history))
		var nonNullCorrelations int
		Expect(dbHelper.GetContext(ctx, &nonNullCorrelations, dbHelper.Rebind("SELECT COUNT(*) FROM scalinghistory WHERE appid = ? AND correlationid IS NOT NULL"), appID)).To(Succeed())
		Expect(nonNullCorrelations).To(Equal(1))
	})

	DescribeTable("rejects invalid open request data before insertion", func(mutate func(*models.AppScalingHistory)) {
		mutate(history)
		Expect(sdb.SaveScalingHistory(history)).NotTo(Succeed())
		Expect(getScalingHistoryForApp(appID)).To(BeZero())
	},
		Entry("missing correlation", func(h *models.AppScalingHistory) { h.CorrelationID = "" }),
		Entry("correlation too long", func(h *models.AppScalingHistory) { h.CorrelationID = strings.Repeat("c", 37) }),
		Entry("missing app", func(h *models.AppScalingHistory) { h.AppId = "" }),
		Entry("app too long for the open guard", func(h *models.AppScalingHistory) { h.AppId = strings.Repeat("a", 256) }),
		Entry("missing dimension", func(h *models.AppScalingHistory) { h.ScalingDimension = "" }),
		Entry("zero cooldown", func(h *models.AppScalingHistory) { h.CoolDownSeconds = 0 }),
		Entry("negative cooldown", func(h *models.AppScalingHistory) { h.CoolDownSeconds = -1 }),
		Entry("cooldown exceeds database integer", func(h *models.AppScalingHistory) { h.CoolDownSeconds = math.MaxInt32 + 1 }),
		Entry("already completed", func(h *models.AppScalingHistory) { h.CompletedAt = 123 }),
		Entry("deployment GUID too long", func(h *models.AppScalingHistory) { h.DeploymentGUID = strings.Repeat("d", 37) }),
	)

	It("rejects a nil scaling history rather than panicking", func() {
		Expect(sdb.SaveScalingHistory(nil)).NotTo(Succeed())
	})

	It("truncates vertical diagnostics to MySQL character limits without changing caller data", func() {
		history.Reason, history.Message, history.Error = strings.Repeat("ä", 300), strings.Repeat("é", 300), strings.Repeat("ö", 300)
		Expect(sdb.SaveScalingHistory(history)).To(Succeed())
		stored := retrieve()
		Expect(stored.Reason).To(Equal(strings.Repeat("ä", 255)))
		Expect(stored.Message).To(Equal(strings.Repeat("é", 255)))
		Expect(stored.Error).To(Equal(strings.Repeat("ö", 255)))
		Expect(history.Error).To(Equal(strings.Repeat("ö", 300)))
	})

	It("enforces a nullable, unique correlation across different apps", func() {
		Expect(sdb.SaveScalingHistory(history)).To(Succeed())
		duplicate := *history
		duplicate.AppId = secondApp
		Expect(sdb.SaveScalingHistory(&duplicate)).NotTo(Succeed())
		Expect(getScalingHistoryForApp(secondApp)).To(BeZero())
	})

	It("allows exactly one concurrent pending save for an app before any deployment POST", func() {
		const requests = 12
		start := make(chan struct{})
		results := make(chan error, requests)
		for index := 0; index < requests; index++ {
			candidate := *history
			candidate.CorrelationID = fmt.Sprintf("concurrent-correlation-%d", index)
			candidate.Timestamp += int64(index)
			go func() {
				<-start
				results <- sdb.SaveScalingHistory(&candidate)
			}()
		}
		close(start)
		accepted := 0
		for index := 0; index < requests; index++ {
			if <-results == nil {
				accepted++ // Only this successful save may be followed by a POST.
			}
		}
		Expect(accepted).To(Equal(1))
		Expect(getScalingHistoryForApp(appID)).To(Equal(1))
	})

	It("recovers an unknown POST after a restart without releasing the open app guard", func() {
		Expect(sdb.SaveScalingHistory(history)).To(Succeed())
		Expect(sdb.MarkScalingRequestUnknown(ctx, history.CorrelationID, strings.Repeat("ü", 300))).To(Succeed())
		Expect(sdb.MarkScalingRequestUnknown(ctx, history.CorrelationID, strings.Repeat("ü", 300))).To(Succeed())
		history.Status, history.Error = models.ScalingStatusUnknown, strings.Repeat("ü", 255)
		Expect(retrieve()).To(Equal(history))
		duplicate := *history
		duplicate.Timestamp++
		duplicate.CorrelationID = "second-correlation"
		Expect(sdb.SaveScalingHistory(&duplicate)).NotTo(Succeed())
		Expect(sdb.Close()).To(Succeed())
		var err error
		sdb, err = NewScalingEngineSQLDB(config, logger)
		Expect(err).NotTo(HaveOccurred())
		open, err := sdb.GetOpenScalingHistories(ctx)
		Expect(err).NotTo(HaveOccurred())
		Expect(open).To(ContainElement(Equal(history)))
		Expect(sdb.SetScalingDeployment(ctx, history.CorrelationID, "deployment-guid")).To(Succeed())
		history.Status, history.Error, history.DeploymentGUID = models.ScalingStatusPending, "", "deployment-guid"
		Expect(retrieve()).To(Equal(history))
		Expect(sdb.HasOpenScalingHistory(ctx, appID)).To(BeTrue())
	})

	It("never silently replaces an assigned deployment GUID", func() {
		Expect(sdb.SaveScalingHistory(history)).To(Succeed())
		Expect(sdb.SetScalingDeployment(ctx, history.CorrelationID, "first-deployment")).To(Succeed())
		Expect(sdb.SetScalingDeployment(ctx, history.CorrelationID, "first-deployment")).To(Succeed())
		Expect(sdb.SetScalingDeployment(ctx, history.CorrelationID, "different-deployment")).To(MatchError(db.ErrConflict))
		Expect(sdb.MarkScalingRequestUnknown(ctx, history.CorrelationID, "late POST timeout")).To(Succeed())
		stored := retrieve()
		Expect(stored.DeploymentGUID).To(Equal("first-deployment"))
		Expect(stored.Status).To(Equal(models.ScalingStatusPending))
		Expect(stored.Error).To(BeEmpty())
	})

	It("serializes competing GUID acknowledgements and late unknown marks", func() {
		Expect(sdb.SaveScalingHistory(history)).To(Succeed())
		start := make(chan struct{})
		results := make(chan error, 2)
		for _, guid := range []string{"first-deployment", "second-deployment"} {
			go func() {
				<-start
				results <- sdb.SetScalingDeployment(ctx, history.CorrelationID, guid)
			}()
		}
		close(start)
		acknowledgements := []error{<-results, <-results}
		Expect(acknowledgements).To(ContainElement(BeNil()))
		Expect(acknowledgements).To(ContainElement(MatchError(db.ErrConflict)))
		Expect(sdb.MarkScalingRequestUnknown(ctx, history.CorrelationID, "late error")).To(Succeed())
		Expect(retrieve().Status).To(Equal(models.ScalingStatusPending))
	})

	It("reports missing histories or invalid GUIDs rather than silently accepting acknowledgements", func() {
		Expect(sdb.SetScalingDeployment(ctx, "missing", "deployment-guid")).To(MatchError(db.ErrDoesNotExist))
		Expect(sdb.MarkScalingRequestUnknown(ctx, "missing", "timeout")).To(MatchError(db.ErrDoesNotExist))
		Expect(sdb.SetScalingDeployment(ctx, history.CorrelationID, "")).NotTo(Succeed())
		Expect(sdb.SetScalingDeployment(ctx, "", "deployment-guid")).NotTo(Succeed())
		Expect(sdb.SetScalingDeployment(ctx, history.CorrelationID, strings.Repeat("d", 37))).NotTo(Succeed())
		Expect(sdb.MarkScalingRequestUnknown(ctx, "", "timeout")).NotTo(Succeed())
	})

	It("completes once with the stored duration and permits the next open operation", func() {
		Expect(sdb.SaveScalingHistory(history)).To(Succeed())
		Expect(sdb.SetScalingDeployment(ctx, history.CorrelationID, "deployment-guid")).To(Succeed())
		completedAt := time.Now().UnixNano()
		Expect(sdb.CompleteScalingHistory(ctx, history.CorrelationID, models.ScalingStatusSucceeded, completedAt, "deployed", "")).To(BeTrue())
		stored := retrieve()
		Expect(stored.Status).To(Equal(models.ScalingStatusSucceeded))
		Expect(stored.CompletedAt).To(Equal(completedAt))
		Expect(stored.Message).To(Equal("deployed"))
		canScale, expiry, err := sdb.CanScaleApp(appID)
		Expect(err).NotTo(HaveOccurred())
		Expect(canScale).To(BeFalse())
		Expect(expiry).To(Equal(completedAt + int64(history.CoolDownSeconds)*int64(time.Second)))
		Expect(sdb.CompleteScalingHistory(ctx, history.CorrelationID, models.ScalingStatusSucceeded, completedAt+int64(time.Hour), "duplicate", "")).To(BeFalse())
		Expect(sdb.HasOpenScalingHistory(ctx, appID)).To(BeFalse())
		Expect(sdb.SetScalingDeployment(ctx, history.CorrelationID, "deployment-guid")).To(Succeed())
		Expect(sdb.SetScalingDeployment(ctx, history.CorrelationID, "different-guid")).To(MatchError(db.ErrConflict))
		Expect(sdb.MarkScalingRequestUnknown(ctx, history.CorrelationID, "late timeout")).To(Succeed())
		Expect(retrieve()).To(Equal(stored))
		_, repeatedExpiry, err := sdb.CanScaleApp(appID)
		Expect(err).NotTo(HaveOccurred())
		Expect(repeatedExpiry).To(Equal(expiry))
		Expect(sdb.Close()).To(Succeed())
		sdb, err = NewScalingEngineSQLDB(config, logger)
		Expect(err).NotTo(HaveOccurred())
		Expect(sdb.CompleteScalingHistory(ctx, history.CorrelationID, models.ScalingStatusSucceeded, completedAt+int64(time.Hour), "after restart", "")).To(BeFalse())
		next := *history
		next.CorrelationID, next.Timestamp = "next-correlation", 101
		Expect(sdb.SaveScalingHistory(&next)).To(Succeed())
		Expect(sdb.HasOpenScalingHistory(ctx, appID)).To(BeTrue())
	})

	It("allows only one of multiple completion checkers to set cooldown", func() {
		Expect(sdb.SaveScalingHistory(history)).To(Succeed())
		Expect(sdb.SetScalingDeployment(ctx, history.CorrelationID, "deployment-guid")).To(Succeed())
		const checkers = 12
		start := make(chan struct{})
		type completion struct {
			changed bool
			err     error
		}
		results := make(chan completion, checkers)
		completedAt := time.Now().UnixNano()
		for index := 0; index < checkers; index++ {
			go func() {
				<-start
				changed, err := sdb.CompleteScalingHistory(ctx, history.CorrelationID, models.ScalingStatusSucceeded, completedAt+int64(index)*int64(time.Second), "deployed", "")
				results <- completion{changed: changed, err: err}
			}()
		}
		close(start)
		winners := 0
		for index := 0; index < checkers; index++ {
			result := <-results
			Expect(result.err).NotTo(HaveOccurred())
			if result.changed {
				winners++
			}
		}
		Expect(winners).To(Equal(1))
		stored := retrieve()
		_, expiry, err := sdb.CanScaleApp(appID)
		Expect(err).NotTo(HaveOccurred())
		Expect(expiry).To(Equal(stored.CompletedAt + int64(history.CoolDownSeconds)*int64(time.Second)))
		var count int
		Expect(dbHelper.GetContext(ctx, &count, dbHelper.Rebind("SELECT COUNT(*) FROM scalingcooldown WHERE appid = ?"), appID)).To(Succeed())
		Expect(count).To(Equal(1))
	})

	DescribeTable("terminal non-success does not create or replace an existing cooldown", func(status models.ScalingStatus, previousExpiry int64) {
		Expect(sdb.SaveScalingHistory(history)).To(Succeed())
		Expect(sdb.MarkScalingRequestUnknown(ctx, history.CorrelationID, "uncertain response")).To(Succeed())
		if previousExpiry != 0 {
			Expect(sdb.UpdateScalingCooldownExpireTime(appID, previousExpiry)).To(Succeed())
		}
		completedAt := time.Now().UnixNano()
		Expect(sdb.CompleteScalingHistory(ctx, history.CorrelationID, status, completedAt, strings.Repeat("ä", 300), strings.Repeat("ö", 300))).To(BeTrue())
		stored := retrieve()
		Expect(stored.Status).To(Equal(status))
		Expect(stored.CompletedAt).To(Equal(completedAt))
		Expect(stored.Message).To(Equal(strings.Repeat("ä", 255)))
		Expect(stored.Error).To(Equal(strings.Repeat("ö", 255)))
		Expect(sdb.HasOpenScalingHistory(ctx, appID)).To(BeFalse())
		Expect(sdb.CompleteScalingHistory(ctx, history.CorrelationID, models.ScalingStatusSucceeded, completedAt+1, "late success", "")).To(BeFalse())
		_, expiry, err := sdb.CanScaleApp(appID)
		Expect(err).NotTo(HaveOccurred())
		Expect(expiry).To(Equal(previousExpiry))
	},
		Entry("failed with no cooldown", models.ScalingStatusFailed, int64(0)),
		Entry("aborted with no cooldown", models.ScalingStatusAborted, int64(0)),
		Entry("superseded with no cooldown", models.ScalingStatusSuperseded, int64(0)),
		Entry("failed preserves cooldown", models.ScalingStatusFailed, int64(123)),
		Entry("aborted preserves cooldown", models.ScalingStatusAborted, int64(123)),
		Entry("superseded preserves cooldown", models.ScalingStatusSuperseded, int64(123)),
	)

	It("does not reopen a completed unknown request without a GUID", func() {
		Expect(sdb.SaveScalingHistory(history)).To(Succeed())
		Expect(sdb.CompleteScalingHistory(ctx, history.CorrelationID, models.ScalingStatusFailed, time.Now().UnixNano(), "request failed", "quota")).To(BeTrue())
		Expect(sdb.SetScalingDeployment(ctx, history.CorrelationID, "late-guid")).To(MatchError(db.ErrConflict))
		Expect(sdb.MarkScalingRequestUnknown(ctx, history.CorrelationID, "late error")).To(Succeed())
		Expect(retrieve().Status).To(Equal(models.ScalingStatusFailed))
	})

	It("cannot mark a request successful before its deployment GUID is persisted", func() {
		Expect(sdb.SaveScalingHistory(history)).To(Succeed())
		changed, err := sdb.CompleteScalingHistory(ctx, history.CorrelationID, models.ScalingStatusSucceeded, time.Now().UnixNano(), "deployed", "")
		Expect(err).To(HaveOccurred())
		Expect(changed).To(BeFalse())
		Expect(retrieve()).To(Equal(history))
		Expect(sdb.HasOpenScalingHistory(ctx, appID)).To(BeTrue())
		_, expiry, err := sdb.CanScaleApp(appID)
		Expect(err).NotTo(HaveOccurred())
		Expect(expiry).To(BeZero())
	})

	DescribeTable("refuses success if persisted deployment references are incomplete", func(assignment string) {
		history.DeploymentGUID = "deployment-guid"
		Expect(sdb.SaveScalingHistory(history)).To(Succeed())
		_, err := dbHelper.ExecContext(ctx, dbHelper.Rebind("UPDATE scalinghistory SET "+assignment+" WHERE correlationid = ?"), history.CorrelationID)
		Expect(err).NotTo(HaveOccurred())
		changed, err := sdb.CompleteScalingHistory(ctx, history.CorrelationID, models.ScalingStatusSucceeded, time.Now().UnixNano(), "deployed", "")
		Expect(err).To(HaveOccurred())
		Expect(changed).To(BeFalse())
		Expect(retrieve().Status).To(Equal(models.ScalingStatusPending))
		Expect(sdb.HasOpenScalingHistory(ctx, appID)).To(BeTrue())
		_, expiry, err := sdb.CanScaleApp(appID)
		Expect(err).NotTo(HaveOccurred())
		Expect(expiry).To(BeZero())
	},
		Entry("NULL deployment GUID", "deploymentguid = NULL"),
		Entry("empty deployment GUID", "deploymentguid = ''"),
		Entry("NULL dimension", "scalingdimension = NULL"),
		Entry("horizontal dimension", "scalingdimension = 'horizontal'"),
	)

	DescribeTable("rejects invalid completions and leaves the request open", func(status models.ScalingStatus, completedAt int64) {
		history.DeploymentGUID = "deployment-guid"
		Expect(sdb.SaveScalingHistory(history)).To(Succeed())
		changed, err := sdb.CompleteScalingHistory(ctx, history.CorrelationID, status, completedAt, "invalid", "")
		Expect(err).To(HaveOccurred())
		Expect(changed).To(BeFalse())
		Expect(retrieve()).To(Equal(history))
		_, expiry, err := sdb.CanScaleApp(appID)
		Expect(err).NotTo(HaveOccurred())
		Expect(expiry).To(BeZero())
	},
		Entry("pending", models.ScalingStatusPending, int64(123)),
		Entry("unknown", models.ScalingStatusUnknown, int64(123)),
		Entry("ignored", models.ScalingStatusIgnored, int64(123)),
		Entry("undefined status", models.ScalingStatus(99), int64(123)),
		Entry("zero timestamp", models.ScalingStatusSucceeded, int64(0)),
		Entry("negative timestamp", models.ScalingStatusSucceeded, int64(-1)),
		Entry("timestamp plus duration overflows", models.ScalingStatusSucceeded, int64(math.MaxInt64)),
	)

	It("does not complete a missing history", func() {
		Expect(sdb.CompleteScalingHistory(ctx, "missing-correlation", models.ScalingStatusSucceeded, time.Now().UnixNano(), "done", "")).To(BeFalse())
		changed, err := sdb.CompleteScalingHistory(ctx, "", models.ScalingStatusSucceeded, time.Now().UnixNano(), "done", "")
		Expect(err).To(HaveOccurred())
		Expect(changed).To(BeFalse())
	})

	DescribeTable("rolls back completion if stored cooldown is invalid", func(value any) {
		history.DeploymentGUID = "deployment-guid"
		Expect(sdb.SaveScalingHistory(history)).To(Succeed())
		_, err := dbHelper.ExecContext(ctx, dbHelper.Rebind("UPDATE scalinghistory SET cooldownsecs = ? WHERE correlationid = ?"), value, history.CorrelationID)
		Expect(err).NotTo(HaveOccurred())
		changed, err := sdb.CompleteScalingHistory(ctx, history.CorrelationID, models.ScalingStatusSucceeded, time.Now().UnixNano(), "deployed", "")
		Expect(err).To(HaveOccurred())
		Expect(changed).To(BeFalse())
		Expect(retrieve().Status).To(Equal(models.ScalingStatusPending))
		Expect(sdb.HasOpenScalingHistory(ctx, appID)).To(BeTrue())
		_, expiry, err := sdb.CanScaleApp(appID)
		Expect(err).NotTo(HaveOccurred())
		Expect(expiry).To(BeZero())
	}, Entry("NULL", nil), Entry("zero", 0), Entry("negative", -1))

	It("rolls back history and the previous cooldown when the cooldown insert fails", func() {
		history.DeploymentGUID = "deployment-guid"
		Expect(sdb.SaveScalingHistory(history)).To(Succeed())
		Expect(sdb.UpdateScalingCooldownExpireTime(appID, 123)).To(Succeed())
		_, err := dbHelper.ExecContext(ctx, fmt.Sprintf("ALTER TABLE scalingcooldown ADD CONSTRAINT vertical_test_expiry CHECK (appid <> '%s' OR expireat <= 123)", appID))
		Expect(err).NotTo(HaveOccurred())
		DeferCleanup(func() {
			statement := "ALTER TABLE scalingcooldown DROP CONSTRAINT vertical_test_expiry"
			if dbHelper.DriverName() == db.MysqlDriverName {
				statement = "ALTER TABLE scalingcooldown DROP CHECK vertical_test_expiry"
			}
			_, err := dbHelper.Exec(statement)
			Expect(err).NotTo(HaveOccurred())
		})
		changed, err := sdb.CompleteScalingHistory(ctx, history.CorrelationID, models.ScalingStatusSucceeded, time.Now().UnixNano(), "deployed", "")
		Expect(err).To(HaveOccurred())
		Expect(changed).To(BeFalse())
		Expect(retrieve()).To(Equal(history))
		Expect(sdb.HasOpenScalingHistory(ctx, appID)).To(BeTrue())
		_, expiry, err := sdb.CanScaleApp(appID)
		Expect(err).NotTo(HaveOccurred())
		Expect(expiry).To(Equal(int64(123)), "the old cooldown DELETE must also be rolled back")
		Expect(sdb.UpdateScalingCooldownExpireTime(appID, 124)).NotTo(Succeed())
		_, expiry, err = sdb.CanScaleApp(appID)
		Expect(err).NotTo(HaveOccurred())
		Expect(expiry).To(Equal(int64(123)), "the legacy horizontal update uses the same atomic helper")
	})

	It("preserves pending and unknown histories while pruning old terminal and horizontal records", func() {
		Expect(sdb.SaveScalingHistory(history)).To(Succeed())
		unknown := *history
		unknown.AppId, unknown.CorrelationID, unknown.Status = secondApp, "unknown-correlation", models.ScalingStatusUnknown
		Expect(sdb.SaveScalingHistory(&unknown)).To(Succeed())
		Expect(sdb.SaveScalingHistory(&models.AppScalingHistory{AppId: appID, Timestamp: 99, Status: models.ScalingStatusSucceeded})).To(Succeed())
		Expect(sdb.PruneScalingHistories(ctx, 100)).To(Succeed())
		Expect(retrieve()).To(Equal(history))
		Expect(getScalingHistoryForApp(secondApp)).To(Equal(1))
		Expect(sdb.HasOpenScalingHistory(ctx, appID)).To(BeTrue())
		Expect(sdb.HasOpenScalingHistory(ctx, secondApp)).To(BeTrue())
		Expect(sdb.CompleteScalingHistory(ctx, history.CorrelationID, models.ScalingStatusAborted, time.Now().UnixNano(), "aborted", "")).To(BeTrue())
		Expect(sdb.PruneScalingHistories(ctx, 100)).To(Succeed())
		Expect(getScalingHistoryForApp(appID)).To(BeZero())
		Expect(getScalingHistoryForApp(secondApp)).To(Equal(1))
	})

	It("also preserves an open status if the nullable open marker is absent", func() {
		history.DeploymentGUID = "deployment-guid"
		Expect(sdb.SaveScalingHistory(history)).To(Succeed())
		_, err := dbHelper.ExecContext(ctx, dbHelper.Rebind("UPDATE scalinghistory SET openappid = NULL WHERE correlationid = ?"), history.CorrelationID)
		Expect(err).NotTo(HaveOccurred())
		Expect(sdb.PruneScalingHistories(ctx, 100)).To(Succeed())
		Expect(retrieve()).To(Equal(history))
		changed, err := sdb.CompleteScalingHistory(ctx, history.CorrelationID, models.ScalingStatusSucceeded, time.Now().UnixNano(), "deployed", "")
		Expect(err).To(MatchError(db.ErrConflict))
		Expect(changed).To(BeFalse())
		_, expiry, err := sdb.CanScaleApp(appID)
		Expect(err).NotTo(HaveOccurred())
		Expect(expiry).To(BeZero())
	})

	It("propagates cancelled context errors without changing persisted state", func() {
		Expect(sdb.SaveScalingHistory(history)).To(Succeed())
		cancelled, cancel := context.WithCancel(ctx)
		cancel()
		_, err := sdb.GetOpenScalingHistories(cancelled)
		Expect(err).To(MatchError(context.Canceled))
		_, err = sdb.HasOpenScalingHistory(cancelled, appID)
		Expect(err).To(MatchError(context.Canceled))
		Expect(sdb.SetScalingDeployment(cancelled, history.CorrelationID, "deployment-guid")).To(MatchError(context.Canceled))
		Expect(sdb.MarkScalingRequestUnknown(cancelled, history.CorrelationID, "timeout")).To(MatchError(context.Canceled))
		changed, err := sdb.CompleteScalingHistory(cancelled, history.CorrelationID, models.ScalingStatusSucceeded, time.Now().UnixNano(), "done", "")
		Expect(err).To(MatchError(context.Canceled))
		Expect(changed).To(BeFalse())
		Expect(retrieve()).To(Equal(history))
	})

	It("handles late unknown updates racing with successful completion", func() {
		Expect(sdb.SaveScalingHistory(history)).To(Succeed())
		Expect(sdb.SetScalingDeployment(ctx, history.CorrelationID, "deployment-guid")).To(Succeed())
		var workers sync.WaitGroup
		results := make(chan error, 2)
		workers.Add(2)
		go func() {
			defer workers.Done()
			results <- sdb.MarkScalingRequestUnknown(ctx, history.CorrelationID, "late timeout")
		}()
		go func() {
			defer workers.Done()
			_, err := sdb.CompleteScalingHistory(ctx, history.CorrelationID, models.ScalingStatusSucceeded, time.Now().UnixNano(), "done", "")
			results <- err
		}()
		workers.Wait()
		Expect(<-results).To(Succeed())
		Expect(<-results).To(Succeed())
		stored := retrieve()
		Expect(stored.Status).To(Equal(models.ScalingStatusSucceeded))
		Expect(stored.Error).To(BeEmpty())
		var marker sql.NullString
		Expect(dbHelper.GetContext(ctx, &marker, dbHelper.Rebind("SELECT openappid FROM scalinghistory WHERE correlationid = ?"), history.CorrelationID)).To(Succeed())
		Expect(marker.Valid).To(BeFalse())
	})
})
