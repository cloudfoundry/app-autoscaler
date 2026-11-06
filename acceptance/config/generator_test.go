package config_test

import (
	acceptanceconfig "acceptance/config"
	"os"
	"os/exec"
	"path/filepath"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("Acceptance config generator", func() {
	const password = "password with \"quotes\", \\slashes and\na newline"
	var configPath string

	BeforeEach(func() {
		if _, err := exec.LookPath("jq"); err != nil {
			Skip("config generator requires jq")
		}
		configPath = filepath.Join(GinkgoT().TempDir(), "config.json")
		for key, value := range map[string]string{
			"PR_NUMBER": "123", "DEPLOYMENT_NAME": "test-autoscaler",
			"SYSTEM_DOMAIN": "example.com", "CF_ADMIN_PASSWORD": password,
			"ACCEPTANCE_CONFIG_PATH": configPath, "CONFIG": configPath,
			"FIPS_MODE_EXPECTED": "", "APISERVER_HOST": "",
			"EVENTGENERATOR_HOST": "", "SCALINGENGINE_HOST": "",
			"METRICSFORWARDER_HOST": "", "OPERATOR_HOST": "", "SCHEDULER_CF_HOST": "",
		} {
			GinkgoT().Setenv(key, value)
		}
	})

	generate := func() *acceptanceconfig.Config {
		cmd := exec.Command("bash", "../../scripts/acceptance-tests-config.sh")
		cmd.Env = os.Environ()
		output, err := cmd.CombinedOutput()
		Expect(err).NotTo(HaveOccurred(), "%s", output)
		return acceptanceconfig.LoadConfig(AbortSuite)
	}

	It("generates a local config with non-FIPS defaults and escaped credentials", func() {
		cfg := generate()
		Expect(cfg.FipsModeExpected).To(BeFalse())
		Expect(cfg.HealthEndpoints).To(BeEmpty())
		Expect(cfg.AdminPassword).To(Equal(password))
		Expect(cfg.ASApiEndpoint).To(Equal("https://test-autoscaler.example.com"))
	})

	DescribeTable("generates reachable MTA health endpoints in either mode", func(fipsMode string, expected bool) {
		GinkgoT().Setenv("FIPS_MODE_EXPECTED", fipsMode)
		for key, value := range map[string]string{
			"APISERVER_HOST": "custom-api", "EVENTGENERATOR_HOST": "eventgenerator",
			"SCALINGENGINE_HOST": "scalingengine", "METRICSFORWARDER_HOST": "metricsforwarder",
			"OPERATOR_HOST": "operator", "SCHEDULER_CF_HOST": "cf-scheduler",
			"EVENTGENERATOR_HEALTH_PASSWORD": password, "SCALINGENGINE_HEALTH_PASSWORD": password,
			"METRICSFORWARDER_HEALTH_PASSWORD": password, "OPERATOR_HEALTH_PASSWORD": password,
		} {
			GinkgoT().Setenv(key, value)
		}
		cfg := generate()
		Expect(cfg.FipsModeExpected).To(Equal(expected))
		Expect(cfg.ASApiEndpoint).To(Equal("https://custom-api.example.com"))
		Expect(cfg.HealthEndpoints).To(HaveLen(5))
		Expect(cfg.HealthEndpoints).NotTo(HaveKey("apiserver"))
		Expect(cfg.HealthEndpoints["scheduler"]).To(Equal(acceptanceconfig.HealthEndpointConfig{
			Endpoint: "https://cf-scheduler.example.com/health",
			Username: "test-user", Password: "test-password",
		}))
		for _, name := range []string{"eventgenerator", "scalingengine", "metricsforwarder", "operator"} {
			Expect(cfg.HealthEndpoints[name]).To(Equal(acceptanceconfig.HealthEndpointConfig{
				Endpoint: "https://" + name + ".example.com/health",
				Username: name, Password: password,
			}))
		}
	},
		Entry("FIPS", "true", true),
		Entry("non-FIPS", "false", false),
	)
})
