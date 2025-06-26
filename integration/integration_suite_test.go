package integration_test

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"code.cloudfoundry.org/app-autoscaler/src/autoscaler/cf"
	"code.cloudfoundry.org/app-autoscaler/src/autoscaler/cf/mocks"
	"code.cloudfoundry.org/app-autoscaler/src/autoscaler/db"
	"code.cloudfoundry.org/app-autoscaler/src/autoscaler/models"
	. "code.cloudfoundry.org/app-autoscaler/src/autoscaler/testhelpers"
	"github.com/google/uuid"

	"code.cloudfoundry.org/lager/v3"
	_ "github.com/go-sql-driver/mysql"
	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/jmoiron/sqlx"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/tedsuo/ifrit"
	"github.com/tedsuo/ifrit/ginkgomon_v2"
	"github.com/tedsuo/ifrit/grouper"
)

const (
	serviceId     = "autoscaler-guid"
	planId        = "autoscaler-free-plan-id"
	testUserId    = "testUserId"
	testUserToken = "testUserOauthToken" // #nosec G101
)

var (
	components            Components
	schedulerConfPath     string
	scalingEngineConfPath string
	operatorConfPath      string
	brokerAuth            string
	dbUrl                 string
	LOGLEVEL              string
	dbHelper              *sqlx.DB
	fakeCCNOAAUAA         *mocks.Server
	testUserScope         = []string{"cloud_controller.read", "cloud_controller.write", "password.write", "openid", "network.admin", "network.write", "uaa.user"}
	processMap            = map[string]ifrit.Process{}
	mockLogCache          = &MockLogCache{}

	defaultHttpClientTimeout = 10 * time.Second

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

	dbUrl = GetDbUrl()
	if dbUrl == "" {
		Fail("environment variable $DBURL is not set")
	}

	database, err := db.GetConnection(dbUrl)
	Expect(err).NotTo(HaveOccurred())

	dbHelper, err = sqlx.Open(database.DriverName, database.DataSourceName)
	Expect(err).NotTo(HaveOccurred())

	clearDatabase()

	return payload
}, func(encodedBuiltArtifacts []byte) {
	err := json.Unmarshal(encodedBuiltArtifacts, &components)
	Expect(err).NotTo(HaveOccurred())
	components.Ports = PreparePorts()

	dbUrl = GetDbUrl()
	database, err := db.GetConnection(dbUrl)
	Expect(err).NotTo(HaveOccurred())

	dbHelper, err = sqlx.Open(database.DriverName, database.DataSourceName)
	Expect(err).NotTo(HaveOccurred())

	LOGLEVEL = os.Getenv("LOGLEVEL")
	if LOGLEVEL == "" {
		LOGLEVEL = "info"
	}
})

var _ = BeforeEach(func() {
	httpClient = NewApiClient()
	httpClientForPublicApi = NewPublicApiClient()
	logger = lager.NewLogger("test")
	logger.RegisterSink(lager.NewWriterSink(GinkgoWriter, lager.DEBUG))
})

func CompileTestedExecutables() Executables {
	builtExecutables := Executables{}
	var err error
	workingDir, err := os.Getwd()
	Expect(err).NotTo(HaveOccurred())
	rootDir := path.Join(workingDir, "..", "..", "..")

	builtExecutables[Scheduler] = path.Join(rootDir, "src", "autoscaler", "scheduler", "target", "scheduler-1.0-SNAPSHOT.war")
	builtExecutables[EventGenerator] = path.Join(rootDir, "src", "autoscaler", "build", "eventgenerator")
	builtExecutables[ScalingEngine] = path.Join(rootDir, "src", "autoscaler", "build", "scalingengine")
	builtExecutables[Operator] = path.Join(rootDir, "src", "autoscaler", "build", "operator")
	builtExecutables[GolangAPIServer] = path.Join(rootDir, "src", "autoscaler", "build", "api")
	builtExecutables[GoRouterProxy] = path.Join(rootDir, "src", "autoscaler", "build", "gorouterproxy")

	return builtExecutables
}

