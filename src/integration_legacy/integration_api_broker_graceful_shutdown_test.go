package integration_legacy

import (
	"encoding/base64"
	"fmt"
	"net/http"
	"path/filepath"

	"code.cloudfoundry.org/cfhttp"

	"time"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/gbytes"
	"github.com/onsi/gomega/ghttp"
	"github.com/tedsuo/ifrit/ginkgomon"
)

var _ = Describe("integration_legacy_Api_Broker_Graceful_Shutdown", func() {

	var (
		runner        *ginkgomon.Runner
		buffer        *gbytes.Buffer
		fakeApiServer *ghttp.Server

		serviceInstanceId  string
		bindingId          string
		orgId              string
		spaceId            string
		appId              string
		schedulePolicyJson []byte
	)

	AfterEach(func() {
		stopAll()
	})

	Describe("Shutdown", func() {
		Context("ApiServer", func() {
			BeforeEach(func() {
				initializeHttpClient("api.crt", "api.key", "autoscaler-ca.crt", apiSchedulerHttpRequestTimeout)
				fakeScheduler = ghttp.NewServer()
				apiServerConfPath = components.PrepareApiServerConfig(components.Ports[APIServer], components.Ports[APIPublicServer], false, 200, "", dbUrl, fakeScheduler.URL(), fmt.Sprintf("https://127.0.0.1:%d", components.Ports[ScalingEngine]), fmt.Sprintf("https://127.0.0.1:%d", components.Ports[MetricsCollector]), fmt.Sprintf("https://127.0.0.1:%d", components.Ports[EventGenerator]), fmt.Sprintf("https://127.0.0.1:%d", components.Ports[ServiceBrokerInternal]), true, defaultHttpClientTimeout, 30, 30, tmpDir)
				runner = startApiServer()
				buffer = runner.Buffer()
			})

			AfterEach(func() {
				fakeScheduler.Close()
			})

			Context("with a SIGUSR2 signal", func() {
				It("stops receiving new connections", func() {
					fakeScheduler.AppendHandlers(ghttp.RespondWith(http.StatusOK, "successful"))

					policyStr := readPolicyFromFile("fakePolicyWithSchedule.json")
					resp, err := attachPolicy(getRandomId(), policyStr, components.Ports[APIServer], httpClient)
					Expect(err).NotTo(HaveOccurred())
					Expect(resp.StatusCode).To(Equal(http.StatusCreated))
					resp.Body.Close()

					sendSigusr2Signal(APIServer)
					Eventually(buffer, 5*time.Second).Should(gbytes.Say(`Received SIGUSR2 signal`))

					resp, err = attachPolicy(getRandomId(), policyStr, components.Ports[APIServer], httpClient)
					Expect(err).To(HaveOccurred())

					Eventually(processMap[APIServer].Wait()).Should(Receive())
					Expect(runner.ExitCode()).Should(Equal(0))
				})

				It("waits for in-flight request to finish", func() {
					ch := make(chan struct{})
					fakeScheduler.AppendHandlers(func(w http.ResponseWriter, r *http.Request) {
						defer GinkgoRecover()
						ch <- struct{}{}

						Eventually(ch, 5*time.Second).Should(BeClosed())
						w.WriteHeader(http.StatusOK)
					})

					done := make(chan struct{})
					go func() {
						defer GinkgoRecover()
						policyStr := readPolicyFromFile("fakePolicyWithSchedule.json")
						resp, err := attachPolicy(getRandomId(), policyStr, components.Ports[APIServer], httpClient)
						Expect(err).NotTo(HaveOccurred())
						Expect(resp.StatusCode).To(Equal(http.StatusCreated))
						resp.Body.Close()
						close(done)
					}()

					Eventually(ch, 5*time.Second).Should(Receive())
					sendSigusr2Signal(APIServer)

					Eventually(buffer, 5*time.Second).Should(gbytes.Say(`Received SIGUSR2 signal`))
					Consistently(processMap[APIServer].Wait()).ShouldNot(Receive())
					close(ch)
					Eventually(done).Should(BeClosed())

					Eventually(processMap[APIServer].Wait()).Should(Receive())
					Expect(runner.ExitCode()).Should(BeZero())
				})
			})

			Context("with a SIGKILL signal", func() {
				It("terminates the server immediately without finishing in-flight requests", func() {
					ch := make(chan struct{})
					fakeScheduler.AppendHandlers(func(w http.ResponseWriter, r *http.Request) {
						defer GinkgoRecover()
						ch <- struct{}{}

						Eventually(ch, 5*time.Second).Should(BeClosed())
						w.WriteHeader(http.StatusOK)
					})

					done := make(chan struct{})
					go func() {
						defer GinkgoRecover()
						policyStr := readPolicyFromFile("fakePolicyWithSchedule.json")
						_, err := attachPolicy(getRandomId(), policyStr, components.Ports[APIServer], httpClient)
						Expect(err).To(HaveOccurred())
						close(done)
					}()

					Eventually(ch, 5*time.Second).Should(Receive())

					sendKillSignal(APIServer)
					Eventually(processMap[APIServer].Wait(), 5*time.Second).Should(Receive())

					close(ch)
					Eventually(done).Should(BeClosed())
				})
			})

		})

		Context("Service Broker", func() {
			BeforeEach(func() {
				fakeApiServer = ghttp.NewServer()
				brokerTLSConfig, err := cfhttp.NewTLSConfig(
					filepath.Join(testCertDir, "servicebroker.crt"),
					filepath.Join(testCertDir, "servicebroker.key"),
					filepath.Join(testCertDir, "autoscaler-ca.crt"),
				)
				Expect(err).NotTo(HaveOccurred())
				httpClient.Timeout = brokerApiHttpRequestTimeout
				httpClient.Transport.(*http.Transport).TLSClientConfig = brokerTLSConfig
				serviceBrokerConfPath = components.PrepareServiceBrokerConfig(components.Ports[ServiceBroker], components.Ports[ServiceBrokerInternal], brokerUserName, brokerPassword, false, dbUrl, fakeApiServer.URL(), brokerApiHttpRequestTimeout, tmpDir)
				runner = startServiceBroker()
				buffer = runner.Buffer()

				brokerAuth = base64.StdEncoding.EncodeToString([]byte("username:password"))
				serviceInstanceId = getRandomId()
				orgId = getRandomId()
				spaceId = getRandomId()
				bindingId = getRandomId()
				appId = getRandomId()
				//add a service instance
				resp, err := provisionServiceInstance(serviceInstanceId, orgId, spaceId, nil, components.Ports[ServiceBroker], httpClient)
				Expect(err).NotTo(HaveOccurred())
				Expect(resp.StatusCode).To(Equal(http.StatusCreated))
				resp.Body.Close()

				schedulePolicyJson = readPolicyFromFile("fakePolicyWithSchedule.json")
			})

			AfterEach(func() {
				fakeApiServer.Close()
			})

			Context("with a SIGUSR2 signal", func() {
				It("stops receiving new connections", func() {
					fakeApiServer.AppendHandlers(ghttp.RespondWith(http.StatusCreated, "created"))

					resp, err := bindService(bindingId, appId, serviceInstanceId, schedulePolicyJson, components.Ports[ServiceBroker], httpClient)
					Expect(err).NotTo(HaveOccurred())
					Expect(resp.StatusCode).To(Equal(http.StatusCreated))
					resp.Body.Close()

					sendSigusr2Signal(ServiceBroker)
					Eventually(buffer, 5*time.Second).Should(gbytes.Say(`Received SIGUSR2 signal`))

					_, err = bindService(getRandomId(), appId, serviceInstanceId, schedulePolicyJson, components.Ports[ServiceBroker], httpClient)
					Expect(err).To(HaveOccurred())

					Eventually(processMap[ServiceBroker].Wait(), 10*time.Second).Should(Receive())
					Expect(runner.ExitCode()).Should(BeZero())
				})

				It("waits for in-flight request to finish", func() {
					ch := make(chan struct{})
					fakeApiServer.AppendHandlers(func(w http.ResponseWriter, r *http.Request) {
						defer GinkgoRecover()
						ch <- struct{}{}

						Eventually(ch, 5*time.Second).Should(BeClosed())
						w.WriteHeader(http.StatusCreated)
					})

					done := make(chan struct{})
					go func() {
						defer GinkgoRecover()
						resp, err := bindService(bindingId, appId, serviceInstanceId, schedulePolicyJson, components.Ports[ServiceBroker], httpClient)
						Expect(err).NotTo(HaveOccurred())
						Expect(resp.StatusCode).To(Equal(http.StatusCreated))
						resp.Body.Close()
						close(done)
					}()

					Eventually(ch, 5*time.Second).Should(Receive())
					sendSigusr2Signal(ServiceBroker)

					Eventually(buffer, 5*time.Second).Should(gbytes.Say(`Received SIGUSR2 signal`))
					Consistently(processMap[ServiceBroker].Wait()).ShouldNot(Receive())
					close(ch)
					Eventually(done).Should(BeClosed())

					Eventually(processMap[ServiceBroker].Wait(), 10*time.Second).Should(Receive())
					Expect(runner.ExitCode()).Should(BeZero())
				})
			})

			Context("with a SIGKILL signal", func() {
				It("terminates the server immediately without finishing in-flight requests", func() {
					ch := make(chan struct{})
					fakeApiServer.AppendHandlers(func(w http.ResponseWriter, r *http.Request) {
						defer GinkgoRecover()
						ch <- struct{}{}

						Eventually(ch, 5*time.Second).Should(BeClosed())
						w.WriteHeader(http.StatusCreated)
					})

					done := make(chan struct{})
					go func() {
						defer GinkgoRecover()
						_, err := bindService(bindingId, appId, serviceInstanceId, schedulePolicyJson, components.Ports[ServiceBroker], httpClient)
						Expect(err).To(HaveOccurred())
						close(done)
					}()

					Eventually(ch, 5*time.Second).Should(Receive())

					sendKillSignal(ServiceBroker)
					Eventually(processMap[ServiceBroker].Wait(), 5*time.Second).Should(Receive())

					close(ch)
					Eventually(done).Should(BeClosed())
				})
			})
		})
	})

})
