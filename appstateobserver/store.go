package appstateobserver

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"sort"
	"time"

	"code.cloudfoundry.org/app-autoscaler/src/autoscaler/appstate"
	"code.cloudfoundry.org/app-autoscaler/src/autoscaler/cf"
	"code.cloudfoundry.org/app-autoscaler/src/autoscaler/db"
	_ "github.com/jackc/pgx/v5/stdlib"
)

// Store owns its tables. The PoC uses the existing scalingengine database,
// avoiding a new database provisioning requirement; consumers access only HTTP.
type Store struct{ db *sql.DB }

func NewStore(ctx context.Context, conf db.DatabaseConfig) (*Store, error) {
	connection, err := db.GetConnection(conf.URL)
	if err != nil {
		return nil, err
	}
	if connection.DriverName != db.PostgresDriverName {
		return nil, fmt.Errorf("appstateobserver requires PostgreSQL")
	}
	database, err := sql.Open(connection.DriverName, connection.DataSourceName)
	if err != nil {
		return nil, err
	}
	database.SetMaxOpenConns(8)
	database.SetConnMaxLifetime(conf.ConnectionMaxLifetime)
	if err := database.PingContext(ctx); err != nil {
		_ = database.Close()
		return nil, err
	}
	return &Store{db: database}, nil
}

func (s *Store) Close() error { return s.db.Close() }

func (s *Store) ReaderCheckedAt(ctx context.Context) (time.Time, error) {
	var checkedAt time.Time
	err := s.db.QueryRowContext(ctx, "SELECT checked_at FROM appstate_meta WHERE id=1").Scan(&checkedAt)
	return checkedAt, err
}

type Window struct {
	Until   time.Time
	Events  []cf.AuditEvent
	Managed map[string]bool
}

// Poll holds a transaction-scoped leader lock while fetching a bounded event
// window. Checkpoint, membership, deduplication and invalidations commit together.
// Losing the connection releases the lock and rolls back the entire window.
func (s *Store) Poll(ctx context.Context, fetch func(time.Time) (Window, error)) error {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback() }()
	var leader bool
	if err = tx.QueryRowContext(ctx, "SELECT pg_try_advisory_xact_lock(132083)").Scan(&leader); err != nil || !leader {
		return err
	}
	var checkpoint time.Time
	var revision int64
	if err = tx.QueryRowContext(ctx, "SELECT checkpoint, revision FROM appstate_meta WHERE id=1 FOR UPDATE").Scan(&checkpoint, &revision); err != nil {
		return err
	}
	window, err := fetch(checkpoint)
	if err != nil {
		return err
	}
	rows, err := tx.QueryContext(ctx, "SELECT observation FROM appstate_observations")
	if err != nil {
		return err
	}
	observations, err := decodeRows(rows)
	if err != nil {
		return err
	}
	byID := make(map[string]appstate.Observation, len(observations))
	changed := make(map[string]bool)
	for _, item := range observations {
		byID[item.AppID] = item
		if item.Managed != window.Managed[item.AppID] {
			item.Managed = window.Managed[item.AppID]
			item.Dirty = item.Managed
			byID[item.AppID] = item
			changed[item.AppID] = true
		}
	}
	for appID := range window.Managed {
		if _, exists := byID[appID]; !exists {
			byID[appID] = appstate.Observation{AppID: appID, Managed: true, Dirty: true}
			changed[appID] = true
		}
	}
	for _, event := range window.Events {
		result, err := tx.ExecContext(ctx, "INSERT INTO appstate_seen(guid, created_at) VALUES($1,$2) ON CONFLICT DO NOTHING", event.GUID, event.CreatedAt)
		if err != nil {
			return err
		}
		count, err := result.RowsAffected()
		if err != nil {
			return err
		}
		if count == 0 {
			continue
		}
		if event.TargetType == "app" && window.Managed[event.TargetGUID] {
			changed[event.TargetGUID] = true
		} else if event.TargetType == "route" {
			// Route-target events do not always carry an app GUID. Conservatively
			// invalidate managed apps; app map/unmap events cover the common case.
			for appID := range window.Managed {
				changed[appID] = true
			}
		}
	}
	ids := make([]string, 0, len(changed))
	for appID := range changed {
		ids = append(ids, appID)
	}
	sort.Strings(ids)
	for _, appID := range ids {
		item := byID[appID]
		item.Dirty = item.Managed
		revision++
		item.Revision = revision
		if err := writeObservation(ctx, tx, item); err != nil {
			return err
		}
	}
	if _, err = tx.ExecContext(ctx, "UPDATE appstate_meta SET checkpoint=$1, checked_at=$1, revision=$2 WHERE id=1", window.Until, revision); err != nil {
		return err
	}
	// Keep deduplication longer than the overlap. Tombstones remain in the
	// compacted feed so a restarted consumer can always recover from cursor 0.
	if _, err = tx.ExecContext(ctx, "DELETE FROM appstate_seen WHERE created_at < $1", window.Until.Add(-24*time.Hour)); err != nil {
		return err
	}
	return tx.Commit()
}

