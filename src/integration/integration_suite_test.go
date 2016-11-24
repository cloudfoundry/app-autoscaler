package integration_test

import (
	"autoscaler/db"
	. "integration"

	"database/sql"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"path"
	"strconv"
	"strings"
	"testing"
	"time"

	"code.cloudfoundry.org/cfhttp"
	"code.cloudfoundry.org/lager"
	_ "github.com/lib/pq"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/ghttp"
	"github.com/tedsuo/ifrit"
	"github.com/tedsuo/ifrit/ginkgomon"
	"github.com/tedsuo/ifrit/grouper"
)

var (
	components Components
	tmpDir     string

	plumbing              ifrit.Process
	logger                lager.Logger
	serviceBrokerConfPath string
	apiServerConfPath     string
	brokerUserName        string = "username"
	brokerPassword        string = "password"
	brokerAuth            string
	dbUrl                 string

	dbHelper           *sql.DB
	scheduler          *ghttp.Server
	httpClient         *http.Client
	policyTemplate     string                   = `{ "app_guid": "%s", "parameters": %s }`
	httpRequestTimeout time.Duration            = 1000 * time.Millisecond
	processMap         map[string]ifrit.Process = map[string]ifrit.Process{}
)

var _ = SynchronizedBeforeSuite(func() []byte {
	payload, err := json.Marshal(Components{
		Executables: CompileTestedExecutables(),
	})
	Expect(err).NotTo(HaveOccurred())

	dbUrl = os.Getenv("DBURL")
	if dbUrl == "" {
		Fail("environment variable $DBURL is not set")
	}

	dbHelper, err = sql.Open(db.PostgresDriverName, dbUrl)
	Expect(err).NotTo(HaveOccurred())

	clearDatabase()

	return payload
}, func(encodedBuiltArtifacts []byte) {
	err := json.Unmarshal(encodedBuiltArtifacts, &components)
	Expect(err).NotTo(HaveOccurred())
	components.Ports = PreparePorts()

	tmpDir, err = ioutil.TempDir("", "autoscaler")
	Expect(err).NotTo(HaveOccurred())

	dbUrl = os.Getenv("DBURL")
	dbHelper, err = sql.Open(db.PostgresDriverName, dbUrl)
	Expect(err).NotTo(HaveOccurred())

	scheduler = ghttp.NewServer()
	apiServerConfPath = prepareApiServerConfig(components.Ports[APIServer], dbUrl, scheduler.URL())
	serviceBrokerConfPath = prepareServiceBrokerConfig(components.Ports[ServiceBroker], brokerUserName, brokerPassword, dbUrl, fmt.Sprintf("http://127.0.0.1:%d", components.Ports[APIServer]))

})

var _ = SynchronizedAfterSuite(func() {
	scheduler.Close()

	if len(tmpDir) > 0 {
		os.RemoveAll(tmpDir)
	}
}, func() {
})

var _ = BeforeEach(func() {
	httpClient = cfhttp.NewCustomTimeoutClient(httpRequestTimeout)
	startApiServer()
	startServiceBroker()
	logger = lager.NewLogger("test")
	logger.RegisterSink(lager.NewWriterSink(GinkgoWriter, lager.DEBUG))
})

var _ = AfterEach(func() {
	stopAll()
})

func TestIntegration(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Integration Suite")
}

func CompileTestedExecutables() Executables {
	builtExecutables := Executables{}
	rootDir := os.Getenv("GOPATH")

	builtExecutables[APIServer] = path.Join(rootDir, "api/index.js")
	builtExecutables[ServiceBroker] = path.Join(rootDir, "servicebroker/lib/index.js")

	return builtExecutables
}

func PreparePorts() Ports {
	return Ports{
		APIServer:     10000 + GinkgoParallelNode(),
		ServiceBroker: 11000 + GinkgoParallelNode(),
	}
}

func startApiServer() {
	processMap[APIServer] = ginkgomon.Invoke(grouper.NewOrdered(os.Interrupt, grouper.Members{
		{APIServer, components.ApiServer(apiServerConfPath)},
	}))
}

func startServiceBroker() {
	processMap[ServiceBroker] = ginkgomon.Invoke(grouper.NewOrdered(os.Interrupt, grouper.Members{
		{ServiceBroker, components.ServiceBroker(serviceBrokerConfPath)},
	}))
}

func stopApiServer() {
	ginkgomon.Interrupt(processMap[APIServer], 5*time.Second)
}

func stopAll() {
	for _, process := range processMap {
		if process == nil {
			continue
		}
		ginkgomon.Interrupt(process, 5*time.Second)
	}
}

