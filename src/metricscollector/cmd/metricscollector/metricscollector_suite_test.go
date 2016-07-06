package main_test

import (
	"io/ioutil"
	"log"
	"metricscollector/cf"
	"metricscollector/config"
	"mime/multipart"
	"net/http"
	"os"
	"os/exec"
	"strings"
	"time"

	"github.com/cloudfoundry-incubator/candiedyaml"
	"github.com/cloudfoundry/sonde-go/events"
	"github.com/gogo/protobuf/proto"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/gbytes"
	"github.com/onsi/gomega/gexec"
	"github.com/onsi/gomega/ghttp"

	"testing"
)

var (
	mcPath     string
	cfg        config.Config
	mcPort     int
	configFile *os.File
	ccNOAAUAA  *ghttp.Server
)

func TestMetricsCollector(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "MetricsCollector Suite")
}

var _ = SynchronizedBeforeSuite(func() []byte {
	mc, err := gexec.Build("metricscollector/cmd/metricscollector", "-race")
	Expect(err).NotTo(HaveOccurred())

	return []byte(mc)
}, func(pathsByte []byte) {
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

	ccNOAAUAA.RouteToHandler("GET", "/apps/an-app-id/containermetrics",
		func(rw http.ResponseWriter, r *http.Request) {
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

	cfg.Cf = config.CfConfig{
		Api:       ccNOAAUAA.URL(),
		GrantType: config.GrantTypePassword,
		Username:  "admin",
		Password:  "admin",
	}

	mcPort = 7000 + GinkgoParallelNode()
	cfg.Server.Port = mcPort
	cfg.Logging.Level = "debug"

	cfg.Db.MetricsDbUrl = os.Getenv("DBURL")
	cfg.Db.PolicyDbUrl = os.Getenv("DBURL")

	configFile = writeConfig(&cfg)
})

var _ = SynchronizedAfterSuite(func() {
	ccNOAAUAA.Close()
	os.Remove(configFile.Name())
}, func() {
	gexec.CleanupBuildArtifacts()
})

func writeConfig(c *config.Config) *os.File {
	cfg, err := ioutil.TempFile("", "mc")
	Expect(err).NotTo(HaveOccurred())
	defer cfg.Close()
	e := candiedyaml.NewEncoder(cfg)
	err = e.Encode(c)
	Expect(err).NotTo(HaveOccurred())

	return cfg
}

type MetricsCollectorRunner struct {
	configPath string
	startCheck string
	Session    *gexec.Session
}

func NewMetricsCollectorRunner() *MetricsCollectorRunner {
	return &MetricsCollectorRunner{
		configPath: configFile.Name(),
		startCheck: "metricscollector.started",
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