func writeObservation(ctx context.Context, tx *sql.Tx, item appstate.Observation) error {
	data, err := json.Marshal(item)
	if err != nil {
		return err
	}
	_, err = tx.ExecContext(ctx, `INSERT INTO appstate_observations(app_id,revision,managed,dirty,verified_at,observation)
		VALUES($1,$2,$3,$4,$5,$6) ON CONFLICT(app_id) DO UPDATE SET revision=$2,managed=$3,dirty=$4,verified_at=$5,observation=$6,retry_at='epoch'`,
		item.AppID, item.Revision, item.Managed, item.Dirty, item.VerifiedAt, data)
	return err
}

func decodeRows(rows *sql.Rows) ([]appstate.Observation, error) {
	defer func() { _ = rows.Close() }()
	var items []appstate.Observation
	for rows.Next() {
		var data []byte
		if err := rows.Scan(&data); err != nil {
			return nil, err
		}
		var item appstate.Observation
		if err := json.Unmarshal(data, &item); err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	return items, rows.Err()
}

func (s *Store) Changes(ctx context.Context, after int64) (appstate.Changes, error) {
	tx, err := s.db.BeginTx(ctx, &sql.TxOptions{Isolation: sql.LevelRepeatableRead, ReadOnly: true})
	if err != nil {
		return appstate.Changes{}, err
	}
	defer func() { _ = tx.Rollback() }()
	result := appstate.Changes{Cursor: after}
	var currentRevision int64
	if err = tx.QueryRowContext(ctx, "SELECT checked_at,revision FROM appstate_meta WHERE id=1").Scan(&result.CheckedAt, &currentRevision); err != nil {
		return result, err
	}
	if after > currentRevision {
		after = 0
		result.Cursor = 0
		result.Reset = true
	}
	rows, err := tx.QueryContext(ctx, "SELECT observation FROM appstate_observations WHERE revision>$1 ORDER BY revision LIMIT 500", after)
	if err != nil {
		return result, err
	}
	result.Items, err = decodeRows(rows)
	if err != nil {
		return result, err
	}
	result.More = len(result.Items) == 500
	if len(result.Items) > 0 {
		result.Cursor = result.Items[len(result.Items)-1].Revision
	}
	return result, tx.Commit()
}

func (s *Store) Pending(ctx context.Context, reconcileBefore time.Time) ([]appstate.Observation, error) {
	rows, err := s.db.QueryContext(ctx, `WITH claimed AS (
		SELECT app_id FROM appstate_observations
		WHERE managed AND retry_at<=CURRENT_TIMESTAMP AND (dirty OR verified_at<$1)
		ORDER BY dirty DESC,verified_at FOR UPDATE SKIP LOCKED LIMIT 20
	) UPDATE appstate_observations SET retry_at=CURRENT_TIMESTAMP+interval '90 seconds'
		WHERE app_id IN (SELECT app_id FROM claimed) RETURNING observation`, reconcileBefore)
	if err != nil {
		return nil, err
	}
	return decodeRows(rows)
}

// Save is fenced by the revision captured before the CF read. No late read can
// overwrite a newer event invalidation, membership change or mutation notice.
func (s *Store) Save(ctx context.Context, expected int64, item appstate.Observation) error {
	return s.update(ctx, item.AppID, func(current appstate.Observation) (appstate.Observation, bool) {
		return item, current.Revision == expected && current.Managed
	})
}

func (s *Store) Invalidate(ctx context.Context, appID string) error {
	return s.update(ctx, appID, func(item appstate.Observation) (appstate.Observation, bool) {
		item.Dirty = true
		return item, item.Managed
	})
}

func (s *Store) update(ctx context.Context, appID string, change func(appstate.Observation) (appstate.Observation, bool)) error {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback() }()
	var revision int64
	if err = tx.QueryRowContext(ctx, "SELECT revision FROM appstate_meta WHERE id=1 FOR UPDATE").Scan(&revision); err != nil {
		return err
	}
	var data []byte
	if err = tx.QueryRowContext(ctx, "SELECT observation FROM appstate_observations WHERE app_id=$1", appID).Scan(&data); errors.Is(err, sql.ErrNoRows) {
		return nil
	} else if err != nil {
		return err
	}
	var item appstate.Observation
	if err = json.Unmarshal(data, &item); err != nil {
		return err
	}
	item, changed := change(item)
	if !changed {
		return nil
	}
	revision++
	item.Revision = revision
	if err = writeObservation(ctx, tx, item); err != nil {
		return err
	}
	if _, err = tx.ExecContext(ctx, "UPDATE appstate_meta SET revision=$1 WHERE id=1", revision); err != nil {
		return err
	}
	return tx.Commit()
}

func (s *Store) Retry(ctx context.Context, item appstate.Observation) error {
	_, err := s.db.ExecContext(ctx, "UPDATE appstate_observations SET retry_at=CURRENT_TIMESTAMP+interval '15 seconds' WHERE app_id=$1 AND revision=$2", item.AppID, item.Revision)
	return err
}
