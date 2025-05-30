package publicapiserver_test

import (
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"testing"

	internalscalinghistory "code.cloudfoundry.org/app-autoscaler/src/autoscaler/scalingengine/apis/scalinghistory"
	"code.cloudfoundry.org/app-autoscaler/src/autoscaler/testhelpers"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/ghttp"
	"github.com/tedsuo/ifrit"

	"code.cloudfoundry.org/app-autoscaler/src/autoscaler/api"
	"code.cloudfoundry.org/app-autoscaler/src/autoscaler/api/config"
	"code.cloudfoundry.org/app-autoscaler/src/autoscaler/cf"
	"code.cloudfoundry.org/app-autoscaler/src/autoscaler/fakes"
	"code.cloudfoundry.org/app-autoscaler/src/autoscaler/helpers"
	"code.cloudfoundry.org/app-autoscaler/src/autoscaler/models"
)

const (
	CLIENT_ID                         = "client-id"
	CLIENT_SECRET                     = "client-secret"
	TEST_APP_ID                       = "deadbeef-dead-beef-dead-beef00000075"
	TEST_BEARER_TOKEN                 = "testusertoken"
	TEST_USER_TOKEN                   = "bearer " + TEST_BEARER_TOKEN
	INVALID_USER_TOKEN                = "bearer invalid_user_token invalid_user_token"
	INVALID_USER_TOKEN_WITHOUT_BEARER = "not-bearer testusertoken"
	TEST_INVALID_USER_TOKEN           = "bearer testinvalidusertoken"
	TEST_CLIENT_TOKEN                 = "client-token"
	TEST_USER_ID                      = "test-user-id"
	TEST_METRIC_TYPE                  = "test_metric"
	TEST_METRIC_UNIT                  = "test_unit"
)

