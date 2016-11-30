package integration_test

import (
	"autoscaler/db"
	"bytes"
	. "integration"

	"database/sql"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
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
	serviceBrokerConfPath string
	apiServerConfPath     string
	schedulerConfPath     string
	brokerUserName        string = "username"
	brokerPassword        string = "password"
	brokerAuth            string
	dbUrl                 string

	dbHelper          *sql.DB
	fakeScheduler     *ghttp.Server
	fakeScalingEngine *ghttp.Server
	processMap        map[string]ifrit.Process = map[string]ifrit.Process{}
	schedulerProcess  ifrit.Process

	brokerApiHttpRequestTimeout    time.Duration = 1 * time.Second
	apiSchedulerHttpRequestTimeout time.Duration = 5 * time.Second

	httpClient *http.Client
	logger     lager.Logger
)

func TestIntegration(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Integration Suite")
}

var _ = SynchronizedBeforeSuite(func() []byte {
	components = Components{
		Ports:       PreparePorts(),
		Executables: CompileTestedExecutables(),
	}
	payload, err := json.Marshal(&components)
	Expect(err).NotTo(HaveOccurred())

	dbUrl = os.Getenv("DBURL")
	if dbUrl == "" {
		Fail("environment variable $DBURL is not set")
	}

	dbHelper, err = sql.Open(db.PostgresDriverName, dbUrl)
	Expect(err).NotTo(HaveOccurred())

	clearDatabase()

	tmpDir, err = ioutil.TempDir("", "autoscaler")
	Expect(err).NotTo(HaveOccurred())

	fakeScalingEngine = ghttp.NewServer()
	schedulerConfPath = prepareSchedulerConfig(dbUrl, fakeScalingEngine.URL())
	schedulerProcess = startScheduler()

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
})

var _ = SynchronizedAfterSuite(func() {
	if len(tmpDir) > 0 {
		os.RemoveAll(tmpDir)
	}
}, func() {
	ginkgomon.Kill(schedulerProcess)
	fakeScalingEngine.Close()
})

var _ = BeforeEach(func() {
	httpClient = cfhttp.NewClient()
	logger = lager.NewLogger("test")
	logger.RegisterSink(lager.NewWriterSink(GinkgoWriter, lager.DEBUG))
})

func CompileTestedExecutables() Executables {
	builtExecutables := Executables{}
	rootDir := os.Getenv("GOPATH")

	builtExecutables[APIServer] = path.Join(rootDir, "api/index.js")
	builtExecutables[ServiceBroker] = path.Join(rootDir, "servicebroker/lib/index.js")
	builtExecutables[Scheduler] = path.Join(rootDir, "scheduler/target/scheduler-1.0-SNAPSHOT.war")

	return builtExecutables
}

