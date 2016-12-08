package integration_test

import (
	"autoscaler/cf"
	"autoscaler/db"
	"autoscaler/models"
	"bytes"
	. "integration"

	egConfig "autoscaler/eventgenerator/config"
	mcConfig "autoscaler/metricscollector/config"
	seConfig "autoscaler/scalingengine/config"
	"code.cloudfoundry.org/cfhttp"
	"code.cloudfoundry.org/lager"
	"database/sql"
	"encoding/json"
	"fmt"
	"github.com/cloudfoundry/sonde-go/events"
	"github.com/gogo/protobuf/proto"
	_ "github.com/lib/pq"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/gexec"
	"github.com/onsi/gomega/ghttp"
	"github.com/tedsuo/ifrit"
	"github.com/tedsuo/ifrit/ginkgomon"
	"github.com/tedsuo/ifrit/grouper"
	"gopkg.in/yaml.v2"
	"io/ioutil"
	"log"
	"mime/multipart"
	"net/http"
	"net/url"
	"os"
	"path"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"testing"
	"time"
)

var (
	components               Components
	tmpDir                   string
	isTokenExpired           bool
	eLock                    *sync.Mutex
	plumbing                 ifrit.Process
	serviceBrokerConfPath    string
	apiServerConfPath        string
	schedulerConfPath        string
	metricsCollectorConfPath string
	eventGeneratorConfPath   string
	scalingEngineConfPath    string
	brokerUserName           string = "username"
	brokerPassword           string = "password"
	brokerAuth               string
	dbUrl                    string
	testCertDir              string = "../../test-certs"
	ccNOAAUAARegPath                = regexp.MustCompile(`^/apps/.*/containermetrics$`)
	dbHelper                 *sql.DB
	fakeScheduler            *ghttp.Server
	fakeScalingEngine        *ghttp.Server
	fakeCCNOAAUAA            *ghttp.Server
	processMap               map[string]ifrit.Process = map[string]ifrit.Process{}
	schedulerProcess         ifrit.Process

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
	var err error
	builtExecutables[APIServer] = path.Join(rootDir, "api/index.js")
	builtExecutables[ServiceBroker] = path.Join(rootDir, "servicebroker/lib/index.js")
	builtExecutables[Scheduler] = path.Join(rootDir, "scheduler/target/scheduler-1.0-SNAPSHOT.war")
	builtExecutables[EventGenerator], err = gexec.BuildIn(os.Getenv("GOPATH"), "autoscaler/eventgenerator/cmd/eventgenerator", "-race")
	Expect(err).NotTo(HaveOccurred())

	builtExecutables[MetricsCollector], err = gexec.BuildIn(os.Getenv("GOPATH"), "autoscaler/metricscollector/cmd/metricscollector", "-race")
	Expect(err).NotTo(HaveOccurred())

	builtExecutables[ScalingEngine], err = gexec.BuildIn(os.Getenv("GOPATH"), "autoscaler/scalingengine/cmd/scalingengine", "-race")
	Expect(err).NotTo(HaveOccurred())

	return builtExecutables
}

