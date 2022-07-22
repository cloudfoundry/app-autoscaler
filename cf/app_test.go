package cf_test

import (
	"code.cloudfoundry.org/app-autoscaler/src/autoscaler/cf"
	"code.cloudfoundry.org/app-autoscaler/src/autoscaler/models"
	. "code.cloudfoundry.org/app-autoscaler/src/autoscaler/testhelpers"
	"errors"
	"fmt"
	"io"

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
		conf            *cf.CFConfig
		cfc             cf.CFClient
		fakeCC          *ghttp.Server
		fakeLoginServer *ghttp.Server
		err             error
		app             *models.AppEntity
	)

	BeforeEach(func() {
		fakeCC = ghttp.NewServer()
		fakeLoginServer = ghttp.NewServer()
		fakeCC.RouteToHandler("GET", cf.PathCFInfo, ghttp.RespondWithJSONEncoded(http.StatusOK, cf.Endpoints{
			AuthEndpoint:    fakeLoginServer.URL(),
			TokenEndpoint:   fakeLoginServer.URL(),
			DopplerEndpoint: "test-doppler-endpoint",
		}))
		fakeLoginServer.RouteToHandler("POST", cf.PathCFAuth, ghttp.RespondWithJSONEncoded(http.StatusOK, cf.Tokens{
			AccessToken: "test-access-token",
			ExpiresIn:   12000,
		}))
		conf = &cf.CFConfig{}
		conf.API = fakeCC.URL()
		cfc = cf.NewCFClient(conf, lager.NewLogger("cf"), clock.NewClock())
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

	Describe("GetApp", func() {
		JustBeforeEach(func() {
			app, err = cfc.GetApp("test-app-id")
		})
		Context("when get app succeeds", func() {
			BeforeEach(func() {
				fakeCC.AppendHandlers(
					ghttp.CombineHandlers(
						ghttp.VerifyRequest("GET", "/v3/apps/test-app-id"),
						ghttp.RespondWith(http.StatusOK, LoadFile("testdata/app.json"), http.Header{"Content-Type": []string{"application/json"}}),
					),
					// ghttp.CombineHandlers(
					// 	ghttp.VerifyRequest("GET", "/v3/app/test-app-id/processes"),
					// 	ghttp.RespondWith(http.StatusOK, LoadFile("testdata/app_processes.json"), http.Header{"Content-Type": []string{"application/json"}}),
					// ),
				)
			})

			It("returns correct state", func() {
				Expect(err).NotTo(HaveOccurred())
				Expect(*app.State).To(Equal("STOPPED"))
			})
		})

		Context("when get app usage return 404 status code", func() {
			//TODO ... we need both variations of app/{guid} and app/{guid}/processes with 404 and 200
			BeforeEach(func() {
				fakeCC.AppendHandlers(
					ghttp.CombineHandlers(
						ghttp.RespondWithJSONEncoded(http.StatusNotFound, models.CfError{
							Errors: []models.CfErrorItem{{Code: 10010, Detail: "App usage event not found", Title: "CF-ResourceNotFound"}},
						}),
					),
				)
			})

			It("should error", func() {
				Expect(app).To(BeNil())
				var cfError *models.CfError
				Expect(errors.As(err, &cfError) && cfError.IsNotFound()).To(BeTrue())
				Expect(models.IsNotFound(err)).To(BeTrue())
			})
		})

		Context("when get app/* return non-200 and non-404 status code", func() {
			//TODO ... we need both variations of app/{guid} and app/{guid}/processes with 404 and 200
			BeforeEach(func() {
				fakeCC.AppendHandlers(
					ghttp.CombineHandlers(
						ghttp.RespondWithJSONEncoded(http.StatusInternalServerError, models.CfInternalServerError),
					),
				)
			})

			It("should error", func() {
				Expect(app).To(BeNil())
				Expect(err).To(MatchError(MatchRegexp("failed getting application usage events: *")))
			})
		})

		Context("when get app/*  return non-200 and non-404 status code with non-JSON response", func() {
			//TODO ... we need both variations of app/{guid} and app/{guid}/processes with non 200
			BeforeEach(func() {
				fakeCC.AppendHandlers(
					ghttp.CombineHandlers(
						ghttp.RespondWithJSONEncoded(http.StatusInternalServerError, ""),
					),
				)
			})

			It("should error", func() {
				Expect(app).To(BeNil())
				Expect(err.Error()).To(MatchRegexp("failed getting application usage events:"))
			})

		})

		Context("when cloud controller is not reachable", func() {
			BeforeEach(func() {
				fakeCC.Close()
				fakeCC = nil
			})

			It("should error", func() {
				Expect(app).To(BeNil())
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
				Expect(app).To(BeNil())
				Expect(err).To(MatchError(MatchRegexp("failed to unmarshal")))
				var errType *json.UnmarshalTypeError
				Expect(errors.As(err, &errType)).Should(BeTrue(), "Error was: %#v", interface{}(err))
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
						ghttp.VerifyRequest("PUT", cf.PathApp+"/test-app-id"),
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
