package testhelpers

import (
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"time"

	"code.cloudfoundry.org/cfhttp/v2"
	. "code.cloudfoundry.org/tlsconfig"
)

func NewApiClient() *http.Client {
	return CreateClientFor("api")
}

func NewPublicApiClient() *http.Client {
	return CreateClientFor("api_public")
}

func NewEventGeneratorClient() *http.Client {
	return CreateClientFor("eventgenerator")
}

func NewServiceBrokerClient() *http.Client {
	return CreateClientFor("servicebroker")
}
func NewSchedulerClient() *http.Client {
	return CreateClientFor("scheduler")
}

func NewScalingEngineClient() *http.Client {
	return CreateClientFor("scalingengine")
}

func CreateClientFor(name string) *http.Client {
	certFolder := TestCertFolder()
	return CreateClient(filepath.Join(certFolder, name+".crt"),
		filepath.Join(certFolder, name+".key"),
		filepath.Join(certFolder, "autoscaler-ca.crt"))
}

func CreateClient(certFileName, keyFileName, caCertFileName string) *http.Client {
	clientTls, err := Build(
		WithInternalServiceDefaults(),
		WithIdentityFromFile(certFileName, keyFileName),
	).Client(WithAuthorityFromFile(caCertFileName))
	FailOnError("Failed to setup tls config", err)
	return cfhttp.NewClient(cfhttp.WithTLSConfig(clientTls), cfhttp.WithRequestTimeout(10*time.Second))
}

func TestCertFolder() string {
	dir, err := os.Getwd()
	FailOnError("failed getting working directory", err)

	// Try to find test-certs by walking up the directory tree
	currentDir := dir
	for {
		testCertsPath := filepath.Join(currentDir, "test-certs")
		if _, err := os.Stat(testCertsPath); err == nil {
			return testCertsPath
		}

		// Move up one directory
		parentDir := filepath.Dir(currentDir)
		if parentDir == currentDir {
			// Reached the root without finding test-certs
			break
		}
		currentDir = parentDir
	}

	FailOnError("failed to find test-certs directory", fmt.Errorf("searched from: %s", dir))
	return ""
}
