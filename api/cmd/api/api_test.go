package main_test

import (
	"crypto/rand"
	"crypto/rsa"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"strings"

	"code.cloudfoundry.org/app-autoscaler/src/autoscaler/api/config"
	"code.cloudfoundry.org/app-autoscaler/src/autoscaler/configutil"
	"code.cloudfoundry.org/app-autoscaler/src/autoscaler/db"
	"code.cloudfoundry.org/app-autoscaler/src/autoscaler/testhelpers"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	. "github.com/onsi/gomega/gbytes"
	. "github.com/onsi/gomega/gexec"
)

var _ = Describe("Api", func() {
	var (
		runner *ApiRunner
		rsp    *http.Response

		brokerHttpClient   *http.Client
		healthHttpClient   *http.Client
		apiHttpClient      *http.Client
		cfServerHttpClient *http.Client

		serverURL   *url.URL
		brokerURL   *url.URL
		healthURL   *url.URL
		cfServerURL *url.URL

		vcapPort int
		err      error
	)

	BeforeEach(func() {
		runner = NewApiRunner()

		vcapPort = 8080 + GinkgoParallelProcess()

		brokerHttpClient = testhelpers.NewServiceBrokerClient()
		healthHttpClient = &http.Client{}
		apiHttpClient = testhelpers.NewPublicApiClient()
		cfServerHttpClient = &http.Client{}

		serverURL, err = url.Parse(fmt.Sprintf("https://127.0.0.1:%d", conf.Server.Port))
		Expect(err).NotTo(HaveOccurred())

		brokerURL, err = url.Parse(fmt.Sprintf("https://127.0.0.1:%d", conf.BrokerServer.Port))
		Expect(err).NotTo(HaveOccurred())

		healthURL, err = url.Parse(fmt.Sprintf("http://127.0.0.1:%d", conf.Health.ServerConfig.Port))
		Expect(err).NotTo(HaveOccurred())

		cfServerURL, err = url.Parse(fmt.Sprintf("http://127.0.0.1:%d", vcapPort))

	})

	JustBeforeEach(func() {
		runner.Start()
	})

	Describe("Api configuration check", func() {
		Context("with a missing config file", func() {
			BeforeEach(func() {
				runner.startCheck = ""
				runner.configPath = "bogus"
			})

			It("fails with an error", func() {
				Eventually(runner.Session).Should(Exit(1))
				Expect(runner.Session.Buffer()).To(Say("failed to open config file"))
			})
		})

		Context("with an invalid config file", func() {
			BeforeEach(func() {
				runner.startCheck = ""
				badfile, err := os.CreateTemp("", "bad-ap-config")
				Expect(err).NotTo(HaveOccurred())
				runner.configPath = badfile.Name()
				// #nosec G306
				err = os.WriteFile(runner.configPath, []byte("bogus"), os.ModePerm)
				Expect(err).NotTo(HaveOccurred())
			})

			AfterEach(func() {
				os.Remove(runner.configPath)
			})

			It("fails with an error", func() {
				Eventually(runner.Session).Should(Exit(1))
				Expect(runner.Session.Buffer()).To(Say("failed to read config file"))
			})
		})

		Context("with missing configuration", func() {
			BeforeEach(func() {
				runner.startCheck = ""
				missingConfig := conf

				missingConfig.Db = make(map[string]db.DatabaseConfig)
				missingConfig.Db[db.PolicyDb] = db.DatabaseConfig{URL: ""}
				missingConfig.Db[db.BindingDb] = db.DatabaseConfig{URL: ""}

				var brokerCreds []config.BrokerCredentialsConfig
				missingConfig.BrokerCredentials = brokerCreds

				missingConfig.BrokerServer.Port = 7000 + GinkgoParallelProcess()
				missingConfig.Logging.Level = "debug"
				runner.configPath = writeConfig(&missingConfig).Name()
			})

			AfterEach(func() {
				os.Remove(runner.configPath)
			})

			It("should fail validation", func() {
				Eventually(runner.Session).Should(Exit(1))
				Expect(runner.Session.Buffer()).To(Say("failed to validate configuration"))
			})
		})

	})

	Describe("when interrupt is sent", func() {
		It("should stop", func() {
			runner.Session.Interrupt()
			Eventually(runner.Session, 5).Should(Exit(0))
		})

	})

	Describe("Broker Rest API", func() {
		AfterEach(func() {
			runner.Interrupt()
			Eventually(runner.Session, 5).Should(Exit(0))
		})
		Context("When a request comes to broker catalog", func() {

			It("succeeds with a 200", func() {
				brokerURL.Path = "/v2/catalog"
				req, err := http.NewRequest(http.MethodGet, brokerURL.String(), nil)
				Expect(err).NotTo(HaveOccurred())
				req.Header.Set("X-Broker-API-Version", "2.14")

				req.SetBasicAuth(username, password)

				rsp, err = brokerHttpClient.Do(req)
				Expect(err).ToNot(HaveOccurred())

				Expect(rsp.StatusCode).To(Equal(http.StatusOK))
				if rsp.StatusCode != http.StatusOK {
					Fail(fmt.Sprintf("Not ok:%d", rsp.StatusCode))
				}

				bodyBytes, err := io.ReadAll(rsp.Body)

				testhelpers.FailOnError("Read failed", err)
				if len(bodyBytes) == 0 {
					Fail("body empty")
				}
				Expect(err).NotTo(HaveOccurred())
				Expect(string(bodyBytes)).To(MatchJSON(catalogBytes))
			})
		})
	})

	Describe("Pubic API", func() {
		AfterEach(func() {
			runner.Interrupt()
			Eventually(runner.Session, 5).Should(Exit(0))
		})
		Context("When a request comes to public api info", func() {
			It("succeeds with a 200", func() {
				serverURL.Path = "/v1/info"
				req, err := http.NewRequest(http.MethodGet, serverURL.String(), nil)
				Expect(err).NotTo(HaveOccurred())

				rsp, err = apiHttpClient.Do(req)
				Expect(err).ToNot(HaveOccurred())
				Expect(rsp.StatusCode).To(Equal(http.StatusOK))

				bodyBytes, err := io.ReadAll(rsp.Body)
				Expect(err).NotTo(HaveOccurred())
				Expect(string(bodyBytes)).To(MatchJSON(infoBytes))
			})
		})
	})

	Describe("when Health server is ready to serve RESTful API", func() {
		BeforeEach(func() {
			basicAuthConfig := conf
			basicAuthConfig.Health.BasicAuth.Username = ""
			basicAuthConfig.Health.BasicAuth.Password = ""
			runner.configPath = writeConfig(&basicAuthConfig).Name()
		})
		AfterEach(func() {
			runner.Interrupt()
			Eventually(runner.Session, 5).Should(Exit(0))
		})

		When("a request to query health comes", func() {
			It("returns with a 200", func() {
				testhelpers.CheckHealthResponse(healthHttpClient, healthURL.String(), []string{
					"autoscaler_golangapiserver_concurrent_http_request", "autoscaler_golangapiserver_policyDB",
					"autoscaler_golangapiserver_bindingDB", "go_goroutines", "go_memstats_alloc_bytes",
				})
			})
		})
	})

	Describe("when Health server is ready to serve RESTful API with basic Auth", func() {
		AfterEach(func() {
			runner.Interrupt()
			Eventually(runner.Session, 5).Should(Exit(0))
		})

		When("Health server is ready to serve RESTful API with basic Auth", func() {
			When("username and password are incorrect for basic authentication during health check", func() {
				It("should return 401", func() {
					testhelpers.CheckHealthAuth(GinkgoT(), healthHttpClient, healthURL.String(), "wrongusername", "wrongpassword", http.StatusUnauthorized)
				})
			})

			When("username and password are correct for basic authentication during health check", func() {
				It("should return 200", func() {
					testhelpers.CheckHealthAuth(GinkgoT(), healthHttpClient, healthURL.String(), conf.Health.BasicAuth.Username, conf.Health.BasicAuth.Password, http.StatusOK)
				})
			})
		})
	})

	Describe("can start with default plugin", func() {
		BeforeEach(func() {
			pluginPathConfig := conf
			pluginPathConfig.CredHelperImpl = "default"
			runner.configPath = writeConfig(&pluginPathConfig).Name()
		})
		AfterEach(func() {
			runner.Interrupt()
			Eventually(runner.Session, 5).Should(Exit(0))
		})
		When("a request to query health comes", func() {
			It("returns with a 200", func() {
				serverURL.Path = "/v1/info"
				req, err := http.NewRequest(http.MethodGet, serverURL.String(), nil)
				Expect(err).NotTo(HaveOccurred())

				rsp, err = apiHttpClient.Do(req)
				Expect(err).ToNot(HaveOccurred())
				Expect(rsp.StatusCode).To(Equal(http.StatusOK))

				bodyBytes, err := io.ReadAll(rsp.Body)
				Expect(err).NotTo(HaveOccurred())
				Expect(string(bodyBytes)).To(MatchJSON(infoBytes))
			})
		})
	})

	When("running CF server", func() {
		var (
			cfInstanceKeyFile  string
			cfInstanceCertFile string
		)

		BeforeEach(func() {
			rsaPrivateKey, err := rsa.GenerateKey(rand.Reader, 2048)
			Expect(err).NotTo(HaveOccurred())

			cfInstanceCert, err := testhelpers.GenerateClientCertWithPrivateKey("org-guid", "space-guid", rsaPrivateKey)
			Expect(err).NotTo(HaveOccurred())

			certTmpDir := os.TempDir()

			cfInstanceCertFile, err := configutil.MaterializeContentInFile(certTmpDir, "eventgenerator.crt", string(cfInstanceCert))
			Expect(err).NotTo(HaveOccurred())
			os.Setenv("CF_INSTANCE_CERT", cfInstanceCertFile)

			cfInstanceKey := testhelpers.GenerateClientKeyWithPrivateKey(rsaPrivateKey)
			cfInstanceKeyFile, err = configutil.MaterializeContentInFile(certTmpDir, "eventgenerator.key", string(cfInstanceKey))
			Expect(err).NotTo(HaveOccurred())
			os.Setenv("CF_INSTANCE_KEY", cfInstanceKeyFile)

			os.Setenv("VCAP_APPLICATION", "{}")
			os.Setenv("VCAP_SERVICES", getVcapServices())
			os.Setenv("PORT", fmt.Sprintf("%d", vcapPort))
		})
		AfterEach(func() {
			runner.Interrupt()
			Eventually(runner.Session, 5).Should(Exit(0))

			os.Remove(cfInstanceKeyFile)
			os.Remove(cfInstanceCertFile)

			os.Unsetenv("CF_INSTANCE_KEY")
			os.Unsetenv("CF_INSTANCE_CERT")
			os.Unsetenv("VCAP_APPLICATION")
			os.Unsetenv("VCAP_SERVICES")
			os.Unsetenv("PORT")
		})

		It("should start a cf server", func() {
			req, err := http.NewRequest(http.MethodGet, fmt.Sprintf("%s/v1/info", cfServerURL), nil)
			Expect(err).NotTo(HaveOccurred())

			rsp, err = cfServerHttpClient.Do(req)
			Expect(err).ToNot(HaveOccurred())

			bodyBytes, err := io.ReadAll(rsp.Body)
			Expect(err).ToNot(HaveOccurred())
			Expect(bodyBytes).To(ContainSubstring("Automatically increase or decrease the number of application instances based on a policy you define."))

			req, err = http.NewRequest(http.MethodGet, fmt.Sprintf("%s/v2/catalog", cfServerURL), nil)
			Expect(err).NotTo(HaveOccurred())
			req.Header.Set("X-Broker-API-Version", "2.14")
			req.SetBasicAuth(username, password)

			rsp, err = cfServerHttpClient.Do(req)
			Expect(err).ToNot(HaveOccurred())
			Expect(rsp.StatusCode).To(Equal(http.StatusOK))

			bodyBytes, err = io.ReadAll(rsp.Body)
			Expect(err).ToNot(HaveOccurred())
			Expect(bodyBytes).To(ContainSubstring("autoscaler-free-plan-id"))
		})
	})
})

