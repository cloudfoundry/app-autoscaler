package integration_test

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"mime/multipart"
	"net/http"
	"os"
	"path"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
	"time"

	"code.cloudfoundry.org/app-autoscaler/src/autoscaler/cf"
	"code.cloudfoundry.org/app-autoscaler/src/autoscaler/db"
	"code.cloudfoundry.org/app-autoscaler/src/autoscaler/models"
	as_testhelpers "code.cloudfoundry.org/app-autoscaler/src/autoscaler/testhelpers"

	"code.cloudfoundry.org/cfhttp"
	"code.cloudfoundry.org/go-loggregator/v8/rpc/loggregator_v2"
	"code.cloudfoundry.org/lager"
	"github.com/cloudfoundry/sonde-go/events"
	_ "github.com/go-sql-driver/mysql"
	"github.com/gogo/protobuf/proto"
	"github.com/jmoiron/sqlx"
	_ "github.com/lib/pq"
	uuid "github.com/nu7hatch/gouuid"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/ghttp"
	"github.com/tedsuo/ifrit"
	"github.com/tedsuo/ifrit/ginkgomon_v2"
	"github.com/tedsuo/ifrit/grouper"
)

const (
	serviceId = "autoscaler-guid"
	planId    = "autoscaler-free-plan-id"
)

var (
	components              Components
	tmpDir                  string
	golangApiServerConfPath string
	schedulerConfPath       string
	eventGeneratorConfPath  string
	scalingEngineConfPath   string
	operatorConfPath        string
	metricsGatewayConfPath  string
	metricsServerConfPath   string
	brokerAuth              string
	dbUrl                   string
	LOGLEVEL                string
	noaaPollingRegPath      = regexp.MustCompile(`^/apps/.*/containermetrics$`)
	appSummaryRegPath       = regexp.MustCompile(`^/v2/apps/.*/summary$`)
	appInstanceRegPath      = regexp.MustCompile(`^/v2/apps/.*$`)
	v3appInstanceRegPath    = regexp.MustCompile(`^/v3/apps/.*$`)
	rolesRegPath            = regexp.MustCompile(`^/v3/roles$`)
	serviceInstanceRegPath  = regexp.MustCompile(`^/v2/service_instances/.*$`)
	servicePlanRegPath      = regexp.MustCompile(`^/v2/service_plans/.*$`)
	dbHelper                *sqlx.DB
	fakeCCNOAAUAA           *ghttp.Server
	testUserId              = "testUserId"
	testUserScope           = []string{"cloud_controller.read", "cloud_controller.write", "password.write", "openid", "network.admin", "network.write", "uaa.user"}

	processMap = map[string]ifrit.Process{}

	defaultHttpClientTimeout = 10 * time.Second

	apiSchedulerHttpRequestTimeout           = 10 * time.Second
	apiScalingEngineHttpRequestTimeout       = 10 * time.Second
	apiMetricsCollectorHttpRequestTimeout    = 10 * time.Second
	apiMetricsServerHttpRequestTimeout       = 10 * time.Second
	apiEventGeneratorHttpRequestTimeout      = 10 * time.Second
	schedulerScalingEngineHttpRequestTimeout = 10 * time.Second

	saveInterval              = 1 * time.Second
	aggregatorExecuteInterval = 1 * time.Second
	policyPollerInterval      = 1 * time.Second
	evaluationManagerInterval = 1 * time.Second
	breachDurationSecs        = 5

	httpClient             *http.Client
	httpClientForPublicApi *http.Client
	logger                 lager.Logger

	testCertDir = "../../../test-certs"
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

	database, err := db.GetConnection(dbUrl)
	Expect(err).NotTo(HaveOccurred())

	dbHelper, err = sqlx.Open(database.DriverName, database.DSN)
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
	database, err := db.GetConnection(dbUrl)
	Expect(err).NotTo(HaveOccurred())

	dbHelper, err = sqlx.Open(database.DriverName, database.DSN)
	Expect(err).NotTo(HaveOccurred())

	LOGLEVEL = os.Getenv("LOGLEVEL")
	if LOGLEVEL == "" {
		LOGLEVEL = "info"
	}
})

var _ = SynchronizedAfterSuite(func() {
	if len(tmpDir) > 0 {
		_ = os.RemoveAll(tmpDir)
	}
}, func() {

})

var _ = BeforeEach(func() {
	//nolint:staticcheck //TODO https://github.com/cloudfoundry/app-autoscaler-release/issues/549
	httpClient = cfhttp.NewClient()
	//nolint:staticcheck //TODO https://github.com/cloudfoundry/app-autoscaler-release/issues/549
	httpClientForPublicApi = cfhttp.NewClient()
	logger = lager.NewLogger("test")
	logger.RegisterSink(lager.NewWriterSink(GinkgoWriter, lager.DEBUG))
})

