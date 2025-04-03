package testhelpers

import (
	"net/http"
	"os"
	"path/filepath"
	"strings"
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
	splitPath := strings.Split(dir, string(os.PathSeparator))
	certPath := "/"
	for _, path := range splitPath {
		if path == "autoscaler" {
			break
		}
		certPath = filepath.Join(certPath, path)
	}
	return filepath.Join(certPath, "../test-certs")
}
