package sqldb

import (
	"context"
	"database/sql"
	"fmt"
	"math"
	"time"

	"code.cloudfoundry.org/app-autoscaler/src/autoscaler/db"
	"code.cloudfoundry.org/app-autoscaler/src/autoscaler/models"
	"github.com/jmoiron/sqlx"
)

const scalingHistoryColumns = "appid, timestamp, scalingtype, status, oldinstances, newinstances, reason, " +
	"COALESCE(message, ''), COALESCE(error, ''), COALESCE(scalingdimension, ''), " +
	"COALESCE(correlationid, ''), COALESCE(deploymentguid, ''), COALESCE(oldmemorymb, 0), " +
	"COALESCE(newmemorymb, 0), COALESCE(cooldownsecs, 0), COALESCE(completedat, 0)"

func isOpenScalingStatus(status models.ScalingStatus) bool {
	return status == models.ScalingStatusPending || status == models.ScalingStatusUnknown
}

func scanScalingHistory(row interface{ Scan(...any) error }) (*models.AppScalingHistory, error) {
	h := &models.AppScalingHistory{}
	err := row.Scan(&h.AppId, &h.Timestamp, &h.ScalingType, &h.Status, &h.OldInstances, &h.NewInstances,
		&h.Reason, &h.Message, &h.Error, &h.ScalingDimension, &h.CorrelationID, &h.DeploymentGUID,
		&h.OldMemoryMB, &h.NewMemoryMB, &h.CoolDownSeconds, &h.CompletedAt)
	return h, err
}

func (sdb *ScalingEngineSQLDB) GetOpenScalingHistories(ctx context.Context) ([]*models.AppScalingHistory, error) {
	rows, err := sdb.sqldb.QueryContext(ctx, "SELECT "+scalingHistoryColumns+" FROM scalinghistory WHERE openappid IS NOT NULL ORDER BY timestamp")
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()
	histories := []*models.AppScalingHistory{}
	for rows.Next() {
		h, err := scanScalingHistory(rows)
		if err != nil {
			return nil, err
		}
		histories = append(histories, h)
	}
	return histories, rows.Err()
}

func (sdb *ScalingEngineSQLDB) HasOpenScalingHistory(ctx context.Context, appId string) (bool, error) {
	var count int
	err := sdb.sqldb.GetContext(ctx, &count, sdb.sqldb.Rebind("SELECT COUNT(*) FROM scalinghistory WHERE openappid = ?"), appId)
	return count != 0, err
}

func (sdb *ScalingEngineSQLDB) SetScalingDeployment(ctx context.Context, correlationID, deploymentGUID string) error {
	if correlationID == "" || deploymentGUID == "" || len(deploymentGUID) > 36 {
		return fmt.Errorf("correlation ID and deployment GUID (at most 36 characters) are required")
	}
	tx, err := sdb.sqldb.BeginTxx(ctx, nil)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback() }()
	h, err := lockedScalingHistory(ctx, tx, correlationID)
	if err != nil {
		return err
	}
	if h.DeploymentGUID != "" {
		if h.DeploymentGUID == deploymentGUID {
			return nil // An idempotent acknowledgement must not reopen a completed row.
		}
		return fmt.Errorf("scaling deployment already assigned: %w", db.ErrConflict)
	}
	if !isOpenScalingStatus(h.Status) || h.ScalingDimension != models.ScalingDimensionVertical {
		return fmt.Errorf("scaling history is not open: %w", db.ErrConflict)
	}
	query := tx.Rebind("UPDATE scalinghistory SET deploymentguid = ?, status = ?, error = '' " +
		"WHERE correlationid = ? AND status = ? AND openappid = ?")
	result, err := tx.ExecContext(ctx, query, deploymentGUID, models.ScalingStatusPending, correlationID, h.Status, h.AppId)
	if err != nil {
		return err
	}
	if err = requireScalingHistoryUpdate(result); err != nil {
		return err
	}
	return tx.Commit()
}

func (sdb *ScalingEngineSQLDB) MarkScalingRequestUnknown(ctx context.Context, correlationID, errorMessage string) error {
	if correlationID == "" {
		return fmt.Errorf("correlation ID is empty")
	}
	tx, err := sdb.sqldb.BeginTxx(ctx, nil)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback() }()
	h, err := lockedScalingHistory(ctx, tx, correlationID)
	if err != nil {
		return err
	}
	// A late POST error must not downgrade recovery or completion by another checker.
	if h.DeploymentGUID != "" || !isOpenScalingStatus(h.Status) {
		return nil
	}
	if h.ScalingDimension != models.ScalingDimensionVertical {
		return fmt.Errorf("scaling history is not vertical: %w", db.ErrConflict)
	}
	errorMessage = truncateHistoryText(errorMessage)
	if h.Status == models.ScalingStatusUnknown && h.Error == errorMessage {
		return nil
	}
	query := tx.Rebind("UPDATE scalinghistory SET status = ?, error = ? WHERE correlationid = ? AND status = ? AND openappid = ?")
	result, err := tx.ExecContext(ctx, query, models.ScalingStatusUnknown, errorMessage, correlationID, h.Status, h.AppId)
	if err != nil {
		return err
	}
	if err = requireScalingHistoryUpdate(result); err != nil {
		return err
	}
	return tx.Commit()
}