func CompileTestedExecutables() Executables {
	builtExecutables := Executables{}
	var err error
	workingDir, err := os.Getwd()
	Expect(err).NotTo(HaveOccurred())
	rootDir := path.Join(workingDir, "..", "..", "..")

	builtExecutables[Scheduler] = path.Join(rootDir, "src", "scheduler", "target", "scheduler-1.0-SNAPSHOT.war")
	builtExecutables[EventGenerator] = path.Join(rootDir, "src", "autoscaler", "build", "eventgenerator")
	builtExecutables[ScalingEngine] = path.Join(rootDir, "src", "autoscaler", "build", "scalingengine")
	builtExecutables[Operator] = path.Join(rootDir, "src", "autoscaler", "build", "operator")
	builtExecutables[MetricsGateway] = path.Join(rootDir, "src", "autoscaler", "build", "metricsgateway")
	builtExecutables[MetricsServerHTTP] = path.Join(rootDir, "src", "autoscaler", "build", "metricsserver")
	builtExecutables[GolangAPIServer] = path.Join(rootDir, "src", "autoscaler", "build", "api")

	return builtExecutables
}

func PreparePorts() Ports {
	return Ports{
		GolangAPIServer:     22000 + GinkgoParallelProcess(),
		GolangServiceBroker: 23000 + GinkgoParallelProcess(),
		Scheduler:           15000 + GinkgoParallelProcess(),
		MetricsCollector:    16000 + GinkgoParallelProcess(),
		MetricsServerHTTP:   20000 + GinkgoParallelProcess(),
		MetricsServerWS:     21000 + GinkgoParallelProcess(),
		EventGenerator:      17000 + GinkgoParallelProcess(),
		ScalingEngine:       18000 + GinkgoParallelProcess(),
	}
}

func startGolangApiServer() {
	processMap[GolangAPIServer] = ginkgomon_v2.Invoke(grouper.NewOrdered(os.Interrupt, grouper.Members{
		{GolangAPIServer, components.GolangAPIServer(golangApiServerConfPath)},
	}))
}

func startScheduler() {
	processMap[Scheduler] = ginkgomon_v2.Invoke(grouper.NewOrdered(os.Interrupt, grouper.Members{
		{Scheduler, components.Scheduler(schedulerConfPath)},
	}))
}

func startEventGenerator() {
	processMap[EventGenerator] = ginkgomon_v2.Invoke(grouper.NewOrdered(os.Interrupt, grouper.Members{
		{EventGenerator, components.EventGenerator(eventGeneratorConfPath)},
	}))
}

func startScalingEngine() {
	processMap[ScalingEngine] = ginkgomon_v2.Invoke(grouper.NewOrdered(os.Interrupt, grouper.Members{
		{ScalingEngine, components.ScalingEngine(scalingEngineConfPath)},
	}))
}

func startOperator() {
	processMap[Operator] = ginkgomon_v2.Invoke(grouper.NewOrdered(os.Interrupt, grouper.Members{
		{Operator, components.Operator(operatorConfPath)},
	}))
}

func startMetricsGateway() {
	processMap[MetricsGateway] = ginkgomon_v2.Invoke(grouper.NewOrdered(os.Interrupt, grouper.Members{
		{MetricsGateway, components.MetricsGateway(metricsGatewayConfPath)},
	}))
}

func startMetricsServer() {
	processMap[MetricsServerHTTP] = ginkgomon_v2.Invoke(grouper.NewOrdered(os.Interrupt, grouper.Members{
		{MetricsServerHTTP, components.MetricsServer(metricsServerConfPath)},
	}))
}

func stopGolangApiServer() {
	ginkgomon_v2.Kill(processMap[GolangAPIServer], 5*time.Second)
}
func stopScheduler() {
	ginkgomon_v2.Kill(processMap[Scheduler], 5*time.Second)
}
func stopScalingEngine() {
	ginkgomon_v2.Kill(processMap[ScalingEngine], 5*time.Second)
}
func stopEventGenerator() {
	ginkgomon_v2.Kill(processMap[EventGenerator], 5*time.Second)
}
func stopOperator() {
	ginkgomon_v2.Kill(processMap[Operator], 5*time.Second)
}
func stopMetricsGateway() {
	ginkgomon_v2.Kill(processMap[MetricsGateway], 5*time.Second)
}
func stopMetricsServer() {
	ginkgomon_v2.Kill(processMap[MetricsServerHTTP], 5*time.Second)
}

func getRandomId() string {
	v4, _ := uuid.NewV4()
	return v4.String()
}

func initializeHttpClient(certFileName string, keyFileName string, caCertFileName string, httpRequestTimeout time.Duration) {
	//nolint:staticcheck  // SA1019 TODO: https://github.com/cloudfoundry/app-autoscaler-release/issues/548
	TLSConfig, err := cfhttp.NewTLSConfig(
		filepath.Join(testCertDir, certFileName),
		filepath.Join(testCertDir, keyFileName),
		filepath.Join(testCertDir, caCertFileName),
	)
	Expect(err).NotTo(HaveOccurred())
	httpClient.Transport.(*http.Transport).TLSClientConfig = TLSConfig
	httpClient.Timeout = httpRequestTimeout
}
func initializeHttpClientForPublicApi(certFileName string, keyFileName string, caCertFileName string, httpRequestTimeout time.Duration) {
	//nolint:staticcheck  // SA1019 TODO: https://github.com/cloudfoundry/app-autoscaler-release/issues/548
	TLSConfig, err := cfhttp.NewTLSConfig(
		filepath.Join(testCertDir, certFileName),
		filepath.Join(testCertDir, keyFileName),
		filepath.Join(testCertDir, caCertFileName),
	)
	Expect(err).NotTo(HaveOccurred())
	httpClientForPublicApi.Transport.(*http.Transport).TLSClientConfig = TLSConfig
	httpClientForPublicApi.Timeout = httpRequestTimeout
}

