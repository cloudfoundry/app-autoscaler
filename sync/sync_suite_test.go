package sync_test

import (
	"database/sql"
	"fmt"

	"code.cloudfoundry.org/app-autoscaler/src/autoscaler/db"
	"code.cloudfoundry.org/app-autoscaler/src/autoscaler/testhelpers"

	_ "github.com/go-sql-driver/mysql"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"testing"
)

func TestSync(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Sync Suite")
}

var (
	dbHelper      *sql.DB
	lockTableName string
)
var _ = BeforeSuite(func() {
	var e error
	lockTableName = fmt.Sprintf("lock_table_%d", GinkgoParallelProcess())
	dbUrl := testhelpers.GetDbUrl()

	database, e := db.GetConnection(dbUrl)
	if e != nil {
		Fail("failed to get database URL and drivername: " + e.Error())
	}

	dbHelper, e = sql.Open(database.DriverName, database.DataSourceName)
	if e != nil {
		Fail("can not connect database: " + e.Error())
	}

	e = createLockTable()
	if e != nil {
		Fail("can not create test lock table: " + e.Error())
	}

})

var _ = AfterSuite(func() {
	if dbHelper != nil {
		e := dropLockTable()
		if e != nil {
			Fail("can not drop test lock table: " + e.Error())
		}
		dbHelper.Close()
	}

})

func getLockOwner() string {
	var owner string
	// #nosec G201
	query := fmt.Sprintf("SELECT owner FROM %s", lockTableName)
	row := dbHelper.QueryRow(query)
	err := row.Scan(&owner)
	if err == sql.ErrNoRows {
		owner = ""
	}
	return owner
}

func cleanLockTable() error {
	_, err := dbHelper.Exec(fmt.Sprintf("DELETE FROM %s", lockTableName))
	if err != nil {
		return err
	}
	return nil
}

func dropLockTable() error {
	_, err := dbHelper.Exec(fmt.Sprintf("DROP TABLE %s", lockTableName))
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
	`, lockTableName))
	if err != nil {
		return err
	}
	return nil
}
