package main_test

import (
	"database/sql"
	"io/ioutil"
	"log"
	"mime/multipart"
	"net/http"
	"os"
	"os/exec"
	"strings"
	"sync"
	"testing"
	"time"

	"code.cloudfoundry.org/cfhttp"
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

	mcPort = 7000 + GinkgoParallelNode()
	cfg.Server.Port = mcPort
	cfg.Server.EnableSSL = true

	cfg.Logging.Level = "debug"

	cfg.Db.InstanceMetricsDbUrl = os.Getenv("DBURL")
	cfg.Db.PolicyDbUrl = os.Getenv("DBURL")

	cfg.Collector.PollInterval = 10 * time.Second
	cfg.Collector.RefreshInterval = 30 * time.Second

	initSSLConfig(&cfg)

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
	query := "INSERT INTO policy_json(app_id, policy_json) values($1, $2)"
	_, err = mcDB.Exec(query, "an-app-id", policy)
	Expect(err).NotTo(HaveOccurred())

	err = mcDB.Close()
	Expect(err).NotTo(HaveOccurred())

	tlsConfig, err := cfhttp.NewTLSConfig(cfg.SSL.CertFile, cfg.SSL.KeyFile, cfg.SSL.CACertFile)
	Expect(err).NotTo(HaveOccurred())
	httpClient = &http.Client{
		Transport: &http.Transport{
			TLSClientConfig: tlsConfig,
		},
	}

})

var _ = SynchronizedAfterSuite(func() {
	ccNOAAUAA.Close()
	os.Remove(configFile.Name())
}, func() {
	gexec.CleanupBuildArtifacts()
})

func initSSLConfig(c *config.Config) {
	caCertFile, err := ioutil.TempFile("", "ca_cert")
	Expect(err).NotTo(HaveOccurred())
	defer caCertFile.Close()
	_, err = caCertFile.Write(testCaCert)
	Expect(err).NotTo(HaveOccurred())
	c.SSL.CACertFile = caCertFile.Name()

	certFile, err := ioutil.TempFile("", "cert")
	Expect(err).NotTo(HaveOccurred())
	defer certFile.Close()
	_, err = certFile.Write(testCert)
	Expect(err).NotTo(HaveOccurred())
	c.SSL.CertFile = certFile.Name()

	keyFile, err := ioutil.TempFile("", "key")
	Expect(err).NotTo(HaveOccurred())
	defer keyFile.Close()
	_, err = keyFile.Write(testKey)
	Expect(err).NotTo(HaveOccurred())
	c.SSL.KeyFile = keyFile.Name()

}

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

