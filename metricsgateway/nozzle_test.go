package metricsgateway_test

import (
	"crypto/tls"
	"path/filepath"

	"code.cloudfoundry.org/go-loggregator/rpc/loggregator_v2"
	"code.cloudfoundry.org/lager/lagertest"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	. "github.com/onsi/gomega/gbytes"

	"autoscaler/metricsgateway"
	. "autoscaler/metricsgateway/testhelpers"
)

var _ = Describe("Nozzle", func() {

	var (
		testCertDir                     = "../../../test-certs"
		serverCrtPath                   = filepath.Join(testCertDir, "reverselogproxy.crt")
		serverKeyPath                   = filepath.Join(testCertDir, "reverselogproxy.key")
		clientCrtPath                   = filepath.Join(testCertDir, "reverselogproxy_client.crt")
		clientKeyPath                   = filepath.Join(testCertDir, "reverselogproxy_client.key")
		caPath                          = filepath.Join(testCertDir, "autoscaler-ca.crt")
		fakeLoggregator                 FakeEventProducer
		testAppId                       = "test-app-id"
		envelopes                       []*loggregator_v2.Envelope
		nonContainerMetricGaugeEnvelope = loggregator_v2.Envelope{
			SourceId: "uaa",
			Message: &loggregator_v2.Envelope_Gauge{
				Gauge: &loggregator_v2.Gauge{
					Metrics: map[string]*loggregator_v2.GaugeValue{
						"cpu": &loggregator_v2.GaugeValue{
							Unit:  "percentage",
							Value: 33.2,
						},
						"memory": &loggregator_v2.GaugeValue{
							Unit:  "bytes",
							Value: 1000000000,
						},
					},
				},
			},
		}

		containerMetricEnvelope = loggregator_v2.Envelope{
			SourceId: testAppId,
			Message: &loggregator_v2.Envelope_Gauge{
				Gauge: &loggregator_v2.Gauge{
					Metrics: map[string]*loggregator_v2.GaugeValue{
						"cpu": &loggregator_v2.GaugeValue{
							Unit:  "percentage",
							Value: 20.5,
						},
						"disk": &loggregator_v2.GaugeValue{
							Unit:  "bytes",
							Value: 3000000000,
						},
						"memory": &loggregator_v2.GaugeValue{
							Unit:  "bytes",
							Value: 1000000000,
						},
						"memory_quota": &loggregator_v2.GaugeValue{
							Unit:  "bytes",
							Value: 2000000000,
						},
					},
				},
			},
		}

		customMetricEnvelope = loggregator_v2.Envelope{
			SourceId: testAppId,
			Tags:     map[string]string{"origin": "autoscaler_metrics_forwarder"},
			Message: &loggregator_v2.Envelope_Gauge{
				Gauge: &loggregator_v2.Gauge{
					Metrics: map[string]*loggregator_v2.GaugeValue{
						"queue_length": &loggregator_v2.GaugeValue{
							Unit:  "number",
							Value: 100,
						},
					},
				},
			},
		}

		nonCustomMetricEnvelope = loggregator_v2.Envelope{
			SourceId: testAppId,
			Tags:     map[string]string{"origin": "other-origin"},
			Message: &loggregator_v2.Envelope_Gauge{
				Gauge: &loggregator_v2.Gauge{
					Metrics: map[string]*loggregator_v2.GaugeValue{
						"queue_length": &loggregator_v2.GaugeValue{
							Unit:  "number",
							Value: 100,
						},
					},
				},
			},
		}

		httpStartStopEnvelope = loggregator_v2.Envelope{
			SourceId: testAppId,
			Message: &loggregator_v2.Envelope_Timer{
				Timer: &loggregator_v2.Timer{
					Name:  "http",
					Start: 1542325492043447110,
					Stop:  1542325492045491009,
				},
			},
		}
		nonHttpStartStopTimerEnvelope = loggregator_v2.Envelope{
			SourceId: testAppId,
			Message: &loggregator_v2.Envelope_Timer{
				Timer: &loggregator_v2.Timer{
					Name:  "other_timer",
					Start: 1542325492043307300,
					Stop:  1542325492045818196,
				},
			},
		}

		logger      *lagertest.TestLogger
		index       = 0
		shardID     = "autoscaler"
		envelopChan chan *loggregator_v2.Envelope
		getAppIDs   metricsgateway.GetAppIDsFunc
		nozzle      *metricsgateway.Nozzle
		rlpAddr     string
		tlsConf     *tls.Config
		appIDs      map[string]bool
	)
	BeforeEach(func() {
		envelopChan = make(chan *loggregator_v2.Envelope, 1000)
		logger = lagertest.NewTestLogger("AppManager-test")
		getAppIDs = func() map[string]bool {
			return appIDs
		}
		envelopes = []*loggregator_v2.Envelope{
			&containerMetricEnvelope,
			&httpStartStopEnvelope,
		}

	})
	AfterEach(func() {
		fakeLoggregator.Stop()

	})
	Context("Start", func() {
		JustBeforeEach(func() {
			fakeLoggregator, err := NewFakeEventProducer(serverCrtPath, serverKeyPath, caPath)
			Expect(err).NotTo(HaveOccurred())
			fakeLoggregator.Start()
			tlsConf, err = NewClientMutualTLSConfig(clientCrtPath, clientKeyPath, caPath, "reverselogproxy")
			Expect(err).NotTo(HaveOccurred())
			rlpAddr = fakeLoggregator.GetAddr()
			fakeLoggregator.SetEnvelops(envelopes)
			nozzle = metricsgateway.NewNozzle(logger, index, shardID, rlpAddr, tlsConf, envelopChan, getAppIDs)
			nozzle.Start()
		})
		BeforeEach(func() {
			appIDs = map[string]bool{testAppId: true}
		})
		AfterEach(func() {
			nozzle.Stop()
		})
		Context("when there is no app", func() {
			BeforeEach(func() {
				appIDs = map[string]bool{}
			})
			It("should not accept any envelop", func() {
				Consistently(envelopChan).ShouldNot(Receive())
			})
		})
		Context("when the app ID of the received envelope is not in policy database", func() {
			BeforeEach(func() {
				appIDs = map[string]bool{"another-test-app-id": true}
			})
			It("should not accept the envelope", func() {
				Consistently(envelopChan).ShouldNot(Receive())
			})
		})
		Context("when the guage envelope is not a container metric", func() {
			BeforeEach(func() {
				envelopes = []*loggregator_v2.Envelope{
					&nonContainerMetricGaugeEnvelope,
				}
			})
			It("should not accept the envelope", func() {
				Consistently(envelopChan).ShouldNot(Receive())
			})
		})
		Context("there is container metric envelope", func() {
			BeforeEach(func() {
				envelopes = []*loggregator_v2.Envelope{
					&containerMetricEnvelope,
				}
			})
			It("should accept the envelope", func() {
				Eventually(envelopChan).Should(Receive())

			})
		})

		Context("there is custom metric envelope", func() {
			BeforeEach(func() {
				envelopes = []*loggregator_v2.Envelope{
					&customMetricEnvelope,
				}
			})
			It("should accept the envelope", func() {
				Eventually(envelopChan).Should(Receive())

			})
		})

		Context("when the guage envelope is not a custom metric", func() {
			BeforeEach(func() {
				envelopes = []*loggregator_v2.Envelope{
					&nonCustomMetricEnvelope,
				}
			})
			It("should not accept the envelope", func() {
				Consistently(envelopChan).ShouldNot(Receive())
			})
		})

		Context("when there is httpstartstop envelope", func() {
			BeforeEach(func() {
				envelopes = []*loggregator_v2.Envelope{
					&httpStartStopEnvelope,
				}
			})
			It("should accept the envelope", func() {
				Eventually(envelopChan).Should(Receive())
			})
		})

		Context("when there is non httpstartstop timer envelope", func() {
			BeforeEach(func() {
				envelopes = []*loggregator_v2.Envelope{
					&nonHttpStartStopTimerEnvelope,
				}
			})
			It("should not accept the envelope", func() {
				Eventually(envelopChan).ShouldNot(Receive())
			})
		})

	})

	Context("Stop", func() {
		BeforeEach(func() {
			appIDs = map[string]bool{testAppId: true}
			fakeLoggregator, err := NewFakeEventProducer(serverCrtPath, serverKeyPath, caPath)
			Expect(err).NotTo(HaveOccurred())
			fakeLoggregator.Start()
			tlsConf, err = NewClientMutualTLSConfig(clientCrtPath, clientKeyPath, caPath, "reverselogproxy")
			Expect(err).NotTo(HaveOccurred())
			rlpAddr = fakeLoggregator.GetAddr()
			fakeLoggregator.SetEnvelops(envelopes)
			nozzle = metricsgateway.NewNozzle(logger, index, shardID, rlpAddr, tlsConf, envelopChan, getAppIDs)
			nozzle.Start()
			Eventually(envelopChan).Should(Receive())
			nozzle.Stop()
			Eventually(logger.Buffer).Should(Say("nozzle-stopped"))
		})
		It("should not accept any envelop", func() {
			Eventually(envelopChan).ShouldNot(Receive())
		})

	})
})
