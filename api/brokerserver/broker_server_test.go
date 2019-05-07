package brokerserver_test

import (
	"io/ioutil"
	"net/http"

	. "github.com/onsi/ginkgo"
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
		JustBeforeEach(func() {
			req, err := http.NewRequest(http.MethodGet, serverUrl.String(), nil)
			Expect(err).NotTo(HaveOccurred())

			req.SetBasicAuth(username, password)

			rsp, err = httpClient.Do(req)
		})

		It("should get the catalog", func() {
			Expect(err).ToNot(HaveOccurred())
			Expect(rsp.StatusCode).To(Equal(http.StatusOK))

			bodyBytes, err := ioutil.ReadAll(rsp.Body)
			Expect(err).NotTo(HaveOccurred())

			Expect(bodyBytes).To(Equal(catalogBytes))
		})
	})

	Context("when requesting non existing path", func() {
		BeforeEach(func() {
			serverUrl.Path = "/non-existing-path"
		})
		JustBeforeEach(func() {
			req, err := http.NewRequest(http.MethodGet, serverUrl.String(), nil)
			Expect(err).NotTo(HaveOccurred())

			req.SetBasicAuth(username, password)

			rsp, err = httpClient.Do(req)
		})

		It("should get 404", func() {
			Expect(err).ToNot(HaveOccurred())
			Expect(rsp.StatusCode).To(Equal(http.StatusNotFound))
		})
	})

})
