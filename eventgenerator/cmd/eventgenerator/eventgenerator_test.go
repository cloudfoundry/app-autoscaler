package main_test

import (
	"context"
	"crypto/rand"
	"crypto/tls"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"code.cloudfoundry.org/app-autoscaler/src/autoscaler/models"

	rpc "code.cloudfoundry.org/go-log-cache/v2/rpc/logcache_v1"
	"code.cloudfoundry.org/go-loggregator/v9/rpc/loggregator_v2"
	"google.golang.org/grpc"

	"code.cloudfoundry.org/app-autoscaler/src/autoscaler/eventgenerator/config"
	"code.cloudfoundry.org/app-autoscaler/src/autoscaler/helpers"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	. "github.com/onsi/gomega/gbytes"
	. "github.com/onsi/gomega/gexec"
)

var _ = Describe("Eventgenerator", func() {
	var (
		runner      *EventGeneratorRunner
		testCertDir = "../../../../../test-certs"
	)

	BeforeEach(func() {
		runner = NewEventGeneratorRunner()
	})

	AfterEach(func() {
		runner.KillWithFire()
	})

	Context("with a valid config file", func() {
		BeforeEach(func() {
			runner.Start()
		})

		It("Starts successfully, retrives metrics and  generates events", func() {
			Consistently(runner.Session).ShouldNot(Exit())
			Eventually(func() bool { return len(metricCollector.ReceivedRequests()) >= 1 }, 5*time.Second).Should(BeTrue())
			Eventually(func() bool { return len(scalingEngine.ReceivedRequests()) >= 1 }, time.Duration(2*breachDurationSecs)*time.Second).Should(BeTrue())
		})

	})

	Context("with a missing config file", func() {
		BeforeEach(func() {
			runner.startCheck = ""
			runner.configPath = "bogus"
			runner.Start()
		})

		It("fails with an error", func() {
			Eventually(runner.Session).Should(Exit(1))
			Expect(runner.Session.Buffer()).To(Say("failed to open config file"))
		})
	})

	Context("with an invalid config file", func() {
		BeforeEach(func() {
			runner.startCheck = ""
			badfile, err := os.CreateTemp("", "bad-mc-config")
			Expect(err).NotTo(HaveOccurred())
			runner.configPath = badfile.Name()
			// #nosec G306
			err = os.WriteFile(runner.configPath, []byte("bogus"), os.ModePerm)
			Expect(err).NotTo(HaveOccurred())
			runner.Start()
		})

		AfterEach(func() {
			os.Remove(runner.configPath)
		})

		It("fails with an error", func() {
			Eventually(runner.Session).Should(Exit(1))
			Expect(runner.Session.Buffer()).To(Say("failed to parse config file"))
		})
	})

	Context("with missing configuration", func() {
		BeforeEach(func() {
			runner.startCheck = ""
			conf := &config.Config{
				Logging: helpers.LoggingConfig{
					Level: "debug",
				},
				Aggregator: config.AggregatorConfig{
					AggregatorExecuteInterval: 2 * time.Second,
					PolicyPollerInterval:      2 * time.Second,
					MetricPollerCount:         2,
					AppMonitorChannelSize:     2,
				},
				Evaluator: config.EvaluatorConfig{
					EvaluationManagerInterval: 2 * time.Second,
					EvaluatorCount:            2,
					TriggerArrayChannelSize:   2,
				},
			}
			configFile := writeConfig(conf)
			runner.configPath = configFile.Name()
			runner.Start()
		})

		AfterEach(func() {
			os.Remove(runner.configPath)
		})

		It("should fail validation", func() {
			Eventually(runner.Session).Should(Exit(1))
			Expect(runner.Session.Buffer()).To(Say("failed to validate configuration"))
		})
	})

	Context("when an interrupt is sent", func() {
		BeforeEach(func() {
			runner.Start()
		})

		It("should stop", func() {
			runner.Session.Interrupt()
			Eventually(runner.Session, 5).Should(Exit(0))
		})
	})

	Describe("EventGenerator REST API", func() {
		Context("when a request for aggregated metrics history comes", func() {
			BeforeEach(func() {
				runner.Start()
			})

			It("returns with a 200", func() {
				rsp, err := httpClient.Get(fmt.Sprintf("https://127.0.0.1:%d/v1/apps/an-app-id/aggregated_metric_histories/a-metric-type", egPort))
				Expect(err).NotTo(HaveOccurred())
				Expect(rsp.StatusCode).To(Equal(http.StatusOK))
				rsp.Body.Close()
			})

		})

	})

	Describe("when Health server is ready to serve RESTful API", func() {
		BeforeEach(func() {
			basicAuthConfig := conf
			basicAuthConfig.Health.HealthCheckUsername = ""
			basicAuthConfig.Health.HealthCheckPassword = ""
			runner.configPath = writeConfig(&basicAuthConfig).Name()

			runner.Start()

		})

		Context("when a request to query health comes", func() {
			It("returns with a 200", func() {
				rsp, err := healthHttpClient.Get(fmt.Sprintf("http://127.0.0.1:%d/health", healthport))
				Expect(err).NotTo(HaveOccurred())
				Expect(rsp.StatusCode).To(Equal(http.StatusOK))
				raw, _ := io.ReadAll(rsp.Body)
				healthData := string(raw)
				Expect(healthData).To(ContainSubstring("autoscaler_eventgenerator_concurrent_http_request"))
				Expect(healthData).To(ContainSubstring("autoscaler_eventgenerator_policyDB"))
				Expect(healthData).To(ContainSubstring("autoscaler_eventgenerator_appMetricDB"))
				Expect(healthData).To(ContainSubstring("go_goroutines"))
				Expect(healthData).To(ContainSubstring("go_memstats_alloc_bytes"))
				rsp.Body.Close()

			})
		})
	})

	Describe("when Health server is ready to serve RESTful API with basic Auth", func() {
		BeforeEach(func() {
			runner.Start()
		})
		Context("when username and password are incorrect for basic authentication during health check", func() {
			It("should return 401", func() {

				req, err := http.NewRequest(http.MethodGet, fmt.Sprintf("http://127.0.0.1:%d/health", healthport), nil)
				Expect(err).NotTo(HaveOccurred())

				req.SetBasicAuth("wrongusername", "wrongpassword")

				rsp, err := healthHttpClient.Do(req)
				Expect(err).ToNot(HaveOccurred())
				Expect(rsp.StatusCode).To(Equal(http.StatusUnauthorized))
			})
		})

		Context("when username and password are correct for basic authentication during health check", func() {
			It("should return 200", func() {

				req, err := http.NewRequest(http.MethodGet, fmt.Sprintf("http://127.0.0.1:%d/health", healthport), nil)
				Expect(err).NotTo(HaveOccurred())

				req.SetBasicAuth(conf.Health.HealthCheckUsername, conf.Health.HealthCheckPassword)

				rsp, err := healthHttpClient.Do(req)
				Expect(err).ToNot(HaveOccurred())
				Expect(rsp.StatusCode).To(Equal(http.StatusOK))
			})
		})
	})

	Describe("when Health server is ready to serve RESTful API with basic Auth", func() {
		BeforeEach(func() {
			runner.Start()
		})
		Context("when username and password are incorrect for basic authentication during health check", func() {
			It("should return 401", func() {

				req, err := http.NewRequest(http.MethodGet, fmt.Sprintf("http://127.0.0.1:%d/health", healthport), nil)
				Expect(err).NotTo(HaveOccurred())

				req.SetBasicAuth("wrongusername", "wrongpassword")

				rsp, err := healthHttpClient.Do(req)
				Expect(err).ToNot(HaveOccurred())
				Expect(rsp.StatusCode).To(Equal(http.StatusUnauthorized))
			})
		})

		Context("when username and password are correct for basic authentication during health check", func() {
			It("should return 200", func() {

				req, err := http.NewRequest(http.MethodGet, fmt.Sprintf("http://127.0.0.1:%d/health", healthport), nil)
				Expect(err).NotTo(HaveOccurred())

				req.SetBasicAuth(conf.Health.HealthCheckUsername, conf.Health.HealthCheckPassword)

				rsp, err := healthHttpClient.Do(req)
				Expect(err).ToNot(HaveOccurred())
				Expect(rsp.StatusCode).To(Equal(http.StatusOK))
			})
		})
	})

	Describe("", func() {
		Context("when logCache is enabled", func() {
			var fakeLogCacheServer *stubGrpcLogCache
			BeforeEach(func() {
				caCert := filepath.Join(testCertDir, "autoscaler-ca.crt")
				certFile := filepath.Join(testCertDir, "metricserver.crt")
				keyFile := filepath.Join(testCertDir, "metricserver.key")
				fakeLogCacheServer = newStubGrpcLogCache(caCert, certFile, keyFile)
				logCacheConfig := conf
				logCacheConfig.MetricCollector.UseLogCache = true
				logCacheConfig.MetricCollector.TLSClientCerts.CACertFile = caCert
				logCacheConfig.MetricCollector.TLSClientCerts.CertFile = certFile
				logCacheConfig.MetricCollector.TLSClientCerts.KeyFile = keyFile
				logCacheConfig.MetricCollector.MetricCollectorURL = strings.ReplaceAll(fakeLogCacheServer.addr(), "http://", "")
				runner.configPath = writeConfig(&logCacheConfig).Name()
				runner.Start()
			})

			It("Should create a LogCacheClient", func() {
				Eventually(runner.Session.Buffer()).ShouldNot(Say("eventgenerator.MetricServerClient.GetMetrics"))
				Eventually(runner.Session.Buffer(), 2).Should(Say("eventgenerator.LogCacheClient.GetMetrics"))
			})

			It("Should initialized an envelopeProcessor", func() {
				Eventually(runner.Session.Buffer(), 2).Should(Say("eventgenerator.EnvelopeProcessor.GetGaugeMetrics"))
			})
			AfterEach(func() {
				fakeLogCacheServer.stop()
			})
		})
	})
})

