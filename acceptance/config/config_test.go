package config_test

import (
	. "acceptance/config"
	"fmt"
	"os"
	"testing"

	. "github.com/onsi/ginkgo/v2"

	. "github.com/onsi/gomega"
)

func TestConfigSuite(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "ConfigSuite")
}

var _ = Describe("LoadConfig", func() {
	When("CONFIG env var not set", func() {
		It("terminates suite", func() {
			loadConfigExpectSuiteTerminationWith("Must set $CONFIG to point to a json file")
		})
	})

	When("CONFIG env var set to non-existing file ", func() {
		BeforeEach(func() {
			err := os.Setenv("CONFIG", "this/path/does/not/exist/config.json")
			Expect(err).ToNot(HaveOccurred())
		})

		It("terminates suite", func() {
			loadConfigExpectSuiteTerminationWith("open this/path/does/not/exist/config.json: no such file or directory")
		})
	})

	When("CONFIG env var set to existing file", func() {
		var configFile *os.File
		BeforeEach(func() {
			tmpDir := GinkgoT().TempDir()
			tmpFile, err := os.Create(fmt.Sprintf("%s/config.json", tmpDir))
			Expect(err).ToNot(HaveOccurred())
			configFile = tmpFile

			err = os.Setenv("CONFIG", configFile.Name())
			Expect(err).ToNot(HaveOccurred())
		})

		DescribeTable("missing required fields", func(content string, message string) {
			write(content, configFile)
			loadConfigExpectSuiteTerminationWith(message)
		},
			Entry("terminates suite because api is missing", `{}`, "missing configuration 'api'"),
			Entry("terminates suite because 'admin_user' is missing", `{
				"api": "api"
			}`, "missing configuration 'admin_user'"),
			Entry("terminates suite because admin_password is missing", `{
				"api": "api",
				"admin_user": "admin_user"
			}`, "missing configuration 'admin_password'"),
			Entry("terminates suite because service_name is missing", `{
				"api": "api",
				"admin_user": "admin_user",
				"admin_password": "admin_password"
			}`, "missing configuration 'service_name'"),
			Entry("terminates suite because service_plan is missing", `{
				"api": "api",
				"admin_user": "admin_user",
				"admin_password": "admin_password",
				"service_name": "service_name"
			}`, "missing configuration 'service_plan'"),
			Entry("terminates suite because aggregate_interval is missing", `{
				"api": "api",
				"admin_user": "admin_user",
				"admin_password": "admin_password",
				"service_name": "service_name",
				"service_plan": "service_plan"
			}`, "missing configuration 'aggregate_interval'"),
			Entry("terminates suite because autoscaler_api is missing", `{
				"api": "api",
				"admin_user": "admin_user",
				"admin_password": "admin_password",
				"service_name": "service_name",
				"service_plan": "service_plan",
				"aggregate_interval": 30
			}`, "missing configuration 'autoscaler_api'"),
		)

		When("all required fields set", func() {
			var cfg = &Config{}

			When("timeout_scale not set correctly", func() {
				It("falls back to a correct value", func() {
					writeAndExpectValueSetTo[float64](configWith(`"timeout_scale": 0`), configFile, cfg, &cfg.TimeoutScale, 1.0)
				})
			})

			When("aggregate_interval not set correctly", func() {
				It("falls back to a correct value", func() {
					writeAndExpectValueSetTo[int](configWith(`"aggregate_interval": 59`), configFile, cfg, &cfg.AggregateInterval, 60)
				})
			})

			When("eventgenerator_health_endpoint not set correctly", func() {
				It("falls back to a correct value", func() {
					writeAndExpectValueSetTo[string](configWith(`"eventgenerator_health_endpoint": "foo.bar/"`), configFile, cfg, &cfg.EventgeneratorHealthEndpoint, "https://foo.bar")
				})
			})

			When("scalingengine_health_endpoint not set correctly", func() {
				It("falls back to a correct value", func() {
					writeAndExpectValueSetTo[string](configWith(`"scalingengine_health_endpoint": "foo.bar/"`), configFile, cfg, &cfg.ScalingengineHealthEndpoint, "https://foo.bar")
				})
			})

			When("operator_health_endpoint not set correctly", func() {
				It("falls back to a correct value", func() {
					writeAndExpectValueSetTo[string](configWith(`"operator_health_endpoint": "foo.bar/"`), configFile, cfg, &cfg.OperatorHealthEndpoint, "https://foo.bar")
				})
			})

			When("metricsforwarder_health_endpoint not set correctly", func() {
				It("falls back to a correct value", func() {
					writeAndExpectValueSetTo[string](configWith(`"metricsforwarder_health_endpoint": "foo.bar/"`), configFile, cfg, &cfg.MetricsforwarderHealthEndpoint, "https://foo.bar")
				})
			})

			When("scheduler_health_endpoint not set correctly", func() {
				It("falls back to a correct value", func() {
					writeAndExpectValueSetTo[string](configWith(`"scheduler_health_endpoint": "foo.bar/"`), configFile, cfg, &cfg.SchedulerHealthEndpoint, "https://foo.bar")
				})
			})

			When("cpuutil_scaling_policy_test.app_memory not set", func() {
				It("results in default value", func() {
					writeAndExpectValueSetTo[string](config(), configFile, cfg, &cfg.CPUUtilScalingPolicyTest.AppMemory, "1GB")
				})
			})

			When("cpuutil_scaling_policy_test.app_memory not set", func() {
				It("results in default value", func() {
					writeAndExpectValueSetTo[int](config(), configFile, cfg, &cfg.CPUUtilScalingPolicyTest.AppCPUEntitlement, 25)
				})
			})

			When("cpuutil_scaling_policy_test.app_memory set", func() {
				It("unmarshalls correct", func() {
					writeAndExpectValueSetTo[string](configWith(`"cpuutil_scaling_policy_test": {
						"app_memory": "2GB"
					}`), configFile, cfg, &cfg.CPUUtilScalingPolicyTest.AppMemory, "2GB")
				})
			})

			When("cpuutil_scaling_policy_test.app_cpu_entitlement set", func() {
				It("unmarshalls correct", func() {
					writeAndExpectValueSetTo[int](configWith(`"cpuutil_scaling_policy_test": {
						"app_cpu_entitlement": 33
					}`), configFile, cfg, &cfg.CPUUtilScalingPolicyTest.AppCPUEntitlement, 33)
				})
			})
		})
	})
})