func provisionServiceInstance(serviceInstanceId string, orgId string, spaceId string, defaultPolicy []byte, brokerPort int, httpClient *http.Client) (*http.Response, error) {
	var bindBody map[string]interface{}
	if defaultPolicy != nil {
		defaultPolicy := json.RawMessage(defaultPolicy)
		parameters := map[string]interface{}{
			"default_policy": &defaultPolicy,
		}
		bindBody = map[string]interface{}{
			"organization_guid": orgId,
			"space_guid":        spaceId,
			"service_id":        serviceId,
			"plan_id":           planId,
			"parameters":        parameters,
		}
	} else {
		bindBody = map[string]interface{}{
			"organization_guid": orgId,
			"space_guid":        spaceId,
			"service_id":        serviceId,
			"plan_id":           planId,
		}
	}

	body, err := json.Marshal(bindBody)
	Expect(err).NotTo(HaveOccurred())
	req, err := http.NewRequest("PUT", fmt.Sprintf("https://127.0.0.1:%d/v2/service_instances/%s", brokerPort, serviceInstanceId), bytes.NewReader(body))
	Expect(err).NotTo(HaveOccurred())
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Basic "+brokerAuth)
	return httpClient.Do(req)
}

func updateServiceInstance(serviceInstanceId string, defaultPolicy []byte, brokerPort int, httpClient *http.Client) (*http.Response, error) {
	var updateBody map[string]interface{}
	if defaultPolicy != nil {
		defaultPolicy := json.RawMessage(defaultPolicy)
		parameters := map[string]interface{}{
			"default_policy": &defaultPolicy,
		}
		updateBody = map[string]interface{}{
			"service_id": serviceId,
			"parameters": parameters,
		}
	}

	body, err := json.Marshal(updateBody)
	Expect(err).NotTo(HaveOccurred())

	req, err := http.NewRequest("PATCH", fmt.Sprintf("https://127.0.0.1:%d/v2/service_instances/%s", brokerPort, serviceInstanceId), bytes.NewReader(body))
	Expect(err).NotTo(HaveOccurred())

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Basic "+brokerAuth)
	return httpClient.Do(req)
}

func deProvisionServiceInstance(serviceInstanceId string, brokerPort int, httpClient *http.Client) (*http.Response, error) {
	req, err := http.NewRequest("DELETE", fmt.Sprintf("https://127.0.0.1:%d/v2/service_instances/%s", brokerPort, serviceInstanceId), strings.NewReader(fmt.Sprintf(`{"service_id":"%s","plan_id": "%s"}`, serviceId, planId)))
	Expect(err).NotTo(HaveOccurred())
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Basic "+brokerAuth)
	return httpClient.Do(req)
}

func bindService(bindingId string, appId string, serviceInstanceId string, policy []byte, brokerPort int, httpClient *http.Client) (*http.Response, error) {
	var bindBody map[string]interface{}
	if policy != nil {
		rawParameters := json.RawMessage(policy)
		bindBody = map[string]interface{}{
			"app_guid":   appId,
			"service_id": serviceId,
			"plan_id":    planId,
			"parameters": &rawParameters,
		}
	} else {
		bindBody = map[string]interface{}{
			"app_guid":   appId,
			"service_id": serviceId,
			"plan_id":    planId,
		}
	}

	body, err := json.Marshal(bindBody)
	Expect(err).NotTo(HaveOccurred())
	req, err := http.NewRequest("PUT", fmt.Sprintf("https://127.0.0.1:%d/v2/service_instances/%s/service_bindings/%s", brokerPort, serviceInstanceId, bindingId), bytes.NewReader(body))
	Expect(err).NotTo(HaveOccurred())
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Basic "+brokerAuth)
	return httpClient.Do(req)
}

func unbindService(bindingId string, appId string, serviceInstanceId string, brokerPort int, httpClient *http.Client) (*http.Response, error) {
	req, err := http.NewRequest("DELETE", fmt.Sprintf("https://127.0.0.1:%d/v2/service_instances/%s/service_bindings/%s", brokerPort, serviceInstanceId, bindingId), strings.NewReader(fmt.Sprintf(`{"app_guid":"%s","service_id":"%s","plan_id":"%s"}`, appId, serviceId, planId)))
	Expect(err).NotTo(HaveOccurred())
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Basic "+brokerAuth)
	return httpClient.Do(req)
}