type stubGrpcLogCache struct {
	mu              sync.Mutex
	reqs            []*rpc.ReadRequest
	promInstantReqs []*rpc.PromQL_InstantQueryRequest
	promRangeReqs   []*rpc.PromQL_RangeQueryRequest
	lis             net.Listener
	srv             *grpc.Server
	block           bool
	rpc.UnimplementedEgressServer
	rpc.UnimplementedPromQLQuerierServer
}

func newStubGrpcLogCache(caCert, certFile, keyFile string) *stubGrpcLogCache {
	s := &stubGrpcLogCache{}
	tlsCerts := &models.TLSCerts{CACertFile: caCert, CertFile: certFile, KeyFile: keyFile}
	config, err := tlsCerts.CreateClientConfig()
	Expect(err).NotTo(HaveOccurred())

	config.Rand = rand.Reader
	lis, err := tls.Listen("tcp", "127.0.0.1:0", config)
	Expect(err).ToNot(HaveOccurred())

	s.lis = lis
	s.srv = grpc.NewServer()

	rpc.RegisterEgressServer(s.srv, s)
	rpc.RegisterPromQLQuerierServer(s.srv, s)
	go func() {
		err := s.srv.Serve(lis)
		Expect(err).NotTo(HaveOccurred())
	}()

	return s
}
func (s *stubGrpcLogCache) stop() {
	s.srv.Stop()
}

