package configutil_test

import (
	"encoding/json"
	"io"
	"net/url"
	"os"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	. "code.cloudfoundry.org/app-autoscaler/src/autoscaler/configutil"
	"code.cloudfoundry.org/app-autoscaler/src/autoscaler/models"
)

var _ = Describe("Configutil", func() {
	Describe("VCAPConfiguration", func() {
		var vcapConfiguration *VCAPConfiguration
		var vcapApplicationJson string
		var vcapServicesJson string
		var err error

		JustBeforeEach(func() {
			os.Setenv("VCAP_APPLICATION", vcapApplicationJson)
			os.Setenv("VCAP_SERVICES", vcapServicesJson)
			vcapConfiguration, err = NewVCAPConfigurationReader()
			Expect(err).NotTo(HaveOccurred())
		})

		AfterEach(func() {
			os.Unsetenv("VCAP_SERVICES")
			os.Unsetenv("VCAP_APPLICATION")
		})

		Describe("IsRunningOnCF", func() {
			When("VCAP_APPLICATION is not set", func() {
				BeforeEach(func() {
					vcapApplicationJson = ""
				})

				It("returns false when vcap", func() {
					Expect(vcapConfiguration.IsRunningOnCF()).To(BeFalse())
				})
			})
		})

		Describe("MaterializeTLSConfigFromService", func() {
			BeforeEach(func() {
				vcapApplicationJson = `{}`
			})

			When("service has tls ca, cert and key credentials", func() {
				var expectedClientCertContent = "client-cert-content"
				var expectedClientKeyContent = "client-key-content"
				var expectedServerCAContent = "server-ca-content"

				BeforeEach(func() {
					vcapServicesJson = getDbVcapServices(map[string]string{
						"client_cert": expectedClientCertContent,
						"client_key":  expectedClientKeyContent,
						"server_ca":   expectedServerCAContent,
					}, "some-custom-tls-service", "postgres")
				})

				It("returns a tls.Config with the expected values", func() {
					actualTLSConfig, err := vcapConfiguration.MaterializeTLSConfigFromService("some-custom-tls-service")
					Expect(err).NotTo(HaveOccurred())

					expectedTLSConfig := models.TLSCerts{
						KeyFile:    "/tmp/some-custom-tls-service/client_key.sslkey",
						CertFile:   "/tmp/some-custom-tls-service/client_cert.sslcert",
						CACertFile: "/tmp/some-custom-tls-service/server_ca.sslrootcert",
					}

					Expect(actualTLSConfig).To(Equal(expectedTLSConfig))

					By("writing certs to /tmp and assigns them to the TLS config")
					assertCertFile(actualTLSConfig.CertFile, expectedClientCertContent)
					assertCertFile(actualTLSConfig.KeyFile, expectedClientKeyContent)
					assertCertFile(actualTLSConfig.CACertFile, expectedServerCAContent)
				})
			})
		})

		Describe("MaterializeDBFromService", func() {
			BeforeEach(func() {
				vcapApplicationJson = ""
				vcapApplicationJson = `{}`
			})

			When("VCAP_SERVICES has relational db service bind to app", func() {
				When("when uri is wrong", func() {
					BeforeEach(func() {
						vcapServicesJson = getDbVcapServices(map[string]string{
							"uri": "http://example.com/path\x00with/invalid/character",
						}, "some-db", "postgres")
					})

					It("errors", func() {
						_, err = vcapConfiguration.MaterializeDBFromService("some-db")
						Expect(err).To(HaveOccurred())
					})
				})

				When("service uri is present", func() {
					var expectedClientCertContent = "client-cert-content"
					var expectedClientKeyContent = "client-key-content"
					var expectedServerCAContent = "server-ca-content"

					When("postgresDB", func() {
						BeforeEach(func() {
							vcapServicesJson = getDbVcapServices(map[string]string{
								"uri":         "postgres://foo:bar@postgres.example.com:5432/some-db",
								"client_cert": expectedClientCertContent,
								"client_key":  expectedClientKeyContent,
								"server_ca":   expectedServerCAContent,
							}, "some-db", "postgres")
						})

						It("loads the db config from VCAP_SERVICES for postgres db", func() {
							expectedDbUrl := "postgres://foo:bar@postgres.example.com:5432/some-db?sslcert=%2Ftmp%2Fsome-db%2Fclient_cert.sslcert&sslkey=%2Ftmp%2Fsome-db%2Fclient_key.sslkey&sslrootcert=%2Ftmp%2Fsome-db%2Fserver_ca.sslrootcert" // #nosec G101
							dbUrl, err := vcapConfiguration.MaterializeDBFromService("some-db")
							Expect(err).NotTo(HaveOccurred())
							Expect(dbUrl).To(Equal(expectedDbUrl))

							By("writing certs to /tmp and assigns them to the DB config")
							Expect(err).NotTo(HaveOccurred())
							parsedURL, err := url.Parse(dbUrl)
							Expect(err).NotTo(HaveOccurred())
							queryParams := parsedURL.Query()

							actualSSLCertPath := queryParams.Get("sslcert")
							actualSSLKeyPath := queryParams.Get("sslkey")
							actualSSLRootCertPath := queryParams.Get("sslrootcert")

							assertCertFile(actualSSLCertPath, expectedClientCertContent)
							assertCertFile(actualSSLKeyPath, expectedClientKeyContent)
							assertCertFile(actualSSLRootCertPath, expectedServerCAContent)
						})

					})

					When("mysqlDB", func() {
						BeforeEach(func() {
							vcapServicesJson = getDbVcapServices(map[string]string{
								"uri":         "mysql://foo:bar@mysql:3306/some-db",
								"client_cert": expectedClientCertContent,
								"client_key":  expectedClientKeyContent,
								"server_ca":   expectedServerCAContent,
							}, "some-db", "mysql")
						})

						XIt("loads the db config from VCAP_SERVICES for postgres db", func() {
							expectedDbUrl := "mysql://foo:bar@mysql:3306/some-db?ssl-ca=%2Ftmp%2Fsome-db%2Fserver_ca.ssl-ca&ssl-cert=%2Ftmp%2Fsome-db%2Fclient_cert.ssl-cert&ssl-key=%2Ftmp%2Fsome-db%2Fclient_key.ssl-key" // #nosec G101

							dbUrl, err := vcapConfiguration.MaterializeDBFromService("some-db")
							Expect(err).NotTo(HaveOccurred())
							Expect(dbUrl).To(Equal(expectedDbUrl))

							By("writing certs to /tmp and assigns them to the DB config")
							Expect(err).NotTo(HaveOccurred())
							parsedURL, err := url.Parse(dbUrl)
							Expect(err).NotTo(HaveOccurred())
							queryParams := parsedURL.Query()

							actualSSLCertPath := queryParams.Get("ssl-cert")
							actualSSLKeyPath := queryParams.Get("ssl-key")
							actualSSLRootCertPath := queryParams.Get("ssl-ca")

							assertCertFile(actualSSLCertPath, expectedClientCertContent)
							assertCertFile(actualSSLKeyPath, expectedClientKeyContent)
							assertCertFile(actualSSLRootCertPath, expectedServerCAContent)
						})

					})
					AfterEach(func() {
						os.Remove("/tmp/some-db/client_cert.sslcert")
						os.Remove("/tmp/some-db/client_key.sslkey")
						os.Remove("/tmp/some-db/server_ca.sslrootcert")
					})
				})
			})
		})
	})
})

func getDbVcapServices(creds map[string]string, serviceName string, dbType string) string {
	credentials, err := json.Marshal(creds)
	Expect(err).NotTo(HaveOccurred())
	return `{
		"user-provided": [ { "name": "config", "credentials": { "metricsforwarder": { } }}],
		"autoscaler": [ {
			"name": "some-service",
			"credentials": ` + string(credentials) + `,
			"syslog_drain_url": "",
			"tags": ["` + serviceName + `", "` + dbType + `"]
			}
		]}` // #nosec G101
}

func assertCertFile(actualCertPath, expectedContent string) {
	Expect(actualCertPath).NotTo(BeEmpty())
	file, err := os.Open(actualCertPath)
	Expect(err).NotTo(HaveOccurred())
	defer file.Close()
	actualContent, err := io.ReadAll(file)
	Expect(err).NotTo(HaveOccurred())
	Expect(string(actualContent)).To(Equal(expectedContent))
}