func PreparePorts() Ports {
	return Ports{
		APIServer:     10000 + GinkgoParallelNode(),
		ServiceBroker: 11000 + GinkgoParallelNode(),
		Scheduler:     12000,
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
func startScheduler() ifrit.Process {
	return ginkgomon.Invoke(components.Scheduler(schedulerConfPath))
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
		HttpRequestTimeout: int(brokerApiHttpRequestTimeout / time.Millisecond),
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

func prepareSchedulerConfig(dbUri string, scalingEngineUri string) string {
	dbUrl, _ := url.Parse(dbUri)
	scheme := dbUrl.Scheme
	host := dbUrl.Host
	path := dbUrl.Path
	userInfo := dbUrl.User
	userName := userInfo.Username()
	password, _ := userInfo.Password()
	jdbcDBUri := fmt.Sprintf("jdbc:%s://%s%s", scheme, host, path)
	settingStrTemplate := `
#datasource for application and quartz
spring.datasource.driverClassName=org.postgresql.Driver
spring.datasource.url=%s
spring.datasource.username=%s
spring.datasource.password=%s
#quartz job
scalingenginejob.reschedule.interval.millisecond=10000
scalingenginejob.reschedule.maxcount=6
scalingengine.notification.reschedule.maxcount=3
# scaling engine url
autoscaler.scalingengine.url=%s
  `
	settingJonsStr := fmt.Sprintf(settingStrTemplate, jdbcDBUri, userName, password, scalingEngineUri)
	cfgFile, err := ioutil.TempFile(tmpDir, Scheduler)
	Expect(err).NotTo(HaveOccurred())
	ioutil.WriteFile(cfgFile.Name(), []byte(settingJonsStr), 0777)
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

func bindService(bindingId string, appId string, serviceInstanceId string, policy []byte) (*http.Response, error) {
	rawParameters := json.RawMessage(policy)
	bindBody := map[string]interface{}{
		"app_guid":   appId,
		"parameters": &rawParameters,
	}
	body, err := json.Marshal(bindBody)
	req, err := http.NewRequest("PUT", fmt.Sprintf("http://127.0.0.1:%d/v2/service_instances/%s/service_bindings/%s", components.Ports[ServiceBroker], serviceInstanceId, bindingId), bytes.NewReader(body))
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

func getPolicy(appId string) (*http.Response, error) {
	req, err := http.NewRequest("GET", fmt.Sprintf("http://127.0.0.1:%d/v1/policies/%s", components.Ports[APIServer], appId), nil)
	Expect(err).NotTo(HaveOccurred())
	return httpClient.Do(req)
}

func detachPolicy(appId string) (*http.Response, error) {
	req, err := http.NewRequest("DELETE", fmt.Sprintf("http://127.0.0.1:%d/v1/policies/%s", components.Ports[APIServer], appId), strings.NewReader(""))
	Expect(err).NotTo(HaveOccurred())
	req.Header.Set("Content-Type", "application/json")
	return httpClient.Do(req)
}

func attachPolicy(appId string, policy []byte) (*http.Response, error) {
	req, err := http.NewRequest("PUT", fmt.Sprintf("http://127.0.0.1:%d/v1/policies/%s", components.Ports[APIServer], appId), bytes.NewReader(policy))
	Expect(err).NotTo(HaveOccurred())
	req.Header.Set("Content-Type", "application/json")
	return httpClient.Do(req)
}

func getSchedules(appId string) (*http.Response, error) {
	req, err := http.NewRequest("GET", fmt.Sprintf("http://127.0.0.1:%d/v2/schedules/%s", components.Ports["scheduler"], appId), strings.NewReader(""))
	Expect(err).NotTo(HaveOccurred())
	req.Header.Set("Content-Type", "application/json")
	return httpClient.Do(req)
}
func readPolicyFromFile(filename string) []byte {
	content, err := ioutil.ReadFile(filename)
	Expect(err).NotTo(HaveOccurred())
	return content
}
func clearDatabase() {
	_, err := dbHelper.Exec("DELETE FROM policy_json")
	Expect(err).NotTo(HaveOccurred())

	_, err = dbHelper.Exec("DELETE FROM binding")
	Expect(err).NotTo(HaveOccurred())

	_, err = dbHelper.Exec("DELETE FROM service_instance")
	Expect(err).NotTo(HaveOccurred())

	_, err = dbHelper.Exec("DELETE FROM app_scaling_recurring_schedule")
	Expect(err).NotTo(HaveOccurred())

	_, err = dbHelper.Exec("DELETE FROM app_scaling_specific_date_schedule")
	Expect(err).NotTo(HaveOccurred())
}

type GetResponse func(id string) (*http.Response, error)

func checkResponseContent(getResponse GetResponse, id string, expectHttpStatus int, expectResponseMap map[string]interface{}) {
	resp, err := getResponse(id)
	Expect(err).NotTo(HaveOccurred())
	Expect(resp.StatusCode).To(Equal(expectHttpStatus))
	var actual map[string]interface{}
	err = json.NewDecoder(resp.Body).Decode(&actual)
	Expect(err).NotTo(HaveOccurred())
	Expect(actual).To(Equal(expectResponseMap))
	resp.Body.Close()
}

func checkSchedule(getResponse GetResponse, id string, expectHttpStatus int, expectResponseMap map[string]int) {
	resp, err := getResponse(id)
	Expect(err).NotTo(HaveOccurred())
	Expect(resp.StatusCode).To(Equal(expectHttpStatus))
	var actual map[string]interface{}
	err = json.NewDecoder(resp.Body).Decode(&actual)
	Expect(err).NotTo(HaveOccurred())
	var schedules map[string]interface{} = actual["schedules"].(map[string]interface{})
	var recurring []interface{} = schedules["recurring_schedule"].([]interface{})
	var specificDate []interface{} = schedules["specific_date"].([]interface{})
	Expect(len(specificDate)).To(Equal(expectResponseMap["specific_date"]))
	Expect(len(recurring)).To(Equal(expectResponseMap["recurring_schedule"]))
	resp.Body.Close()
}
