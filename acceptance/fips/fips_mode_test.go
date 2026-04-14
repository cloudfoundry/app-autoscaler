package fips_test

import (
	"bufio"
	"crypto/tls"
	"fmt"
	"io"
	"net"
	"net/http"
	"strconv"
	"strings"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("FIPS Mode Verification", func() {
	var client *http.Client

	BeforeEach(func() {
		client = &http.Client{
			Transport: &http.Transport{
				Proxy: http.ProxyFromEnvironment,
				DialContext: (&net.Dialer{
					Timeout:   30 * time.Second,
					KeepAlive: 30 * time.Second,
				}).DialContext,
				TLSHandshakeTimeout: 10 * time.Second,
				DisableCompression:  true,
				DisableKeepAlives:   true,
				TLSClientConfig: &tls.Config{
					InsecureSkipVerify: cfg.SkipSSLValidation,
				},
			},
			Timeout: 30 * time.Second,
		}
	})

	It("verifies FIPS mode status on all configured health endpoints", func() {
		Expect(cfg.HealthEndpoints).NotTo(BeEmpty(),
			"No health_endpoints configured — cannot verify FIPS mode")

		expectedValue := float64(0)
		if cfg.FipsModeExpected {
			expectedValue = 1
		}

		for name, ep := range cfg.HealthEndpoints {
			url := ep.Endpoint
			if !strings.HasPrefix(url, "http") {
				url = "https://" + url
			}

			By(fmt.Sprintf("Checking FIPS status on %s at %s", name, url))

			req, err := http.NewRequest("GET", url, nil)
			Expect(err).NotTo(HaveOccurred())

			if ep.Username != "" {
				req.SetBasicAuth(ep.Username, ep.Password)
			}

			resp, err := client.Do(req)
			Expect(err).NotTo(HaveOccurred(), "Failed to reach %s health endpoint", name)
			defer resp.Body.Close()

			Expect(resp.StatusCode).To(Equal(http.StatusOK),
				"Health endpoint for %s returned status %d", name, resp.StatusCode)

			fipsValue, found := parseFipsMetric(resp.Body)
			Expect(found).To(BeTrue(),
				"autoscaler_fips_enabled metric not found on %s", name)
			Expect(fipsValue).To(Equal(expectedValue),
				"FIPS mode mismatch on %s: expected %v, got %v", name, expectedValue, fipsValue)
		}
	})
})

// parseFipsMetric scans Prometheus text output for the autoscaler_fips_enabled metric.
func parseFipsMetric(body io.Reader) (float64, bool) {
	scanner := bufio.NewScanner(body)
	for scanner.Scan() {
		line := scanner.Text()
		if strings.HasPrefix(line, "#") {
			continue
		}
		if strings.HasPrefix(line, "autoscaler_fips_enabled") {
			parts := strings.Fields(line)
			if len(parts) >= 2 {
				val, err := strconv.ParseFloat(parts[1], 64)
				if err == nil {
					return val, true
				}
			}
		}
	}
	return 0, false
}