var (
	serverProcess ifrit.Process
	serverUrl     *url.URL
	cfServerUrl   *url.URL
	healthUrl     *url.URL
	conf          *config.Config

	infoBytes  []byte
	httpClient *http.Client

	scalingEngineServer    *ghttp.Server
	metricsCollectorServer *ghttp.Server
	eventGeneratorServer   *ghttp.Server
	schedulerServer        *ghttp.Server

	scalingEngineStatus    int
	metricsCollectorStatus int
	eventGeneratorStatus   int
	schedulerStatus        int
	schedulerErrJson       string

	scalingEngineResponse    internalscalinghistory.History
	metricsCollectorResponse []models.AppInstanceMetric
	eventGeneratorResponse   []models.AppMetric

	fakeCFClient        *fakes.FakeCFClient
	fakePolicyDB        *fakes.FakePolicyDB
	fakeBindingDB       *fakes.FakeBindingDB
	fakeRateLimiter     *fakes.FakeLimiter
	httpStatusCollector *fakes.FakeHTTPStatusCollector
	fakeCredentials     *fakes.FakeCredentials
	fakeBrokerServer    *fakes.FakeBrokerServer
	checkBindingFunc    api.CheckBindingFunc
	hasBinding          = true
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

	testCertDir := testhelpers.TestCertFolder()

	conf = &config.Config{
		Logging: helpers.LoggingConfig{
			Level: "debug",
		},
		Server: helpers.ServerConfig{
			Port: 12000 + GinkgoParallelProcess(),
		},
		Health: helpers.HealthConfig{
			ServerConfig: helpers.ServerConfig{
				Port: 13000 + GinkgoParallelProcess(),
			},
		},
		CFServer: helpers.ServerConfig{
			Port: 14000 + GinkgoParallelProcess(),
		},
		PolicySchemaPath: "../policyvalidator/policy_json.schema.json",
		Scheduler: config.SchedulerConfig{
			SchedulerURL: schedulerServer.URL(),
		},
		InfoFilePath: "../exampleconfig/info-file.json",
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
		MetricsForwarder: config.MetricsForwarderConfig{
			MetricsForwarderUrl: "http://localhost:8088",
		},
		CF: cf.Config{
			API:          "http://api.bosh-lite.com",
			ClientID:     CLIENT_ID,
			Secret:       CLIENT_SECRET,
			ClientConfig: cf.ClientConfig{SkipSSLValidation: true},
		},
		APIClientId: "api-client-id",
	}

	// verify MetricCollector certs
	_, err := os.ReadFile(conf.EventGenerator.TLSClientCerts.KeyFile)
	Expect(err).NotTo(HaveOccurred())

	_, err = os.ReadFile(conf.EventGenerator.TLSClientCerts.CertFile)
	Expect(err).NotTo(HaveOccurred())

	_, err = os.ReadFile(conf.EventGenerator.TLSClientCerts.CACertFile)
	Expect(err).NotTo(HaveOccurred())

	// verify ScalingEngine certs
	_, err = os.ReadFile(conf.ScalingEngine.TLSClientCerts.KeyFile)
	Expect(err).NotTo(HaveOccurred())

	_, err = os.ReadFile(conf.ScalingEngine.TLSClientCerts.CertFile)
	Expect(err).NotTo(HaveOccurred())

	_, err = os.ReadFile(conf.ScalingEngine.TLSClientCerts.CACertFile)
	Expect(err).NotTo(HaveOccurred())

	fakePolicyDB = &fakes.FakePolicyDB{}
	fakeBindingDB = &fakes.FakeBindingDB{}
	checkBindingFunc = func(appId string) bool {
		return hasBinding
	}
	fakeCFClient = &fakes.FakeCFClient{}
	httpStatusCollector = &fakes.FakeHTTPStatusCollector{}
	fakeRateLimiter = &fakes.FakeLimiter{}
	fakeCredentials = &fakes.FakeCredentials{}
	fakeBrokerServer = &fakes.FakeBrokerServer{}

	serverUrl, err = url.Parse("http://127.0.0.1:" + strconv.Itoa(conf.Server.Port))
	Expect(err).NotTo(HaveOccurred())

	cfServerUrl, err = url.Parse("http://127.0.0.1:" + strconv.Itoa(conf.CFServer.Port))
	Expect(err).NotTo(HaveOccurred())

	healthUrl, err = url.Parse("http://127.0.0.1:" + strconv.Itoa(conf.Health.ServerConfig.Port))
	Expect(err).NotTo(HaveOccurred())

	httpClient = &http.Client{}

	infoBytes, err = os.ReadFile("../exampleconfig/info-file.json")
	Expect(err).NotTo(HaveOccurred())

	scalingHistoryPathMatcher, err := regexp.Compile(`/v1/apps/[A-Za-z0-9\-]+/scaling_histories`)
	Expect(err).NotTo(HaveOccurred())
	scalingEngineServer.RouteToHandler(http.MethodGet, scalingHistoryPathMatcher, ghttp.RespondWithJSONEncodedPtr(&scalingEngineStatus, &scalingEngineResponse))

	metricsCollectorPathMatcher, err := regexp.Compile(`/v1/apps/[A-Za-z0-9\-]+/metric_histories/[a-zA-Z0-9_]+`)
	Expect(err).NotTo(HaveOccurred())
	metricsCollectorServer.RouteToHandler(http.MethodGet, metricsCollectorPathMatcher, ghttp.RespondWithJSONEncodedPtr(&metricsCollectorStatus, &metricsCollectorResponse))

	eventGeneratorPathMatcher, err := regexp.Compile(`/v1/apps/[A-Za-z0-9\-]+/aggregated_metric_histories/[a-zA-Z0-9_]+`)
	Expect(err).NotTo(HaveOccurred())
	eventGeneratorServer.RouteToHandler(http.MethodGet, eventGeneratorPathMatcher, ghttp.RespondWithJSONEncodedPtr(&eventGeneratorStatus, &eventGeneratorResponse))

	schedulerPathMatcher, err := regexp.Compile(`/v1/apps/[A-Za-z0-9\-]+/schedules`)
	Expect(err).NotTo(HaveOccurred())
	schedulerErrJson = "{}"
	schedulerServer.RouteToHandler(http.MethodPut, schedulerPathMatcher, ghttp.RespondWithPtr(&schedulerStatus, &schedulerErrJson))
	schedulerServer.RouteToHandler(http.MethodDelete, schedulerPathMatcher, ghttp.RespondWithPtr(&schedulerStatus, &schedulerErrJson))
})

var _ = AfterSuite(func() {
	scalingEngineServer.Close()
	metricsCollectorServer.Close()
	eventGeneratorServer.Close()
})