func prepareServiceBrokerConfig(port int, username string, password string, dbUri string, apiServerUri string) string {
	brokerConfig := ServiceBrokerConfig{
		Port:     port,
		Username: username,
		Password: password,
		DB: DBConfig{
			URI:            dbUri,
			MinConnections: 1,
			MaxConnections: 10,
			IdleTimeout:    1000,
		},
		APIServerUri:       apiServerUri,
		HttpRequestTimeout: int(httpRequestTimeout / time.Millisecond),
	}

	cfgFile, err := ioutil.TempFile(tmpDir, ServiceBroker)
	w := json.NewEncoder(cfgFile)
	err = w.Encode(brokerConfig)
	Expect(err).NotTo(HaveOccurred())
	cfgFile.Close()
	return cfgFile.Name()
}

func prepareApiServerConfig(port int, dbUri string, schedulerUri string) string {
	apiConfig := APIServerConfig{
		Port: port,

		DB: DBConfig{
			URI:            dbUri,
			MinConnections: 1,
			MaxConnections: 10,
			IdleTimeout:    1000,
		},

		SchedulerUri: schedulerUri,
	}

	cfgFile, err := ioutil.TempFile(tmpDir, APIServer)
	w := json.NewEncoder(cfgFile)
	err = w.Encode(apiConfig)
	Expect(err).NotTo(HaveOccurred())
	cfgFile.Close()
	return cfgFile.Name()
}

func getRandomId() string {
	return strconv.FormatInt(time.Now().UnixNano(), 10)
}

func provisionServiceInstance(serviceInstanceId string, orgId string, spaceId string) (*http.Response, error) {
	req, err := http.NewRequest("PUT", fmt.Sprintf("http://127.0.0.1:%d/v2/service_instances/%s", components.Ports[ServiceBroker], serviceInstanceId), strings.NewReader(fmt.Sprintf(`{"organization_guid":"%s","space_guid":"%s"}`, orgId, spaceId)))
	Expect(err).NotTo(HaveOccurred())
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Basic "+brokerAuth)
	return httpClient.Do(req)
}

func deprovisionServiceInstance(serviceInstanceId string) (*http.Response, error) {
	req, err := http.NewRequest("DELETE", fmt.Sprintf("http://127.0.0.1:%d/v2/service_instances/%s", components.Ports[ServiceBroker], serviceInstanceId), strings.NewReader(""))
	Expect(err).NotTo(HaveOccurred())
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Basic "+brokerAuth)
	return httpClient.Do(req)
}

func bindService(bindingId string, appId string, serviceInstanceId string, policyStr string) (*http.Response, error) {
	policy := fmt.Sprintf(policyTemplate, appId, policyStr)
	req, err := http.NewRequest("PUT", fmt.Sprintf("http://127.0.0.1:%d/v2/service_instances/%s/service_bindings/%s", components.Ports[ServiceBroker], serviceInstanceId, bindingId), strings.NewReader(policy))
	Expect(err).NotTo(HaveOccurred())
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Basic "+brokerAuth)
	return httpClient.Do(req)
}

func unbindService(bindingId string, appId string, serviceInstanceId string) (*http.Response, error) {
	req, err := http.NewRequest("DELETE", fmt.Sprintf("http://127.0.0.1:%d/v2/service_instances/%s/service_bindings/%s", components.Ports[ServiceBroker], serviceInstanceId, bindingId), strings.NewReader(fmt.Sprintf(`{"app_guid":"%s"}`, appId)))
	Expect(err).NotTo(HaveOccurred())
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Basic "+brokerAuth)
	return httpClient.Do(req)
}

func detachPolicy(appId string) (*http.Response, error) {
	req, err := http.NewRequest("DELETE", fmt.Sprintf("http://127.0.0.1:%d/v1/policies/%s", components.Ports[APIServer], appId), strings.NewReader(""))
	Expect(err).NotTo(HaveOccurred())
	req.Header.Set("Content-Type", "application/json")
	return httpClient.Do(req)
}

func attachPolicy(appId string, policyStr string) (*http.Response, error) {
	req, err := http.NewRequest("PUT", fmt.Sprintf("http://127.0.0.1:%d/v1/policies/%s", components.Ports[APIServer], appId), strings.NewReader(policyStr))
	Expect(err).NotTo(HaveOccurred())
	req.Header.Set("Content-Type", "application/json")
	return httpClient.Do(req)
}

func clearDatabase() {
	_, err := dbHelper.Exec("DELETE FROM policy_json")
	Expect(err).NotTo(HaveOccurred())

	_, err = dbHelper.Exec("DELETE FROM binding")
	Expect(err).NotTo(HaveOccurred())

	_, err = dbHelper.Exec("DELETE FROM service_instance")
	Expect(err).NotTo(HaveOccurred())
}
