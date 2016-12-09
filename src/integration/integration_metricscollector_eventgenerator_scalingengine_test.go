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
		metricsCollectorConfPath = components.PrepareMetricsCollectorConfig(dbUrl, components.Ports[MetricsCollector], fakeCCNOAAUAA.URL(), cf.GrantTypePassword, tmpDir)
		eventGeneratorConfPath = components.PrepareEventGeneratorConfig(dbUrl, components.Ports[EventGenerator], fmt.Sprintf("http://127.0.0.1:%d", components.Ports[MetricsCollector]), fmt.Sprintf("http://127.0.0.1:%d", components.Ports[ScalingEngine]), tmpDir)
		scalingEngineConfPath = components.PrepareScalingEngineConfig(dbUrl, components.Ports[ScalingEngine], fakeCCNOAAUAA.URL(), cf.GrantTypePassword, tmpDir)
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
