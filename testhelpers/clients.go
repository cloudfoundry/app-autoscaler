package testhelpers

//nolint:stylecheck
import (
	"net/http"
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

func NewMetricServerClient() *http.Client {
	return CreateClientFor("metricserver")
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

func CreateClientFor(name string) *http.Client {
	return CreateClient(filepath.Join(testCertDir, name+".crt"),
		filepath.Join(testCertDir, name+".key"),
		filepath.Join(testCertDir, "autoscaler-ca.crt"))
}

func CreateClient(certFileName, keyFileName, caCertFileName string) *http.Client {
	tlsConf, err := Build(WithIdentityFromFile(certFileName, keyFileName)).Client(WithAuthorityFromFile(caCertFileName))
	FailOnError("Failed to setup tls config", err)
	return cfhttp.NewClient(cfhttp.WithTLSConfig(tlsConf), cfhttp.WithRequestTimeout(10*time.Second))
}
