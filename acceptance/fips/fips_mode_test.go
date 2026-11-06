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

	"acceptance/config"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("FIPS Mode Verification", func() {
	var client *http.Client

	BeforeEach(func() {
		cfg = config.LoadConfig(config.DefaultTerminateSuite)
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

			fipsMetrics, err := parseFipsMetrics(resp.Body)
			Expect(err).NotTo(HaveOccurred(), "Failed to parse FIPS metrics on %s", name)
			for _, metricName := range []string{"autoscaler_fips_enabled", "autoscaler_fips_enforced"} {
				fipsValue, found := fipsMetrics[metricName]
				Expect(found).To(BeTrue(), "%s metric not found on %s", metricName, name)
				Expect(fipsValue).To(Equal(expectedValue),
					"%s mismatch on %s: expected %v, got %v", metricName, name, expectedValue, fipsValue)
			}
		}
	})
})

// parseFipsMetrics reads both runtime state and startup enforcement from
// Prometheus text output, preserving missing metrics for the caller to detect.
func parseFipsMetrics(body io.Reader) (map[string]float64, error) {
	metrics := make(map[string]float64)
	scanner := bufio.NewScanner(body)
	for scanner.Scan() {
		line := scanner.Text()
		parts := strings.Fields(line)
		if len(parts) == 0 || strings.HasPrefix(parts[0], "#") {
			continue
		}
		metricName, _, _ := strings.Cut(parts[0], "{")
		if metricName != "autoscaler_fips_enabled" && metricName != "autoscaler_fips_enforced" {
			continue
		}
		if len(parts) < 2 {
			return nil, fmt.Errorf("missing value for %s", metricName)
		}
		val, err := strconv.ParseFloat(parts[1], 64)
		if err != nil {
			return nil, fmt.Errorf("invalid value for %s: %w", metricName, err)
		}
		metrics[metricName] = val
	}
	return metrics, scanner.Err()
}
