package models_test

import (
	"code.cloudfoundry.org/app-autoscaler/src/autoscaler/models"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"gopkg.in/yaml.v2"
)

var (
	healthConfigBytes []byte
	healthConfig      models.HealthConfig
)

var _ = Describe("Health Config", func() {
	Context("Readiness enabled", func() {
		BeforeEach(func() {
			healthConfigBytes = []byte(`
port: 9999
username: test-username
username_hash: test-username-hash
password: password
password_hash: password-hash
readiness_enabled: false
`)
		})
		It("should have config set", func() {
			err := yaml.Unmarshal(healthConfigBytes, &healthConfig)
			if err != nil {
				Fail("unable to unmarshal to health config" + err.Error())
			}
			Expect(healthConfig).To(Equal(models.HealthConfig{
				Port:                    9999,
				HealthCheckUsername:     "test-username",
				HealthCheckUsernameHash: "test-username-hash",
				HealthCheckPassword:     "password",
				HealthCheckPasswordHash: "password-hash",
				ReadinessCheckEnabled:   false,
			}))
		})
	})

})
