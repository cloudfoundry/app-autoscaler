package sqldb

import (
	"os"

	"code.cloudfoundry.org/app-autoscaler/src/autoscaler/db"
	"code.cloudfoundry.org/lager"
)

func CreatePolicyDb(dbConf db.DatabaseConfig, logger lager.Logger) *PolicySQLDB {
	policyDB, err := NewPolicySQLDB(dbConf, logger.Session("policy-db"))
	if err != nil {
		logger.Fatal("Failed To connect to policyDB", err, lager.Data{"dbConfig": dbConf})
		os.Exit(1)
	}
	return policyDB
}
