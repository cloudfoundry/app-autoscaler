package integration_test

import (
	"fmt"
	. "integration"

	"net"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/ghttp"
)

var _ = Describe("Integration_Api_Broker_Graceful_Shutdown", func() {

	const (
		MessageForServer = "message_for_server"
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
				startApiServer()
			})

			It("stops receiving new connections after being interrupted", func() {
				conn, err := net.Dial("tcp", fmt.Sprintf("127.0.0.1:%d", components.Ports[APIServer]))
				Expect(err).NotTo(HaveOccurred())

				_, err = conn.Write([]byte(MessageForServer))
				Expect(err).NotTo(HaveOccurred())

				newConn, err := net.Dial("tcp", fmt.Sprintf("127.0.0.1:%d", components.Ports[APIServer]))
				Expect(err).NotTo(HaveOccurred())

				conn.Close()
				newConn.Close()

				sendSigusr2Signal(APIServer)

				_, err = net.Dial("tcp", fmt.Sprintf("127.0.0.1:%d", components.Ports[APIServer]))
				Expect(err).To(HaveOccurred())

			})

			It("waits for in-flight request to finish before shutting down", func() {
				conn, err := net.Dial("tcp", fmt.Sprintf("127.0.0.1:%d", components.Ports[APIServer]))
				Expect(err).NotTo(HaveOccurred())

				sendSigusr2Signal(APIServer)

				_, err = net.Dial("tcp", fmt.Sprintf("127.0.0.1:%d", components.Ports[APIServer]))
				Expect(err).To(HaveOccurred())

				_, err = conn.Write([]byte(MessageForServer))
				Expect(err).NotTo(HaveOccurred())

				conn.Close()

			})

		})

		Context("Service Broker", func() {

			BeforeEach(func() {
				startServiceBroker()
			})

			It("stops receiving new connections after being interrupted", func() {
				conn, err := net.Dial("tcp", fmt.Sprintf("127.0.0.1:%d", components.Ports[ServiceBroker]))
				Expect(err).NotTo(HaveOccurred())

				_, err = conn.Write([]byte(MessageForServer))
				Expect(err).NotTo(HaveOccurred())

				newConn, err := net.Dial("tcp", fmt.Sprintf("127.0.0.1:%d", components.Ports[ServiceBroker]))
				Expect(err).NotTo(HaveOccurred())

				conn.Close()
				newConn.Close()

				sendSigusr2Signal(ServiceBroker)

				_, err = net.Dial("tcp", fmt.Sprintf("127.0.0.1:%d", components.Ports[ServiceBroker]))
				Expect(err).To(HaveOccurred())

			})

			It("waits for in-flight request to finish before shutting down", func() {
				conn, err := net.Dial("tcp", fmt.Sprintf("127.0.0.1:%d", components.Ports[ServiceBroker]))
				Expect(err).NotTo(HaveOccurred())

				sendSigusr2Signal(ServiceBroker)

				_, err = net.Dial("tcp", fmt.Sprintf("127.0.0.1:%d", components.Ports[ServiceBroker]))
				Expect(err).To(HaveOccurred())

				_, err = conn.Write([]byte(MessageForServer))
				Expect(err).NotTo(HaveOccurred())

				conn.Close()

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