func lockedScalingHistory(ctx context.Context, tx *sqlx.Tx, correlationID string) (*models.AppScalingHistory, error) {
	h, err := scanScalingHistory(tx.QueryRowContext(ctx, tx.Rebind("SELECT "+scalingHistoryColumns+" FROM scalinghistory WHERE correlationid = ? FOR UPDATE"), correlationID))
	if err == sql.ErrNoRows {
		return nil, db.ErrDoesNotExist
	}
	return h, err
}

func requireScalingHistoryUpdate(result sql.Result) error {
	count, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if count != 1 {
		return fmt.Errorf("scaling history changed or is not open: %w", db.ErrConflict)
	}
	return nil
}

// CompleteScalingHistory serializes completion on the history row. The winner updates
// both history and the existing app cooldown in this transaction; subsequent checkers
// do nothing, including after a restart or a policy change.
func (sdb *ScalingEngineSQLDB) CompleteScalingHistory(ctx context.Context, correlationID string, status models.ScalingStatus, completedAt int64, message, errorMessage string) (bool, error) {
	if status != models.ScalingStatusSucceeded && status != models.ScalingStatusFailed &&
		status != models.ScalingStatusSuperseded && status != models.ScalingStatusAborted {
		return false, fmt.Errorf("invalid completion status %d", status)
	}
	if correlationID == "" || completedAt <= 0 {
		return false, fmt.Errorf("correlation ID and positive completion time are required")
	}
	tx, err := sdb.sqldb.BeginTxx(ctx, nil)
	if err != nil {
		return false, err
	}
	defer func() { _ = tx.Rollback() }()
	h, err := lockedScalingHistory(ctx, tx, correlationID)
	if err == db.ErrDoesNotExist {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	if !isOpenScalingStatus(h.Status) {
		return false, nil
	}
	if h.ScalingDimension != models.ScalingDimensionVertical {
		return false, fmt.Errorf("scaling history is not vertical: %w", db.ErrConflict)
	}
	if status == models.ScalingStatusSucceeded && (h.CorrelationID == "" || len(h.CorrelationID) > 36 || h.DeploymentGUID == "" || len(h.DeploymentGUID) > 36) {
		return false, fmt.Errorf("successful scaling completion requires stored correlation ID and deployment GUID")
	}
	query := tx.Rebind("UPDATE scalinghistory SET status = ?, completedat = ?, message = ?, error = ?, openappid = NULL WHERE correlationid = ? AND status = ? AND openappid = ?")
	result, err := tx.ExecContext(ctx, query, status, completedAt, truncateHistoryText(message), truncateHistoryText(errorMessage), correlationID, h.Status, h.AppId)
	if err != nil {
		return false, err
	}
	if err = requireScalingHistoryUpdate(result); err != nil {
		return false, err
	}
	if status == models.ScalingStatusSucceeded {
		if err = validateScalingCooldown(h.CoolDownSeconds); err != nil {
			return false, err
		}
		if int64(h.CoolDownSeconds) > (math.MaxInt64-completedAt)/int64(time.Second) {
			return false, fmt.Errorf("scaling cooldown expiry overflows int64")
		}
		if err = updateScalingCooldown(ctx, tx, h.AppId, completedAt+int64(h.CoolDownSeconds)*int64(time.Second)); err != nil {
			return false, err
		}
	}
	if err = tx.Commit(); err != nil {
		return false, err
	}
	return true, nil
}

func updateScalingCooldown(ctx context.Context, tx *sqlx.Tx, appId string, expireAt int64) error {
	if _, err := tx.ExecContext(ctx, tx.Rebind("DELETE FROM scalingcooldown WHERE appid = ?"), appId); err != nil {
		return err
	}
	_, err := tx.ExecContext(ctx, tx.Rebind("INSERT INTO scalingcooldown(appid, expireat) VALUES(?, ?)"), appId, expireAt)
	return err
}

func validateScalingCooldown(seconds int) error {
	if seconds <= 0 || int64(seconds) > math.MaxInt32 {
		return fmt.Errorf("scaling cooldown must be positive and fit the database integer column")
	}
	return nil
}

// Legacy MySQL reason, message and error columns are varchar(255). Count Unicode
// characters rather than bytes, so diagnostic text is never cut mid-codepoint.
func truncateHistoryText(message string) string {
	runes := []rune(message)
	if len(runes) > 255 {
		return string(runes[:255])
	}
	return message
}
