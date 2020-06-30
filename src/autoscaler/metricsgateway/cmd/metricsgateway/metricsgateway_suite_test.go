package main_test

import (
	"autoscaler/models"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"code.cloudfoundry.org/go-loggregator/rpc/loggregator_v2"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/gbytes"
	"github.com/onsi/gomega/gexec"
	"github.com/onsi/gomega/ghttp"
	"google.golang.org/grpc/grpclog"
	"github.com/jmoiron/sqlx"
	"gopkg.in/yaml.v2"

	"testing"

	"autoscaler/db"
	"autoscaler/helpers"
	"autoscaler/metricsgateway/config"
	"autoscaler/testhelpers"
)

var (
	conf             config.Config
	configFile       *os.File
	mgPath           string
	healthHttpClient *http.Client
	healthport       int

	testCertDir              = "../../../../../test-certs"
	loggregatorServerCrtPath = filepath.Join(testCertDir, "reverselogproxy.crt")
	loggregatorServerKeyPath = filepath.Join(testCertDir, "reverselogproxy.key")
	loggregatorClientCrtPath = filepath.Join(testCertDir, "reverselogproxy_client.crt")
	loggregatorClientKeyPath = filepath.Join(testCertDir, "reverselogproxy_client.key")

	metricServerCrtPath       = filepath.Join(testCertDir, "metricserver.crt")
	metricServerKeyPath       = filepath.Join(testCertDir, "metricserver.key")
	metricServerClientCrtPath = filepath.Join(testCertDir, "metricserver_client.crt")
	metricServerClientKeyPath = filepath.Join(testCertDir, "metricserver_client.key")

	caPath = filepath.Join(testCertDir, "autoscaler-ca.crt")

	fakeLoggregator testhelpers.FakeEventProducer
	rlpAddr         string

	fakeMetricServer    *ghttp.Server
	metricServerAddress string

	testAppId = "test-app-id"
	envelopes = []*loggregator_v2.Envelope{
		{
			SourceId: testAppId,
			DeprecatedTags: map[string]*loggregator_v2.Value{
				"peer_type": {Data: &loggregator_v2.Value_Text{Text: "Client"}},
			},
			Message: &loggregator_v2.Envelope_Timer{
				Timer: &loggregator_v2.Timer{
					Name:  "http",
					Start: 1542325492043447110,
					Stop:  1542325492045491009,
				},
			},
		},
	}
	messageChan  chan []byte
	pingPongChan chan int
)

func TestMetricsgateway(t *testing.T) {
	grpclog.SetLogger(log.New(GinkgoWriter, "", 0))
	log.SetOutput(GinkgoWriter)
	RegisterFailHandler(Fail)
	RunSpecs(t, "Metricsgateway Suite")
}

var _ = SynchronizedBeforeSuite(func() []byte {
	eg, err := gexec.Build("autoscaler/metricsgateway/cmd/metricsgateway", "-race")
	Expect(err).NotTo(HaveOccurred())
	initDB()
	return []byte(eg)
}, func(pathByte []byte) {
	mgPath = string(pathByte)
	initFakeServers()
	initConfig()
	healthHttpClient = &http.Client{}
})

var _ = SynchronizedAfterSuite(func() {
	os.Remove(configFile.Name())
	fakeLoggregator.Stop()
	fakeMetricServer.Close()
}, func() {
	gexec.CleanupBuildArtifacts()
})

func initDB() {
	database, err := db.GetConnection(os.Getenv("DBURL"))
	Expect(err).NotTo(HaveOccurred())

	mgDB, err := sqlx.Open(database.DriverName, database.DSN)
	Expect(err).NotTo(HaveOccurred())

	_, err = mgDB.Exec("DELETE from policy_json")
	Expect(err).NotTo(HaveOccurred())

	policy := fmt.Sprintf(`
		{
		   "instance_min_count":1,
		   "instance_max_count":5,
		   "scaling_rules":[
		      {
		         "metric_type":"a-metric-type",
		         "breach_duration_secs":120,
		         "threshold":300,
		         "operator":">",
		         "cool_down_secs":300,
		         "adjustment":"+1"
		      }
		   ]
		}`)
	query := mgDB.Rebind("INSERT INTO policy_json(app_id, policy_json, guid) values(?, ?, ?)")
	_, err = mgDB.Exec(query, testAppId, policy, "1234")
	Expect(err).NotTo(HaveOccurred())

	err = mgDB.Close()
	Expect(err).NotTo(HaveOccurred())
}

