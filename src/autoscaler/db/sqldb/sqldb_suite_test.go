package sqldb_test

import (
	"database/sql"
	"encoding/json"
	"os"
	"testing"

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

func hasMetric(appId, name string, timestamp int64) bool {
	query := "SELECT * FROM applicationmetrics WHERE appid = $1 AND name = $2 AND timestamp = $3"
	rows, e := dbHelper.Query(query, appId, name, timestamp)
	if e != nil {
		Fail("can not query table applicationmetrics: " + e.Error())
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

	query := "INSERT INTO policy_json(app_id, policy_json) values($1, $2)"
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
