package integration_test

import (
	. "integration"

	"autoscaler/cf"
	"fmt"
	. "github.com/onsi/ginkgo"
	// . "github.com/onsi/gomega"
)

var _ = Describe("Integration_Metricscollector_Eventgenerator_Scalingengine", func() {

	BeforeEach(func() {
		startFakeCCNOAAUAA()
		metricsCollectorConfPath = prepareMetricsCollectorConfig(dbUrl, components.Ports[MetricsCollector], fakeCCNOAAUAA.URL(), cf.GrantTypePassword)
		eventGeneratorConfPath = prepareEventGeneratorConfig(dbUrl, components.Ports[EventGenerator], fmt.Sprintf("http://127.0.0.1:%d", components.Ports[MetricsCollector]), fmt.Sprintf("http://127.0.0.1:%d", components.Ports[ScalingEngine]))
		scalingEngineConfPath = prepareScalingEngineConfig(dbUrl, components.Ports[ScalingEngine], fakeCCNOAAUAA.URL(), cf.GrantTypePassword)
		startMetricsCollector()
		startEventGenerator()
		startScalingEngine()
	})

	AfterEach(func() {
		stopAll()
	})
	Describe("", func() {
		Context("", func() {
			It("", func() {
			})

		})
	})
})
