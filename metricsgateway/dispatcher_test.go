package metricsgateway_test

import (
	"code.cloudfoundry.org/app-autoscaler/src/autoscaler/fakes"
	. "code.cloudfoundry.org/app-autoscaler/src/autoscaler/metricsgateway"

	"code.cloudfoundry.org/go-loggregator/v8/rpc/loggregator_v2"
	"code.cloudfoundry.org/lager/lagertest"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("Dispatcher", func() {
	var (
		logger      *lagertest.TestLogger
		envelopChan chan *loggregator_v2.Envelope
		resultChans []chan *loggregator_v2.Envelope
		emitters    []Emitter
		dispatcher  *Dispatcher
		testAppId1  = "the-test-appid1"
		testAppId2  = "the-test-appid2"
		testAppId3  = "the-test-appid3"
	)
	BeforeEach(func() {
		envelopChan = make(chan *loggregator_v2.Envelope, 10)
		resultChans = []chan *loggregator_v2.Envelope{}
		logger = lagertest.NewTestLogger("Dispatcher")

	})
	Context("Start", func() {
		AfterEach(func() {
			dispatcher.Stop()
		})
		JustBeforeEach(func() {
			dispatcher.Start()
			envelopChan <- &loggregator_v2.Envelope{
				SourceId: testAppId1,
			}
			envelopChan <- &loggregator_v2.Envelope{
				SourceId: testAppId2,
			}
			envelopChan <- &loggregator_v2.Envelope{
				SourceId: testAppId3,
			}
		})
		Context("when there is only one emitter", func() {
			BeforeEach(func() {
				resultChans = []chan *loggregator_v2.Envelope{
					make(chan *loggregator_v2.Envelope, 100),
				}
				emitters = []Emitter{
					&fakes.FakeEmitter{
						AcceptStub: func(envelope *loggregator_v2.Envelope) {
							resultChans[0] <- envelope
						},
					},
				}
				dispatcher = NewDispatcher(logger, envelopChan, emitters)
			})
			It("dispatch envelope to emitter", func() {
				Eventually(resultChans[0]).Should(Receive(Equal(&loggregator_v2.Envelope{
					SourceId: testAppId1,
				})))
				Eventually(resultChans[0]).Should(Receive(Equal(&loggregator_v2.Envelope{
					SourceId: testAppId2,
				})))
				Eventually(resultChans[0]).Should(Receive(Equal(&loggregator_v2.Envelope{
					SourceId: testAppId3,
				})))
			})
		})

		Context("when there 4 emitters", func() {
			BeforeEach(func() {
				resultChans = []chan *loggregator_v2.Envelope{
					make(chan *loggregator_v2.Envelope, 5),
					make(chan *loggregator_v2.Envelope, 5),
					make(chan *loggregator_v2.Envelope, 5),
					make(chan *loggregator_v2.Envelope, 5),
				}
				emitters = []Emitter{
					&fakes.FakeEmitter{
						AcceptStub: func(envelope *loggregator_v2.Envelope) {
							resultChans[0] <- envelope
						},
					},
					&fakes.FakeEmitter{
						AcceptStub: func(envelope *loggregator_v2.Envelope) {
							resultChans[1] <- envelope
						},
					},
					&fakes.FakeEmitter{
						AcceptStub: func(envelope *loggregator_v2.Envelope) {
							resultChans[2] <- envelope
						},
					},
					&fakes.FakeEmitter{
						AcceptStub: func(envelope *loggregator_v2.Envelope) {
							resultChans[3] <- envelope
						},
					},
				}
				dispatcher = NewDispatcher(logger, envelopChan, emitters)
			})
			It("dispatch envelope to emitters", func() {
				Eventually(resultChans[3]).Should(Receive(Equal(&loggregator_v2.Envelope{
					SourceId: testAppId1,
				})))
				Eventually(resultChans[2]).Should(Receive(Equal(&loggregator_v2.Envelope{
					SourceId: testAppId2,
				})))
				Eventually(resultChans[1]).Should(Receive(Equal(&loggregator_v2.Envelope{
					SourceId: testAppId3,
				})))
				Consistently(resultChans[0]).ShouldNot(Receive())
			})
		})

	})

	Context("Stop", func() {
		BeforeEach(func() {
			resultChans = []chan *loggregator_v2.Envelope{
				make(chan *loggregator_v2.Envelope, 100),
			}
			emitters = []Emitter{
				&fakes.FakeEmitter{
					AcceptStub: func(envelope *loggregator_v2.Envelope) {
						resultChans[0] <- envelope
					},
				},
			}
			dispatcher = NewDispatcher(logger, envelopChan, emitters)
			dispatcher.Start()
			envelopChan <- &loggregator_v2.Envelope{
				SourceId: testAppId1,
			}
			Eventually(resultChans[0]).Should(Receive())
			dispatcher.Stop()
			envelopChan <- &loggregator_v2.Envelope{
				SourceId: testAppId1,
			}
		})
		It("stop dispathing", func() {
			Consistently(resultChans[0]).ShouldNot(Receive())
		})
	})
})
