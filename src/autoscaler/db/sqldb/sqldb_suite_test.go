package sqldb_test

import (
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"testing"
	"time"

	"autoscaler/db"
	"autoscaler/models"

	_ "github.com/lib/pq"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var dbHelper *sql.DB

func TestSqldb(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Sqldb Suite")
}

var _ = BeforeSuite(func() {
	var e error

	dbUrl := os.Getenv("DBURL")
	if dbUrl == "" {
		Fail("environment variable $DBURL is not set")
	}

	dbHelper, e = sql.Open(db.PostgresDriverName, dbUrl)
	if e != nil {
		Fail("can not connect database: " + e.Error())
	}

	e = createLockTable()
	if e != nil {
		Fail("can not create test lock table: " + e.Error())
	}
})

var _ = AfterSuite(func() {
	e := dropLockTable()
	if e != nil {
		Fail("can not drop test lock table: " + e.Error())
	}
	if dbHelper != nil {
		dbHelper.Close()
	}

})

func cleanInstanceMetricsTable() {
	_, e := dbHelper.Exec("DELETE FROM appinstancemetrics")
	if e != nil {
		Fail("can not clean table appinstancemetrics:" + e.Error())
	}
}

func hasInstanceMetric(appId string, index int, name string, timestamp int64) bool {
	query := "SELECT * FROM appinstancemetrics WHERE appid = $1 AND instanceindex = $2 AND name = $3 AND timestamp = $4"
	rows, e := dbHelper.Query(query, appId, index, name, timestamp)
	if e != nil {
		Fail("can not query table appinstancemetrics: " + e.Error())
	}
	defer rows.Close()
	return rows.Next()
}

func getNumberOfInstanceMetrics() int {
	var num int
	e := dbHelper.QueryRow("SELECT COUNT(*) FROM appinstancemetrics").Scan(&num)
	if e != nil {
		Fail("can not count the number of records in table appinstancemetrics: " + e.Error())
	}
	return num
}

func cleanServiceBindingTable() {
	_, e := dbHelper.Exec("DELETE from binding")
	if e != nil {
		Fail("can not clean table binding: " + e.Error())
	}
}

func cleanServiceInstanceTable() {
	_, e := dbHelper.Exec("DELETE from service_instance")
	if e != nil {
		Fail("can not clean table service_instance: " + e.Error())
	}
}

func hasServiceInstance(serviceInstanceId string) bool {
	query := "SELECT * FROM service_instance WHERE service_instance_id = $1 "
	rows, e := dbHelper.Query(query, serviceInstanceId)
	if e != nil {
		Fail("can not query table service_instance: " + e.Error())
	}
	defer rows.Close()
	return rows.Next()
}

func hasServiceBinding(bindingId string, serviceInstanceId string) bool {
	query := "SELECT * FROM binding WHERE binding_id = $1 AND service_instance_id = $2 "
	rows, e := dbHelper.Query(query, bindingId, serviceInstanceId)
	if e != nil {
		Fail("can not query table binding: " + e.Error())
	}
	defer rows.Close()
	return rows.Next()
}

func cleanPolicyTable() {
	_, e := dbHelper.Exec("DELETE from policy_json")
	if e != nil {
		Fail("can not clean table policy_json: " + e.Error())
	}
}

func insertPolicy(appId string, scalingPolicy *models.ScalingPolicy) {
	policyJson, e := json.Marshal(scalingPolicy)
	if e != nil {
		Fail("failed to marshall scaling policy" + e.Error())
	}

	query := "INSERT INTO policy_json(app_id, policy_json, guid) VALUES($1, $2, $3)"
	_, e = dbHelper.Exec(query, appId, string(policyJson), "1234")

	if e != nil {
		Fail("can not insert data to table policy_json: " + e.Error())
	}
}

func getAppPolicy(appId string) string {
	query := "SELECT policy_json FROM policy_json WHERE app_id=$1 "
	rows, err := dbHelper.Query(query, appId)
	if err != nil {
		Fail("failed to get policy" + err.Error())
	}
	defer rows.Close()
	var policyJsonStr string
	if rows.Next() {
		err = rows.Scan(&policyJsonStr)
		if err != nil {
			Fail("failed to scan policy" + err.Error())
		}
	}
	return policyJsonStr
}

func cleanAppMetricTable() {
	_, e := dbHelper.Exec("DELETE from app_metric")
	if e != nil {
		Fail("can not clean table app_metric : " + e.Error())
	}
}

func hasAppMetric(appId, metricType string, timestamp int64, value string) bool {
	query := "SELECT * FROM app_metric WHERE app_id = $1 AND metric_type = $2 AND timestamp = $3 AND value = $4"
	rows, e := dbHelper.Query(query, appId, metricType, timestamp, value)
	if e != nil {
		Fail("can not query table app_metric: " + e.Error())
	}
	defer rows.Close()
	return rows.Next()
}

func getNumberOfAppMetrics() int {
	var num int
	e := dbHelper.QueryRow("SELECT COUNT(*) FROM app_metric").Scan(&num)
	if e != nil {
		Fail("can not count the number of records in table app_metric: " + e.Error())
	}
	return num
}

func cleanScalingHistoryTable() {
	_, e := dbHelper.Exec("DELETE from scalinghistory")
	if e != nil {
		Fail("can not clean table scalinghistory: " + e.Error())
	}
}

func hasScalingHistory(appId string, timestamp int64) bool {
	query := "SELECT * FROM scalinghistory WHERE appid = $1 AND timestamp = $2"
	rows, e := dbHelper.Query(query, appId, timestamp)
	if e != nil {
		Fail("can not query table scalinghistory: " + e.Error())
	}
	defer rows.Close()
	return rows.Next()
}

func getNumberOfScalingHistories() int {
	var num int
	e := dbHelper.QueryRow("SELECT COUNT(*) FROM scalinghistory").Scan(&num)
	if e != nil {
		Fail("can not count the number of records in table scalinghistory: " + e.Error())
	}
	return num
}

func cleanScalingCooldownTable() {
	_, e := dbHelper.Exec("DELETE from scalingcooldown")
	if e != nil {
		Fail("can not clean table scalingcooldown: " + e.Error())
	}
}

func hasScalingCooldownRecord(appId string, expireAt int64) bool {
	query := "SELECT * FROM scalingcooldown WHERE appid = $1 AND expireat = $2"
	rows, e := dbHelper.Query(query, appId, expireAt)
	if e != nil {
		Fail("can not query table scalingcooldown: " + e.Error())
	}
	defer rows.Close()
	return rows.Next()
}
func GetInt64Pointer(value int64) *int64 {
	tmp := value
	return &tmp
}

func cleanActiveScheduleTable() error {
	_, e := dbHelper.Exec("DELETE from activeschedule")
	return e
}

func insertActiveSchedule(appId, scheduleId string, instanceMin, instanceMax, instanceMinInitial int) error {
	query := "INSERT INTO activeschedule(appid, scheduleid, instancemincount, instancemaxcount, initialmininstancecount) " +
		" VALUES ($1, $2, $3, $4, $5)"
	_, e := dbHelper.Exec(query, appId, scheduleId, instanceMin, instanceMax, instanceMinInitial)
	return e
}

func cleanSchedulerActiveScheduleTable() error {
	_, e := dbHelper.Exec("DELETE from app_scaling_active_schedule")
	return e
}

func insertSchedulerActiveSchedule(id int, appId string, startJobIdentifier int, instanceMin, instanceMax, instanceMinInitial int) error {
	var e error
	var query string
	if instanceMinInitial <= 0 {
		query = "INSERT INTO app_scaling_active_schedule(id, app_id, start_job_identifier, instance_min_count, instance_max_count) " +
			" VALUES ($1, $2, $3, $4, $5)"
		_, e = dbHelper.Exec(query, id, appId, startJobIdentifier, instanceMin, instanceMax)
	} else {
		query = "INSERT INTO app_scaling_active_schedule(id, app_id, start_job_identifier, instance_min_count, instance_max_count, initial_min_instance_count) " +
			" VALUES ($1, $2, $3, $4, $5, $6)"
		_, e = dbHelper.Exec(query, id, appId, startJobIdentifier, instanceMin, instanceMax, instanceMinInitial)
	}
	return e
}

func insertCustomMetricsBindingCredentials(appid string, username string, password string) error {

	var err error
	var query string

	query = "INSERT INTO credentials(id, username, password, updated_at) values($1, $2, $3, $4)"
	_, err = dbHelper.Exec(query, appid, username, password, "2011-05-18 15:36:38")
	return err

}

func cleanCredentialsTable() error {
	_, err := dbHelper.Exec("DELETE FROM credentials")
	if err != nil {
		return err
	}
	return nil
}

func insertLockDetails(lock *models.Lock) (sql.Result, error) {
	query := "INSERT INTO test_lock (owner,lock_timestamp,ttl) VALUES ($1,$2,$3)"
	result, err := dbHelper.Exec(query, lock.Owner, lock.LastModifiedTimestamp, int64(lock.Ttl/time.Second))
	return result, err
}

func cleanLockTable() error {
	_, err := dbHelper.Exec("DELETE FROM test_lock")
	if err != nil {
		return err
	}
	return nil
}

func dropLockTable() error {
	_, err := dbHelper.Exec("DROP TABLE test_lock")
	if err != nil {
		return err
	}
	return nil
}

func createLockTable() error {
	_, err := dbHelper.Exec(`
		CREATE TABLE IF NOT EXISTS test_lock (
			owner VARCHAR(255) PRIMARY KEY,
			lock_timestamp TIMESTAMP  NOT NULL,
			ttl BIGINT DEFAULT 0
		);
	`)
	if err != nil {
		return err
	}
	return nil
}

func validateLockInDB(ownerid string, expectedLock *models.Lock) error {
	var (
		timestamp time.Time
		ttl       time.Duration
		owner     string
	)
	query := "SELECT owner,lock_timestamp,ttl FROM test_lock WHERE owner=$1"
	row := dbHelper.QueryRow(query, ownerid)
	err := row.Scan(&owner, &timestamp, &ttl)
	if err != nil {
		return err
	}
	errMsg := ""
	if expectedLock.Owner != owner {
		errMsg += fmt.Sprintf("mismatch owner (%s, %s),", expectedLock.Owner, owner)
	}
	if expectedLock.Ttl != time.Second*time.Duration(ttl) {
		errMsg += fmt.Sprintf("mismatch ttl (%d, %d),", expectedLock.Ttl, time.Second*time.Duration(ttl))
	}
	if errMsg != "" {
		return errors.New(errMsg)
	}
	return nil
}

func validateLockNotInDB(owner string) error {
	var (
		timestamp time.Time
		ttl       time.Duration
	)
	query := "SELECT owner,lock_timestamp,ttl FROM test_lock WHERE owner=$1"
	row := dbHelper.QueryRow(query, owner)
	err := row.Scan(&owner, &timestamp, &ttl)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil
		}
		return err
	}
	return fmt.Errorf("lock exists with owner (%s)", owner)
}
