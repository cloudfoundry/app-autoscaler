package server_test

import (
	"metrics-collector/cf"
	"metrics-collector/metrics"
	"metrics-collector/mhttp"
	. "metrics-collector/server"
	"metrics-collector/server/fakes"

	"github.com/cloudfoundry/sonde-go/events"
	"github.com/gogo/protobuf/proto"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/ghttp"
	"github.com/pivotal-golang/lager"

	"bytes"
	"encoding/json"
	"math/rand"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"strings"
)

const TEST_PATH_MEMORY_METRICS = "/v1/apps/test-app-id/metrics/memory"
const TEST_PATH_CONTAINER_METRICS = "/apps/test-app-id/containermetrics"

var _ = Describe("Handler", func() {

	var (
		cfc         cf.CfClient
		handler     *Handler
		testServer  *httptest.Server
		fakeDoppler *ghttp.Server
		method      string
		url         string
		resp        *http.Response
		err         error
	)

	BeforeEach(func() {
		fakeDoppler = ghttp.NewServer()
		cfc = fakes.NewFakeCfClient("test-access-token", strings.Replace(fakeDoppler.URL(), "http", "ws", 1))
		cfc.Login()

		logger := lager.NewLogger("handler-test")
		//logger.RegisterSink(lager.NewWriterSink(os.Stdout, lager.DEBUG))

		handler = NewHandler(cfc, logger)
		testServer = httptest.NewServer(handler)

	})

	AfterEach(func() {
		if fakeDoppler != nil {
			fakeDoppler.Close()
		}

		testServer.Close()
	})

	Describe("NewHandler", func() {
		It("creates a http handler", func() {
			Expect(handler.ServeHTTP).To(BeAssignableToTypeOf(http.NotFoundHandler()))
		})
	})

	Describe("ServeHttp", func() {
		JustBeforeEach(func() {
			resp, err = mhttp.DoRequest(method, url, "", map[string]string{}, nil)
		})

		Context("when requesting the wrong path", func() {
			BeforeEach(func() {
				url = "http://" + testServer.Listener.Addr().String() + "/not-exist-path"
				method = "GET"
			})

			It("should return 404", func() {
				Expect(err).To(BeNil())
				Expect(resp.StatusCode).To(Equal(http.StatusNotFound))
			})
		})

		Context("when requesting the wrong method", func() {
			BeforeEach(func() {
				url = "http://" + testServer.Listener.Addr().String() + TEST_PATH_MEMORY_METRICS
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
			url = "http://" + testServer.Listener.Addr().String() + TEST_PATH_MEMORY_METRICS
			resp, err = mhttp.DoRequest("GET", url, "", map[string]string{}, nil)
		})

		Context("when doppler server fails", func() {
			BeforeEach(func() {
				fakeDoppler.Close()
				fakeDoppler = nil
			})

			It("should return 500 error response", func() {
				Expect(err).To(BeNil())
				Expect(resp.StatusCode).To(Equal(http.StatusInternalServerError))

				By("checking the json error response body")

				errJson := &mhttp.ErrorResponse{}
				d := json.NewDecoder(resp.Body)
				err = d.Decode(errJson)

				Expect(err).To(BeNil())
				Expect(errJson.Code).To(Equal("Interal-Server-Error"))
				Expect(errJson.Message).To(Equal("Error getting memory metrics from doppler"))
			})
		})

		Context("when doppler return 500 error response", func() {
			BeforeEach(func() {
				fakeDoppler.AppendHandlers(
					ghttp.RespondWith(500, ""),
				)
			})

			It("should return 500 error response", func() {
				Expect(err).To(BeNil())
				Expect(resp.StatusCode).To(Equal(http.StatusInternalServerError))

				By("checking the json error response body")

				errJson := &mhttp.ErrorResponse{}
				d := json.NewDecoder(resp.Body)
				err = d.Decode(errJson)

				Expect(err).To(BeNil())
				Expect(errJson.Code).To(Equal("Interal-Server-Error"))
				Expect(errJson.Message).To(Equal("Error getting memory metrics from doppler"))

			})
		})

		Context("when doppler return 401 error response", func() {
			BeforeEach(func() {
				fakeDoppler.AppendHandlers(
					ghttp.RespondWith(401, ""),
				)
			})

			It("should return 500 error response", func() {
				Expect(err).To(BeNil())
				Expect(resp.StatusCode).To(Equal(http.StatusInternalServerError))

				By("checking the json error response body")

				errJson := &mhttp.ErrorResponse{}
				d := json.NewDecoder(resp.Body)
				err = d.Decode(errJson)

				Expect(err).To(BeNil())
				Expect(errJson.Code).To(Equal("Interal-Server-Error"))
				Expect(errJson.Message).To(Equal("Error getting memory metrics from doppler"))

			})
		})

		Context("when doppler return 200 response", func() {

			Context("container metrics is empty", func() {
				BeforeEach(func() {
					appendDopplerHandler200OK(0, fakeDoppler)
				})

				It("should return 200 response with empty metrics", func() {
					Expect(err).To(BeNil())
					Expect(resp.StatusCode).To(Equal(http.StatusOK))

					By("checking the memroy metrics in response body")

					metric := &metrics.Metric{}
					d := json.NewDecoder(resp.Body)
					err = d.Decode(metric)

					Expect(err).To(BeNil())
					Expect(metric.AppId).To(Equal("test-app-id"))
					Expect(metric.Name).To(Equal(metrics.MEMORY_METRIC_NAME))
					Expect(metric.Unit).To(Equal(metrics.UNIT_BYTES))
					Expect(metric.Instances).To(BeEmpty())
				})
			})

			Context("container metrics is not empty", func() {
				BeforeEach(func() {
					appendDopplerHandler200OK(2, fakeDoppler)
				})

				It("should return 200 response with empty metrics", func() {
					Expect(err).To(BeNil())
					Expect(resp.StatusCode).To(Equal(http.StatusOK))

					By("checking the memroy metrics in response body")

					metric := &metrics.Metric{}
					d := json.NewDecoder(resp.Body)
					err = d.Decode(metric)

					Expect(err).To(BeNil())
					Expect(metric.AppId).To(Equal("test-app-id"))
					Expect(metric.Name).To(Equal(metrics.MEMORY_METRIC_NAME))
					Expect(metric.Unit).To(Equal(metrics.UNIT_BYTES))
					Expect(len(metric.Instances)).To(Equal(2))
				})
			})

		})

	})
})

func appendDopplerHandler200OK(num int, doppler *ghttp.Server) {
	buffer := bytes.NewBuffer([]byte{})
	mp := multipart.NewWriter(buffer)

	header := http.Header{}
	header.Add("Content-Type", `multipart/x-protobuf; boundary=`+mp.Boundary())

	origin := "fake-doppler"
	appId := "test-app-id"
	for i := 0; i < num; i++ {
		index := int32(i)
		cpu := rand.Float64()
		memory := uint64(rand.Int63())
		disk := uint64(rand.Int63())
		cm := &events.ContainerMetric{
			ApplicationId: &appId,
			InstanceIndex: &index,
			CpuPercentage: &cpu,
			MemoryBytes:   &memory,
			DiskBytes:     &disk,
		}
		envelope := &events.Envelope{
			Origin:          &origin,
			EventType:       events.Envelope_ContainerMetric.Enum(),
			ContainerMetric: cm,
		}

		ebytes, _ := proto.Marshal(envelope)
		partWriter, _ := mp.CreatePart(nil)
		partWriter.Write(ebytes)
	}
	mp.Close()

	doppler.AppendHandlers(
		ghttp.CombineHandlers(
			ghttp.VerifyRequest("GET", TEST_PATH_CONTAINER_METRICS),
			ghttp.VerifyHeaderKV("Authorization", TOKEN_TYPE_BEARER+" test-access-token"),
			ghttp.RespondWith(200, buffer.Bytes(), header),
		),
	)
}
