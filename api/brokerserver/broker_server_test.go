package brokerserver_test

import (
	"io"
	"net/http"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("BrokerServer", func() {
	const (
		catalogPath = "/v2/catalog"
	)
	var (
		rsp *http.Response
		err error
	)
	Context("when username and password are not provided", func() {
		BeforeEach(func() {
			serverUrl.Path = catalogPath
		})

		JustBeforeEach(func() {
			req, err := http.NewRequest(http.MethodGet, serverUrl.String(), nil)
			Expect(err).NotTo(HaveOccurred())
			rsp, err = httpClient.Do(req)
			Expect(err).NotTo(HaveOccurred())
		})

		It("should return 401", func() {
			Expect(err).ToNot(HaveOccurred())
			Expect(rsp.StatusCode).To(Equal(http.StatusUnauthorized))
		})
	})

	Context("when username and password are incorrect", func() {
		BeforeEach(func() {
			serverUrl.Path = catalogPath
		})
		JustBeforeEach(func() {
			req, err := http.NewRequest(http.MethodGet, serverUrl.String(), nil)
			Expect(err).NotTo(HaveOccurred())

			req.SetBasicAuth("wrongusername", "wrongpassword")
			rsp, err = httpClient.Do(req)
			Expect(err).NotTo(HaveOccurred())
		})

		It("should return 401", func() {
			Expect(err).ToNot(HaveOccurred())
			Expect(rsp.StatusCode).To(Equal(http.StatusUnauthorized))
		})
	})

	Context("when username and password are correct", func() {
		BeforeEach(func() {
			serverUrl.Path = catalogPath
		})

		Context("when the 1st username and password are correct", func() {
			JustBeforeEach(func() {
				req, err := http.NewRequest(http.MethodGet, serverUrl.String(), nil)
				Expect(err).NotTo(HaveOccurred())
				req.Header.Set("X-Broker-API-Version", "2.14")

				req.SetBasicAuth(username, password)

				rsp, err = httpClient.Do(req)
				Expect(err).NotTo(HaveOccurred())
			})

			It("should get the catalog", func() {
				Expect(err).ToNot(HaveOccurred())
				Expect(rsp.StatusCode).To(Equal(http.StatusOK))

				bodyBytes, err := io.ReadAll(rsp.Body)
				Expect(err).NotTo(HaveOccurred())

				Expect(bodyBytes).To(MatchJSON(catalogBytes))
			})
		})

		Context("when the 2st username and password are correct", func() {
			JustBeforeEach(func() {
				req, err := http.NewRequest(http.MethodGet, serverUrl.String(), nil)
				Expect(err).NotTo(HaveOccurred())
				req.Header.Set("X-Broker-API-Version", "2.14")

				req.SetBasicAuth(username2, password2)

				rsp, err = httpClient.Do(req)
				Expect(err).ToNot(HaveOccurred())
			})

			It("should get the catalog", func() {
				Expect(rsp.StatusCode).To(Equal(http.StatusOK))

				bodyBytes, err := io.ReadAll(rsp.Body)
				Expect(err).NotTo(HaveOccurred())

				Expect(bodyBytes).To(MatchJSON(catalogBytes))
			})
		})
		BeforeEach(func() {
			serverUrl.Path = catalogPath
		})
		JustBeforeEach(func() {
			req, err := http.NewRequest(http.MethodGet, serverUrl.String(), nil)
			Expect(err).NotTo(HaveOccurred())
			req.Header.Set("X-Broker-API-Version", "2.14")

			req.SetBasicAuth(username, password)

			rsp, err = httpClient.Do(req)
			Expect(err).ToNot(HaveOccurred())
		})
	})

	Context("when requesting non existing path", func() {
		BeforeEach(func() {
			serverUrl.Path = "/non-existing-path"
		})
		JustBeforeEach(func() {
			req, err := http.NewRequest(http.MethodGet, serverUrl.String(), nil)
			Expect(err).NotTo(HaveOccurred())
			req.Header.Set("X-Broker-API-Version", "2.14")

			req.SetBasicAuth(username, password)

			rsp, err = httpClient.Do(req)
			Expect(err).ToNot(HaveOccurred())
		})

		It("should get 404", func() {
			Expect(rsp.StatusCode).To(Equal(http.StatusNotFound))
		})
	})

	Context("when requesting the health endpoint", func() {
		BeforeEach(func() {
			serverUrl.Path = "/health"
		})
		JustBeforeEach(func() {
			req, err := http.NewRequest(http.MethodGet, serverUrl.String(), nil)
			Expect(err).NotTo(HaveOccurred())
			req.Header.Set("X-Broker-API-Version", "2.14")

			rsp, err = httpClient.Do(req)
			Expect(err).ToNot(HaveOccurred())
		})

		It("should get 200", func() {
			Expect(rsp.StatusCode).To(Equal(http.StatusOK))
		})
	})

})
