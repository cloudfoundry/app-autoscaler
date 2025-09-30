package helpers_test

import (
	"errors"

	"code.cloudfoundry.org/app-autoscaler/src/autoscaler/helpers"
	"code.cloudfoundry.org/app-autoscaler/src/autoscaler/models"
	. "code.cloudfoundry.org/app-autoscaler/src/autoscaler/testhelpers"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"go.yaml.in/yaml/v4"
)

var (
	healthConfigBytes []byte
	healthConfig      helpers.HealthConfig
)

var _ = Describe("Health Config", func() {
	BeforeEach(func() {
		healthConfigBytes = []byte{}
		healthConfig = helpers.HealthConfig{}
	})

	When("Readiness is not supplied", func() {
		BeforeEach(func() {
			healthConfigBytes = []byte(`
basic_auth:
  username: test-username
  password: password
readiness_enabled: false
`)
		})
		It("should default to false", func() {
			err := yaml.Unmarshal(healthConfigBytes, &healthConfig)
			FailOnError("unable to unmarshal to health config", err)
			err = healthConfig.Validate()
			Expect(err).ToNot(HaveOccurred())

			Expect(healthConfig).To(Equal(helpers.HealthConfig{
				BasicAuth: models.BasicAuth{
					Username: "test-username",
					Password: "password",
				},
				ReadinessCheckEnabled: false,
			}))
		})
	})
	When("readiness is set to true", func() {
		BeforeEach(func() {
			healthConfigBytes = []byte(`
port: 9999
basic_auth:
  username: test-username
  password: password
readiness_enabled: true
`)
		})
		It("should have readiness true when supplied", func() {
			err := yaml.Unmarshal(healthConfigBytes, &healthConfig)
			FailOnError("unable to unmarshal to health config", err)
			err = healthConfig.Validate()
			Expect(err).ToNot(HaveOccurred())

			Expect(healthConfig).To(Equal(helpers.HealthConfig{
				BasicAuth: models.BasicAuth{
					Username: "test-username",
					Password: "password",
				},
				ReadinessCheckEnabled: true,
			}))
		})
	})

	When("both password password_hash are supplied", func() {
		BeforeEach(func() {
			healthConfigBytes = []byte(`
basic_auth:
  username: test-username
  password: password
  password_hash: password_hash
`)
		})
		It("should fail validation", func() {
			err := yaml.Unmarshal(healthConfigBytes, &healthConfig)
			FailOnError("unable to unmarshal to health config", err)
			err = healthConfig.Validate()
			Expect(err).To(HaveOccurred())
			Expect(errors.Is(err, helpers.ErrConfiguration)).To(BeTrue())
			Expect(err.Error()).To(Equal("configuration error: both healthcheck password and healthcheck password_hash are provided, please provide only one of them"))
		})
	})
})
