package sqldb_test

import (
	"database/sql"
	"os"
	"testing"

	_ "github.com/lib/pq"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	. "metricscollector/db/sqldb"
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
		Fail("can not clean policy table: " + e.Error())
	}
}

func insertPolicy(appId string) {
	policy := `
		{
 			"instance_min_count": 1,
  			"instance_max_count": 5
		}`
	query := "INSERT INTO policy_json(app_id, policy_json) values($1, $2)"
	_, e := dbHelper.Exec(query, appId, policy)

	if e != nil {
		Fail("can not insert data to policy table: " + e.Error())
	}

}
