package integration_test

import (
	"autoscaler/db"
	"autoscaler/integration"
	"code.cloudfoundry.org/cfhttp"
	"code.cloudfoundry.org/lager"
	"database/sql"
	"encoding/json"
	"fmt"
	_ "github.com/lib/pq"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/ghttp"
	"github.com/tedsuo/ifrit"
	"github.com/tedsuo/ifrit/ginkgomon"
	"github.com/tedsuo/ifrit/grouper"
	"gopkg.in/yaml.v2"
	"io/ioutil"
	"net/http"
	"os"
	"path"
	"strconv"
	"strings"
	"testing"
	"time"
)

var (
	components            integration.Components
	plumbing              ifrit.Process
	logger                lager.Logger
	serviceBrokerConfPath string
	apiServerConfPath     string
	dbHelper              *sql.DB
	brokerUserName        string = "username"
	brokerPassword        string = "password"
	brokerAuth            string
	dbUrl                 string
	scheduler             *ghttp.Server
	httpClient            *http.Client
	policyTemplate        string                   = `{ "app_guid": "%s", "parameters": %s }`
	httpRequestTimeout    time.Duration            = 1000 * time.Millisecond
	processMap            map[string]ifrit.Process = map[string]ifrit.Process{}
)

var _ = SynchronizedBeforeSuite(func() []byte {

	payload, err := json.Marshal(integration.Components{
		Executables: CompileTestedExecutables(),
		Ports:       PreparePorts(),
	})
	Expect(err).NotTo(HaveOccurred())

	var e error
	dbUrl = os.Getenv("DBURL")
	if dbUrl == "" {
		Fail("environment variable $DBURL is not set")
	}

	dbHelper, e = sql.Open(db.PostgresDriverName, dbUrl)
	if e != nil {
		Fail("can not connect database: " + e.Error())
	}
	clearDatabase()

	return payload
}, func(encodedBuiltArtifacts []byte) {
	err := json.Unmarshal(encodedBuiltArtifacts, &components)
	Expect(err).NotTo(HaveOccurred())
})

var _ = SynchronizedAfterSuite(func() {

}, func() {
	if dbHelper != nil {
		dbHelper.Close()
	}
})
var _ = BeforeEach(func() {
	httpClient = cfhttp.NewCustomTimeoutClient(httpRequestTimeout)
	scheduler = ghttp.NewServer()
	apiServerConfPath = prepareApiServerConfig(components.Ports["apiServer"], dbUrl, scheduler.URL())
	serviceBrokerConfPath = prepareServiceBrokerConfig(components.Ports["serviceBroker"], brokerUserName, brokerPassword, dbUrl, fmt.Sprintf("http://127.0.0.1:%d", components.Ports["apiServer"]))
	startApiServer()
	startServiceBroker()
	logger = lager.NewLogger("test")
	logger.RegisterSink(lager.NewWriterSink(GinkgoWriter, lager.DEBUG))
})

var _ = AfterEach(func() {
	stopAll()
	scheduler.Close()

})

func TestIntegration(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Integration Suite")
}

func CompileTestedExecutables() integration.Executables {

	builtExecutables := integration.Executables{}

	builtExecutables["apiServer"] = path.Join(os.Getenv("GOPATH"), "api/index.js")

	builtExecutables["serviceBroker"] = path.Join(os.Getenv("GOPATH"), "servicebroker/lib/index.js")

	return builtExecutables
}
func PreparePorts() integration.Ports {
	ports := integration.Ports{

		"apiServer":     10000 + GinkgoParallelNode(),
		"serviceBroker": 11000 + GinkgoParallelNode(),
	}
	return ports
}
func startApiServer() {
	processMap["apiServer"] = ginkgomon.Invoke(grouper.NewParallel(os.Kill, grouper.Members{
		{"apiServer", components.ApiServer(apiServerConfPath)},
	}))
}
func startServiceBroker() {
	processMap["serviceBroker"] = ginkgomon.Invoke(grouper.NewParallel(os.Kill, grouper.Members{
		{"serviceBroker", components.ServiceBroker(serviceBrokerConfPath)},
	}))
}
func stopApiServer() {
	ginkgomon.Interrupt(processMap["apiServer"], 5*time.Second)
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
	settingStrTemplate := `
{
    "port": %d,
    "username": "%s",
    "password": "%s",
    "db": {
        "maxConnections": 10,
        "minConnections": 0,
        "idleTimeout": 1000,
        "uri": "%s"
    },
    "apiServerUri": "%s",
    "httpRequestTimeout" : %d
}
  `
	settingJonsStr := fmt.Sprintf(settingStrTemplate, port, username, password, dbUri, apiServerUri, httpRequestTimeout)
	configFile := writeStringConfig(settingJonsStr)
	return configFile.Name()
}
func prepareApiServerConfig(port int, dbUri string, schedulerUri string) string {
	settingStrTemplate := `
{
  "port": %d,
  "db": {
    "maxConnections": 10,
    "minConnections": 0,
    "idleTimeout": 1000,
    "uri": "%s"
  },
  "schedulerUri": "%s"
}
  `
	settingJonsStr := fmt.Sprintf(settingStrTemplate, port, dbUri, schedulerUri)
	configFile := writeStringConfig(settingJonsStr)
	return configFile.Name()
}

