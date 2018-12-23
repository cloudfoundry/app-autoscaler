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
		testCertDir      = "../../../test-certs"
		serverCrtPath    = filepath.Join(testCertDir, "metron.crt")
		serverKeyPath    = filepath.Join(testCertDir, "metron.key")
		clientCrtPath    = filepath.Join(testCertDir, "metron_client.crt")
		clientKeyPath    = filepath.Join(testCertDir, "metron_client.key")
		caPath           = filepath.Join(testCertDir, "autoscaler-ca.crt")
		fakeLoggregator  FakeEventProducer
		testAppId        = "test-app-id"
		needlessEnvelope = loggregator_v2.Envelope{
			SourceId: testAppId,
			Message: &loggregator_v2.Envelope_Event{
				Event: &loggregator_v2.Event{
					Title: "event-name",
					Body:  "event-body",
				},
			},
		}
		guageEnvelope = loggregator_v2.Envelope{
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
							Value: 20.5,
						},
						"memory": &loggregator_v2.GaugeValue{
							Unit:  "bytes",
							Value: 20.5,
						},
						"memory_quota": &loggregator_v2.GaugeValue{
							Unit:  "bytes",
							Value: 20.5,
						},
					},
				},
			},
		}
		timerEnvelope = loggregator_v2.Envelope{
			SourceId: testAppId,
			Message: &loggregator_v2.Envelope_Timer{
				Timer: &loggregator_v2.Timer{
					Name:  "http",
					Start: 111,
					Stop:  222,
				},
			},
		}
		envelops = []*loggregator_v2.Envelope{
			&needlessEnvelope,
			&guageEnvelope,
			&timerEnvelope,
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

	})
	AfterEach(func() {
		fakeLoggregator.Stop()

	})
	Context("Start", func() {
		JustBeforeEach(func() {
			fakeLoggregator, err := NewFakeEventProducer(serverCrtPath, serverKeyPath, caPath)
			Expect(err).NotTo(HaveOccurred())
			fakeLoggregator.Start()
			tlsConf, err = NewClientMutualTLSConfig(clientCrtPath, clientKeyPath, caPath, "metron")
			Expect(err).NotTo(HaveOccurred())
			rlpAddr = fakeLoggregator.GetAddr()
			fakeLoggregator.SetEnvelops(envelops)
			nozzle = metricsgateway.NewNozzle(logger, index, shardID, rlpAddr, tlsConf, envelopChan, getAppIDs)
			nozzle.Start()
		})
		BeforeEach(func() {
			appIDs = map[string]bool{"test-app-id": true}
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
		Context("when the app of received envelopes is not in policy database", func() {
			BeforeEach(func() {
				appIDs = map[string]bool{"another-test-app-id": true}
			})
			It("should not accept any envelop", func() {
				Consistently(envelopChan).ShouldNot(Receive())
			})
		})
		Context("when there are needless envelopes", func() {
			BeforeEach(func() {
				envelops = []*loggregator_v2.Envelope{
					&needlessEnvelope,
				}
			})
			It("should accept needless envelopes", func() {
				Consistently(envelopChan).ShouldNot(Receive())
			})
		})
		Context("accept timer envelops", func() {
			BeforeEach(func() {
				envelops = []*loggregator_v2.Envelope{
					&timerEnvelope,
				}
			})
			It("should accept timer envelopes", func() {
				Eventually(envelopChan).Should(Receive())
			})
		})
		Context("accept guage envelops", func() {
			BeforeEach(func() {
				envelops = []*loggregator_v2.Envelope{
					&guageEnvelope,
				}
			})
			It("should accept guage envelopes", func() {
				Eventually(envelopChan).Should(Receive())

			})
		})

	})

	Context("Stop", func() {
		BeforeEach(func() {
			appIDs = map[string]bool{"test-app-id": true}
			fakeLoggregator, err := NewFakeEventProducer(serverCrtPath, serverKeyPath, caPath)
			Expect(err).NotTo(HaveOccurred())
			fakeLoggregator.Start()
			tlsConf, err = NewClientMutualTLSConfig(clientCrtPath, clientKeyPath, caPath, "metron")
			Expect(err).NotTo(HaveOccurred())
			rlpAddr = fakeLoggregator.GetAddr()
			fakeLoggregator.SetEnvelops(envelops)
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