func loadConfigExpectSuiteTerminationWith(expectedMessage string) {
	terminated := false
	actualMessage := ""
	var mockTerminateSuite TerminateSuite = func(message string, _ ...int) {
		terminated = true
		actualMessage = message
		panic(nil)
	}

	defer func() {
		if r := recover(); r == nil {
			Fail("expected a panic to recover from")
		}
		Expect(terminated).To(BeTrue())
		Expect(actualMessage).To(Equal(expectedMessage))
	}()

	LoadConfig(mockTerminateSuite)
}

func configWith(keyValue string) string {
	// template contains all required fields
	template := `{
		"api": "api",
		"admin_user": "admin_user",
		"admin_password": "admin_password",
		"service_name": "service_name",
		"service_plan": "service_plan",
		"aggregate_interval": 30,
		"autoscaler_api": "autoscaler_api",
		%s
	}`

	return fmt.Sprintf(template, keyValue)
}

func config() string {
	// passing dummy stuff to get a config JSON that comes with all required fields
	return configWith(`"dummyKey": "dummyValue"`)
}

func write(content string, file *os.File) {
	_, err := file.Write([]byte(content))
	Expect(err).ToNot(HaveOccurred())
}

func writeAndExpectValueSetTo[T any](content string, file *os.File, cfg *Config, actual *T, expected T) {
	write(content, file)
	// copy values from one config to another
	// it's a workaround to successfully pass pointers to values of cfg
	*cfg = *LoadConfig(AbortSuite)
	Expect(*actual).To(Equal(expected))
}
