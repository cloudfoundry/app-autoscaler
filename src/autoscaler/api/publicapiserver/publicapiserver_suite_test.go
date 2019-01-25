package publicapiserver_test

import (
	"autoscaler/api/config"
	"autoscaler/api/publicapiserver"
	"autoscaler/cf"
	"autoscaler/fakes"
	"autoscaler/helpers"
	"autoscaler/models"
	"io/ioutil"
	"net/http"
	"net/url"
	"path/filepath"
	"regexp"
	"strconv"

	"github.com/onsi/gomega/ghttp"

	"code.cloudfoundry.org/lager"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/tedsuo/ifrit"
	"github.com/tedsuo/ifrit/ginkgomon"

	"testing"
)

const (
	CLIENT_ID        = "client-id"
	CLIENT_SECRET    = "client-secret"
	TEST_APP_ID      = "test-app-id"
	TEST_USER_TOKEN  = "bearer testusertoken"
	TEST_USER_ID     = "test-user-id"
	TEST_METRIC_TYPE = "test_metric"
	TEST_METRIC_UNIT = "test_unit"
)

var (
	serverProcess ifrit.Process
	serverUrl     *url.URL
	httpClient    *http.Client
	conf          *config.Config
	logger        lager.Logger
	infoBytes     []byte

	ccServer    *ghttp.Server
	tokenServer *ghttp.Server

	scalingEngineServer    *ghttp.Server
	metricsCollectorServer *ghttp.Server
	eventGeneratorServer   *ghttp.Server

	scalingEngineStatus    int
	metricsCollectorStatus int
	eventGeneratorStatus   int

	scalingEngineResponse    []models.AppScalingHistory
	metricsCollectorResponse []models.AppInstanceMetric
	eventGeneratorResponse   []models.AppMetric
)

type info struct {
	TokenEndpoint string `json:"token_endpoint"`
}

type userInfo struct {
	UserId string `json:"user_id"`
}

type userScope struct {
	Scope []string `json:"scope"`
}

type spaceDeveoper struct {
	Total int `json:"total_results"`
}

func TestPublicapiserver(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Publicapiserver Suite")
}

var _ = BeforeSuite(func() {
	scalingEngineServer = ghttp.NewServer()
	metricsCollectorServer = ghttp.NewServer()
	eventGeneratorServer = ghttp.NewServer()
	ccServer = ghttp.NewServer()
	tokenServer = ghttp.NewServer()

	testCertDir := "../../../../test-certs"
	apiPort := 11000 + GinkgoParallelNode()
	conf = &config.Config{
		Logging: helpers.LoggingConfig{
			Level: "debug",
		},
		PublicApiServer: config.ServerConfig{
			Port: apiPort,
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
			API:               ccServer.URL(),
			GrantType:         cf.GrantTypeClientCredentials,
			ClientID:          CLIENT_ID,
			Secret:            CLIENT_SECRET,
			SkipSSLValidation: true,
		},
	}

	fakePolicyDB := &fakes.FakePolicyDB{}

	httpServer, err := publicapiserver.NewPublicApiServer(lager.NewLogger("test"), conf, fakePolicyDB)
	Expect(err).NotTo(HaveOccurred())

	serverUrl, err = url.Parse("http://127.0.0.1:" + strconv.Itoa(apiPort))
	Expect(err).NotTo(HaveOccurred())

	serverProcess = ginkgomon.Invoke(httpServer)

	httpClient = &http.Client{}

	infoBytes, err = ioutil.ReadFile("../exampleconfig/info-file.json")
	Expect(err).NotTo(HaveOccurred())

	logger = helpers.InitLoggerFromConfig(&conf.Logging, "test")

	ccServer.RouteToHandler(http.MethodGet, "/v2/info", ghttp.RespondWithJSONEncoded(http.StatusOK, info{
		TokenEndpoint: tokenServer.URL(),
	}))

	tokenServer.RouteToHandler(http.MethodGet, "/userinfo", ghttp.RespondWithJSONEncoded(http.StatusOK, userInfo{
		UserId: TEST_USER_ID,
	}))

	tokenServer.RouteToHandler(http.MethodPost, "/check_token", ghttp.RespondWithJSONEncoded(http.StatusOK, userScope{
		Scope: []string{""},
	}))

	spaceDeveloperPathMatcher, _ := regexp.Compile("/v2/users/[A-Za-z0-9\\-]+/spaces")
	ccServer.RouteToHandler(http.MethodGet, spaceDeveloperPathMatcher, ghttp.RespondWithJSONEncoded(http.StatusOK, spaceDeveoper{
		Total: 1,
	}))

	scalingHistoryPathMatcher, err := regexp.Compile("/v1/apps/[A-Za-z0-9\\-]+/scaling_histories")
	Expect(err).NotTo(HaveOccurred())
	scalingEngineServer.RouteToHandler(http.MethodGet, scalingHistoryPathMatcher, ghttp.RespondWithJSONEncodedPtr(&scalingEngineStatus, &scalingEngineResponse))

	metricsCollectorPathMatcher, err := regexp.Compile("/v1/apps/[A-Za-z0-9\\-]+/metric_histories/[a-zA-Z0-9_]+")
	Expect(err).NotTo(HaveOccurred())
	metricsCollectorServer.RouteToHandler(http.MethodGet, metricsCollectorPathMatcher, ghttp.RespondWithJSONEncodedPtr(&metricsCollectorStatus, &metricsCollectorResponse))

	eventGeneratorPathMatcher, err := regexp.Compile("/v1/apps/[A-Za-z0-9\\-]+/aggregated_metric_histories/[a-zA-Z0-9_]+")
	Expect(err).NotTo(HaveOccurred())
	eventGeneratorServer.RouteToHandler(http.MethodGet, eventGeneratorPathMatcher, ghttp.RespondWithJSONEncodedPtr(&eventGeneratorStatus, &eventGeneratorResponse))

})

var _ = AfterSuite(func() {
	ginkgomon.Interrupt(serverProcess)

	scalingEngineServer.Close()
	metricsCollectorServer.Close()
	eventGeneratorServer.Close()

	ccServer.Close()
	tokenServer.Close()

})

func GetTestHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("Success"))
	}
}
