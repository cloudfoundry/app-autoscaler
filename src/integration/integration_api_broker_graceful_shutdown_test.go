package integration_test

import (
	"fmt"
	. "integration"

	"net"

	"time"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/gbytes"
	"github.com/onsi/gomega/ghttp"
	"github.com/tedsuo/ifrit/ginkgomon"
)

var _ = Describe("Integration_Api_Broker_Graceful_Shutdown", func() {

	const (
		MessageForServer = "message_for_server"
	)

	var (
		runner *ginkgomon.Runner
		buffer *gbytes.Buffer
	)

	BeforeEach(func() {
		fakeScheduler = ghttp.NewServer()
		fakeApiServer := ghttp.NewServer()

		apiServerConfPath = components.PrepareApiServerConfig(components.Ports[APIServer], dbUrl, fakeScheduler.URL(), tmpDir)
		serviceBrokerConfPath = components.PrepareServiceBrokerConfig(components.Ports[ServiceBroker], brokerUserName, brokerPassword, dbUrl, fakeApiServer.URL(), brokerApiHttpRequestTimeout, tmpDir)

	})

	AfterEach(func() {
		stopAll()
	})

	Describe("Graceful Shutdown", func() {

		Context("ApiServer", func() {

			BeforeEach(func() {
				runner = startApiServer()
				buffer = runner.Buffer()
			})

			It("stops receiving new connections on receiving SIGUSR2 signal", func() {
				conn, err := net.Dial("tcp", fmt.Sprintf("127.0.0.1:%d", components.Ports[APIServer]))
				Expect(err).NotTo(HaveOccurred())

				_, err = conn.Write([]byte(MessageForServer))
				Expect(err).NotTo(HaveOccurred())

				newConn, err := net.Dial("tcp", fmt.Sprintf("127.0.0.1:%d", components.Ports[APIServer]))
				Expect(err).NotTo(HaveOccurred())

				conn.Close()
				newConn.Close()

				sendSigusr2Signal(APIServer)
				Eventually(buffer, 5*time.Second).Should(gbytes.Say(`Received SIGUSR2 signal`))

				Eventually(processMap[APIServer].Wait()).Should(Receive())
				Expect(runner.ExitCode()).Should(Equal(0))

				_, err = net.Dial("tcp", fmt.Sprintf("127.0.0.1:%d", components.Ports[APIServer]))
				Expect(err).To(HaveOccurred())

			})

			It("waits for in-flight request to finish before shutting down server on receiving SIGUSR2 signal", func() {
				conn, err := net.Dial("tcp", fmt.Sprintf("127.0.0.1:%d", components.Ports[APIServer]))
				Expect(err).NotTo(HaveOccurred())

				sendSigusr2Signal(APIServer)
				Eventually(buffer, 5*time.Second).Should(gbytes.Say(`Received SIGUSR2 signal`))

				Expect(runner.ExitCode()).Should(Equal(-1))

				_, err = net.Dial("tcp", fmt.Sprintf("127.0.0.1:%d", components.Ports[APIServer]))
				Expect(err).To(HaveOccurred())

				_, err = conn.Write([]byte(MessageForServer))
				Expect(err).NotTo(HaveOccurred())

				conn.Close()

				Eventually(processMap[APIServer].Wait()).Should(Receive())
				Expect(runner.ExitCode()).Should(Equal(0))
			})

		})

		Context("Service Broker", func() {

			BeforeEach(func() {
				runner = startServiceBroker()
				buffer = runner.Buffer()
			})

			It("stops receiving new connections on receiving SIGUSR2 signal", func() {
				conn, err := net.Dial("tcp", fmt.Sprintf("127.0.0.1:%d", components.Ports[ServiceBroker]))
				Expect(err).NotTo(HaveOccurred())

				_, err = conn.Write([]byte(MessageForServer))
				Expect(err).NotTo(HaveOccurred())

				newConn, err := net.Dial("tcp", fmt.Sprintf("127.0.0.1:%d", components.Ports[ServiceBroker]))
				Expect(err).NotTo(HaveOccurred())

				conn.Close()
				newConn.Close()

				sendSigusr2Signal(ServiceBroker)
				Eventually(buffer, 5*time.Second).Should(gbytes.Say(`Received SIGUSR2 signal`))

				Eventually(processMap[ServiceBroker].Wait()).Should(Receive())
				Expect(runner.ExitCode()).Should(Equal(0))

				_, err = net.Dial("tcp", fmt.Sprintf("127.0.0.1:%d", components.Ports[ServiceBroker]))
				Expect(err).To(HaveOccurred())

			})

			It("waits for in-flight request to finish before shutting down server on receiving SIGUSR2 signal", func() {
				conn, err := net.Dial("tcp", fmt.Sprintf("127.0.0.1:%d", components.Ports[ServiceBroker]))
				Expect(err).NotTo(HaveOccurred())

				sendSigusr2Signal(ServiceBroker)
				Eventually(buffer, 5*time.Second).Should(gbytes.Say(`Received SIGUSR2 signal`))

				Expect(runner.ExitCode()).Should(Equal(-1))

				_, err = net.Dial("tcp", fmt.Sprintf("127.0.0.1:%d", components.Ports[ServiceBroker]))
				Expect(err).To(HaveOccurred())

				_, err = conn.Write([]byte(MessageForServer))
				Expect(err).NotTo(HaveOccurred())

				conn.Close()

				Eventually(processMap[ServiceBroker].Wait()).Should(Receive())
				Expect(runner.ExitCode()).Should(Equal(0))
			})

		})

	})

	Describe("Non Graceful Shutdown", func() {

		Context("ApiServer", func() {

			BeforeEach(func() {
				startApiServer()
			})

			It("kills server immediately without serving the in-flight requests", func() {
				conn, err := net.Dial("tcp", fmt.Sprintf("127.0.0.1:%d", components.Ports[APIServer]))
				Expect(err).NotTo(HaveOccurred())

				_, err = conn.Write([]byte(MessageForServer))
				Expect(err).NotTo(HaveOccurred())

				sendKillSignal(APIServer)

				_, err = conn.Write([]byte(MessageForServer))
				Expect(err).To(HaveOccurred())

				_, err = net.Dial("tcp", fmt.Sprintf("127.0.0.1:%d", components.Ports[APIServer]))
				Expect(err).To(HaveOccurred())

			})
		})

		Context("Service Broker", func() {

			BeforeEach(func() {
				startServiceBroker()
			})

			It("kills server immediately without serving the in-flight requests", func() {
				conn, err := net.Dial("tcp", fmt.Sprintf("127.0.0.1:%d", components.Ports[ServiceBroker]))
				Expect(err).NotTo(HaveOccurred())

				_, err = conn.Write([]byte(MessageForServer))
				Expect(err).NotTo(HaveOccurred())

				sendKillSignal(ServiceBroker)

				_, err = conn.Write([]byte(MessageForServer))
				Expect(err).To(HaveOccurred())

				_, err = net.Dial("tcp", fmt.Sprintf("127.0.0.1:%d", components.Ports[ServiceBroker]))
				Expect(err).To(HaveOccurred())

			})
		})

	})

})
