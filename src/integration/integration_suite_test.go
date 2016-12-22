package integration_test

import (
	"autoscaler/cf"
	"autoscaler/db"
	"autoscaler/models"
	"bytes"
	. "integration"

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
	"io/ioutil"
	"log"
	"mime/multipart"
	"net/http"
	"os"
	"path"
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
	ccNOAAUAARegPath         = regexp.MustCompile(`^/apps/.*/containermetrics$`)
	appInstanceRegPath       = regexp.MustCompile(`^/v2/apps/.*$`)
	dbHelper                 *sql.DB
	fakeScheduler            *ghttp.Server
	fakeScalingEngine        *ghttp.Server
	fakeCCNOAAUAA            *ghttp.Server
	processMap               map[string]ifrit.Process = map[string]ifrit.Process{}
	schedulerProcess         ifrit.Process

	brokerApiHttpRequestTimeout    time.Duration = 1 * time.Second
	apiSchedulerHttpRequestTimeout time.Duration = 5 * time.Second

	pollInterval              time.Duration = 1 * time.Second
	refreshInterval           time.Duration = 1 * time.Second
	aggregatorExecuteInterval time.Duration = 1 * time.Second
	policyPollerInterval      time.Duration = 1 * time.Second
	evaluationManagerInterval time.Duration = 1 * time.Second

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
	schedulerConfPath = components.PrepareSchedulerConfig(dbUrl, fakeScalingEngine.URL(), tmpDir)
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

	_, err = dbHelper.Exec("DELETE FROM scalinghistory")
	Expect(err).NotTo(HaveOccurred())

	_, err = dbHelper.Exec("DELETE FROM app_metric")
	Expect(err).NotTo(HaveOccurred())

	_, err = dbHelper.Exec("DELETE FROM appinstancemetrics")
	Expect(err).NotTo(HaveOccurred())
}
func insertPolicy(appId string, scalingPolicy models.ScalingPolicy) {

	query := "INSERT INTO policy_json(app_id, policy_json) VALUES($1, $2)"
	policyBytes, err := json.Marshal(scalingPolicy)
	Expect(err).NotTo(HaveOccurred())
	_, err = dbHelper.Exec(query, appId, string(policyBytes))
	Expect(err).NotTo(HaveOccurred())

}
func getScalingHistoryCount(appId string, oldInstanceCount int, newInstanceCount int) int {
	var count int
	query := "SELECT COUNT(*) FROM scalinghistory WHERE appid=$1 AND oldinstances=$2 AND newinstances=$3"
	err := dbHelper.QueryRow(query, appId, oldInstanceCount, newInstanceCount).Scan(&count)
	Expect(err).NotTo(HaveOccurred())
	return count
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

func startFakeCCNOAAUAA(appId string, instanceCount int) {
	fakeCCNOAAUAA = ghttp.NewServer()
	fakeCCNOAAUAA.RouteToHandler("GET", "/v2/info", ghttp.RespondWithJSONEncoded(http.StatusOK,
		cf.Endpoints{
			AuthEndpoint:    fakeCCNOAAUAA.URL(),
			DopplerEndpoint: strings.Replace(fakeCCNOAAUAA.URL(), "http", "ws", 1),
		}))
	fakeCCNOAAUAA.RouteToHandler("POST", "/oauth/token", ghttp.RespondWithJSONEncoded(http.StatusOK, cf.Tokens{}))

	fakeCCNOAAUAA.RouteToHandler("GET", appInstanceRegPath, ghttp.RespondWithJSONEncoded(http.StatusOK,
		models.AppInfo{Entity: models.AppEntity{Instances: instanceCount}}))
	fakeCCNOAAUAA.RouteToHandler("PUT", appInstanceRegPath, ghttp.RespondWith(http.StatusCreated, ""))

}

func fakeMetrics(appId string, memoryValue uint64) {

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

			guid := appId

			rw.Header().Set("Content-Type", `multipart/x-protobuf; boundary=`+mp.Boundary())
			timestamp := time.Now().UnixNano()
			message1 := marshalMessage(createContainerMetric(appId, 0, 3.0, memoryValue, 2048, timestamp))
			message2 := marshalMessage(createContainerMetric(appId, 1, 4.0, memoryValue, 2048, timestamp))
			message3 := marshalMessage(createContainerMetric(appId, 2, 5.0, memoryValue, 2048, timestamp))

			messages := map[string][][]byte{}
			messages[appId] = [][]byte{message1, message2, message3}
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
