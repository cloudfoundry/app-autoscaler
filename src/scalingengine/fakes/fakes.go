package fakes

//go:generate counterfeiter -o ./fake_cf_client.go ../../cf CfClient
//go:generate counterfeiter -o ./fake_policy_db.go ../../db PolicyDB