func provisionAndBind(serviceInstanceId string, orgId string, spaceId string, defaultPolicy []byte, bindingId string, appId string, policy []byte, brokerPort int, httpClient *http.Client) {
	resp, err := provisionServiceInstance(serviceInstanceId, orgId, spaceId, defaultPolicy, brokerPort, httpClient)
	Expect(err).NotTo(HaveOccurred())
	Expect(resp.StatusCode).To(Equal(http.StatusCreated))
	_ = resp.Body.Close()

	resp, err = bindService(bindingId, appId, serviceInstanceId, policy, brokerPort, httpClient)
	Expect(err).NotTo(HaveOccurred())
	Expect(resp.StatusCode).To(Equal(http.StatusCreated))
	_ = resp.Body.Close()
}

func unbindAndDeProvision(bindingId string, appId string, serviceInstanceId string, brokerPort int, httpClient *http.Client) {
	resp, err := unbindService(bindingId, appId, serviceInstanceId, brokerPort, httpClient)
	Expect(err).NotTo(HaveOccurred())
	Expect(resp.StatusCode).To(Equal(http.StatusOK))
	_ = resp.Body.Close()

	resp, err = deProvisionServiceInstance(serviceInstanceId, brokerPort, httpClient)
	Expect(err).NotTo(HaveOccurred())
	Expect(resp.StatusCode).To(Equal(http.StatusOK))
	_ = resp.Body.Close()
}

func getPolicy(appId string, apiServerPort int, httpClient *http.Client) (*http.Response, error) {
	req, err := http.NewRequest("GET", fmt.Sprintf("https://127.0.0.1:%d/v1/apps/%s/policy", apiServerPort, appId), nil)
	req.Header.Set("Authorization", "bearer fake-token")
	Expect(err).NotTo(HaveOccurred())
	return httpClient.Do(req)
}

func detachPolicy(appId string, apiServerPort int, httpClient *http.Client) (*http.Response, error) {
	req, err := http.NewRequest("DELETE", fmt.Sprintf("https://127.0.0.1:%d/v1/apps/%s/policy", apiServerPort, appId), strings.NewReader(""))
	Expect(err).NotTo(HaveOccurred())
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "bearer fake-token")
	return httpClient.Do(req)
}

func attachPolicy(appId string, policy []byte, apiServerPort int, httpClient *http.Client) (*http.Response, error) {
	req, err := http.NewRequest("PUT", fmt.Sprintf("https://127.0.0.1:%d/v1/apps/%s/policy", apiServerPort, appId), bytes.NewReader(policy))
	Expect(err).NotTo(HaveOccurred())
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "bearer fake-token")
	return httpClient.Do(req)
}

func getSchedules(appId string) (*http.Response, error) {
	req, err := http.NewRequest("GET", fmt.Sprintf("https://127.0.0.1:%d/v1/apps/%s/schedules", components.Ports[Scheduler], appId), strings.NewReader(""))
	Expect(err).NotTo(HaveOccurred())
	req.Header.Set("Content-Type", "application/json")
	return httpClient.Do(req)
}

func createSchedule(appId string, guid string, schedule string) (*http.Response, error) {
	req, err := http.NewRequest("PUT", fmt.Sprintf("https://127.0.0.1:%d/v1/apps/%s/schedules?guid=%s", components.Ports[Scheduler], appId, guid), bytes.NewReader([]byte(schedule)))
	if err != nil {
		panic(err)
	}
	Expect(err).NotTo(HaveOccurred())
	req.Header.Set("Content-Type", "application/json")
	return httpClient.Do(req)
}

func deleteSchedule(appId string) (*http.Response, error) {
	req, err := http.NewRequest("DELETE", fmt.Sprintf("https://127.0.0.1:%d/v1/apps/%s/schedules", components.Ports[Scheduler], appId), strings.NewReader(""))
	Expect(err).NotTo(HaveOccurred())
	req.Header.Set("Content-Type", "application/json")
	return httpClient.Do(req)
}

func getActiveSchedule(appId string) (*http.Response, error) {
	req, err := http.NewRequest("GET", fmt.Sprintf("https://127.0.0.1:%d/v1/apps/%s/active_schedules", components.Ports[ScalingEngine], appId), strings.NewReader(""))
	Expect(err).NotTo(HaveOccurred())
	req.Header.Set("Content-Type", "application/json")
	return httpClient.Do(req)
}

func activeScheduleExists(appId string) bool {
	resp, err := getActiveSchedule(appId)
	Expect(err).NotTo(HaveOccurred())

	return resp.StatusCode == http.StatusOK
}

func setPolicyRecurringDate(policyByte []byte) []byte {
	var policy models.ScalingPolicy
	err := json.Unmarshal(policyByte, &policy)
	Expect(err).NotTo(HaveOccurred())

	if policy.Schedules != nil {
		location, err := time.LoadLocation(policy.Schedules.Timezone)
		Expect(err).NotTo(HaveOccurred())
		now := time.Now().In(location)
		starttime := now.Add(time.Minute * 10)
		endtime := now.Add(time.Minute * 20)
		for _, entry := range policy.Schedules.RecurringSchedules {
			if endtime.Day() != starttime.Day() {
				entry.StartTime = "00:01"
				entry.EndTime = "23:59"
				entry.StartDate = endtime.Format("2006-01-02")
			} else {
				entry.StartTime = starttime.Format("15:04")
				entry.EndTime = endtime.Format("15:04")
			}
		}
	}

	content, err := json.Marshal(policy)
	Expect(err).NotTo(HaveOccurred())
	return content
}

