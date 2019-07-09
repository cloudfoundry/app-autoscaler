package sync_test

import (
	"database/sql"
	"fmt"
	"os"

	"autoscaler/db"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"testing"
)

func TestSync(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Sync Suite")
}

var (
	dbHelper      *sql.DB
	lockTableName = "test_lock"
)
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