var testCaCert = []byte(`
-----BEGIN CERTIFICATE-----
MIIDIDCCAgigAwIBAgIJAKLEYnHtFqiqMA0GCSqGSIb3DQEBBQUAMBQxEjAQBgNV
BAMTCWxvY2FsaG9zdDAeFw0xNjExMjIyMDAxMDVaFw0zMDA4MDEyMDAxMDVaMBQx
EjAQBgNVBAMTCWxvY2FsaG9zdDCCASIwDQYJKoZIhvcNAQEBBQADggEPADCCAQoC
ggEBAMTaBQudTO53Erf2f0OPMfuYZzPGHEwnGnae40XkOTO1kycwNGnS09vHFp/Y
8F7Mp/HwEy5AnlOEKJW6nJtvZKb9fgoqZIk/K3Lt2Q8NBqv3HEJve9Ub8HMEAUCX
4PNeFCjGEDwsB42Y2HLMl5mNiQzppCe7Ja186MDBvBdfbYYAzDeYk47eTtKY+SN+
Debf57yZlJKWcbwiLy2fJm8uaFjMnR17/gYgaOk7JPLL46vPnLqXACih9avnvJEr
G2rhsqGa8FOvphLqVfFQRdfwjsqJD0R3Ws9YVOqPrzieowdHY1Ojw7xCfaNjX2lh
K9+bA0V8Or4V+c0l7syYFEbnWeECAwEAAaN1MHMwHQYDVR0OBBYEFJJyZv3dEuPw
AtOT0csj/G252FSqMEQGA1UdIwQ9MDuAFJJyZv3dEuPwAtOT0csj/G252FSqoRik
FjAUMRIwEAYDVQQDEwlsb2NhbGhvc3SCCQCixGJx7RaoqjAMBgNVHRMEBTADAQH/
MA0GCSqGSIb3DQEBBQUAA4IBAQBrGHc0OoeBI0CK27frNoYSPmLwdwHFiKfNE31I
5+6OZVAsyVP4rI4IZFoIQsOTu8hmbZiRip8X9blp8xO7uzJNi1o3o0tB5ryseuTc
UIC7zx8uS41TGCFuoZbatkyeePK3k7afQ+HsbR94SGh20Je6SaoeYnNFSlQBZvGL
+/oFgoLAQ+doj5RM9v/gWOE1jb2QIekYrS4FXB5BArik68eM7pbAoD+FAkHyfRV/
JC7WjFBTWE8GM2U4PKSLr/N3KYHLezNTcKIIRviW64kpy198y5wtDuttc3UMKFWB
ojtR84p+HKknY8Vi6Felx4elfitEkMGAe420KgtWsrhUG8BM
-----END CERTIFICATE-----
`)

var testCert = []byte(`
-----BEGIN CERTIFICATE-----
MIICpDCCAYwCCQDhR/ajlwZ2azANBgkqhkiG9w0BAQUFADAUMRIwEAYDVQQDEwls
b2NhbGhvc3QwHhcNMTYxMTIyMjAwMjA5WhcNMzAwODAxMjAwMjA5WjAUMRIwEAYD
VQQDEwlsb2NhbGhvc3QwggEiMA0GCSqGSIb3DQEBAQUAA4IBDwAwggEKAoIBAQDJ
aXrW7G1V5m1KBr1bCkAKu0GrgkTTcaTH+tuTxT9zn6qjPCY953sxBCiuvSfswv3x
ndqG0zjpayG04ly8tfB/b9R61o6UUcVabeSoeepn8M+SbMV9MbDBNXazXdEcMLR3
LTq7lE8iRs+yR++7VyazXFOBCjHuABcuNbvvXuO1WQ0GTUQi1vWXnLyEcy3lv0qP
pcHtsSMQ087eYdFtWrinTu9OUXFg0MZAHj75VWaMkJqnYSX5VAU7BYE0nK3OcABs
QA9qE4hFO3Pd5o+Gh6+au8B+j30+8qfU6uvchfNNIjWDIRY0cRPjCzY21Q8XZHBE
OGKUs2qkUEGTzROkPmdnAgMBAAEwDQYJKoZIhvcNAQEFBQADggEBACVowTB2jtUY
0w/ItduwFrZOcLbq10SKvOrA4BS0zNxqqkkO+nCVDRLyvZ+PN334ZKa4AkNjlsMc
WMpUh2kB7l0pHYvOQlSLvf8NomTXl71QV5pdGVJrYDKsGEFsZWeSNB/Hn8v2b+uV
HQVIyXuOzmNm2qMTqCg0xXlSHHfgO/I8cENRl/XPjFLOjOPE82VXyzJltmhLPocL
WHPM/64NeO+n/kRw2fmUjHjG59DtmadjIaX6Ab6BfBkt5nxGRxbxQ/THU4ZDI7IX
LRU3vXhT8lF+ug86I5S4Em+fpZVjKOlF2w5jm0ur0av59ZfjpKdBXkFXDSqs+9Pw
79KHmelyK6M=
-----END CERTIFICATE-----
`)