func setPolicySpecificDateTime(policyByte []byte, start time.Duration, end time.Duration) string {
	timeZone := "GMT"
	location, _ := time.LoadLocation(timeZone)
	timeNowInTimeZone := time.Now().In(location)
	dateTimeFormat := "2006-01-02T15:04"
	startTime := timeNowInTimeZone.Add(start).Format(dateTimeFormat)
	endTime := timeNowInTimeZone.Add(end).Format(dateTimeFormat)

	return fmt.Sprintf(string(policyByte), timeZone, startTime, endTime)
}

func getScalingHistories(apiServerPort int, pathVariables []string, parameters map[string]string) (*http.Response, error) {
	httpClientTmp := httpClientForPublicApi
	url := "https://127.0.0.1:%d/v1/apps/%s/scaling_histories"
	if len(parameters) > 0 {
		url += "?any=any"
		for paramName, paramValue := range parameters {
			url += "&" + paramName + "=" + paramValue
		}
	}
	req, err := http.NewRequest("GET", fmt.Sprintf(url, apiServerPort, pathVariables[0]), strings.NewReader(""))
	Expect(err).NotTo(HaveOccurred())
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "bearer fake-token")
	return httpClientTmp.Do(req)
}

func getAppInstanceMetrics(apiServerPort int, pathVariables []string, parameters map[string]string) (*http.Response, error) {
	httpClientTmp := httpClientForPublicApi
	url := "https://127.0.0.1:%d/v1/apps/%s/metric_histories/%s"
	if len(parameters) > 0 {
		url += "?any=any"
		for paramName, paramValue := range parameters {
			url += "&" + paramName + "=" + paramValue
		}
	}
	req, err := http.NewRequest("GET", fmt.Sprintf(url, apiServerPort, pathVariables[0], pathVariables[1]), strings.NewReader(""))
	Expect(err).NotTo(HaveOccurred())
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "bearer fake-token")
	return httpClientTmp.Do(req)
}

func getAppAggregatedMetrics(apiServerPort int, pathVariables []string, parameters map[string]string) (*http.Response, error) {
	httpClientTmp := httpClientForPublicApi
	url := "https://127.0.0.1:%d/v1/apps/%s/aggregated_metric_histories/%s"
	if len(parameters) > 0 {
		url += "?any=any"
		for paramName, paramValue := range parameters {
			url += "&" + paramName + "=" + paramValue
		}
	}
	req, err := http.NewRequest("GET", fmt.Sprintf(url, apiServerPort, pathVariables[0], pathVariables[1]), strings.NewReader(""))
	Expect(err).NotTo(HaveOccurred())
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "bearer fake-token")
	return httpClientTmp.Do(req)
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

	_, err = dbHelper.Exec("DELETE FROM app_scaling_active_schedule")
	Expect(err).NotTo(HaveOccurred())

	_, err = dbHelper.Exec("DELETE FROM activeschedule")
	Expect(err).NotTo(HaveOccurred())

	_, err = dbHelper.Exec("DELETE FROM scalinghistory")
	Expect(err).NotTo(HaveOccurred())

	_, err = dbHelper.Exec("DELETE FROM app_metric")
	Expect(err).NotTo(HaveOccurred())

	_, err = dbHelper.Exec("DELETE FROM appinstancemetrics")
	Expect(err).NotTo(HaveOccurred())
}

func insertPolicy(appId string, policyStr string, guid string) {
	query := dbHelper.Rebind("INSERT INTO policy_json(app_id, policy_json, guid) VALUES(?, ?, ?)")
	_, err := dbHelper.Exec(query, appId, policyStr, guid)
	Expect(err).NotTo(HaveOccurred())
}

func deletePolicy(appId string) {
	query := dbHelper.Rebind("DELETE FROM policy_json WHERE app_id=?")
	_, err := dbHelper.Exec(query, appId)
	Expect(err).NotTo(HaveOccurred())
}

func insertScalingHistory(history *models.AppScalingHistory) {
	query := dbHelper.Rebind("INSERT INTO scalinghistory" +
		"(appid, timestamp, scalingtype, status, oldinstances, newinstances, reason, message, error) " +
		" VALUES(?, ?, ?, ?, ?, ?, ?, ?, ?)")
	_, err := dbHelper.Exec(query, history.AppId, history.Timestamp, history.ScalingType, history.Status,
		history.OldInstances, history.NewInstances, history.Reason, history.Message, history.Error)

	Expect(err).NotTo(HaveOccurred())
}

func createScalingHistory(appId string, timestamp int64) models.AppScalingHistory {
	return models.AppScalingHistory{
		AppId:        appId,
		OldInstances: 2,
		NewInstances: 4,
		Reason:       "a reason",
		Message:      "a message",
		ScalingType:  models.ScalingTypeDynamic,
		Status:       models.ScalingStatusSucceeded,
		Error:        "",
		Timestamp:    timestamp,
	}
}

