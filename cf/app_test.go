package cf_test

import (
	"errors"
	"fmt"
	"io"

	. "code.cloudfoundry.org/app-autoscaler/src/autoscaler/cf"
	"code.cloudfoundry.org/app-autoscaler/src/autoscaler/models"

	"code.cloudfoundry.org/clock"
	"code.cloudfoundry.org/lager"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/ghttp"

	"encoding/json"
	"net"
	"net/http"
	"net/url"
)

var _ = Describe("App", func() {

	var (
		conf            *CFConfig
		cfc             CFClient
		fakeCC          *ghttp.Server
		fakeLoginServer *ghttp.Server
		err             error
		appState        string
		appEntity       *models.AppEntity
	)

	BeforeEach(func() {
		fakeCC = ghttp.NewServer()
		fakeLoginServer = ghttp.NewServer()
		fakeCC.RouteToHandler("GET", PathCFInfo, ghttp.RespondWithJSONEncoded(http.StatusOK, Endpoints{
			AuthEndpoint:    fakeLoginServer.URL(),
			TokenEndpoint:   fakeLoginServer.URL(),
			DopplerEndpoint: "test-doppler-endpoint",
		}))
		fakeLoginServer.RouteToHandler("POST", PathCFAuth, ghttp.RespondWithJSONEncoded(http.StatusOK, Tokens{
			AccessToken: "test-access-token",
			ExpiresIn:   12000,
		}))
		conf = &CFConfig{}
		conf.API = fakeCC.URL()
		cfc = NewCFClient(conf, lager.NewLogger("cf"), clock.NewClock())
		err = cfc.Login()
		Expect(err).NotTo(HaveOccurred())
	})

	AfterEach(func() {
		if fakeCC != nil {
			fakeCC.Close()
		}
		if fakeLoginServer != nil {
			fakeLoginServer.Close()
		}
	})

	Describe("GetAppEntity", func() {
		JustBeforeEach(func() {
			appEntity, err = cfc.GetApp("test-app-id")
		})
		Context("when get app summary succeeds", func() {
			appState = "test_app_state"
			BeforeEach(func() {
				fakeCC.AppendHandlers(
					ghttp.CombineHandlers(
						ghttp.VerifyRequest("GET", PathApp+"/test-app-id/summary"),
						ghttp.RespondWithJSONEncoded(http.StatusOK,
							models.AppEntity{
								Instances: 6,
								State:     &appState,
							}),
					),
				)
			})

			It("returns correct instance number", func() {
				Expect(err).NotTo(HaveOccurred())
				Expect(appEntity.Instances).To(Equal(6))
				Expect(*appEntity.State).To(Equal("test_app_state"))
			})
		})

		Context("when get app summary return 404 status code", func() {
			BeforeEach(func() {
				fakeCC.AppendHandlers(
					ghttp.CombineHandlers(
						ghttp.RespondWithJSONEncoded(http.StatusNotFound, models.CFErrorResponse{
							Description: "The app could not be found: 7efa8f58-1aba-4493-bf9e-30d69c40dbb42",
							ErrorCode:   "CF-AppNotFound",
							Code:        100004,
						}),
					),
				)
			})

			It("should error", func() {
				Expect(appEntity).To(BeNil())
				Expect(err).To(Equal(models.NewAppNotFoundErr("The app could not be found: 7efa8f58-1aba-4493-bf9e-30d69c40dbb42")))
				Expect(err).To(MatchError(MatchRegexp("The app could not be found: *")))
			})
		})

		Context("when get app summary return non-200 and non-404 status code", func() {
			BeforeEach(func() {
				fakeCC.AppendHandlers(
					ghttp.CombineHandlers(
						ghttp.RespondWithJSONEncoded(http.StatusInternalServerError, models.CFErrorResponse{
							Description: "Server error",
							ErrorCode:   "ServerError",
							Code:        100001,
						}),
					),
				)
			})

			It("should error", func() {
				Expect(appEntity).To(BeNil())
				Expect(err).To(MatchError(MatchRegexp("failed getting application summary: *")))
			})

		})

		Context("when get app summary return non-200 and non-404 status code with non-JSON response", func() {
			BeforeEach(func() {
				fakeCC.AppendHandlers(
					ghttp.CombineHandlers(
						ghttp.RespondWithJSONEncoded(http.StatusInternalServerError, ""),
					),
				)
			})

			It("should error", func() {
				Expect(appEntity).To(BeNil())
				Expect(err).To(MatchError(MatchRegexp("failed getting application summary: *")))
			})

		})

		Context("when cloud controller is not reachable", func() {
			BeforeEach(func() {
				fakeCC.Close()
				fakeCC = nil
			})

			It("should error", func() {
				Expect(appEntity).To(BeNil())
				IsUrlNetOpError(err)
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
				Expect(appEntity).To(BeNil())
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
				responseMap := make(map[string]interface{})
				responseMap["description"] = "You have exceeded the instance memory limit for your space's quota"
				responseMap["error_code"] = "SpaceQuotaInstanceMemoryLimitExceeded"
				fakeCC.AppendHandlers(
					ghttp.CombineHandlers(
						ghttp.RespondWithJSONEncoded(http.StatusBadRequest, responseMap),
					),
				)
			})

			It("should error", func() {
				Expect(err).To(MatchError(MatchRegexp("failed setting application instances: *")))
			})

		})

		Context("when cloud controller is not reachable", func() {
			BeforeEach(func() {
				ccURL := fakeCC.URL()
				fakeCC.Close()
				fakeCC = nil

				Eventually(func() error {
					// #nosec G107
					resp, err := http.Get(ccURL)

					if err != nil {
						return err
					}
					_ = resp.Body.Close()

					return nil
				}).Should(HaveOccurred())
			})

			It("should error", func() {
				IsUrlNetOpError(err)
			})

		})

	})

})

func IsUrlNetOpError(err error) {
	var urlErr *url.Error
	Expect(errors.As(err, &urlErr)).To(BeTrue(), fmt.Sprintf("Expected a (*url.Error) error in the chan got, %T: %+v", err, err))

	var netOpErr *net.OpError
	Expect(errors.As(err, &netOpErr) || errors.Is(err, io.EOF)).
		To(BeTrue(), fmt.Sprintf("Expected a (*net.OpError) or io.EOF error in the chan got, %T: %+v", err, err))
}
