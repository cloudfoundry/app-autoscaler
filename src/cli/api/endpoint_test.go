package api_test

import (
	"fmt"
	"io/ioutil"
	"os"

	. "cli/api"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("Endpoint Helper Test", func() {

	const (
		fakeApiEndpoint = "autoscaler.boshlite.com"
	)

	var (
		endpoint       *APIEndpoint
		configFilePath string
		content        []byte
		err            error
	)

	BeforeEach(func() {
		os.Setenv("AUTOSCALER_CONFIG_FILE", "test_config.json")
		configFilePath = ConfigFile()
	})

	AfterEach(func() {
	})

	Context("Set API endpoint", func() {

		It("Set a valid json in config file", func() {
			err = SetEndpoint(fakeApiEndpoint, false)
			Expect(err).NotTo(HaveOccurred())

			content, err = ioutil.ReadFile(configFilePath)
			Expect(err).NotTo(HaveOccurred())
			Expect(content).Should(MatchJSON(fmt.Sprintf(`{"URL":"%s", "SkipSSLValidation":%t}`, fakeApiEndpoint, false)))
		})
	})

	Context("Unset API endpoint", func() {

		BeforeEach(func() {
			urlConfig := []byte(fmt.Sprintf(`{"URL":"%s"}`, fakeApiEndpoint))
			err = ioutil.WriteFile(configFilePath, urlConfig, 0600)
			Expect(err).NotTo(HaveOccurred())
		})

		It("Succed and set config.json to an empty file", func() {
			err = UnsetEndpoint()
			Expect(err).NotTo(HaveOccurred())

			content, err = ioutil.ReadFile(configFilePath)
			Expect(err).NotTo(HaveOccurred())
			Expect(len(content)).Should(Equal(0))
		})
	})

	Context("Get API Endpoint", func() {

		Context("No endpoint detected when config file is empty", func() {

			BeforeEach(func() {
				err = ioutil.WriteFile(configFilePath, nil, 0600)
				Expect(err).NotTo(HaveOccurred())
			})

			It("Return an empty URL", func() {
				endpoint, err = GetEndpoint()
				Expect(err).NotTo(HaveOccurred())
				Expect(endpoint.URL).Should(Equal(""))
			})
		})

		Context("Load existing URL from config file", func() {

			BeforeEach(func() {
				urlConfig := []byte(fmt.Sprintf(`{"URL":"%s"}`, fakeApiEndpoint))
				err = ioutil.WriteFile(configFilePath, urlConfig, 0600)
				Expect(err).NotTo(HaveOccurred())
			})

			It("Return valid URL", func() {
				endpoint, err = GetEndpoint()
				Expect(err).NotTo(HaveOccurred())
				Expect(endpoint.URL).Should(Equal(fakeApiEndpoint))
			})
		})

		Context("No endpoint detected when config is a invalid JSON file", func() {

			BeforeEach(func() {
				invalidConfig := []byte(`{"invalidJSON": invalidJSON}`)
				err = ioutil.WriteFile(configFilePath, invalidConfig, 0600)
				Expect(err).NotTo(HaveOccurred())
			})

			It("No error thrown out and unset API endpoint", func() {
				endpoint, err = GetEndpoint()
				Expect(err).NotTo(HaveOccurred())
				Expect(endpoint.URL).Should(Equal(""))

				content, err = ioutil.ReadFile(configFilePath)
				Expect(err).NotTo(HaveOccurred())
				Expect(len(content)).Should(Equal(0))
			})

		})

		Context("No endpoint detected when no URL field defined in config file", func() {

			BeforeEach(func() {
				invalidConfig := []byte(`{"invalidJSON": invalidJSON}`)
				err = ioutil.WriteFile(configFilePath, invalidConfig, 0600)
				Expect(err).NotTo(HaveOccurred())
			})

			It("No error thrown out and unset API endpoint when missing URL definition in JSON config", func() {
				endpoint, err = GetEndpoint()
				Expect(err).NotTo(HaveOccurred())
				Expect(endpoint.URL).Should(Equal(""))

				content, err = ioutil.ReadFile(configFilePath)
				Expect(err).NotTo(HaveOccurred())
				Expect(len(content)).Should(Equal(0))
			})
		})
	})
})