func getScalingHistoryCount(appId string, oldInstanceCount int, newInstanceCount int) int {
	var count int
	query := dbHelper.Rebind("SELECT COUNT(*) FROM scalinghistory WHERE appid=? AND oldinstances=? AND newinstances=?")
	err := dbHelper.QueryRow(query, appId, oldInstanceCount, newInstanceCount).Scan(&count)
	Expect(err).NotTo(HaveOccurred())
	return count
}

func getScalingHistoryTotalCount(appId string) int {
	var count int
	query := dbHelper.Rebind("SELECT COUNT(*) FROM scalinghistory WHERE appid=?")
	err := dbHelper.QueryRow(query, appId).Scan(&count)
	Expect(err).NotTo(HaveOccurred())
	return count
}

func insertAppInstanceMetric(appInstanceMetric *models.AppInstanceMetric) {
	query := dbHelper.Rebind("INSERT INTO appinstancemetrics" +
		"(appid, instanceindex, collectedat, name, unit, value, timestamp) " +
		"VALUES(?, ?, ?, ?, ?, ?, ?)")
	_, err := dbHelper.Exec(query, appInstanceMetric.AppId, appInstanceMetric.InstanceIndex, appInstanceMetric.CollectedAt, appInstanceMetric.Name, appInstanceMetric.Unit, appInstanceMetric.Value, appInstanceMetric.Timestamp)
	Expect(err).NotTo(HaveOccurred())
}

func insertAppMetric(appMetrics *models.AppMetric) {
	query := dbHelper.Rebind("INSERT INTO app_metric" +
		"(app_id, metric_type, unit, value, timestamp) " +
		"VALUES(?, ?, ?, ?, ?)")
	_, err := dbHelper.Exec(query, appMetrics.AppId, appMetrics.MetricType, appMetrics.Unit, appMetrics.Value, appMetrics.Timestamp)
	Expect(err).NotTo(HaveOccurred())
}

func getAppInstanceMetricTotalCount(appId string) int {
	var count int
	query := dbHelper.Rebind("SELECT COUNT(*) FROM appinstancemetrics WHERE appid=?")
	err := dbHelper.QueryRow(query, appId).Scan(&count)
	Expect(err).NotTo(HaveOccurred())
	return count
}

func getAppMetricTotalCount(appId string) int {
	var count int
	query := dbHelper.Rebind("SELECT COUNT(*) FROM app_metric WHERE app_id=?")
	err := dbHelper.QueryRow(query, appId).Scan(&count)
	Expect(err).NotTo(HaveOccurred())
	return count
}

type GetResponse func(id string, port int, httpClient *http.Client) (*http.Response, error)
type GetResponseWithParameters func(apiServerPort int, pathVariables []string, parameters map[string]string) (*http.Response, error)

func checkResponseContent(getResponse GetResponse, id string, expectHttpStatus int, expectResponseMap map[string]interface{}, port int, httpClient *http.Client) {
	resp, err := getResponse(id, port, httpClient)
	checkResponse(resp, err, expectHttpStatus, expectResponseMap)
}

func checkPublicAPIResponseContentWithParameters(getResponseWithParameters GetResponseWithParameters, apiServerPort int, pathVariables []string, parameters map[string]string, expectHttpStatus int, expectResponseMap map[string]interface{}) {
	resp, err := getResponseWithParameters(apiServerPort, pathVariables, parameters)
	checkResponse(resp, err, expectHttpStatus, expectResponseMap)
}

func checkResponse(resp *http.Response, err error, expectHttpStatus int, expectResponseMap map[string]interface{}) {
	Expect(err).NotTo(HaveOccurred())
	Expect(resp.StatusCode).To(Equal(expectHttpStatus))
	var actual map[string]interface{}
	err = json.NewDecoder(resp.Body).Decode(&actual)
	Expect(err).NotTo(HaveOccurred())
	Expect(actual).To(Equal(expectResponseMap))
	_ = resp.Body.Close()
}

func checkResponseEmptyAndStatusCode(resp *http.Response, err error, expectedStatus int) {
	Expect(err).NotTo(HaveOccurred())
	defer func() { _ = resp.Body.Close() }()
	body, err := ioutil.ReadAll(resp.Body)
	Expect(err).NotTo(HaveOccurred())
	Expect(body).To(HaveLen(0))
	Expect(resp.StatusCode).To(Equal(expectedStatus))
}

func assertScheduleContents(appId string, expectHttpStatus int, expectResponseMap map[string]int) {
	By("checking the schedule contents")
	resp, err := getSchedules(appId)
	ExpectWithOffset(1, err).NotTo(HaveOccurred(), "Failed to get schedule:%s", err)
	ExpectWithOffset(1, resp.StatusCode).To(Equal(expectHttpStatus), "Unexpected HTTP status")
	defer func() { _ = resp.Body.Close() }()
	var actual map[string]interface{}

	err = json.NewDecoder(resp.Body).Decode(&actual)
	Expect(err).NotTo(HaveOccurred(), "Invalid JSON")

	var schedules = actual["schedules"].(map[string]interface{})
	var recurring = schedules["recurring_schedule"].([]interface{})
	var specificDate = schedules["specific_date"].([]interface{})
	ExpectWithOffset(1, len(specificDate)).To(Equal(expectResponseMap["specific_date"]), "Expected %d specific date schedules, but found %d: %#v\n", expectResponseMap["specific_date"], len(specificDate), specificDate)
	ExpectWithOffset(1, len(recurring)).To(Equal(expectResponseMap["recurring_schedule"]), "Expected %d recurring schedules, but found %d: %#v\n", expectResponseMap["recurring_schedule"], len(recurring), recurring)
}