func getVcapServices() (result string) {
	var dbType string

	// read file
	dbClientCert, err := os.ReadFile("../../../../../test-certs/postgres.crt")
	Expect(err).NotTo(HaveOccurred())
	dbClientKey, err := os.ReadFile("../../../../../test-certs/postgres.key")
	Expect(err).NotTo(HaveOccurred())
	dbClientCA, err := os.ReadFile("../../../../../test-certs/autoscaler-ca.crt")
	Expect(err).NotTo(HaveOccurred())

	catalogBytes, err := os.ReadFile("../../exampleconfig/catalog-example.json")
	Expect(err).NotTo(HaveOccurred())

	dbURL := os.Getenv("DBURL")
	Expect(dbURL).NotTo(BeEmpty())

	if strings.Contains(dbURL, "postgres") {
		dbType = "postgres"
	} else {
		dbType = "mysql"
	}

	result = `{
			"user-provided": [
			  { "name": "apiserver-config", "tags": ["apiserver-config"], "credentials": { "apiserver-config": { } }},
			  { "name": "broker-catalog", "tags": ["broker-catalog"], "credentials": { "broker-catalog": ` + string(catalogBytes) + ` }}
            ],
			"autoscaler": [ {
				"name": "some-service",
				"credentials": {
					"uri": "` + dbURL + `",
					"client_cert": "` + strings.ReplaceAll(string(dbClientCert), "\n", "\\n") + `",
					"client_key": "` + strings.ReplaceAll(string(dbClientKey), "\n", "\\n") + `",
					"server_ca": "` + strings.ReplaceAll(string(dbClientCA), "\n", "\\n") + `"
				},
				"syslog_drain_url": "",
				"tags": ["policy_db", "binding_db", "` + dbType + `"]
			}]}` // #nosec G101

	return result
}
