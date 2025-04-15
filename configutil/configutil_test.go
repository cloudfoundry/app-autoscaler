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

		Describe("ConfigureStoredProcedureDb", func() {
			var dbName string

			var actualDbs *map[string]db.DatabaseConfig
			var expectedDbs *map[string]db.DatabaseConfig
			var storedProcedureUsername string
			var storedProcedurePassword string

			When("storedProcedure_db service is provided and cred_helper_impl is stored_procedure", func() {
				BeforeEach(func() {
					actualDbs = &map[string]db.DatabaseConfig{}
					vcapApplicationJson = `{}`
					dbName = db.StoredProcedureDb
					vcapServicesJson, err = testhelpers.GetStoredProcedureDbVcapServices(map[string]string{
						"uri":         dbUri,
						"client_cert": expectedClientCertContent,
						"client_key":  expectedClientKeyContent,
						"server_ca":   expectedServerCAContent,
					}, dbName, "postgres")
					Expect(err).NotTo(HaveOccurred())
					storedProcedureUsername = "storedProcedureUsername"
					storedProcedurePassword = "storedProcedurePassword"

				})

				It("reads the store procedure service from vcap", func() {
					expectedDbs = &map[string]db.DatabaseConfig{
						dbName: {
							URL: "postgres://storedProcedureUsername:storedProcedurePassword@postgres.example.com:5432/some-db?sslcert=%2Ftmp%2Fstoredprocedure_db%2Fclient_cert.sslcert&sslkey=%2Ftmp%2Fstoredprocedure_db%2Fclient_key.sslkey&sslrootcert=%2Ftmp%2Fstoredprocedure_db%2Fserver_ca.sslrootcert", // #nosec G101
						},
					}
					storedProcedureConfig := &models.StoredProcedureConfig{
						Username: storedProcedureUsername,
						Password: storedProcedurePassword,
					}
					err := vcapConfiguration.ConfigureStoredProcedureDb(dbName, actualDbs, storedProcedureConfig)
					Expect(err).NotTo(HaveOccurred())

					Expect(*actualDbs).To(Equal(*expectedDbs))
				})
			})

		})

		Describe("ConfigureDatabases", func() {
			var actualDbs *map[string]db.DatabaseConfig
			var expectedDbs *map[string]db.DatabaseConfig
			var expectedServerCAContent = "server-ca-content"
			var databaseNames []string

			BeforeEach(func() {
				vcapApplicationJson = `{}`
			})

			When("stored procedure implementation is set to stored_procedure", func() {
				var actualProcedureConfig *models.StoredProcedureConfig

				BeforeEach(func() {
					databaseNames = []string{db.PolicyDb, db.BindingDb, db.StoredProcedureDb}
					vcapServicesJson, err = testhelpers.GetDbVcapServices(map[string]string{
						"uri": dbUri,

						"client_cert": expectedClientCertContent,
						"client_key":  expectedClientKeyContent,
						"server_ca":   expectedServerCAContent,
					}, databaseNames, "postgres")
					Expect(err).NotTo(HaveOccurred())
				})

				When("VCAP_SERVICES has relational db service bind to app for policy db", func() {
					BeforeEach(func() {

						actualDbs = &map[string]db.DatabaseConfig{}
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
						}
					})

					It("loads the db config from VCAP_SERVICES successfully", func() {
						err := vcapConfiguration.ConfigureDatabases(actualDbs, actualProcedureConfig, "stored_procedure")
						Expect(err).NotTo(HaveOccurred())
						Expect(*actualDbs).To(Equal(*expectedDbs))
					})
				})
			})

			When("stored procedure implementation is set to default", func() {
				BeforeEach(func() {
					databaseNames = []string{db.PolicyDb, db.BindingDb}
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
						actualDbs = &map[string]db.DatabaseConfig{}

						expectedDbs = &map[string]db.DatabaseConfig{
							db.PolicyDb: {
								URL: "postgres://foo:bar@postgres.example.com:5432/some-db?sslcert=%2Ftmp%2Fpolicy_db%2Fclient_cert.sslcert&sslkey=%2Ftmp%2Fpolicy_db%2Fclient_key.sslkey&sslrootcert=%2Ftmp%2Fpolicy_db%2Fserver_ca.sslrootcert", // #nosec G101
							},
							db.BindingDb: {
								URL: "postgres://foo:bar@postgres.example.com:5432/some-db?sslcert=%2Ftmp%2Fbinding_db%2Fclient_cert.sslcert&sslkey=%2Ftmp%2Fbinding_db%2Fclient_key.sslkey&sslrootcert=%2Ftmp%2Fbinding_db%2Fserver_ca.sslrootcert", // #nosec G101
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

				When("service uri is present", func() {

					When("postgresDB", func() {
						BeforeEach(func() {
							vcapServicesJson, err = testhelpers.GetDbVcapServices(map[string]string{
								"uri":         dbUri,
								"client_cert": expectedClientCertContent,
								"client_key":  expectedClientKeyContent,
								"server_ca":   expectedServerCAContent,
							}, []string{dbName}, "postgres")
							Expect(err).NotTo(HaveOccurred())
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
							vcapServicesJson, err = testhelpers.GetDbVcapServices(map[string]string{
								"uri":         "mysql://foo:bar@mysql:3306/some-db",
								"client_cert": expectedClientCertContent,
								"client_key":  expectedClientKeyContent,
								"server_ca":   expectedServerCAContent,
							}, []string{dbName}, "mysql")
							Expect(err).NotTo(HaveOccurred())
						})

						XIt("loads the db config from VCAP_SERVICES for postgres db", func() {
							expectedDbUrl := "mysql://foo:bar@mysql:3306/some-db?ssl-ca=%2Ftmp%2Fsome-db%2Fserver_ca.ssl-ca&ssl-cert=%2Ftmp%2Fsome-db%2Fclient_cert.ssl-cert&ssl-key=%2Ftmp%2Fsome-db%2Fclient_key.ssl-key" // #nosec G101

							dbUrl, err := vcapConfiguration.MaterializeDBFromService(dbName)
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

func assertCertFile(actualCertPath, expectedContent string) {
	Expect(actualCertPath).NotTo(BeEmpty())
	file, err := os.Open(actualCertPath)
	Expect(err).NotTo(HaveOccurred())
	defer file.Close()
	actualContent, err := io.ReadAll(file)
	Expect(err).NotTo(HaveOccurred())
	Expect(string(actualContent)).To(Equal(expectedContent))
}
