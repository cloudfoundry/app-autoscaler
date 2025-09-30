package cf_test

import (
	"code.cloudfoundry.org/app-autoscaler/src/autoscaler/cf"
	"go.yaml.in/yaml/v4"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("Config", func() {

	var (
		conf *cf.Config
		err  error
	)
	Describe("Deserialise", func() {
		var (
			objectbytes string
		)
		JustBeforeEach(func() {
			err = yaml.Unmarshal([]byte(objectbytes), &conf)
		})
		Context("Given a valid configuration", func() {
			BeforeEach(func() {
				objectbytes = `
api: https://api.example.com
per_page: 3
client_id: client-id
secret: client-secret
skip_ssl_validation: true
max_retries: 3
max_retry_wait_ms: 27
max_idle_conns_per_host_ms: 200
idle_connection_timeout_ms: 200
`
			})
			It("should deserialise correctly", func() {
				Expect(err).To(Not(HaveOccurred()))
				Expect(conf).To(Equal(&cf.Config{
					API:      "https://api.example.com",
					ClientID: "client-id",
					Secret:   "client-secret",
					PerPage:  3,
					ClientConfig: cf.ClientConfig{
						SkipSSLValidation:       true,
						MaxRetries:              3,
						MaxRetryWaitMs:          27,
						MaxIdleConnsPerHost:     200,
						IdleConnectionTimeoutMs: 200,
					},
				}))
			})
		})

	})

	Describe("Validate", func() {
		BeforeEach(func() {
			conf = &cf.Config{}
			conf.API = "http://api.example.com"
			conf.ClientID = "admin"
			conf.SkipSSLValidation = false
		})

		JustBeforeEach(func() {
			err = conf.Validate()
		})

		Context("when api is not set", func() {
			BeforeEach(func() {
				conf.API = ""
			})

			It("should error", func() {
				Expect(err).To(MatchError("Configuration error: cf api is empty"))
			})
		})

		Context("when SkipSSLValidation is not set", func() {
			It("should set SkipSSLValidation to default false value", func() {
				Expect(err).NotTo(HaveOccurred())
				Expect(conf.SkipSSLValidation).To(Equal(false))
			})
		})

		Context("when SkipSSLValidation is set", func() {
			BeforeEach(func() {
				conf.SkipSSLValidation = true
			})

			It("should not error", func() {
				Expect(err).NotTo(HaveOccurred())
				Expect(conf.SkipSSLValidation).To(Equal(true))
			})
		})

		Context("when api is not a url", func() {
			BeforeEach(func() {
				conf.API = "http://a.com%"
			})

			It("should error", func() {
				Expect(err).To(MatchError("Configuration error: cf api is not a valid url"))
			})
		})

		Context("when api scheme is empty", func() {
			BeforeEach(func() {
				conf.API = "a.com"
			})

			It("should error", func() {
				Expect(err).To(MatchError("Configuration error: cf api scheme is empty"))
			})
		})

		Context("when api has invalid scheme", func() {
			BeforeEach(func() {
				conf.API = "badscheme://a.com"
			})

			It("should error", func() {
				Expect(err).To(MatchError("Configuration error: cf api scheme is invalid"))
			})
		})

		Context("when api is valid but ends with a '/'", func() {
			BeforeEach(func() {
				conf.API = "https://a.com/"
			})

			It("should not error and remove the '/'", func() {
				Expect(err).NotTo(HaveOccurred())
				Expect(conf.API).To(Equal("https://a.com"))
			})
		})

		Context("the client id is empty", func() {
			BeforeEach(func() {
				conf.ClientID = ""
			})

			It("returns error", func() {
				Expect(err).To(MatchError("Configuration error: client_id is empty"))
			})
		})

	})
})