var testKey = []byte(`
-----BEGIN RSA PRIVATE KEY-----
MIIEpAIBAAKCAQEAyWl61uxtVeZtSga9WwpACrtBq4JE03Gkx/rbk8U/c5+qozwm
Ped7MQQorr0n7ML98Z3ahtM46WshtOJcvLXwf2/UetaOlFHFWm3kqHnqZ/DPkmzF
fTGwwTV2s13RHDC0dy06u5RPIkbPskfvu1cms1xTgQox7gAXLjW7717jtVkNBk1E
Itb1l5y8hHMt5b9Kj6XB7bEjENPO3mHRbVq4p07vTlFxYNDGQB4++VVmjJCap2El
+VQFOwWBNJytznAAbEAPahOIRTtz3eaPhoevmrvAfo99PvKn1Orr3IXzTSI1gyEW
NHET4ws2NtUPF2RwRDhilLNqpFBBk80TpD5nZwIDAQABAoIBAFVLkDP4iAz6uonA
9OvbGWfyCUuCrXcGB4yRFfAsdkSW94KNdHx+zVLiuf/WJ1RC95wdB3BVfyKKtgmO
W0unmSO+zjL/Lf5t6q/zHgSjLLu75YvS+IeeRYZtB8nKD0Zq8eIOi3LoyeJwLoEH
qeSiccc/cDWThbWO4EI3i1FB1Bj1zr2YZrUD0Ztoevgq7VoNKjmZjpqrkU8gThfB
Q7WN0bjFj2KIeDc/Ea/YNtOzZXi0xmI1Un32eqJ5Aor6X9iO3pmj6OdQTDdPrPdQ
y6q5DCvoz9Etyqiu4VQWPU9uob3jjNv6gDUPVlyeeX4TxpvROYUMBYmBrOkX2DtA
m6jZSbECgYEA8P2Gx4x7qqlx8TCCzgfjm9wCp8DB64Gmx0c1m+lTLBqEbzHw1Nrv
77qHaqqmQKR3EdpEPYu438rj5OPyvOPrzPtw/nXkC71EAY5IJxiHT62lEI/e8lNy
BjL4Pt3k1P42Hx+kJtitBS2QGm1Cj5l7y3Ll3FblahNoBswn+7Dv+pMCgYEA1fTl
g1g4N7u656hgHf+98Sl7znttnxmvey8PLVqxPQ2QL34+iAxss1VeGJ7NBc55NoiQ
StqMu6ZMGt33J+s1+LwujAa25ceFLaha19Jn/NpIAC4sDVdoaMk6L1mKZfkgyTBg
Bo0noCIcuoaKK/bHvTQA+tE/KDjBssqD15wKIF0CgYEAq60AOeGzK4WgXSCg3mMy
WxLldVnVC4+GHwp0f0g9bvrJA2nBVfFN4iqwU2WUIBLJnBcwa+PAZPTlWmakwrlf
ftxUx4F2XoKLEsyoS+mmYzWhlGXZfinB3farcCWk4bLjHVLuHppWz4yAzsKtGx1M
2kdUxp3EusduYQFJLn0CyC8CgYBm0UcyNUTe06JgDPQTtoCK0gqjEYF+gZNouRas
Dvc5hbkSebKHIqFiFNYhMr8H2U86u6nLrvcOfj34c06AqnHHVHdx5xAhoB4J8Oum
53/9bNBI9edJigWsxXbmpjryAiSVSl/7Bf5S39G6eUkRE4itDb6iycd2BsBesR9m
j5/2KQKBgQDCpkxsGII5FvoZKarQhsWHINdxsQ5mrBX3Z5BrNZLxVW08LrQVCbij
b5D3cprXT/y3tc3J3a/K4yDQUEtoRpslQUUUkD6NVTZoXwsg6NrxffuOZIQIzSrS
P/t3oLbFG4ICv3ywOzHIiJZnMGai70YohLGI0SpTiGftCBJjzO/9/g==
-----END RSA PRIVATE KEY-----
`)
