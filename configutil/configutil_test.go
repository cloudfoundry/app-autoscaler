package configutil_test

import (
	"io"
	"net/url"
	"os"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	. "code.cloudfoundry.org/app-autoscaler/src/autoscaler/configutil"
	"code.cloudfoundry.org/app-autoscaler/src/autoscaler/db"
	"code.cloudfoundry.org/app-autoscaler/src/autoscaler/models"
	"code.cloudfoundry.org/app-autoscaler/src/autoscaler/testhelpers"
)

var _ = Describe("Configutil", func() {
	var _ = Describe("ToJSON", func() {
		type SampleConfig struct {
			Name string `json:"name"`
			Age  int    `json:"age"`
		}

		It("marshals struct to JSON", func() {
			s := SampleConfig{Name: "Alice", Age: 30}
			result, err := ToJSON(s)
			Expect(err).To(BeNil())
			Expect(result).To(MatchJSON(`{"name":"Alice","age":30}`))
		})

		It("fails to marshal unsupported type", func() {
			ch := make(chan int)
			_, err := ToJSON(ch)
			Expect(err).To(HaveOccurred())
		})
	})

	Describe("VCAPConfiguration", func() {
		var (
			vcapConfiguration         *VCAPConfiguration
			vcapApplicationJson       string
			vcapServicesJson          string
			dbUri                     = "postgres://foo:bar@postgres.example.com:5432/some-db" // #nosec G101
			expectedClientKeyContent  = "client-key-content"
			expectedServerCAContent   = "server-ca-content"
			expectedClientCertContent = "client-cert-content"
			err                       error
		)

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

		Describe("GetInstanceTLSCerts", func() {
			BeforeEach(func() {
				os.Setenv("CF_INSTANCE_KEY", "some/path/in/container/cfcert.key")
				os.Setenv("CF_INSTANCE_CERT", "some/path/in/container/cfcert.crt")
				os.Setenv("CF_INSTANCE_CA_CERT", "some/path/in/container/cfcert.crt")
			})

			AfterEach(func() {
				os.Unsetenv("CF_INSTANCE_KEY")
				os.Unsetenv("CF_INSTANCE_CERT")
				os.Unsetenv("CF_INSTANCE_CA_CERT")
			})

			It("returns cf instance TlSClientCert", func() {
				result := vcapConfiguration.GetInstanceTLSCerts()
				Expect(result.CACertFile).To(Equal("some/path/in/container/cfcert.crt"))
				Expect(result.CertFile).To(Equal("some/path/in/container/cfcert.crt"))
				Expect(result.KeyFile).To(Equal("some/path/in/container/cfcert.key"))
			})

		})

		Describe("GetOrgGuid", func() {
			BeforeEach(func() {
				vcapApplicationJson = `{"organization_id":"some-org-id"}`
				vcapServicesJson = `{}`
			})
			It("returns the org guid", func() {
				Expect(vcapConfiguration.GetOrgGuid()).To(Equal("some-org-id"))
			})
		})

		Describe("GetSpaceGuid", func() {
			BeforeEach(func() {
				vcapApplicationJson = `{"space_id":"some-space-id"}`
				vcapServicesJson = `{}`
			})

			It("returns the space guid", func() {
				Expect(vcapConfiguration.GetSpaceGuid()).To(Equal("some-space-id"))
			})

		})

		Describe("GetInstanceIndex", func() {
			BeforeEach(func() {
				os.Setenv("CF_INSTANCE_INDEX", "1")
			})
			It("returns the instance index", func() {
				Expect(vcapConfiguration.GetInstanceIndex()).To(Equal(1))
			})
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

				BeforeEach(func() {
					vcapServicesJson, err = testhelpers.GetDbVcapServices(map[string]string{
						"client_cert": expectedClientCertContent,
						"client_key":  expectedClientKeyContent,
						"server_ca":   expectedServerCAContent,
					}, []string{"some-custom-tls-service"}, "postgres")
					Expect(err).NotTo(HaveOccurred())
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

		Describe("ConfigureDatabases", func() {
			var actualDbs *map[string]db.DatabaseConfig
			var expectedDbs *map[string]db.DatabaseConfig
			var expectedServerCAContent = "server-ca-content"

			BeforeEach(func() {
				vcapApplicationJson = `{}`
				actualDbs = &map[string]db.DatabaseConfig{}
			})
			When("stored procedure implementation is set to stored_procedure", func() {
				var actualProcedureConfig *models.StoredProcedureConfig

				BeforeEach(func() {
					var databaseNames = []string{db.PolicyDb, db.BindingDb, db.StoredProcedureDb}
					vcapServicesJson, err = testhelpers.GetDbVcapServices(map[string]string{
						"uri":         dbUri,
						"client_cert": expectedClientCertContent,
						"client_key":  expectedClientKeyContent,
						"server_ca":   expectedServerCAContent,
					}, databaseNames, "postgres")

					Expect(err).NotTo(HaveOccurred())
				})

				When("VCAP_SERVICES has relational db service bind to app for policy db", func() {
					BeforeEach(func() {

						actualProcedureConfig = &models.StoredProcedureConfig{}

						expectedDbs = &map[string]db.DatabaseConfig{
							db.PolicyDb: {
								URL: "postgres://foo:bar@postgres.example.com:5432/some-db?sslcert=%2Ftmp%2Fpolicy_db%2Fclient_cert.sslcert&sslkey=%2Ftmp%2Fpolicy_db%2Fclient_key.sslkey&sslrootcert=%2Ftmp%2Fpolicy_db%2Fserver_ca.sslrootcert", // #nosec G101
							},
							db.BindingDb: {
								URL: "postgres://foo:bar@postgres.example.com:5432/some-db?sslcert=%2Ftmp%2Fbinding_db%2Fclient_cert.sslcert&sslkey=%2Ftmp%2Fbinding_db%2Fclient_key.sslkey&sslrootcert=%2Ftmp%2Fbinding_db%2Fserver_ca.sslrootcert", // #nosec G101
							},
							db.StoredProcedureDb: {
								URL: "postgres://foo:bar@postgres.example.com:5432/some-db?sslcert=%2Ftmp%2Fstoredprocedure_db%2Fclient_cert.sslcert&sslkey=%2Ftmp%2Fstoredprocedure_db%2Fclient_key.sslkey&sslrootcert=%2Ftmp%2Fstoredprocedure_db%2Fserver_ca.sslrootcert",
							},
							db.LockDb:          {},
							db.ScalingEngineDb: {},
							db.AppMetricsDb:    {},
							db.SchedulerDb:     {},
						}
					})

					It("loads the db config from VCAP_SERVICES successfully", func() {
						err := vcapConfiguration.ConfigureDatabases(actualDbs, actualProcedureConfig, "stored_procedure")
						Expect(err).NotTo(HaveOccurred())
						Expect(*actualDbs).To(Equal(*expectedDbs))
					})
				})

				When("stored procedure username and password are provided", func() {
					BeforeEach(func() {
						actualProcedureConfig = &models.StoredProcedureConfig{
							Username: "storedProcedureUsername",
							Password: "storedProcedurePassword",
						}
					})

					It("overrides default url credentials with stored procedure config username and password", func() {
						expectedDbs = &map[string]db.DatabaseConfig{
							db.PolicyDb: {
								URL: "postgres://foo:bar@postgres.example.com:5432/some-db?sslcert=%2Ftmp%2Fpolicy_db%2Fclient_cert.sslcert&sslkey=%2Ftmp%2Fpolicy_db%2Fclient_key.sslkey&sslrootcert=%2Ftmp%2Fpolicy_db%2Fserver_ca.sslrootcert", // #nosec G101
							},
							db.BindingDb: {
								URL: "postgres://foo:bar@postgres.example.com:5432/some-db?sslcert=%2Ftmp%2Fbinding_db%2Fclient_cert.sslcert&sslkey=%2Ftmp%2Fbinding_db%2Fclient_key.sslkey&sslrootcert=%2Ftmp%2Fbinding_db%2Fserver_ca.sslrootcert", // #nosec G101
							},
							db.StoredProcedureDb: {
								URL: "postgres://storedProcedureUsername:storedProcedurePassword@postgres.example.com:5432/some-db?sslcert=%2Ftmp%2Fstoredprocedure_db%2Fclient_cert.sslcert&sslkey=%2Ftmp%2Fstoredprocedure_db%2Fclient_key.sslkey&sslrootcert=%2Ftmp%2Fstoredprocedure_db%2Fserver_ca.sslrootcert", // #nosec G101
							},
							db.AppMetricsDb:    {},
							db.ScalingEngineDb: {},
							db.LockDb:          {},
							db.SchedulerDb:     {},
						}

						err := vcapConfiguration.ConfigureDatabases(actualDbs, actualProcedureConfig, "stored_procedure")
						Expect(err).NotTo(HaveOccurred())
						Expect(*actualDbs).To(Equal(*expectedDbs))
					})
				})
			})

			When("stored procedure implementation is set to default", func() {

				BeforeEach(func() {
					vcapServicesJson, err = testhelpers.GetDbVcapServices(map[string]string{
						"uri":         dbUri,
						"client_cert": expectedClientCertContent,
						"client_key":  expectedClientKeyContent,
						"server_ca":   expectedServerCAContent,
					}, AvailableDatabases, "postgres")
					Expect(err).NotTo(HaveOccurred())
				})

				When("VCAP_SERVICES has relational db service bind to app for policy db", func() {
					BeforeEach(func() {
						actualDbs = &map[string]db.DatabaseConfig{}

						expectedDbs = &map[string]db.DatabaseConfig{
							db.PolicyDb: {
								URL: "postgres://foo:bar@postgres.example.com:5432/some-db?sslcert=%2Ftmp%2Fpolicy_db%2Fclient_cert.sslcert&sslkey=%2Ftmp%2Fpolicy_db%2Fclient_key.sslkey&sslrootcert=%2Ftmp%2Fpolicy_db%2Fserver_ca.sslrootcert", // #nosec G101
							},
							db.BindingDb: {
								URL: "postgres://foo:bar@postgres.example.com:5432/some-db?sslcert=%2Ftmp%2Fbinding_db%2Fclient_cert.sslcert&sslkey=%2Ftmp%2Fbinding_db%2Fclient_key.sslkey&sslrootcert=%2Ftmp%2Fbinding_db%2Fserver_ca.sslrootcert", // #nosec G101
							},
							db.AppMetricsDb: {
								URL: "postgres://foo:bar@postgres.example.com:5432/some-db?sslcert=%2Ftmp%2Fappmetrics_db%2Fclient_cert.sslcert&sslkey=%2Ftmp%2Fappmetrics_db%2Fclient_key.sslkey&sslrootcert=%2Ftmp%2Fappmetrics_db%2Fserver_ca.sslrootcert", // #nosec G101
							},
							db.SchedulerDb: {
								URL: "postgres://foo:bar@postgres.example.com:5432/some-db?sslcert=%2Ftmp%2Fscheduler_db%2Fclient_cert.sslcert&sslkey=%2Ftmp%2Fscheduler_db%2Fclient_key.sslkey&sslrootcert=%2Ftmp%2Fscheduler_db%2Fserver_ca.sslrootcert", // #nosec G101
							},
							db.LockDb: {
								URL: "postgres://foo:bar@postgres.example.com:5432/some-db?sslcert=%2Ftmp%2Flock_db%2Fclient_cert.sslcert&sslkey=%2Ftmp%2Flock_db%2Fclient_key.sslkey&sslrootcert=%2Ftmp%2Flock_db%2Fserver_ca.sslrootcert", // #nosec G101
							},
							db.ScalingEngineDb: {
								URL: "postgres://foo:bar@postgres.example.com:5432/some-db?sslcert=%2Ftmp%2Fscalingengine_db%2Fclient_cert.sslcert&sslkey=%2Ftmp%2Fscalingengine_db%2Fclient_key.sslkey&sslrootcert=%2Ftmp%2Fscalingengine_db%2Fserver_ca.sslrootcert", // #nosec G101
							},
						}
					})

					It("loads the db config from VCAP_SERVICES successfully", func() {
						err := vcapConfiguration.ConfigureDatabases(actualDbs, nil, "default")
						Expect(err).NotTo(HaveOccurred())
						Expect(*actualDbs).To(Equal(*expectedDbs))
					})
				})
			})
		})

		Describe("MaterializeDBFromService", func() {
			var dbName string
			BeforeEach(func() {
				dbName = "some-db"
				vcapApplicationJson = `{}`
			})

			When("VCAP_SERVICES has relational db service bind to app", func() {
				When("when uri is wrong", func() {
					BeforeEach(func() {
						vcapServicesJson, err = testhelpers.GetDbVcapServices(map[string]string{
							"uri": "http://example.com/path\x00with/invalid/character",
						}, []string{dbName}, "postgres")
						Expect(err).NotTo(HaveOccurred())
					})

					It("errors", func() {
						_, err = vcapConfiguration.MaterializeDBFromService(dbName)
						Expect(err).To(HaveOccurred())
					})
				})

				When("service postgresDB uri is present", func() {
					BeforeEach(func() {
						vcapServicesJson, err = testhelpers.GetDbVcapServices(map[string]string{
							"uri":         dbUri,
							"client_cert": expectedClientCertContent,
							"client_key":  expectedClientKeyContent,
							"server_ca":   expectedServerCAContent,
						}, []string{dbName}, "postgres")
						Expect(err).NotTo(HaveOccurred())
					})

					XIt("loads the db config from VCAP_SERVICES for postgres db", func() {
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

				AfterEach(func() {
					os.Remove("/tmp/some-db/client_cert.sslcert")
					os.Remove("/tmp/some-db/client_key.sslkey")
					os.Remove("/tmp/some-db/server_ca.sslrootcert")
				})
			})
		})
	})
})

func assertCertFile(actualCertPath, expectedContent string) {
	Expect(actualCertPath).NotTo(BeEmpty())
	file, err := os.Open(actualCertPath)
	Expect(err).NotTo(HaveOccurred())
	defer file.Close()
	actualContent, err := io.ReadAll(file)
	Expect(err).NotTo(HaveOccurred())
	Expect(string(actualContent)).To(Equal(expectedContent))
}