func checkScheduleContents(appId string, expectHttpStatus int, expectResponseMap map[string]int) bool {
	resp, err := getSchedules(appId)
	ExpectWithOffset(1, err).NotTo(HaveOccurred(), "Get schedules failed with: %s", err)
	ExpectWithOffset(1, resp.StatusCode).To(Equal(expectHttpStatus), "Unexpected HTTP status")
	defer func() { _ = resp.Body.Close() }()
	var actual map[string]interface{}
	err = json.NewDecoder(resp.Body).Decode(&actual)
	ExpectWithOffset(1, err).NotTo(HaveOccurred(), "Invalid JSON")
	var schedules = actual["schedules"].(map[string]interface{})
	var recurring = schedules["recurring_schedule"].([]interface{})
	var specificDate = schedules["specific_date"].([]interface{})
	return len(specificDate) == expectResponseMap["specific_date"] && len(recurring) == expectResponseMap["recurring_schedule"]
}

func startFakeCCNOAAUAA(instanceCount int) {
	fakeCCNOAAUAA = ghttp.NewServer()
	fakeCCNOAAUAA.RouteToHandler("GET", "/v2/info", ghttp.RespondWithJSONEncoded(http.StatusOK,
		cf.Endpoints{
			AuthEndpoint:    fakeCCNOAAUAA.URL(),
			TokenEndpoint:   fakeCCNOAAUAA.URL(),
			DopplerEndpoint: strings.Replace(fakeCCNOAAUAA.URL(), "http", "ws", 1),
		}))
	fakeCCNOAAUAA.RouteToHandler("POST", "/oauth/token", ghttp.RespondWithJSONEncoded(http.StatusOK, cf.Tokens{}))
	appState := models.AppStatusStarted
	fakeCCNOAAUAA.RouteToHandler("GET", appSummaryRegPath, ghttp.RespondWithJSONEncoded(http.StatusOK,
		models.AppEntity{Instances: instanceCount, State: &appState}))
	fakeCCNOAAUAA.RouteToHandler("PUT", appInstanceRegPath, ghttp.RespondWith(http.StatusCreated, ""))
	fakeCCNOAAUAA.RouteToHandler("POST", "/check_token", ghttp.RespondWithJSONEncoded(http.StatusOK,
		struct {
			Scope []string `json:"scope"`
		}{
			testUserScope,
		}))
	fakeCCNOAAUAA.RouteToHandler("GET", "/userinfo", ghttp.RespondWithJSONEncoded(http.StatusOK,
		struct {
			UserId string `json:"user_id"`
		}{
			testUserId,
		}))
	fakeCCNOAAUAA.RouteToHandler("GET", v3appInstanceRegPath, ghttp.RespondWithJSONEncoded(http.StatusOK,
		struct {
			TotalResults int `json:"total_results"`
		}{
			1,
		}))

	app := struct {
		Relationships struct {
			Space struct {
				Data struct {
					GUID string `json:"guid"`
				} `json:"data"`
			} `json:"space"`
		} `json:"relationships"`
	}{}
	app.Relationships.Space.Data.GUID = "test_space_guid"
	fakeCCNOAAUAA.RouteToHandler("GET", v3appInstanceRegPath, ghttp.RespondWithJSONEncoded(http.StatusOK,
		app))

	roles := struct {
		Pagination struct {
			Total int `json:"total_results"`
		} `json:"pagination"`
	}{}
	roles.Pagination.Total = 1
	fakeCCNOAAUAA.RouteToHandler("GET", rolesRegPath, ghttp.RespondWithJSONEncoded(http.StatusOK,
		roles))

	type ServiceInstanceEntity struct {
		ServicePlanGuid string `json:"service_plan_guid"`
	}
	fakeCCNOAAUAA.RouteToHandler("GET", serviceInstanceRegPath, ghttp.RespondWithJSONEncoded(http.StatusOK,
		struct {
			ServiceInstanceEntity `json:"entity"`
		}{
			ServiceInstanceEntity{"cc-free-plan-id"},
		}))

	type ServicePlanEntity struct {
		UniqueId string `json:"unique_id"`
	}
	fakeCCNOAAUAA.RouteToHandler("GET", servicePlanRegPath, ghttp.RespondWithJSONEncoded(http.StatusOK,
		struct {
			ServicePlanEntity `json:"entity"`
		}{
			ServicePlanEntity{"autoscaler-free-plan-id"},
		}))
}

