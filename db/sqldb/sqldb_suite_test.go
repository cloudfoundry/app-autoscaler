package sqldb_test

import (
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"strings"
	"testing"
	"time"

	. "code.cloudfoundry.org/app-autoscaler/src/autoscaler/testhelpers"

	"code.cloudfoundry.org/app-autoscaler/src/autoscaler/db"
	"code.cloudfoundry.org/app-autoscaler/src/autoscaler/models"

	_ "github.com/go-sql-driver/mysql"
	"github.com/jmoiron/sqlx"
	_ "github.com/lib/pq"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var (
	dbHelper  *sqlx.DB
	lockTable string
)

func TestSqldb(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Sqldb Suite")
}

var _ = SynchronizedBeforeSuite(func() []byte {
	var err error

	dbUrl := os.Getenv("DBURL")
	if dbUrl == "" {
		Fail("environment variable $DBURL is not set")
	}
	database, err := db.GetConnection(dbUrl)
	FailOnError("failed to parse database connection", err)

	dbHelper, err = sqlx.Open(database.DriverName, database.DSN)
	FailOnError("can not connect database: ", err)

	_, err = dbHelper.Exec("DELETE from binding")
	FailOnError("can not clean table binding", err)

	_, err = dbHelper.Exec("DELETE from service_instance")
	FailOnError("can not clean table service_instance", err)

	if strings.Contains(os.Getenv("DBURL"), "postgres") && getPostgresMajorVersion() >= 12 {
		deleteAllFunctions()
		addPSQLFunctions()
	}

	_ = dbHelper.Close()
	dbHelper = nil

	return []byte{}
}, func([]byte) {
	var e error
	lockTable = fmt.Sprintf("test_lock_%d", GinkgoParallelProcess())
	dbUrl := os.Getenv("DBURL")
	if dbUrl == "" {
		Fail("environment variable $DBURL is not set")
	}
	database, err := db.GetConnection(dbUrl)
	FailOnError("failed to parse database connection: ", err)

	dbHelper, e = sqlx.Open(database.DriverName, database.DSN)
	if e != nil {
		Fail("can not connect database: " + e.Error())
	}

	err = createLockTable()
	FailOnError("can not create test lock table: ", err)

})

var _ = SynchronizedAfterSuite(func() {
	if dbHelper != nil && GinkgoParallelProcess() != 1 {
		_ = dbHelper.Close()
	}
}, func() {
	e := dropLockTable()
	if e != nil {
		Fail("can not drop test lock table: " + e.Error())
	}
	if dbHelper != nil && GinkgoParallelProcess() == 1 {
		_ = dbHelper.Close()
	}
})

func cleanInstanceMetricsTableForApp(appId string) {
	_, e := dbHelper.Exec(dbHelper.Rebind("DELETE FROM appinstancemetrics where appid=?"), appId)
	FailOnError("can not clean table appinstancemetrics:", e)
}

func hasInstanceMetric(appId string, index int, name string, timestamp int64) bool {
	query := dbHelper.Rebind("SELECT * FROM appinstancemetrics WHERE appid = ? AND instanceindex = ? AND name = ? AND timestamp = ?")
	rows, e := dbHelper.Query(query, appId, index, name, timestamp)
	if e != nil {
		Fail("can not query table appinstancemetrics: " + e.Error())
	}
	defer func() {
		_ = rows.Close()
		_ = rows.Err()
	}()
	return rows.Next()
}

func getNumberOfInstanceMetricsForApp(appId string) int {
	var num int
	e := dbHelper.QueryRow(dbHelper.Rebind("SELECT COUNT(*) FROM appinstancemetrics where appid = ?"), appId).Scan(&num)
	if e != nil {
		Fail("can not count the number of records in table appinstancemetrics: " + e.Error())
	}
	return num
}

func hasServiceInstance(serviceInstanceId string) bool {
	query := dbHelper.Rebind("SELECT * FROM service_instance WHERE service_instance_id = ?")
	rows, e := dbHelper.Query(query, serviceInstanceId)
	if e != nil {
		Fail("can not query table service_instance: " + e.Error())
	}
	defer func() {
		_ = rows.Close()
		_ = rows.Err()
	}()
	return rows.Next()
}

func hasServiceInstanceWithNullDefaultPolicy(serviceInstanceId string) bool {
	query := dbHelper.Rebind("SELECT * FROM service_instance WHERE service_instance_id = ? AND default_policy IS NULL AND default_policy_guid IS NULL")
	rows, e := dbHelper.Query(query, serviceInstanceId)
	if e != nil {
		Fail("can not query table service_instance: " + e.Error())
	}
	defer func() {
		_ = rows.Close()
		_ = rows.Err()
	}()

	return rows.Next()
}

func hasServiceBinding(bindingId string, serviceInstanceId string) bool {
	query := dbHelper.Rebind("SELECT * FROM binding WHERE binding_id = ? AND service_instance_id = ? ")
	rows, e := dbHelper.Query(query, bindingId, serviceInstanceId)
	if e != nil {
		Fail("can not query table binding: " + e.Error())
	}
	defer func() {
		_ = rows.Close()
		_ = rows.Err()
	}()
	return rows.Next()
}

func cleanPolicyTable() {
	_, e := dbHelper.Exec("DELETE from policy_json")
	if e != nil {
		Fail("can not clean table policy_json: " + e.Error())
	}
}

func insertPolicy(appId string, scalingPolicy *models.ScalingPolicy, policyGuid string) {
	policyJson, e := json.Marshal(scalingPolicy)
	if e != nil {
		Fail("failed to marshall scaling policy" + e.Error())
	}

	query := dbHelper.Rebind("INSERT INTO policy_json(app_id, policy_json, guid) VALUES(?, ?, ?)")
	_, e = dbHelper.Exec(query, appId, string(policyJson), policyGuid)

	if e != nil {
		Fail(fmt.Sprintf("can not insert app:%s data to table policy_json: %s", appId, e.Error()))
	}
}

func insertPolicyWithGuid(appId string, scalingPolicy *models.ScalingPolicy, guid string) {
	By("Insert policy:" + guid)
	policyJson, e := json.Marshal(scalingPolicy)
	if e != nil {
		Fail("failed to marshall scaling policy" + e.Error())
	}

	query := dbHelper.Rebind("INSERT INTO policy_json(app_id, policy_json, guid) VALUES(?, ?, ?)")
	_, e = dbHelper.Exec(query, appId, string(policyJson), guid)

	if e != nil {
		Fail("can not insert data to table policy_json: " + e.Error())
	}
}

func getAppPolicy(appId string) string {
	query := dbHelper.Rebind("SELECT policy_json FROM policy_json WHERE app_id=? ")
	rows, err := dbHelper.Query(query, appId)
	FailOnError("failed to get policy", err)
	defer func() {
		_ = rows.Close()
		_ = rows.Err()
	}()
	var policyJsonStr string
	if rows.Next() {
		err = rows.Scan(&policyJsonStr)
		FailOnError("failed to scan policy", err)
	}
	return policyJsonStr
}

func cleanAppMetricTable(appId string) {
	query := dbHelper.Rebind("DELETE from app_metric where app_id = ?")
	_, err := dbHelper.Exec(query, appId)
	FailOnError("can not clean table app_metric or app_id", err)
}

func hasAppMetric(appId, metricType string, timestamp int64, value string) bool {
	query := dbHelper.Rebind("SELECT * FROM app_metric WHERE app_id = ? AND metric_type = ? AND timestamp = ? AND value = ?")
	rows, e := dbHelper.Query(query, appId, metricType, timestamp, value)
	if e != nil {
		Fail("can not query table app_metric: " + e.Error())
	}
	defer func() {
		_ = rows.Close()
		_ = rows.Err()
	}()
	return rows.Next()
}

func getNumberOfMetricsForApp(appId string) int {
	var num int
	query := dbHelper.Rebind("SELECT COUNT(*) FROM app_metric where app_id = ?")
	err := dbHelper.QueryRow(query, appId).Scan(&num)
	FailOnError("can not count the number of records in table app_metric: ", err)
	return num
}

func removeScalingHistoryForApp(appId string) {
	query := dbHelper.Rebind("DELETE from scalinghistory where appId = ?")
	_, err := dbHelper.Exec(query, appId)
	FailOnError("can not clean table scalinghistory: ", err)
}

func removeCooldownForApp(appId string) {
	query := dbHelper.Rebind("DELETE from scalingcooldown where appId = ?")
	_, err := dbHelper.Exec(query, appId)
	FailOnError("can not clean table scalingcooldown: ", err)
}
func removeActiveScheduleForApp(appId string) {
	query := dbHelper.Rebind("DELETE from activeschedule where appId = ?")
	_, err := dbHelper.Exec(query, appId)
	FailOnError("can not clean table scalingcooldown: ", err)
}

func hasScalingHistory(appId string, timestamp int64) bool {
	query := dbHelper.Rebind("SELECT * FROM scalinghistory WHERE appid = ? AND timestamp = ?")
	rows, e := dbHelper.Query(query, appId, timestamp)
	if e != nil {
		Fail("can not query table scalinghistory: " + e.Error())
	}
	defer func() {
		_ = rows.Close()
		_ = rows.Err()
	}()
	return rows.Next()
}

func getScalingHistoryForApp(appId string) int {
	var num int
	query := dbHelper.Rebind("SELECT COUNT(*) FROM scalinghistory WHERE appid = ?")
	row := dbHelper.QueryRow(query, appId)
	err := row.Scan(&num)
	FailOnError("can not count the number of records in table scalinghistory: ", err)
	return num
}

func hasScalingCooldownRecord(appId string, expireAt int64) bool {
	query := dbHelper.Rebind("SELECT * FROM scalingcooldown WHERE appid = ? AND expireat = ?")
	rows, e := dbHelper.Query(query, appId, expireAt)
	if e != nil {
		Fail("can not query table scalingcooldown: " + e.Error())
	}
	defer func() {
		_ = rows.Close()
		_ = rows.Err()
	}()
	return rows.Next()
}

func insertActiveSchedule(appId, scheduleId string, instanceMin, instanceMax, instanceMinInitial int) error {
	query := dbHelper.Rebind("INSERT INTO activeschedule(appid, scheduleid, instancemincount, instancemaxcount, initialmininstancecount) " +
		" VALUES (?, ?, ?, ?, ?)")
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
		query = dbHelper.Rebind("INSERT INTO app_scaling_active_schedule(id, app_id, start_job_identifier, instance_min_count, instance_max_count) " +
			" VALUES (?, ?, ?, ?, ?)")
		_, e = dbHelper.Exec(query, id, appId, startJobIdentifier, instanceMin, instanceMax)
	} else {
		query = dbHelper.Rebind("INSERT INTO app_scaling_active_schedule(id, app_id, start_job_identifier, instance_min_count, instance_max_count, initial_min_instance_count) " +
			" VALUES (?, ?, ?, ?, ?, ?)")
		_, e = dbHelper.Exec(query, id, appId, startJobIdentifier, instanceMin, instanceMax, instanceMinInitial)
	}
	return e
}

