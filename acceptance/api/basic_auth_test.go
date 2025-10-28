package api_test

import (
	"net/http"
	"strings"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("AutoScaler Basic Auth Tests", func() {

	urlfor := func(name string) func() string {
		return func() string {
			override := ""
			switch name {
			case "eventgenerator":
				override = cfg.EventgeneratorHealthEndpoint
			case "scalingengine":
				override = cfg.ScalingengineHealthEndpoint
			case "operator":
				override = cfg.OperatorHealthEndpoint
			case "metricsforwarder":
				override = cfg.MetricsforwarderHealthEndpoint
			case "scheduler":
				override = cfg.SchedulerHealthEndpoint
			}
			if override != "" {
				return override
			}
			return strings.Replace(healthURL, cfg.ServiceName, cfg.ServiceName+"-"+name, 1)
		}
	}
	DescribeTable("basic auth tests",
		func(url func() string, statusCode func() int) {
			Expect(Get(url())).To(Equal(statusCode()), "to get status code %d when getting %s", statusCode(), url())
		},
		Entry("API Server", func() string { return healthURL }, func() int { return 200 }),
		Entry("Eventgenerator", urlfor("eventgenerator"), getStatus),
		Entry("Scaling Engine", urlfor("scalingengine"), getStatus),
		Entry("Operator", urlfor("operator"), getStatus),
		Entry("Metrics Forwarder", urlfor("metricsforwarder"), getStatus),
		Entry("Scheduler", urlfor("scheduler"), getStatus),
	)

})

func getStatus() int {
	if cfg.HealthEndpointsBasicAuthEnabled {
		return 401
	} else {
		return 200
	}
}

func Get(url string) int {
	req, err := http.NewRequest(http.MethodGet, url, nil)
	Expect(err).ShouldNot(HaveOccurred())
	resp, err := client.Do(req)
	Expect(err).ShouldNot(HaveOccurred())
	defer func() { _ = resp.Body.Close() }()
	Expect(err).ShouldNot(HaveOccurred())
	return resp.StatusCode
}
