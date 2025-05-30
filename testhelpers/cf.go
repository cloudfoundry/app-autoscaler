package testhelpers

import (
	"encoding/json"
	"os"

	. "github.com/onsi/ginkgo/v2"

	. "github.com/onsi/gomega"
)

func GetDbVcapServices(creds map[string]string, databaseNames []string, dbType string) (string, error) {
	credentials, err := json.Marshal(creds)
	if err != nil {
		return "", err
	}

	return getDbVcapServices(string(credentials), databaseNames, dbType, "default")
}

// supports multiple db tags in the same vcap user provided service
func getDbVcapServices(creds string, databaseNames []string, dbType string, credHelperImpl string) (string, error) {
	tag := append(databaseNames, dbType)
	vcapServices := map[string]any{
		"user-provided": []map[string]any{
			{
				"name": "config",
				"credentials": map[string]any{
					"metricsforwarder": map[string]any{
						"cred_helper_impl": credHelperImpl,
					},
				},
			},
		},
		"autoscaler": []map[string]any{
			{
				"name":             "some-service",
				"credentials":      json.RawMessage(creds),
				"syslog_drain_url": "",
				"tags":             tag,
			},
		},
	}

	vcapServicesJson, err := json.Marshal(vcapServices)
	if err != nil {
		return "", err
	}

	return string(vcapServicesJson), nil
}

func GetVcapServices(userProvidedServiceName string, configJson string) string {
	GinkgoHelper()
	dbURL := os.Getenv("DBURL")

	catalogBytes, err := os.ReadFile("../api/exampleconfig/catalog-example.json")
	Expect(err).NotTo(HaveOccurred())

	vcapServices := map[string]any{
		"user-provided": []map[string]any{
			{
				"name":        userProvidedServiceName,
				"tags":        []string{userProvidedServiceName},
				"credentials": map[string]any{userProvidedServiceName: json.RawMessage(configJson)},
			},
			{
				"name":        "broker-catalog",
				"tags":        []string{"broker-catalog"},
				"credentials": map[string]any{"broker-catalog": json.RawMessage(catalogBytes)},
			},
		},
		"autoscaler": []map[string]any{
			{
				"name": "some-service",
				"credentials": map[string]any{
					"uri": dbURL,
				},
				"syslog_drain_url": "",
				"tags":             []string{"policy_db", "binding_db", "postgres"},
			},
		},
	}

	vcapServicesJson, err := json.Marshal(vcapServices)
	Expect(err).NotTo(HaveOccurred())

	return string(vcapServicesJson)
}