func PreparePorts() Ports {
	return Ports{
		GolangAPIServer:     22000 + GinkgoParallelProcess(),
		GolangAPICFServer:   22500 + GinkgoParallelProcess(),
		GolangServiceBroker: 23000 + GinkgoParallelProcess(),
		Scheduler:           15000 + GinkgoParallelProcess(),
		SchedulerCFServer:   15500 + GinkgoParallelProcess(),
		MetricsCollector:    16000 + GinkgoParallelProcess(),
		EventGenerator:      17000 + GinkgoParallelProcess(),
		CfEventGenerator:    17500 + GinkgoParallelProcess(),
		ScalingEngine:       18000 + GinkgoParallelProcess(),
		GoRouterProxy:       19000 + GinkgoParallelProcess(),
	}
}

func startGolangApiServer(golangApiServerConfPath string) {
	GinkgoHelper()
	processMap[GolangAPIServer] = ginkgomon_v2.Invoke(grouper.NewOrdered(os.Interrupt, grouper.Members{
		{GolangAPIServer, components.GolangAPIServer(golangApiServerConfPath)},
	}))
}

func startGolangApiCFServer() {
	GinkgoHelper()
	processMap[GolangAPIServer] = ginkgomon_v2.Invoke(grouper.NewOrdered(os.Interrupt, grouper.Members{
		{GolangAPIServer, components.GolangAPICFServer()},
	}))
}

func startGoRouterProxyTo(portToForward int) {
	processMap[GoRouterProxy] = ginkgomon_v2.Invoke(grouper.NewOrdered(os.Interrupt, grouper.Members{
		{"gorouter-proxy", components.GoRouterProxy(portToForward)},
	}))
}

func startScheduler() {
	processMap[Scheduler] = ginkgomon_v2.Invoke(grouper.NewOrdered(os.Interrupt, grouper.Members{
		{Scheduler, components.Scheduler(schedulerConfPath)},
	}))
}

