package fips_test

import (
	"bufio"
	"strings"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("FIPS metrics parser", Label("unit"), func() {
	DescribeTable("reads runtime and enforcement gauges", func(body string, expected map[string]float64) {
		metrics, err := parseFipsMetrics(strings.NewReader(body))
		Expect(err).NotTo(HaveOccurred())
		Expect(metrics).To(Equal(expected))
	},
		Entry("Go integer values",
			"# HELP autoscaler_fips_enabled FIPS state\n"+
				"autoscaler_fips_enabled 1\n"+
				"autoscaler_fips_enforced 1\n",
			map[string]float64{"autoscaler_fips_enabled": 1, "autoscaler_fips_enforced": 1}),
		Entry("Java decimal values",
			"autoscaler_fips_enabled 1.0\nautoscaler_fips_enforced 1.0\n",
			map[string]float64{"autoscaler_fips_enabled": 1, "autoscaler_fips_enforced": 1}),
		Entry("runtime enabled without enforcement",
			"autoscaler_fips_enabled 1\nautoscaler_fips_enforced 0\n",
			map[string]float64{"autoscaler_fips_enabled": 1, "autoscaler_fips_enforced": 0}),
		Entry("missing enforcement stays absent",
			"autoscaler_fips_enabled 0\n",
			map[string]float64{"autoscaler_fips_enabled": 0}),
		Entry("ignores similarly named metrics",
			"autoscaler_fips_enabled_other 1\n",
			map[string]float64{}),
		Entry("ignores blank lines and unrelated metrics",
			"\n# TYPE unrelated gauge\nunrelated 1\n",
			map[string]float64{}),
		Entry("accepts labeled gauges",
			"autoscaler_fips_enabled{instance=\"0\"} 1\n",
			map[string]float64{"autoscaler_fips_enabled": 1}),
	)

	DescribeTable("reports malformed metric output", func(body string) {
		_, err := parseFipsMetrics(strings.NewReader(body))
		Expect(err).To(HaveOccurred())
	},
		Entry("invalid numeric value", "autoscaler_fips_enforced invalid\n"),
		Entry("missing value", "autoscaler_fips_enabled\n"),
		Entry("scanner limit exceeded", strings.Repeat("x", bufio.MaxScanTokenSize+1)),
	)
})
