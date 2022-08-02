package cf_test

import (
	"crypto/tls"
	"fmt"
	"net/http"
	"os"
	"testing"

	"code.cloudfoundry.org/app-autoscaler/src/autoscaler/cf"
	"code.cloudfoundry.org/clock"
	"code.cloudfoundry.org/lager"
	"github.com/stretchr/testify/assert"
)

/* TestClient_GetAppProcesses
 * This test is for checking the integration with the cf API using the client library.
 * Currently you will need to supply the CLIENT_ID and CLIENT_SECRET in
 * environment variables to be able to run the test. It will also require you to deploy the test app.
 * To deploy the test app run the deploy script in testdata/test_app/deploy.sh
 */
func TestClient_GetAppProcesses(t *testing.T) {
	t.Skip("Only for testing integrations with cf api")
	//TODO We should consider moving this to the acceptance tests.
	logger := lager.NewLogger("cf")
	logger.RegisterSink(lager.NewWriterSink(os.Stdout, lager.DEBUG))
	conf := &cf.Config{}
	conf.API = "https://api.autoscaler.ci.cloudfoundry.org"
	conf.MaxRetries = 3
	conf.ClientID = getEnv("CLIENT_ID")
	conf.Secret = getEnv("CLIENT_SECRET")
	conf.SkipSSLValidation = true
	conf.PerPage = 2
	client := cf.NewCFClient(conf, logger, clock.NewClock())
	err := client.Login()
	assert.Nil(t, err)
	// #nosec G402
	http.DefaultTransport.(*http.Transport).TLSClientConfig = &tls.Config{InsecureSkipVerify: true}
	resp, err := http.Get("https://test_app.autoscaler.ci.cloudfoundry.org")
	if err != nil {
		panic("Please deploy the app in testdata/test_app using deploy.sh")
	}
	assert.Equal(t, resp.StatusCode, 200)

	processes, err := client.GetAppProcesses("67ae15a0-2d76-4b04-99d4-807bcf95b9f5")
	assert.Nil(t, err)
	assert.Equal(t, processes.GetInstances(), 7)
}

//nolint:unused
func getEnv(key string) string {
	envVar := os.Getenv(key)
	if envVar == "" {
		panic(fmt.Sprintf("To run this test please set env var %s", key))
	}
	return envVar
}