func insertCredential(appid string, username string, password string) error {
	var err error
	query := dbHelper.Rebind("INSERT INTO credentials(id, username, password, updated_at) values(?, ?, ?, ?)")
	_, err = dbHelper.Exec(query, appid, username, password, "2011-05-18 15:36:38")
	return err
}

func getCredential(appId string) (string, string, error) {
	query := dbHelper.Rebind("SELECT username,password FROM credentials WHERE id=? ")
	rows, err := dbHelper.Query(query, appId)
	FailOnError("failed to get credential", err)
	defer func() {
		_ = rows.Close()
		_ = rows.Err()
	}()
	var username, password string
	if rows.Next() {
		err = rows.Scan(&username, &password)
		FailOnError("failed to scan credential", err)
	}
	return username, password, nil
}
func hasCredential(appId string) bool {
	query := dbHelper.Rebind("SELECT * FROM credentials WHERE id=?")
	rows, e := dbHelper.Query(query, appId)
	if e != nil {
		Fail("can not query table credentials: " + e.Error())
	}
	defer func() {
		_ = rows.Close()
		_ = rows.Err()
	}()
	return rows.Next()
}

func insertLockDetails(lock *models.Lock) (sql.Result, error) {
	query := dbHelper.Rebind(fmt.Sprintf("INSERT INTO %s (owner,lock_timestamp,ttl) VALUES (?,?,?)", lockTable))
	result, err := dbHelper.Exec(query, lock.Owner, lock.LastModifiedTimestamp, int64(lock.Ttl/time.Second))
	return result, err
}