func PreparePorts() Ports {
	return Ports{
		APIServer:        10000 + GinkgoParallelNode(),
		ServiceBroker:    11000 + GinkgoParallelNode(),
		Scheduler:        12000,
		MetricsCollector: 13000 + GinkgoParallelNode(),
		EventGenerator:   14000 + GinkgoParallelNode(),
		ScalingEngine:    15000 + GinkgoParallelNode(),
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

func startMetricsCollector() {
	processMap[MetricsCollector] = ginkgomon.Invoke(grouper.NewOrdered(os.Interrupt, grouper.Members{
		{MetricsCollector, components.MetricsCollector(metricsCollectorConfPath)},
	}))
}
func startEventGenerator() {
	processMap[EventGenerator] = ginkgomon.Invoke(grouper.NewOrdered(os.Interrupt, grouper.Members{
		{EventGenerator, components.EventGenerator(eventGeneratorConfPath)},
	}))
}
func startScalingEngine() {
	processMap[ScalingEngine] = ginkgomon.Invoke(grouper.NewOrdered(os.Interrupt, grouper.Members{
		{ScalingEngine, components.ScalingEngine(scalingEngineConfPath)},
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

func prepareMetricsCollectorConfig(dbUri string, port int, ccNOAAUAAUrl string, cfGrantTypePassword string) string {
	cfg := mcConfig.Config{
		Cf: cf.CfConfig{
			Api:       ccNOAAUAAUrl,
			GrantType: cfGrantTypePassword,
			Username:  "admin",
			Password:  "admin",
		},
		Server: mcConfig.ServerConfig{
			Port: port,
			TLS: models.TLSCerts{
				KeyFile:    filepath.Join(testCertDir, "metricscollector.key"),
				CertFile:   filepath.Join(testCertDir, "metricscollector.crt"),
				CACertFile: filepath.Join(testCertDir, "autoscaler-ca.crt"),
			},
		},
		Logging: mcConfig.LoggingConfig{
			Level: "debug",
		},
		Db: mcConfig.DbConfig{
			InstanceMetricsDbUrl: dbUri,
			PolicyDbUrl:          dbUri,
		},
		Collector: mcConfig.CollectorConfig{
			PollInterval:    10,
			RefreshInterval: 30,
		},
	}

	return writeYmlConfig(tmpDir, MetricsCollector, &cfg)
}

func prepareEventGeneratorConfig(dbUri string, port int, metricsCollectorUrl string, scalingEngineUrl string) string {
	conf := &egConfig.Config{
		Server: egConfig.ServerConfig{
			Port: port,
		},
		Logging: egConfig.LoggingConfig{
			Level: "debug",
		},
		Aggregator: egConfig.AggregatorConfig{
			AggregatorExecuteInterval: 1 * time.Second,
			PolicyPollerInterval:      1 * time.Second,
			MetricPollerCount:         1,
			AppMonitorChannelSize:     1,
		},
		Evaluator: egConfig.EvaluatorConfig{
			EvaluationManagerInterval: 1 * time.Second,
			EvaluatorCount:            1,
			TriggerArrayChannelSize:   1,
		},
		DB: egConfig.DBConfig{
			PolicyDBUrl:    dbUri,
			AppMetricDBUrl: dbUri,
		},
		ScalingEngine: egConfig.ScalingEngineConfig{
			ScalingEngineUrl: scalingEngineUrl,
			TLSClientCerts: models.TLSCerts{
				KeyFile:    filepath.Join(testCertDir, "eventgenerator.key"),
				CertFile:   filepath.Join(testCertDir, "eventgenerator.crt"),
				CACertFile: filepath.Join(testCertDir, "autoscaler-ca.crt"),
			},
		},
		MetricCollector: egConfig.MetricCollectorConfig{
			MetricCollectorUrl: metricsCollectorUrl,
			TLSClientCerts: models.TLSCerts{
				KeyFile:    filepath.Join(testCertDir, "eventgenerator.key"),
				CertFile:   filepath.Join(testCertDir, "eventgenerator.crt"),
				CACertFile: filepath.Join(testCertDir, "autoscaler-ca.crt"),
			},
		},
	}
	return writeYmlConfig(tmpDir, EventGenerator, &conf)
}

func prepareScalingEngineConfig(dbUri string, port int, ccUAAUrl string, cfGrantTypePassword string) string {
	conf := seConfig.Config{
		Cf: cf.CfConfig{
			Api:       ccUAAUrl,
			GrantType: cfGrantTypePassword,
			Username:  "admin",
			Password:  "admin",
		},
		Server: seConfig.ServerConfig{
			Port: port,
			TLS: models.TLSCerts{
				KeyFile:    filepath.Join(testCertDir, "scalingengine.key"),
				CertFile:   filepath.Join(testCertDir, "scalingengine.crt"),
				CACertFile: filepath.Join(testCertDir, "autoscaler-ca.crt"),
			},
		},
		Logging: seConfig.LoggingConfig{
			Level: "debug",
		},
		Db: seConfig.DbConfig{
			PolicyDbUrl:        dbUri,
			ScalingEngineDbUrl: dbUri,
			SchedulerDbUrl:     dbUri,
		},
		Synchronizer: seConfig.SynchronizerConfig{
			ActiveScheduleSyncInterval: 10 * time.Minute,
		},
	}

	return writeYmlConfig(tmpDir, ScalingEngine, &conf)
}

func writeYmlConfig(dir string, componentName string, c interface{}) string {
	cfgFile, err := ioutil.TempFile(dir, componentName)
	Expect(err).NotTo(HaveOccurred())
	defer cfgFile.Close()
	configBytes, err := yaml.Marshal(c)
	ioutil.WriteFile(cfgFile.Name(), configBytes, 0777)
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

func startFakeCCNOAAUAA() {
	fakeCCNOAAUAA = ghttp.NewServer()
	fakeCCNOAAUAA.RouteToHandler("GET", "/v2/info", ghttp.RespondWithJSONEncoded(http.StatusOK,
		cf.Endpoints{
			AuthEndpoint:    fakeCCNOAAUAA.URL(),
			DopplerEndpoint: strings.Replace(fakeCCNOAAUAA.URL(), "http", "ws", 1),
		}))

	fakeCCNOAAUAA.RouteToHandler("POST", "/oauth/token", ghttp.RespondWithJSONEncoded(http.StatusOK,
		cf.Tokens{}))

	message1 := marshalMessage(createContainerMetric("an-app-id", 0, 3.0, 1024, 2048, 0))
	message2 := marshalMessage(createContainerMetric("an-app-id", 1, 4.0, 1024, 2048, 0))
	message3 := marshalMessage(createContainerMetric("an-app-id", 2, 5.0, 1024, 2048, 0))

	messages := map[string][][]byte{}
	messages["an-app-id"] = [][]byte{message1, message2, message3}

	eLock = &sync.Mutex{}
	fakeCCNOAAUAA.RouteToHandler("GET", ccNOAAUAARegPath,
		func(rw http.ResponseWriter, r *http.Request) {
			eLock.Lock()
			defer eLock.Unlock()
			if isTokenExpired {
				isTokenExpired = false
				rw.WriteHeader(http.StatusUnauthorized)
				return
			}

			mp := multipart.NewWriter(rw)
			defer mp.Close()

			guid := "some-process-guid"

			rw.Header().Set("Content-Type", `multipart/x-protobuf; boundary=`+mp.Boundary())

			for _, msg := range messages[guid] {
				partWriter, _ := mp.CreatePart(nil)
				partWriter.Write(msg)
			}
		},
	)
}

func createContainerMetric(appId string, instanceIndex int32, cpuPercentage float64, memoryBytes uint64, diskByte uint64, timestamp int64) *events.Envelope {
	if timestamp == 0 {
		timestamp = time.Now().UnixNano()
	}

	cm := &events.ContainerMetric{
		ApplicationId: proto.String(appId),
		InstanceIndex: proto.Int32(instanceIndex),
		CpuPercentage: proto.Float64(cpuPercentage),
		MemoryBytes:   proto.Uint64(memoryBytes),
		DiskBytes:     proto.Uint64(diskByte),
	}

	return &events.Envelope{
		ContainerMetric: cm,
		EventType:       events.Envelope_ContainerMetric.Enum(),
		Origin:          proto.String("fake-origin-1"),
		Timestamp:       proto.Int64(timestamp),
	}
}

func marshalMessage(message *events.Envelope) []byte {
	data, err := proto.Marshal(message)
	if err != nil {
		log.Println(err.Error())
	}

	return data
}