func startEventGenerator(eventGeneratorConfPath string) {
	GinkgoHelper()
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

func startMockLogCache() {
	tlsConfig, err := NewTLSConfig(
		filepath.Join(testCertDir, "autoscaler-ca.crt"),
		filepath.Join(testCertDir, "log-cache.crt"),
		filepath.Join(testCertDir, "log-cache.key"),
		"log-cache",
	)
	Expect(err).ToNot(HaveOccurred())

	mockLogCache = NewMockLogCache(tlsConfig)
	err = mockLogCache.Start(20000 + GinkgoParallelProcess())
	Expect(err).ToNot(HaveOccurred())
}

func stopGolangApiServer() {
	ginkgomon_v2.Kill(processMap[GolangAPIServer], 5*time.Second)
}

func stopGoRouterProxy() {
	ginkgomon_v2.Kill(processMap[GoRouterProxy], 5*time.Second)
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

func stopMockLogCache() {
	mockLogCache.Stop()
}

func getRandomIdRef(ref string) string {
	report := CurrentSpecReport()
	// 0123456789012345678901234567890123456789
	// operator_others:189,11,instance:a5f63cbf 7c204c417941d91d21cb3bd0
	// |filename|:|linenumber|,|ref|process|:|random|
	// |15|1|3-4|1|14|2|1|3-4| == 40 (max id length)
	if len(ref) > 13 {
		GinkgoT().Logf("WARNING: %s:%d using a ref that is being truncated '%s' should be <= 13 chars", report.FileName(), report.LineNumber(), ref)
		ref = ref[:13]
	}
	id := fmt.Sprintf("%s:%d,%s,%d:%s", testFileFragment(report.FileName()), report.LineNumber(), ref, GinkgoParallelProcess(), randomBits())
	if len(id) > 40 {
		id = id[:40]
	}
	return id
}

func randomBits() string {
	return strings.ReplaceAll(uuid.NewString(), "-", "")
}

func testFileFragment(filename string) string {
	base := filepath.Base(filename)
	base = strings.TrimSuffix(base, "_test.go")
	base = strings.TrimPrefix(base, "integration_")
	if len(base) > 15 {
		return base[(len(base) - 15):]
	}
	return base
}

func provisionServiceInstance(brokerUrl *url.URL, serviceInstanceId string, orgId string, spaceId string, defaultPolicy []byte, httpClient *http.Client) (*http.Response, error) {
	By("provisionServiceInstance")
	var bindBody map[string]interface{}
	if defaultPolicy != nil {
		defaultPolicy := json.RawMessage(defaultPolicy)
		parameters := map[string]interface{}{
			"default_policy": defaultPolicy,
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
	req, err := http.NewRequest("PUT", fmt.Sprintf("%s/v2/service_instances/%s", brokerUrl.String(), serviceInstanceId), bytes.NewReader(body))
	Expect(err).NotTo(HaveOccurred())
	req.Header.Set("X-Broker-API-Version", "2.14")
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Basic "+brokerAuth)
	return httpClient.Do(req)
}

func updateServiceInstance(serviceInstanceId string, defaultPolicy []byte, brokerPort int, httpClient *http.Client) (*http.Response, error) {
	By("updateServiceInstance")
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
	req.Header.Set("X-Broker-API-Version", "2.14")
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Basic "+brokerAuth)
	return httpClient.Do(req)
}

func deProvisionServiceInstance(brokerUrl *url.URL, serviceInstanceId string, httpClient *http.Client) (*http.Response, error) {
	By("deProvisionServiceInstance")
	req, err := http.NewRequest("DELETE", fmt.Sprintf("%s/v2/service_instances/%s?service_id=%s&plan_id=%s", brokerUrl.String(), serviceInstanceId, serviceId, planId), nil)
	ExpectWithOffset(2, err).NotTo(HaveOccurred())
	req.Header.Set("X-Broker-API-Version", "2.14")
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Basic "+brokerAuth)
	return httpClient.Do(req)
}

func bindService(bindingId string, appId string, serviceInstanceId string, policy []byte, brokerPort int, httpClient *http.Client) (*http.Response, error) {
	By("bindService")
	var bindBody map[string]interface{}
	if policy != nil {
		rawParameters := json.RawMessage(policy)
		bindBody = map[string]interface{}{
			"app_guid":   appId,
			"service_id": serviceId,
			"plan_id":    planId,
			"parameters": rawParameters,
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
	req.Header.Set("X-Broker-API-Version", "2.14")
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Basic "+brokerAuth)
	return httpClient.Do(req)
}

func unbindService(brokerUrl *url.URL, bindingId string, appId string, serviceInstanceId string, httpClient *http.Client) (*http.Response, error) {
	By("unbindService")
	req, err := http.NewRequest("DELETE", fmt.Sprintf("%s/v2/service_instances/%s/service_bindings/%s?service_id=%s&plan_id=%s", brokerUrl.String(), serviceInstanceId, bindingId, serviceId, planId), nil)
	Expect(err).NotTo(HaveOccurred())
	req.Header.Set("X-Broker-API-Version", "2.14")
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Basic "+brokerAuth)
	return httpClient.Do(req)
}

// It is used to provision and bind a service instance to an app
func provisionAndBind(brokerUrl *url.URL, serviceInstanceId string, orgId string, spaceId string, bindingId string, appId string, httpClient *http.Client) {
	brokerPort := components.Ports[GolangServiceBroker]
	resp, err := provisionServiceInstance(brokerUrl, serviceInstanceId, orgId, spaceId, nil, httpClient)
	Expect(err).WithOffset(1).NotTo(HaveOccurred())
	Expect(resp.StatusCode).WithOffset(1).To(Equal(http.StatusCreated), fmt.Sprintf("response was '%s'", MustReadAll(resp.Body)))
	_ = resp.Body.Close()

	resp, err = bindService(bindingId, appId, serviceInstanceId, nil, brokerPort, httpClient)
	Expect(err).WithOffset(1).NotTo(HaveOccurred())
	Expect(resp.StatusCode).WithOffset(1).To(Equal(http.StatusCreated), fmt.Sprintf("response was '%s'", MustReadAll(resp.Body)))
	_ = resp.Body.Close()
}

func unbindAndDeProvision(brokerUrl *url.URL, bindingId string, appId string, serviceInstanceId string, httpClient *http.Client) {
	resp, err := unbindService(brokerUrl, bindingId, appId, serviceInstanceId, httpClient)
	ExpectWithOffset(1, err).NotTo(HaveOccurred())
	ExpectWithOffset(1, resp.StatusCode).To(Equal(http.StatusOK))
	_ = resp.Body.Close()

	resp, err = deProvisionServiceInstance(brokerUrl, serviceInstanceId, httpClient)
	ExpectWithOffset(1, err).NotTo(HaveOccurred())
	ExpectWithOffset(1, resp.StatusCode).To(Equal(http.StatusOK))
	_ = resp.Body.Close()
}

func getPolicy(appId string, apiServerPort int, httpClient *http.Client) (*http.Response, error) {
	By("getPolicy")
	req, err := http.NewRequest("GET", fmt.Sprintf("https://127.0.0.1:%d/v1/apps/%s/policy", apiServerPort, appId), nil)
	req.Header.Set("Authorization", "bearer fake-token")
	Expect(err).NotTo(HaveOccurred())
	return httpClient.Do(req)
}

func detachPolicy(appId string, apiServerPort int, httpClient *http.Client) (*http.Response, error) {
	By("detachPolicy")
	req, err := http.NewRequest("DELETE", fmt.Sprintf("https://127.0.0.1:%d/v1/apps/%s/policy", apiServerPort, appId), strings.NewReader(""))
	Expect(err).NotTo(HaveOccurred())
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "bearer fake-token")
	return httpClient.Do(req)
}

func attachPolicy(appId string, policy []byte, apiServerPort int, httpClient *http.Client) (*http.Response, error) {
	By("attachPolicy")
	req, err := http.NewRequest("PUT", fmt.Sprintf("https://127.0.0.1:%d/v1/apps/%s/policy", apiServerPort, appId), bytes.NewReader(policy))
	ExpectWithOffset(1, err).NotTo(HaveOccurred())
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "bearer fake-token")
	return httpClient.Do(req)
}

func getSchedules(appId string) (*http.Response, error) {
	By("getSchedules")
	req, err := http.NewRequest("GET", fmt.Sprintf("https://127.0.0.1:%d/v1/apps/%s/schedules", components.Ports[Scheduler], appId), strings.NewReader(""))
	Expect(err).NotTo(HaveOccurred())
	req.Header.Set("Content-Type", "application/json")
	return httpClient.Do(req)
}

func createSchedule(appId string, guid string, schedule string) (*http.Response, error) {
	By("createSchedule")
	req, err := http.NewRequest("PUT", fmt.Sprintf("https://127.0.0.1:%d/v1/apps/%s/schedules?guid=%s", components.Ports[Scheduler], appId, guid), bytes.NewReader([]byte(schedule)))
	if err != nil {
		panic(err)
	}
	Expect(err).NotTo(HaveOccurred())
	req.Header.Set("Content-Type", "application/json")
	return httpClient.Do(req)
}

func deleteSchedule(appId string) (*http.Response, error) {
	By("deleteSchedule")
	req, err := http.NewRequest("DELETE", fmt.Sprintf("https://127.0.0.1:%d/v1/apps/%s/schedules", components.Ports[Scheduler], appId), strings.NewReader(""))
	Expect(err).NotTo(HaveOccurred())
	req.Header.Set("Content-Type", "application/json")
	return httpClient.Do(req)
}

func getActiveSchedule(appId string) (*http.Response, error) {
	By("getActiveSchedule")
	req, err := http.NewRequest("GET", fmt.Sprintf("https://127.0.0.1:%d/v1/apps/%s/active_schedules", components.Ports[ScalingEngine], appId), strings.NewReader(""))
	Expect(err).NotTo(HaveOccurred())
	req.Header.Set("Content-Type", "application/json")
	return httpClient.Do(req)
}

func activeScheduleExists(appId string) bool {
	resp, err := getActiveSchedule(appId)
	if err == nil {
		defer func() { _ = resp.Body.Close() }()
	}
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
	By("getScalingHistories")
	httpClientTmp := httpClientForPublicApi
	getScalingHistoriesURL := fmt.Sprintf("https://127.0.0.1:%d/v1/apps/%s/scaling_histories", apiServerPort, pathVariables[0])
	if len(parameters) > 0 {
		parsedURL, err := url.Parse(getScalingHistoriesURL)
		Expect(err).ToNot(HaveOccurred())

		params := url.Values{}
		for paramName, paramValue := range parameters {
			params.Add(paramName, paramValue)
		}
		parsedURL.RawQuery = params.Encode()
		getScalingHistoriesURL = parsedURL.String()
	}
	req, err := http.NewRequest("GET", getScalingHistoriesURL, strings.NewReader(""))
	Expect(err).NotTo(HaveOccurred())
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "bearer fake-token")
	return httpClientTmp.Do(req)
}

func getAppAggregatedMetrics(apiServerPort int, pathVariables []string, parameters map[string]string) (*http.Response, error) {
	By("getAppAggregatedMetrics")
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
	content, err := os.ReadFile(filename)
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

func createScalingHistoryError(appId string, timestamp int64) models.AppScalingHistory {
	return models.AppScalingHistory{
		AppId:        appId,
		OldInstances: -1,
		NewInstances: -1,
		Reason:       "a reason",
		Message:      "a message",
		ScalingType:  models.ScalingTypeDynamic,
		Status:       models.ScalingStatusFailed,
		Error:        "an error",
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

func insertAppMetric(appMetrics *models.AppMetric) {
	query := dbHelper.Rebind("INSERT INTO app_metric" +
		"(app_id, metric_type, unit, value, timestamp) " +
		"VALUES(?, ?, ?, ?, ?)")
	_, err := dbHelper.Exec(query, appMetrics.AppId, appMetrics.MetricType, appMetrics.Unit, appMetrics.Value, appMetrics.Timestamp)
	Expect(err).NotTo(HaveOccurred())
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
	defer func() { _ = resp.Body.Close() }()
	checkResponse(resp, err, expectHttpStatus, expectResponseMap)
}

func checkPublicAPIResponseContentWithParameters(getResponseWithParameters GetResponseWithParameters, apiServerPort int, pathVariables []string, parameters map[string]string, expectHttpStatus int, expectResponseMap map[string]interface{}) {
	resp, err := getResponseWithParameters(apiServerPort, pathVariables, parameters)
	defer func() { _ = resp.Body.Close() }()
	checkResponse(resp, err, expectHttpStatus, expectResponseMap)
}

func checkResponse(resp *http.Response, err error, expectHttpStatus int, expectResponseMap map[string]interface{}) {
	Expect(err).WithOffset(2).NotTo(HaveOccurred())
	Expect(resp.StatusCode).WithOffset(2).To(Equal(expectHttpStatus))
	var actual map[string]interface{}
	err = json.NewDecoder(resp.Body).Decode(&actual)
	Expect(err).WithOffset(2).NotTo(HaveOccurred())
	Expect(actual).WithOffset(2).To(Equal(expectResponseMap))
}

func checkResponseEmptyAndStatusCode(resp *http.Response, err error, expectedStatus int) {
	Expect(err).NotTo(HaveOccurred())
	defer func() { _ = resp.Body.Close() }()
	body, err := io.ReadAll(resp.Body)
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
	fakeCCNOAAUAA = mocks.NewServer()
	fakeCCNOAAUAA.Add().
		GetApp(models.AppStatusStarted, http.StatusOK, "test_space_guid").
		GetAppProcesses(instanceCount).
		ScaleAppWebProcess().
		Roles(http.StatusOK, cf.Role{Type: cf.RoleSpaceDeveloper}).
		ServiceInstance("cc-free-plan-id").
		ServicePlan("autoscaler-free-plan-id").
		Info(fakeCCNOAAUAA.URL()).
		OauthToken(testUserToken).
		Introspect(testUserScope).
		UserInfo(http.StatusOK, testUserId)
}