func fakeMetricsPolling(appId string, memoryValue uint64, memQuota uint64) {
	fakeCCNOAAUAA.RouteToHandler("GET", noaaPollingRegPath,
		func(rw http.ResponseWriter, r *http.Request) {
			mp := multipart.NewWriter(rw)
			defer func() { _ = mp.Close() }()

			rw.Header().Set("Content-Type", `multipart/x-protobuf; boundary=`+mp.Boundary())
			timestamp := time.Now().UnixNano()
			message1 := marshalMessage(createContainerMetric(appId, 0, 3.0, memoryValue, 2048000000, memQuota, 4096000000, timestamp))
			message2 := marshalMessage(createContainerMetric(appId, 1, 4.0, memoryValue, 2048000000, memQuota, 4096000000, timestamp))
			message3 := marshalMessage(createContainerMetric(appId, 2, 5.0, memoryValue, 2048000000, memQuota, 4096000000, timestamp))

			messages := [][]byte{message1, message2, message3}
			for _, msg := range messages {
				partWriter, _ := mp.CreatePart(nil)
				_, _ = partWriter.Write(msg)
			}
		},
	)
}

func startFakeRLPServer(appId string, envelopes []*loggregator_v2.Envelope, emitInterval time.Duration) *as_testhelpers.FakeEventProducer {
	fakeRLPServer, err := as_testhelpers.NewFakeEventProducer(filepath.Join(testCertDir, "reverselogproxy.crt"), filepath.Join(testCertDir, "reverselogproxy.key"), filepath.Join(testCertDir, "autoscaler-ca.crt"), emitInterval)
	Expect(err).NotTo(HaveOccurred())
	fakeRLPServer.SetEnvelops(envelopes)
	fakeRLPServer.Start()
	return fakeRLPServer
}

func stopFakeRLPServer(fakeRLPServer *as_testhelpers.FakeEventProducer) {
	stopped := fakeRLPServer.Stop()
	Expect(stopped).To(Equal(true))
}

func createContainerMetric(appId string, instanceIndex int32, cpuPercentage float64, memoryBytes uint64, diskByte uint64, memQuota uint64, diskQuota uint64, timestamp int64) *events.Envelope {
	if timestamp == 0 {
		timestamp = time.Now().UnixNano()
	}
	cm := &events.ContainerMetric{
		ApplicationId:    proto.String(appId),
		InstanceIndex:    proto.Int32(instanceIndex),
		CpuPercentage:    proto.Float64(cpuPercentage),
		MemoryBytes:      proto.Uint64(memoryBytes),
		DiskBytes:        proto.Uint64(diskByte),
		MemoryBytesQuota: proto.Uint64(memQuota),
		DiskBytesQuota:   proto.Uint64(diskQuota),
	}

	return &events.Envelope{
		ContainerMetric: cm,
		EventType:       events.Envelope_ContainerMetric.Enum(),
		Origin:          proto.String("fake-origin-1"),
		Timestamp:       proto.Int64(timestamp),
	}
}

func createContainerEnvelope(appId string, instanceIndex int32, cpuPercentage float64, memoryBytes float64, diskByte float64, memQuota float64) []*loggregator_v2.Envelope {
	return []*loggregator_v2.Envelope{
		{
			SourceId: appId,
			Message: &loggregator_v2.Envelope_Gauge{
				Gauge: &loggregator_v2.Gauge{
					Metrics: map[string]*loggregator_v2.GaugeValue{
						"cpu": {
							Unit:  "percentage",
							Value: cpuPercentage,
						},
						"disk": {
							Unit:  "bytes",
							Value: diskByte,
						},
						"memory": {
							Unit:  "bytes",
							Value: memoryBytes,
						},
						"memory_quota": {
							Unit:  "bytes",
							Value: memQuota,
						},
					},
				},
			},
		},
	}
}

func createHTTPTimerEnvelope(appId string, start int64, end int64) []*loggregator_v2.Envelope {
	return []*loggregator_v2.Envelope{
		{
			SourceId: appId,
			Message: &loggregator_v2.Envelope_Timer{
				Timer: &loggregator_v2.Timer{
					Name:  "http",
					Start: start,
					Stop:  end,
				},
			},
			DeprecatedTags: map[string]*loggregator_v2.Value{
				"peer_type": {Data: &loggregator_v2.Value_Text{Text: "Client"}},
			},
		},
	}
}

func createCustomEnvelope(appId string, name string, unit string, value float64) []*loggregator_v2.Envelope {
	return []*loggregator_v2.Envelope{
		{
			SourceId: appId,
			DeprecatedTags: map[string]*loggregator_v2.Value{
				"origin": {
					Data: &loggregator_v2.Value_Text{
						Text: "autoscaler_metrics_forwarder",
					},
				},
			},
			Message: &loggregator_v2.Envelope_Gauge{
				Gauge: &loggregator_v2.Gauge{
					Metrics: map[string]*loggregator_v2.GaugeValue{
						name: {
							Unit:  unit,
							Value: value,
						},
					},
				},
			},
		},
	}
}

func marshalMessage(message *events.Envelope) []byte {
	data, err := proto.Marshal(message)
	if err != nil {
		log.Println(err.Error())
	}

	return data
}
