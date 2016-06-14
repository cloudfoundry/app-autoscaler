package server_test

import (
	"metrics-collector/cf"
	"metrics-collector/metrics"
	. "metrics-collector/server"
	"metrics-collector/server/fakes"
	"metrics-collector/util"

	"encoding/json"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"strings"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("Handler", func() {

	var (
		cfc         cf.CfClient
		handler     *Handler
		testServer  *httptest.Server
		method      string
		url         string
		accessToken string
		dopplerUrl  string
		appId       string
		resp        *http.Response
		err         error
	)

	Describe("NewHandler", func() {
		JustBeforeEach(func() {
			handler = NewHandler(cfc)
		})
	})

	Describe("ServeHttp", func() {
		BeforeEach(func() {
			cfc = fakes.NewCfClient(fakes.FAKE_DOPPLER_ACCESS_TOKEN, fakes.FAKE_DOPPLER_URL)
			cfc.Login()

			handler = NewHandler(cfc)
			testServer = httptest.NewServer(handler)

		})

		AfterEach(func() {
			testServer.Close()
		})

		JustBeforeEach(func() {
			resp, err = util.DoRequest(method, url, "", map[string]string{}, nil)
		})

		Context("when request the wrong server", func() {

			BeforeEach(func() {
				url = "http://www.not-exist.com" + strings.Replace(PATH_MEMORY_METRIC, "{appid}", fakes.FAKE_APP_ID, -1)
				method = "GET"
			})

			It("should err", func() {
				Expect(err).NotTo(BeNil())
			})
		})

		Context("when request the wrong path", func() {

			BeforeEach(func() {
				url = "http://" + testServer.Listener.Addr().String() + "/not-exist-path"
				method = "GET"
			})

			It("should return 404", func() {
				Expect(err).To(BeNil())
				Expect(resp.StatusCode).To(Equal(http.StatusNotFound))
			})
		})

		Context("when request the wrong method", func() {

			BeforeEach(func() {
				url = "http://" + testServer.Listener.Addr().String() + strings.Replace(PATH_MEMORY_METRIC, "{appid}", fakes.FAKE_APP_ID, -1)
				method = "PUT"
			})

			It("should return 404", func() {
				Expect(err).To(BeNil())
				Expect(resp.StatusCode).To(Equal(http.StatusNotFound))
			})
		})
	})

	Describe("GetMemoryMetric", func() {

		JustBeforeEach(func() {
			cfc = fakes.NewCfClient(accessToken, dopplerUrl)
			cfc.Login()

			handler = NewHandler(cfc)
			testServer = httptest.NewServer(handler)

			url = "http://" + testServer.Listener.Addr().String() + strings.Replace(PATH_MEMORY_METRIC, "{appid}", appId, -1)
			resp, err = util.DoRequest("GET", url, "", map[string]string{}, nil)
		})

		AfterEach(func() {
			testServer.Close()
		})

		Context("when doppler address is not valid", func() {
			BeforeEach(func() {
				accessToken = fakes.FAKE_DOPPLER_ACCESS_TOKEN
				dopplerUrl = "ws://www.not-exist.com"
				appId = fakes.FAKE_APP_ID
			})

			It("should not error and return 500", func() {
				Expect(err).To(BeNil())
				Expect(resp.StatusCode).To(Equal(http.StatusInternalServerError))
			})
		})

		Context("when  token to access doppler is not valid", func() {
			BeforeEach(func() {
				accessToken = "invalid-token"
				dopplerUrl = testDopplerUrl
				appId = fakes.FAKE_APP_ID
			})

			It("should not error and return 500", func() {
				Expect(err).To(BeNil())
				Expect(resp.StatusCode).To(Equal(http.StatusInternalServerError))
			})
		})

		Context("when  appid is not valid", func() {
			BeforeEach(func() {
				accessToken = fakes.FAKE_DOPPLER_ACCESS_TOKEN
				dopplerUrl = testDopplerUrl
				appId = "not-exist-appid"
			})

			It("should not error and return 500", func() {
				Expect(err).To(BeNil())
				Expect(resp.StatusCode).To(Equal(http.StatusInternalServerError))
			})
		})

		Context("when request with right appid ", func() {
			BeforeEach(func() {
				accessToken = fakes.FAKE_DOPPLER_ACCESS_TOKEN
				dopplerUrl = testDopplerUrl
				appId = fakes.FAKE_APP_ID
			})

			It("should not error and return correct result", func() {
				Expect(err).To(BeNil())
				Expect(resp.StatusCode).To(Equal(http.StatusOK))

				By("checking the memroy metrics in response body")
				b, _ := ioutil.ReadAll(resp.Body)

				metric := &metrics.Metric{}
				e := json.Unmarshal(b, metric)
				Expect(e).To(BeNil())
				Expect(metric.AppId).To(Equal(appId))
				Expect(metric.Name).To(Equal(metrics.MEMORY_METRIC_NAME))
				Expect(metric.Unit).To(Equal(metrics.UNIT_BYTES))
			})
		})

	})

})
