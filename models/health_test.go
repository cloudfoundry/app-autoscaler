package models_test

import (
	"code.cloudfoundry.org/app-autoscaler/src/autoscaler/models"
	"errors"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"gopkg.in/yaml.v2"
)

var (
	healthConfigBytes []byte
	healthConfig      models.HealthConfig
)

var _ = Describe("Health Config", func() {
	When("Readiness is not supplied", func() {
		BeforeEach(func() {
			healthConfigBytes = []byte(`
port: 9999
username: test-username
password: password
readiness_enabled: false
`)
		})
		It("should default to false", func() {
			err := yaml.Unmarshal(healthConfigBytes, &healthConfig)
			if err != nil {
				Fail("unable to unmarshal to health config" + err.Error())
			}
			err = healthConfig.Validate()
			Expect(err).ToNot(HaveOccurred())

			Expect(healthConfig).To(Equal(models.HealthConfig{
				Port:                  9999,
				HealthCheckUsername:   "test-username",
				HealthCheckPassword:   "password",
				ReadinessCheckEnabled: false,
			}))
		})
	})
	When("readiness is set to true", func() {
		BeforeEach(func() {
			healthConfigBytes = []byte(`
port: 9999
username: test-username
password: password
readiness_enabled: true
`)
		})
		It("should have readiness true when supplied", func() {
			err := yaml.Unmarshal(healthConfigBytes, &healthConfig)
			if err != nil {
				Fail("unable to unmarshal to health config" + err.Error())
			}
			err = healthConfig.Validate()
			Expect(err).ToNot(HaveOccurred())

			Expect(healthConfig).To(Equal(models.HealthConfig{
				Port:                  9999,
				HealthCheckUsername:   "test-username",
				HealthCheckPassword:   "password",
				ReadinessCheckEnabled: true,
			}))
		})
	})

	When("both password password_hash are supplied", func() {
		BeforeEach(func() {
			healthConfigBytes = []byte(`
port: 9999
username: test-username
password: password
password_hash: password_hash
`)
		})
		It("should fail validation", func() {
			err := yaml.Unmarshal(healthConfigBytes, &healthConfig)
			if err != nil {
				Fail("unable to unmarshal to health config" + err.Error())
			}
			err = healthConfig.Validate()
			Expect(err).To(HaveOccurred())
			Expect(errors.Is(err, models.ConfigurationErr)).To(BeTrue())
			Expect(err.Error()).To(Equal("configuration error: both healthcheck password and healthcheck password_hash are provided, please provide only one of them"))
		})
	})
})
