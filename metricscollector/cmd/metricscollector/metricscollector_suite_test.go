package main_test

import (
	"database/sql"
	"io/ioutil"
	"log"
	"mime/multipart"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"code.cloudfoundry.org/cfhttp"
	"code.cloudfoundry.org/consuladapter/consulrunner"
	"github.com/cloudfoundry/sonde-go/events"
	"github.com/gogo/protobuf/proto"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/gbytes"
	"github.com/onsi/gomega/gexec"
	"github.com/onsi/gomega/ghttp"
	"gopkg.in/yaml.v2"

	"autoscaler/cf"
	"autoscaler/db"
	"autoscaler/metricscollector/config"

	"code.cloudfoundry.org/locket"
)

var (
	mcPath         string
	cfg            config.Config
	mcPort         int
	configFile     *os.File
	ccNOAAUAA      *ghttp.Server
	isTokenExpired bool
	eLock          *sync.Mutex
	httpClient     *http.Client
	consulRunner   *consulrunner.ClusterRunner
)

func TestMetricsCollector(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "MetricsCollector Suite")
}

var _ = SynchronizedBeforeSuite(func() []byte {
	mc, err := gexec.Build("autoscaler/metricscollector/cmd/metricscollector", "-race")
	Expect(err).NotTo(HaveOccurred())

	return []byte(mc)
}, func(pathsByte []byte) {

	consulRunner = consulrunner.NewClusterRunner(
		consulrunner.ClusterRunnerConfig{
			StartingPort: 9001 + GinkgoParallelNode()*consulrunner.PortOffsetLength,
			NumNodes:     1,
			Scheme:       "http",
		},
	)
	consulRunner.Start()
	consulRunner.WaitUntilReady()
	mcPath = string(pathsByte)

	ccNOAAUAA = ghttp.NewServer()
	ccNOAAUAA.RouteToHandler("GET", "/v2/info", ghttp.RespondWithJSONEncoded(http.StatusOK,
		cf.Endpoints{
			AuthEndpoint:    ccNOAAUAA.URL(),
			DopplerEndpoint: strings.Replace(ccNOAAUAA.URL(), "http", "ws", 1),
		}))

	ccNOAAUAA.RouteToHandler("POST", "/oauth/token", ghttp.RespondWithJSONEncoded(http.StatusOK,
		cf.Tokens{}))

	message1 := marshalMessage(createContainerMetric("an-app-id", 0, 3.0, 1024, 2048, 0))
	message2 := marshalMessage(createContainerMetric("an-app-id", 1, 4.0, 1024, 2048, 0))
	message3 := marshalMessage(createContainerMetric("an-app-id", 2, 5.0, 1024, 2048, 0))

	messages := map[string][][]byte{}
	messages["an-app-id"] = [][]byte{message1, message2, message3}

	eLock = &sync.Mutex{}
	ccNOAAUAA.RouteToHandler("GET", "/apps/an-app-id/containermetrics",
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

	cfg.Cf = cf.CfConfig{
		Api:       ccNOAAUAA.URL(),
		GrantType: cf.GrantTypePassword,
		Username:  "admin",
		Password:  "admin",
	}

	testCertDir := "../../../../../test-certs"
	mcPort = 7000 + GinkgoParallelNode()
	cfg.Server.Port = mcPort
	cfg.Server.TLS.KeyFile = filepath.Join(testCertDir, "metricscollector.key")
	cfg.Server.TLS.CertFile = filepath.Join(testCertDir, "metricscollector.crt")
	cfg.Server.TLS.CACertFile = filepath.Join(testCertDir, "autoscaler-ca.crt")

	cfg.Logging.Level = "debug"

	cfg.Db.InstanceMetricsDbUrl = os.Getenv("DBURL")
	cfg.Db.PolicyDbUrl = os.Getenv("DBURL")

	cfg.Collector.PollInterval = 10 * time.Second
	cfg.Collector.RefreshInterval = 30 * time.Second

	cfg.Lock.ConsulClusterConfig = consulRunner.ConsulCluster()
	cfg.Lock.LockRetryInterval = locket.RetryInterval
	cfg.Lock.LockTTL = locket.DefaultSessionTTL

	configFile = writeConfig(&cfg)

	mcDB, err := sql.Open(db.PostgresDriverName, os.Getenv("DBURL"))
	Expect(err).NotTo(HaveOccurred())

	_, err = mcDB.Exec("DELETE FROM appinstancemetrics")
	Expect(err).NotTo(HaveOccurred())

	_, err = mcDB.Exec("DELETE from policy_json")
	Expect(err).NotTo(HaveOccurred())

	policy := `
		{
 			"instance_min_count": 1,
  			"instance_max_count": 5
		}`
	query := "INSERT INTO policy_json(app_id, policy_json, guid) values($1, $2, $3)"
	_, err = mcDB.Exec(query, "an-app-id", policy, "1234")
	Expect(err).NotTo(HaveOccurred())

	err = mcDB.Close()
	Expect(err).NotTo(HaveOccurred())

	tlsConfig, err := cfhttp.NewTLSConfig(
		filepath.Join(testCertDir, "eventgenerator.crt"),
		filepath.Join(testCertDir, "eventgenerator.key"),
		filepath.Join(testCertDir, "autoscaler-ca.crt"))
	Expect(err).NotTo(HaveOccurred())
	httpClient = &http.Client{
		Transport: &http.Transport{
			TLSClientConfig: tlsConfig,
		},
	}
})

var _ = SynchronizedAfterSuite(func() {
	if consulRunner != nil {
		consulRunner.Stop()
	}
	ccNOAAUAA.Close()
	os.Remove(configFile.Name())
}, func() {
	gexec.CleanupBuildArtifacts()
})

func writeConfig(c *config.Config) *os.File {
	cfg, err := ioutil.TempFile("", "mc")
	Expect(err).NotTo(HaveOccurred())
	defer cfg.Close()

	bytes, err := yaml.Marshal(c)
	Expect(err).NotTo(HaveOccurred())

	_, err = cfg.Write(bytes)
	Expect(err).NotTo(HaveOccurred())

	return cfg
}

type MetricsCollectorRunner struct {
	configPath        string
	startCheck        string
	acquiredLockCheck string
	Session           *gexec.Session
}

func NewMetricsCollectorRunner() *MetricsCollectorRunner {
	return &MetricsCollectorRunner{
		configPath:        configFile.Name(),
		startCheck:        "metricscollector.started",
		acquiredLockCheck: "metricscollector.lock.acquire-lock-succeeded",
	}
}

func (mc *MetricsCollectorRunner) Start() {
	mcSession, err := gexec.Start(exec.Command(
		mcPath,
		"-c",
		mc.configPath,
	),
		gexec.NewPrefixedWriter("\x1b[32m[o]\x1b[32m[mc]\x1b[0m ", GinkgoWriter),
		gexec.NewPrefixedWriter("\x1b[91m[e]\x1b[32m[mc]\x1b[0m ", GinkgoWriter),
	)
	Expect(err).NotTo(HaveOccurred())

	if mc.startCheck != "" {
		Eventually(mcSession.Buffer(), 2).Should(gbytes.Say(mc.startCheck))
	}

	mc.Session = mcSession
}

func (mc *MetricsCollectorRunner) Interrupt() {
	if mc.Session != nil {
		mc.Session.Interrupt().Wait(5 * time.Second)
	}
}

func (mc *MetricsCollectorRunner) KillWithFire() {
	if mc.Session != nil {
		mc.Session.Kill().Wait(5 * time.Second)
	}
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
