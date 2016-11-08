package sqldb_test

import (
	"database/sql"
	"encoding/json"
	"os"
	"testing"
	"time"

	. "autoscaler/db/sqldb"
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

	dbHelper, e = sql.Open(PostgresDriverName, dbUrl)
	if e != nil {
		Fail("can not connect database: " + e.Error())
	}

})

var _ = AfterSuite(func() {
	if dbHelper != nil {
		dbHelper.Close()
	}

})

func cleanMetricsTable() {
	_, e := dbHelper.Exec("DELETE FROM applicationmetrics")
	if e != nil {
		Fail("can not clean table applicationmetrics:" + e.Error())
	}
}

func cleanInstanceMetricsTable() {
	_, e := dbHelper.Exec("DELETE FROM appinstancemetrics")
	if e != nil {
		Fail("can not clean table appinstancemetrics:" + e.Error())
	}
}

func hasMetric(appId, name string, timestamp int64) bool {
	query := "SELECT * FROM applicationmetrics WHERE appid = $1 AND name = $2 AND timestamp = $3"
	rows, e := dbHelper.Query(query, appId, name, timestamp)
	if e != nil {
		Fail("can not query table applicationmetrics: " + e.Error())
	}
	defer rows.Close()
	return rows.Next()
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

func getNumberOfMetrics() int {
	var num int
	e := dbHelper.QueryRow("SELECT COUNT(*) FROM applicationmetrics").Scan(&num)
	if e != nil {
		Fail("can not count the number of records in table applicationmetrics: " + e.Error())
	}
	return num
}

func getNumberOfInstanceMetrics() int {
	var num int
	e := dbHelper.QueryRow("SELECT COUNT(*) FROM appinstancemetrics").Scan(&num)
	if e != nil {
		Fail("can not count the number of records in table appinstancemetrics: " + e.Error())
	}
	return num
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

	query := "INSERT INTO policy_json(app_id, policy_json) VALUES($1, $2)"
	_, e = dbHelper.Exec(query, appId, string(policyJson))

	if e != nil {
		Fail("can not insert data to table policy_json: " + e.Error())
	}
}

func cleanAppMetricTable() {
	_, e := dbHelper.Exec("DELETE from app_metric")
	if e != nil {
		Fail("can not clean table app_metric : " + e.Error())
	}
}

func hasAppMetric(appId, metricType string, timestamp int64) bool {
	query := "SELECT * FROM app_metric WHERE app_id = $1 AND metric_type = $2 AND timestamp = $3"
	rows, e := dbHelper.Query(query, appId, metricType, timestamp)
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
	_, e := dbHelper.Exec("DELETE from app_scaling_active_schedule")
	return e
}

func insertActiveSchedule(appId string, instanceMin, instanceMax, instanceMinInitial int, status string) error {
	var query string
	var err error
	id := time.Now().UnixNano()
	if instanceMinInitial <= 0 {
		query = "INSERT INTO app_scaling_active_schedule(id, app_id, instance_min_count, instance_max_count, status) " +
			"i VALUES ($1, $2, $3, $4, $5)"
		_, err = dbHelper.Exec(query, id, appId, instanceMin, instanceMax, status)
	} else {
		query = "INSERT INTO app_scaling_active_schedule(id, app_id, instance_min_count, instance_max_count, initial_min_instance_count, status) " +
			" VALUES ($1, $2, $3, $4, $5, $6)"
		_, err = dbHelper.Exec(query, id, appId, instanceMin, instanceMax, instanceMinInitial, status)
	}
	return err
}