func (s *stubGrpcLogCache) addr() string {
	return s.lis.Addr().String()
}

func (s *stubGrpcLogCache) Read(c context.Context, r *rpc.ReadRequest) (*rpc.ReadResponse, error) {
	if s.block {
		var block chan struct{}
		<-block
	}

	s.mu.Lock()
	defer s.mu.Unlock()
	s.reqs = append(s.reqs, r)

	return &rpc.ReadResponse{
		Envelopes: &loggregator_v2.EnvelopeBatch{
			Batch: []*loggregator_v2.Envelope{
				{Timestamp: 99, SourceId: "some-id"},
				{Timestamp: 100, SourceId: "some-id"},
			},
		},
	}, nil
}

func (s *stubGrpcLogCache) InstantQuery(c context.Context, r *rpc.PromQL_InstantQueryRequest) (*rpc.PromQL_InstantQueryResult, error) {
	if s.block {
		var block chan struct{}
		<-block
	}

	s.mu.Lock()
	defer s.mu.Unlock()
	s.promInstantReqs = append(s.promInstantReqs, r)

	return &rpc.PromQL_InstantQueryResult{
		Result: &rpc.PromQL_InstantQueryResult_Scalar{
			Scalar: &rpc.PromQL_Scalar{
				Time:  "99.000",
				Value: 101,
			},
		},
	}, nil
}

func (s *stubGrpcLogCache) RangeQuery(c context.Context, r *rpc.PromQL_RangeQueryRequest) (*rpc.PromQL_RangeQueryResult, error) {
	if s.block {
		var block chan struct{}
		<-block
	}

	s.mu.Lock()
	defer s.mu.Unlock()
	s.promRangeReqs = append(s.promRangeReqs, r)

	return &rpc.PromQL_RangeQueryResult{
		Result: &rpc.PromQL_RangeQueryResult_Matrix{
			Matrix: &rpc.PromQL_Matrix{
				Series: []*rpc.PromQL_Series{
					{
						Metric: map[string]string{
							"__name__": "test",
						},
						Points: []*rpc.PromQL_Point{
							{
								Time:  "99.000",
								Value: 101,
							},
						},
					},
				},
			},
		},
	}, nil
}
