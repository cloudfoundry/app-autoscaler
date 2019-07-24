package publicapiserver_test

import (
	"encoding/json"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"net/url"
	"path/filepath"
	"regexp"
	"strconv"
	"testing"

	"code.cloudfoundry.org/lager/lagertest"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/ghttp"
	"github.com/tedsuo/ifrit"
	"github.com/tedsuo/ifrit/ginkgomon"

	"autoscaler/api/config"
	"autoscaler/api/publicapiserver"
	"autoscaler/cf"
	"autoscaler/fakes"
	"autoscaler/helpers"
	"autoscaler/models"
)

const (
	CLIENT_ID                         = "client-id"
	CLIENT_SECRET                     = "client-secret"
	TEST_APP_ID                       = "test-app-id"
	TEST_USER_TOKEN                   = "bearer testusertoken"
	INVALID_USER_TOKEN                = "bearer invalid_user_token invalid_user_token"
	INVALID_USER_TOKEN_WITHOUT_BEARER = "not-bearer testusertoken"
	TEST_INVALID_USER_TOKEN           = "bearer testinvalidusertoken"
	TEST_USER_ID                      = "test-user-id"
	TEST_METRIC_TYPE                  = "test_metric"
	TEST_METRIC_UNIT                  = "test_unit"
)

var (
	serverProcess ifrit.Process
	serverUrl     *url.URL
	httpClient    *http.Client
	conf          *config.Config
	infoBytes     []byte

	scalingEngineServer    *ghttp.Server
	metricsCollectorServer *ghttp.Server
	eventGeneratorServer   *ghttp.Server
	schedulerServer        *ghttp.Server

	scalingEngineStatus    int
	metricsCollectorStatus int
	eventGeneratorStatus   int
	schedulerStatus        int

	scalingEngineResponse    []models.AppScalingHistory
	metricsCollectorResponse []models.AppInstanceMetric
	eventGeneratorResponse   []models.AppMetric

	fakeCFClient *fakes.FakeCFClient
	fakePolicyDB *fakes.FakePolicyDB
)

func TestPublicapiserver(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Publicapiserver Suite")
}

var _ = BeforeSuite(func() {
	scalingEngineServer = ghttp.NewServer()
	metricsCollectorServer = ghttp.NewServer()
	eventGeneratorServer = ghttp.NewServer()
	schedulerServer = ghttp.NewServer()

	testCertDir := "../../../../test-certs"
	apiPort := 11000 + GinkgoParallelNode()
	conf = &config.Config{
		Logging: helpers.LoggingConfig{
			Level: "debug",
		},
		PublicApiServer: config.ServerConfig{
			Port: apiPort,
		},
		PolicySchemaPath: "../policyvalidator/policy_json.schema.json",
		Scheduler: config.SchedulerConfig{
			SchedulerURL: schedulerServer.URL(),
		},
		InfoFilePath: "../exampleconfig/info-file.json",
		MetricsCollector: config.MetricsCollectorConfig{
			MetricsCollectorUrl: metricsCollectorServer.URL(),
			TLSClientCerts: models.TLSCerts{
				KeyFile:    filepath.Join(testCertDir, "metricscollector.key"),
				CertFile:   filepath.Join(testCertDir, "metricscollector.crt"),
				CACertFile: filepath.Join(testCertDir, "autoscaler-ca.crt"),
			},
		},
		EventGenerator: config.EventGeneratorConfig{
			EventGeneratorUrl: eventGeneratorServer.URL(),
			TLSClientCerts: models.TLSCerts{
				KeyFile:    filepath.Join(testCertDir, "eventgenerator.key"),
				CertFile:   filepath.Join(testCertDir, "eventgenerator.crt"),
				CACertFile: filepath.Join(testCertDir, "autoscaler-ca.crt"),
			},
		},
		ScalingEngine: config.ScalingEngineConfig{
			ScalingEngineUrl: scalingEngineServer.URL(),
			TLSClientCerts: models.TLSCerts{
				KeyFile:    filepath.Join(testCertDir, "scalingengine.key"),
				CertFile:   filepath.Join(testCertDir, "scalingengine.crt"),
				CACertFile: filepath.Join(testCertDir, "autoscaler-ca.crt"),
			},
		},
		CF: cf.CFConfig{
			API:               "http://api.bosh-lite.com",
			ClientID:          CLIENT_ID,
			Secret:            CLIENT_SECRET,
			SkipSSLValidation: true,
		},
	}

	fakePolicyDB = &fakes.FakePolicyDB{}
	fakeCFClient = &fakes.FakeCFClient{}

	httpServer, err := publicapiserver.NewPublicApiServer(lagertest.NewTestLogger("publicapiserver"), conf, fakePolicyDB, fakeCFClient)
	Expect(err).NotTo(HaveOccurred())

	serverUrl, err = url.Parse("http://127.0.0.1:" + strconv.Itoa(apiPort))
	Expect(err).NotTo(HaveOccurred())

	serverProcess = ginkgomon.Invoke(httpServer)

	httpClient = &http.Client{}

	infoBytes, err = ioutil.ReadFile("../exampleconfig/info-file.json")
	Expect(err).NotTo(HaveOccurred())

	scalingHistoryPathMatcher, err := regexp.Compile("/v1/apps/[A-Za-z0-9\\-]+/scaling_histories")
	Expect(err).NotTo(HaveOccurred())
	scalingEngineServer.RouteToHandler(http.MethodGet, scalingHistoryPathMatcher, ghttp.RespondWithJSONEncodedPtr(&scalingEngineStatus, &scalingEngineResponse))

	metricsCollectorPathMatcher, err := regexp.Compile("/v1/apps/[A-Za-z0-9\\-]+/metric_histories/[a-zA-Z0-9_]+")
	Expect(err).NotTo(HaveOccurred())
	metricsCollectorServer.RouteToHandler(http.MethodGet, metricsCollectorPathMatcher, ghttp.RespondWithJSONEncodedPtr(&metricsCollectorStatus, &metricsCollectorResponse))

	eventGeneratorPathMatcher, err := regexp.Compile("/v1/apps/[A-Za-z0-9\\-]+/aggregated_metric_histories/[a-zA-Z0-9_]+")
	Expect(err).NotTo(HaveOccurred())
	eventGeneratorServer.RouteToHandler(http.MethodGet, eventGeneratorPathMatcher, ghttp.RespondWithJSONEncodedPtr(&eventGeneratorStatus, &eventGeneratorResponse))

	schedulerPathMatcher, err := regexp.Compile("/v1/apps/[A-Za-z0-9\\-]+/schedules")
	Expect(err).NotTo(HaveOccurred())
	schedulerServer.RouteToHandler(http.MethodPut, schedulerPathMatcher, ghttp.RespondWithJSONEncodedPtr(&schedulerStatus, nil))
	schedulerServer.RouteToHandler(http.MethodDelete, schedulerPathMatcher, ghttp.RespondWithJSONEncodedPtr(&schedulerStatus, nil))

})

var _ = AfterSuite(func() {
	ginkgomon.Interrupt(serverProcess)

	scalingEngineServer.Close()
	metricsCollectorServer.Close()
	eventGeneratorServer.Close()
})

func GetTestHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("Success"))
	}
}

func CheckResponse(resp *httptest.ResponseRecorder, statusCode int, errResponse models.ErrorResponse) {
	Expect(resp.Code).To(Equal(statusCode))
	var errResp models.ErrorResponse
	err := json.NewDecoder(resp.Body).Decode(&errResp)
	Expect(err).NotTo(HaveOccurred())
	Expect(errResp).To(Equal(errResponse))
}