func cleanLockTable() error {
	_, err := dbHelper.Exec(fmt.Sprintf("DELETE FROM %s", lockTable))
	if err != nil {
		return err
	}
	return nil
}

func dropLockTable() error {
	_, err := dbHelper.Exec(fmt.Sprintf("DROP TABLE %s", lockTable))
	if err != nil {
		return err
	}
	return nil
}

func createLockTable() error {
	_, err := dbHelper.Exec(fmt.Sprintf(`
		CREATE TABLE IF NOT EXISTS %s (
			owner VARCHAR(255) PRIMARY KEY,
			lock_timestamp TIMESTAMP  NOT NULL,
			ttl BIGINT DEFAULT 0
		);
	`, lockTable))
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
	query := dbHelper.Rebind(fmt.Sprintf("SELECT owner,lock_timestamp,ttl FROM %s WHERE owner=?", lockTable))
	row := dbHelper.QueryRow(query, ownerid)
	err := row.Scan(&owner, &timestamp, &ttl)
	if err != nil {
		return err
	}
	errMsg := ""
	if expectedLock.Owner != owner {
		errMsg += fmt.Sprintf("mismatch owner (%s, %s),", expectedLock.Owner, owner)
	}
	if expectedLock.Ttl != time.Second*ttl {
		errMsg += fmt.Sprintf("mismatch ttl (%d, %d),", expectedLock.Ttl, time.Second*ttl)
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
	query := dbHelper.Rebind(fmt.Sprintf("SELECT owner,lock_timestamp,ttl FROM %s WHERE owner=?", lockTable))
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

func formatPolicyString(policyStr string) (string, error) {
	scalingPolicy := &models.ScalingPolicy{}
	err := json.Unmarshal([]byte(policyStr), &scalingPolicy)
	if err != nil {
		return "", fmt.Errorf("failed to unmarshal policyJson string %s", policyStr)
	}
	policyJsonStr, err := json.Marshal(scalingPolicy)
	if err != nil {
		return "", fmt.Errorf("failed to marshal ScalingPolicy %v", scalingPolicy)
	}
	return string(policyJsonStr), nil
}

func expectServiceInstancesToEqual(actual *models.ServiceInstance, expected *models.ServiceInstance) {
	ExpectWithOffset(1, actual.ServiceInstanceId).To(Equal(expected.ServiceInstanceId))
	ExpectWithOffset(1, actual.OrgId).To(Equal(expected.OrgId))
	ExpectWithOffset(1, actual.SpaceId).To(Equal(expected.SpaceId))
	ExpectWithOffset(1, actual.DefaultPolicy).To(MatchJSON(expected.DefaultPolicy))
	ExpectWithOffset(1, actual.DefaultPolicyGuid).To(Equal(expected.DefaultPolicyGuid))
}