func writeYmlConfig(c interface{}) *os.File {
	cfg, err := ioutil.TempFile("", "autoscaler")
	Expect(err).NotTo(HaveOccurred())
	defer cfg.Close()
	configBytes, err1 := yaml.Marshal(c)
	ioutil.WriteFile(cfg.Name(), configBytes, 0777)
	Expect(err1).NotTo(HaveOccurred())
	return cfg

}
func writeStringConfig(c string) *os.File {
	cfg, err := ioutil.TempFile("", "autoscaler")
	Expect(err).NotTo(HaveOccurred())
	defer cfg.Close()
	configBytes := ([]byte)(c)
	ioutil.WriteFile(cfg.Name(), configBytes, 0777)
	return cfg

}
func getRandomId() string {
	return strconv.FormatInt(time.Now().UnixNano(), 10)
}
func provisionServiceInstance(serviceInstanceId string, orgId string, spaceId string) (*http.Response, error) {
	req, err := http.NewRequest("PUT", fmt.Sprintf("http://127.0.0.1:%d/v2/service_instances/%s", components.Ports["serviceBroker"], serviceInstanceId), strings.NewReader(fmt.Sprintf(`{"organization_guid":"%s","space_guid":"%s"}`, orgId, spaceId)))
	Expect(err).NotTo(HaveOccurred())
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Basic "+brokerAuth)
	return httpClient.Do(req)
}
func deprovisionServiceInstance(serviceInstanceId string) (*http.Response, error) {
	req, err := http.NewRequest("DELETE", fmt.Sprintf("http://127.0.0.1:%d/v2/service_instances/%s", components.Ports["serviceBroker"], serviceInstanceId), strings.NewReader(""))
	Expect(err).NotTo(HaveOccurred())
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Basic "+brokerAuth)
	return httpClient.Do(req)
}
func bindService(bindingId string, appId string, serviceInstanceId string, policyStr string) (*http.Response, error) {
	policy := fmt.Sprintf(policyTemplate, appId, policyStr)
	req, err := http.NewRequest("PUT", fmt.Sprintf("http://127.0.0.1:%d/v2/service_instances/%s/service_bindings/%s", components.Ports["serviceBroker"], serviceInstanceId, bindingId), strings.NewReader(policy))
	Expect(err).NotTo(HaveOccurred())
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Basic "+brokerAuth)
	return httpClient.Do(req)
}
func unbindService(bindingId string, appId string, serviceInstanceId string) (*http.Response, error) {
	req, err := http.NewRequest("DELETE", fmt.Sprintf("http://127.0.0.1:%d/v2/service_instances/%s/service_bindings/%s", components.Ports["serviceBroker"], serviceInstanceId, bindingId), strings.NewReader(fmt.Sprintf(`{"app_guid":"%s"}`, appId)))
	Expect(err).NotTo(HaveOccurred())
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Basic "+brokerAuth)
	return httpClient.Do(req)
}

func detachPolicy(appId string) (*http.Response, error) {
	req, err := http.NewRequest("DELETE", fmt.Sprintf("http://127.0.0.1:%d/v1/policies/%s", components.Ports["apiServer"], appId), strings.NewReader(""))
	Expect(err).NotTo(HaveOccurred())
	req.Header.Set("Content-Type", "application/json")
	return httpClient.Do(req)
}
func attachPolicy(appId string, policyStr string) (*http.Response, error) {
	req, err := http.NewRequest("PUT", fmt.Sprintf("http://127.0.0.1:%d/v1/policies/%s", components.Ports["apiServer"], appId), strings.NewReader(policyStr))
	Expect(err).NotTo(HaveOccurred())
	req.Header.Set("Content-Type", "application/json")
	return httpClient.Do(req)
}

func getNumberOfPolicyJson(appId string) int {
	var num int
	err := dbHelper.QueryRow("SELECT COUNT(*) FROM policy_json WHERE app_id=$1", appId).Scan(&num)
	Expect(err).NotTo(HaveOccurred())
	return num
}
func getNumberOfBinding(bindingId string, appId string, serviceInstanceId string) int {
	var num int
	err := dbHelper.QueryRow("SELECT COUNT(*) FROM binding WHERE binding_id=$1 AND app_id=$2 AND service_instance_id=$3 ", bindingId, appId, serviceInstanceId).Scan(&num)
	Expect(err).NotTo(HaveOccurred())
	return num
}
func getNumberOfServiceInstance(serviceInstanceId string, orgId string, spaceId string) int {
	var num int
	err := dbHelper.QueryRow("SELECT COUNT(*) FROM service_instance WHERE service_instance_id=$1 AND org_id=$2 AND space_id=$3", serviceInstanceId, orgId, spaceId).Scan(&num)
	Expect(err).NotTo(HaveOccurred())
	return num
}
func removeBinding(bindingId string) {
	_, err := dbHelper.Exec("DELETE FROM binding WHERE binding_id=$1", bindingId)
	Expect(err).NotTo(HaveOccurred())
}
func removePolicy(appId string) {
	_, err := dbHelper.Exec("DELETE FROM policy_json WHERE app_id=$1", appId)
	Expect(err).NotTo(HaveOccurred())
}
func clearDatabase() {

	_, err := dbHelper.Exec("DELETE FROM policy_json")
	Expect(err).NotTo(HaveOccurred())

	_, err = dbHelper.Exec("DELETE FROM binding")
	Expect(err).NotTo(HaveOccurred())

	_, err = dbHelper.Exec("DELETE FROM service_instance")
	Expect(err).NotTo(HaveOccurred())

}