func initConfig() {

	healthport = 8000 + GinkgoParallelNode()
	conf = config.Config{
		Logging: helpers.LoggingConfig{
			Level: "info",
		},
		EnvelopChanSize:   500,
		NozzleCount:       1,
		MetricServerAddrs: []string{metricServerAddress},
		AppManager: config.AppManagerConfig{
			AppRefreshInterval: 10 * time.Second,
			PolicyDB: db.DatabaseConfig{
				URL:                   os.Getenv("DBURL"),
				MaxOpenConnections:    10,
				MaxIdleConnections:    5,
				ConnectionMaxLifetime: 60 * time.Second,
			},
		},
		Emitter: config.EmitterConfig{
			BufferSize:         500,
			KeepAliveInterval:  1 * time.Second,
			HandshakeTimeout:   1 * time.Second,
			MaxSetupRetryCount: 3,
			MaxCloseRetryCount: 3,
			RetryDelay:         1 * time.Second,
			MetricsServerClientTLS: &models.TLSCerts{
				KeyFile:    metricServerClientKeyPath,
				CertFile:   metricServerClientCrtPath,
				CACertFile: caPath,
			},
		},
		Nozzle: config.NozzleConfig{
			RLPAddr: rlpAddr,
			ShardID: "autoscaler",
			RLPClientTLS: &models.TLSCerts{
				KeyFile:    loggregatorClientKeyPath,
				CertFile:   loggregatorClientCrtPath,
				CACertFile: caPath,
			},
		},
		Health: models.HealthConfig{
			Port:                healthport,
			HealthCheckUsername: "metricsgatewayhealthcheckuser",
			HealthCheckPassword: "metricsgatewayhealthcheckpassword",
		},
	}
	configFile = writeConfig(&conf)

}

func initFakeServers() {
	fakeLoggregator, err := testhelpers.NewFakeEventProducer(loggregatorServerCrtPath, loggregatorServerKeyPath, caPath, 500*time.Millisecond)
	Expect(err).NotTo(HaveOccurred())
	fakeLoggregator.Start()
	rlpAddr = fakeLoggregator.GetAddr()
	fakeLoggregator.SetEnvelops(envelopes)

	fakeMetricServer = ghttp.NewServer()
	metricServerAddress = strings.Replace(fakeMetricServer.URL(), "http", "ws", 1)

	messageChan = make(chan []byte, 10)
	pingPongChan = make(chan int, 10)
	wsh := testhelpers.NewWebsocketHandler(messageChan, pingPongChan, 5*time.Second)
	fakeMetricServer.RouteToHandler("GET", "/v1/envelopes", wsh.ServeWebsocket)
}
func writeConfig(c *config.Config) *os.File {
	cfg, err := ioutil.TempFile("", "mg")
	Expect(err).NotTo(HaveOccurred())
	defer cfg.Close()
	configBytes, err1 := yaml.Marshal(c)
	ioutil.WriteFile(cfg.Name(), configBytes, 0777)
	Expect(err1).NotTo(HaveOccurred())
	return cfg

}

type MetricsGatewayRunner struct {
	configPath string
	startCheck string
	Session    *gexec.Session
}

func NewMetricsGatewayRunner() *MetricsGatewayRunner {
	return &MetricsGatewayRunner{
		configPath: configFile.Name(),
		startCheck: "metricsgateway.started",
	}
}

func (mg *MetricsGatewayRunner) Start() {
	mgSession, err := gexec.Start(exec.Command(
		mgPath,
		"-c",
		mg.configPath,
	),
		gexec.NewPrefixedWriter("\x1b[32m[o]\x1b[32m[mg]\x1b[0m ", GinkgoWriter),
		gexec.NewPrefixedWriter("\x1b[91m[e]\x1b[32m[mg]\x1b[0m ", GinkgoWriter),
	)
	Expect(err).NotTo(HaveOccurred())

	if mg.startCheck != "" {
		Eventually(mgSession.Buffer, 2).Should(gbytes.Say(mg.startCheck))
	}

	mg.Session = mgSession
}

func (mg *MetricsGatewayRunner) Interrupt() {
	if mg.Session != nil {
		mg.Session.Interrupt().Wait(5 * time.Second)
	}
}

func (mg *MetricsGatewayRunner) KillWithFire() {
	if mg.Session != nil {
		mg.Session.Kill().Wait(5 * time.Second)
	}
}
