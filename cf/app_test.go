package cf_test

import (
	. "autoscaler/cf"
	"autoscaler/models"

	"code.cloudfoundry.org/clock"
	"code.cloudfoundry.org/lager"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/ghttp"

	"encoding/json"
	"net"
	"net/http"
	"net/url"
)

var _ = Describe("App", func() {

	var (
		conf            *CfConfig
		cfc             CfClient
		fakeCC          *ghttp.Server
		fakeLoginServer *ghttp.Server
		instances       int
		err             error
	)

	BeforeEach(func() {
		fakeCC = ghttp.NewServer()
		fakeLoginServer = ghttp.NewServer()
		fakeCC.RouteToHandler("GET", PathCfInfo, ghttp.RespondWithJSONEncoded(http.StatusOK, Endpoints{
			AuthEndpoint:    fakeLoginServer.URL(),
			TokenEndpoint:   "test-token-endpoint",
			DopplerEndpoint: "test-doppler-endpoint",
		}))
		fakeLoginServer.RouteToHandler("POST", PathCfAuth, ghttp.RespondWithJSONEncoded(http.StatusOK, Tokens{
			AccessToken:  "test-access-token",
			RefreshToken: "test-refresh-token",
			ExpiresIn:    12000,
		}))
		conf = &CfConfig{}
		conf.Api = fakeCC.URL()
		cfc = NewCfClient(conf, lager.NewLogger("cf"), clock.NewClock())
		cfc.Login()
	})

	AfterEach(func() {
		if fakeCC != nil {
			fakeCC.Close()
		}
		if fakeLoginServer != nil {
			fakeLoginServer.Close()
		}
	})

	Describe("GetAppInstances", func() {
		JustBeforeEach(func() {
			instances, err = cfc.GetAppInstances("test-app-id")
		})
		Context("when get app summary succeeds", func() {
			BeforeEach(func() {
				fakeCC.AppendHandlers(
					ghttp.CombineHandlers(
						ghttp.VerifyRequest("GET", PathApp+"/test-app-id"),
						ghttp.RespondWithJSONEncoded(http.StatusOK, models.AppInfo{
							models.AppEntity{Instances: 6},
						}),
					),
				)
			})

			It("returns correct instance number", func() {
				Expect(err).NotTo(HaveOccurred())
				Expect(instances).To(Equal(6))
			})
		})

		Context("when get app summary return non-200 status code", func() {
			BeforeEach(func() {
				fakeCC.AppendHandlers(
					ghttp.CombineHandlers(
						ghttp.RespondWithJSONEncoded(http.StatusNotFound, ""),
					),
				)
			})

			It("should error", func() {
				Expect(instances).To(Equal(-1))
				Expect(err).To(MatchError(MatchRegexp("failed getting application summary: *")))
			})

		})

		Context("when cloud controller is not reachable", func() {
			BeforeEach(func() {
				fakeCC.Close()
				fakeCC = nil
			})

			It("should error", func() {
				Expect(instances).To(Equal(-1))
				Expect(err).To(BeAssignableToTypeOf(&url.Error{}))
				urlErr := err.(*url.Error)
				Expect(urlErr.Err).To(BeAssignableToTypeOf(&net.OpError{}))
			})

		})

		Context("when cloud controller returns incorrect message body", func() {
			BeforeEach(func() {
				fakeCC.AppendHandlers(
					ghttp.CombineHandlers(
						ghttp.RespondWithJSONEncoded(http.StatusOK, `{"entity":{"instances:"abc"}}`),
					),
				)
			})

			It("should error", func() {
				Expect(instances).To(Equal(-1))
				Expect(err).To(BeAssignableToTypeOf(&json.UnmarshalTypeError{}))
			})

		})
	})

	Describe("SetAppInstances", func() {
		JustBeforeEach(func() {
			err = cfc.SetAppInstances("test-app-id", 6)
		})
		Context("when set app instances succeeds", func() {
			BeforeEach(func() {
				fakeCC.AppendHandlers(
					ghttp.CombineHandlers(
						ghttp.VerifyRequest("PUT", PathApp+"/test-app-id"),
						ghttp.VerifyJSONRepresenting(models.AppEntity{Instances: 6}),
						ghttp.RespondWith(http.StatusCreated, ""),
					),
				)
			})

			It("should not error", func() {
				Expect(err).NotTo(HaveOccurred())
			})
		})

		Context("when updating app instances returns non-200 status code", func() {
			BeforeEach(func() {
				fakeCC.AppendHandlers(
					ghttp.CombineHandlers(
						ghttp.RespondWithJSONEncoded(http.StatusNotFound, ""),
					),
				)
			})

			It("should error", func() {
				Expect(err).To(MatchError(MatchRegexp("failed setting application instances: *")))
			})

		})

		Context("when cloud controller is not reachable", func() {
			BeforeEach(func() {
				fakeCC.Close()
				fakeCC = nil
			})

			It("should error", func() {
				Expect(err).To(BeAssignableToTypeOf(&url.Error{}))
				urlErr := err.(*url.Error)
				Expect(urlErr.Err).To(BeAssignableToTypeOf(&net.OpError{}))
			})

		})

	})

})
