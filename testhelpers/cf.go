package testhelpers

import (
	"encoding/json"
	"os"
)

func GetDbVcapServices(creds map[string]string, serviceName string, dbType string) (string, error) {
	credentials, err := json.Marshal(creds)
	if err != nil {
		return "", err
	}

	return `{
		"user-provided": [ { "name": "config", "credentials": { "metricsforwarder": { } }}],
		"autoscaler": [ {
			"name": "some-service",
			"credentials": ` + string(credentials) + `,
			"syslog_drain_url": "",
			"tags": ["` + serviceName + `", "` + dbType + `"]
			}
		]}`, nil // #nosec G101
}

func GetVcapServices(userProvidedServiceName string, configJson string) string {
	dbURL := os.Getenv("DBURL")

	return `{
		"user-provided": [ {
			"tags": [ "` + userProvidedServiceName + `" ],
			"name": "` + userProvidedServiceName + `",
			"credentials": {
				"` + userProvidedServiceName + `": ` + configJson + `
			}

		}],
		"autoscaler": [ {
			"name": "some-service",
			"credentials": {
				"uri": "` + dbURL + `"
				},
			"syslog_drain_url": "",
			"tags": [ "policy_db","binding_db", "postgres" ]

	   }]}`
}
