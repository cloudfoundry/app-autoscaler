package integration_test

import (
	"autoscaler/db"
	"autoscaler/integration"
	"autoscaler/integration/helper"
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
	"os"
	"path"
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
	nodeDbUri             string = "postgres://postgres:123@127.0.0.1:5432/autoscaler"
	golangDbUri           string = "postgres://postgres:123@127.0.0.1:5432/autoscaler?sslmode=disable"
	testAppId             string = "testAppId"
	testServiceInstanceId string = "testServiceInstanceId"
	testOrgId             string = "testOrgId"
	testSpaceId           string = "testSpaceId"
	testBindingId         string = "testBindingId"
	scheduler             *ghttp.Server
)

var _ = SynchronizedBeforeSuite(func() []byte {

	payload, err := json.Marshal(integration.Components{
		Executables: CompileTestedExecutables(),
		Ports:       PreparePorts(),
	})
	Expect(err).NotTo(HaveOccurred())
	return payload
}, func(encodedBuiltArtifacts []byte) {
	err := json.Unmarshal(encodedBuiltArtifacts, &components)
	Expect(err).NotTo(HaveOccurred())

	var e error

	dbUrl := golangDbUri
	if dbUrl == "" {
		Fail("environment variable $DBURL is not set")
	}

	dbHelper, e = sql.Open(db.PostgresDriverName, dbUrl)
	if e != nil {
		Fail("can not connect database: " + e.Error())
	}
})

var _ = AfterSuite(func() {
	if dbHelper != nil {
		dbHelper.Close()
	}

})

var _ = BeforeEach(func() {
	scheduler = ghttp.NewServer()
	apiServerConfPath = prepareApiServerConfig(components.Ports["apiServer"], nodeDbUri, scheduler.URL())
	serviceBrokerConfPath = prepareServiceBrokerConfig(components.Ports["serviceBroker"], brokerUserName, brokerPassword, nodeDbUri, fmt.Sprintf("http://127.0.0.1:%d", components.Ports["apiServer"]))
	plumbing = ginkgomon.Invoke(grouper.NewParallel(os.Kill, grouper.Members{
		{"apiServer", components.ApiServer(apiServerConfPath)},
		{"serviceBroker", components.ServiceBroker(serviceBrokerConfPath)},
	}))
	logger = lager.NewLogger("test")
	logger.RegisterSink(lager.NewWriterSink(GinkgoWriter, lager.DEBUG))
})

var _ = AfterEach(func() {
	helper.StopProcesses(plumbing)

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
    "apiServerUri": "%s"
}
  `
	settingJonsStr := fmt.Sprintf(settingStrTemplate, port, username, password, dbUri, apiServerUri)
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
func addServiceInstance(serviceInstanceId string, orgId string, spaceId string) {
	_, err := dbHelper.Exec("INSERT INTO service_instance(service_instance_id,org_id,space_id) VALUES ($1,$2,$3)", serviceInstanceId, orgId, spaceId)
	Expect(err).NotTo(HaveOccurred())
}
func addBinding(bindingId string, appId string, serviceInstanceId string, createAt time.Time) {
	_, err := dbHelper.Exec("INSERT INTO binding(binding_id,app_id,service_instance_id,created_at) VALUES ($1,$2,$3,$4)", bindingId, appId, serviceInstanceId, createAt)
	Expect(err).NotTo(HaveOccurred())
}
func addPolicy(appId string, policyJson string, updateAt time.Time) {
	_, err := dbHelper.Exec("INSERT INTO policy_json(app_id,policy_json,updated_at) VALUES ($1,$2,$3)", appId, policyJson, updateAt)
	Expect(err).NotTo(HaveOccurred())
}
func cleanPolicyJsonTable() {
	_, err := dbHelper.Exec("DELETE FROM policy_json")
	Expect(err).NotTo(HaveOccurred())
}
func getNumberOfPolicyJson() int {
	var num int
	err := dbHelper.QueryRow("SELECT COUNT(*) FROM policy_json").Scan(&num)
	Expect(err).NotTo(HaveOccurred())
	return num
}
func getNumberOfBinding() int {
	var num int
	err := dbHelper.QueryRow("SELECT COUNT(*) FROM binding").Scan(&num)
	Expect(err).NotTo(HaveOccurred())
	return num
}
func getNumberOfServiceInstance() int {
	var num int
	err := dbHelper.QueryRow("SELECT COUNT(*) FROM service_instance").Scan(&num)
	Expect(err).NotTo(HaveOccurred())
	return num
}
func clearDatabase() {

	_, err := dbHelper.Exec("DELETE FROM policy_json")
	Expect(err).NotTo(HaveOccurred())

	_, err = dbHelper.Exec("DELETE FROM binding")
	Expect(err).NotTo(HaveOccurred())

	_, err = dbHelper.Exec("DELETE FROM service_instance")
	Expect(err).NotTo(HaveOccurred())

}
