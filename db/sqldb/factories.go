package sqldb

import (
	"os"

	"code.cloudfoundry.org/app-autoscaler/src/autoscaler/db"
	"code.cloudfoundry.org/lager/v3"
)

func CreatePolicyDb(dbConf db.DatabaseConfig, logger lager.Logger) *PolicySQLDB {
	policyDB, err := NewPolicySQLDB(dbConf, logger.Session("policy-db"))
	if err != nil {
		logger.Error("failed to connect policy db", err, lager.Data{"dbConfig": dbConf})
		os.Exit(1)
	}
	return policyDB
}

func CreateBindingDB(dbConf db.DatabaseConfig, logger lager.Logger) *BindingSQLDB {
	bindingDB, err := NewBindingSQLDB(dbConf, logger.Session("binding-db"))
	if err != nil {
		logger.Fatal("Failed To connect to bindingDB", err, lager.Data{"dbConfig": dbConf})
		os.Exit(1)
	}
	return bindingDB
}
