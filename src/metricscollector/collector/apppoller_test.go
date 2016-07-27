package collector_test

import (
	"metricscollector/cf"
	. "metricscollector/collector"
	"metricscollector/collector/fakes"
	"metricscollector/metrics"

	"code.cloudfoundry.org/lager"
	"github.com/cloudfoundry/sonde-go/events"
	"github.com/gogo/protobuf/proto"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"errors"
	"time"
)

var _ = Describe("Apppoller", func() {

	var (
		cfc      *fakes.FakeCfClient
		noaa     *fakes.FakeNoaaConsumer
		database *fakes.FakeDB
		poller   *AppPoller
	)

	BeforeEach(func() {
		cfc = &fakes.FakeCfClient{}
		noaa = &fakes.FakeNoaaConsumer{}
		database = &fakes.FakeDB{}
		logger := lager.NewLogger("apppoller-test")

		poller = NewAppPoller(logger, "test-app-id", TestPollInterval*time.Second, cfc, noaa, database)
	})

	Describe("Start", func() {

		JustBeforeEach(func() {
			poller.Start()
			time.Sleep(2500 * time.Millisecond)
			poller.Stop()
		})

		BeforeEach(func() {
			cfc.GetTokensReturns(cf.Tokens{AccessToken: "test-access-token"})

			database.SaveMetricStub = func(metric *metrics.Metric) error {
				Expect(metric.AppId).To(Equal("test-app-id"))
				Expect(metric.Name).To(Equal(metrics.MetricNameMemory))
				Expect(metric.Unit).To(Equal(metrics.UnitBytes))
				Expect(metric.Instances).To(ConsistOf(metrics.InstanceMetric{Index: 0, Value: "1234"}))
				return nil
			}

		})

		Context("when retrieving container metrics all succeeds", func() {

			Context("when all container metrics are not empty", func() {

				BeforeEach(func() {
					noaa.ContainerMetricsStub = func(appid string, token string) ([]*events.ContainerMetric, error) {
						Expect(appid).To(Equal("test-app-id"))
						Expect(token).To(Equal("bearer test-access-token"))

						return []*events.ContainerMetric{
							&events.ContainerMetric{
								ApplicationId: proto.String("test-app-id"),
								InstanceIndex: proto.Int32(0),
								MemoryBytes:   proto.Uint64(1234),
							},
						}, nil
					}
				})

				It("saves all the metrics to database", func() {
					Expect(database.SaveMetricCallCount()).To(Equal(3))
				})
			})

			Context("when all container metrics are empty", func() {
				BeforeEach(func() {
					noaa.ContainerMetricsStub = func(appid string, token string) ([]*events.ContainerMetric, error) {
						Expect(appid).To(Equal("test-app-id"))
						Expect(token).To(Equal("bearer test-access-token"))

						return []*events.ContainerMetric{}, nil
					}
				})

				It("saves nothings to database", func() {
					Expect(database.SaveMetricCallCount()).To(BeZero())
				})

			})

			Context("when container metrics are partially empty", func() {
				BeforeEach(func() {
					noaa.ContainerMetricsStub = func(appid string, token string) ([]*events.ContainerMetric, error) {
						Expect(appid).To(Equal("test-app-id"))
						Expect(token).To(Equal("bearer test-access-token"))

						if noaa.ContainerMetricsCallCount()%2 == 0 {
							return []*events.ContainerMetric{}, nil
						} else {
							return []*events.ContainerMetric{
								&events.ContainerMetric{
									ApplicationId: proto.String("test-app-id"),
									InstanceIndex: proto.Int32(0),
									MemoryBytes:   proto.Uint64(1234),
								},
							}, nil
						}

					}
				})

				It("saves non-empty metrics to database", func() {
					Expect(database.SaveMetricCallCount()).To(Equal(2))
				})
			})
		})

		Context("when retrieving container metrics all fails", func() {

			BeforeEach(func() {
				noaa.ContainerMetricsReturns(nil, errors.New("an error"))
			})

			It("saves nothing to database", func() {
				Expect(database.SaveMetricCallCount()).To(Equal(0))
			})
		})

		Context("when retrieving container metrics partially fails", func() {
			BeforeEach(func() {
				noaa.ContainerMetricsStub = func(appid string, token string) ([]*events.ContainerMetric, error) {
					Expect(appid).To(Equal("test-app-id"))
					Expect(token).To(Equal("bearer test-access-token"))

					if noaa.ContainerMetricsCallCount()%2 == 0 {
						return nil, errors.New("an error")
					} else {
						return []*events.ContainerMetric{
							&events.ContainerMetric{
								ApplicationId: proto.String("test-app-id"),
								InstanceIndex: proto.Int32(0),
								MemoryBytes:   proto.Uint64(1234),
							},
						}, nil
					}
				}
			})

			It("saves successful results to database", func() {
				Expect(database.SaveMetricCallCount()).To(Equal(2))
			})

		})

	})

	Describe("Stop", func() {
		JustBeforeEach(func() {
			poller.Stop()
		})

		Context("when apppoller is started", func() {
			BeforeEach(func() {
				poller.Start()
				time.Sleep(2500 * time.Millisecond)
			})

			It("stops the polling", func() {
				num := noaa.ContainerMetricsCallCount()
				time.Sleep(1500 * time.Millisecond)
				Expect(noaa.ContainerMetricsCallCount()).To(Equal(num))
			})
		})

		Context("when apppoller is not started", func() {
			It("does nothing", func() {
				Expect(noaa.ContainerMetricsCallCount()).To(BeZero())
			})
		})

	})

})
