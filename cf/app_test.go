package cf_test

import (
	"errors"
	"fmt"
	"io"
	"time"

	"code.cloudfoundry.org/app-autoscaler/src/autoscaler/cf"
	"code.cloudfoundry.org/app-autoscaler/src/autoscaler/models"
	. "code.cloudfoundry.org/app-autoscaler/src/autoscaler/testhelpers"

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

var _ = Describe("Cf client App", func() {

	var (
		conf            *cf.CFConfig
		cfc             cf.CFClient
		fakeCC          *ghttp.Server
		fakeLoginServer *ghttp.Server
		err             error
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

		When("get app succeeds", func() {
			BeforeEach(func() {
				fakeCC.AppendHandlers(
					ghttp.CombineHandlers(
						ghttp.VerifyRequest("GET", "/v3/apps/test-app-id"),
						ghttp.RespondWith(http.StatusOK, LoadFile("testdata/app.json"), http.Header{"Content-Type": []string{"application/json"}}),
					),
				)
			})

			It("returns correct state", func() {
				app, err := cfc.GetApp("test-app-id")
				Expect(err).NotTo(HaveOccurred())
				created, err := time.Parse(time.RFC3339, "2022-07-21T13:42:30Z")
				Expect(err).NotTo(HaveOccurred())
				updated, err := time.Parse(time.RFC3339, "2022-07-21T14:30:17Z")
				Expect(err).NotTo(HaveOccurred())
				Expect(app).To(Equal(&cf.App{
					Guid:      "663e9a25-30ba-4fb4-91fa-9b784f4a8542",
					Name:      "autoscaler-1--0cde0e473e3e47f4",
					State:     "STOPPED",
					CreatedAt: created,
					UpdatedAt: updated,
				}))
			})
		})

		When("get app usage return 404 status code", func() {
			BeforeEach(func() {
				fakeCC.AppendHandlers(
					ghttp.CombineHandlers(
						ghttp.VerifyRequest("GET", "/v3/apps/404"),
						ghttp.RespondWithJSONEncoded(http.StatusNotFound, models.CfResourceNotFound),
					),
				)
			})

			It("should error", func() {
				app, err := cfc.GetApp("404")
				Expect(app).To(BeNil())
				var cfError *models.CfError
				Expect(errors.As(err, &cfError) && cfError.IsNotFound()).To(BeTrue())
				Expect(models.IsNotFound(err)).To(BeTrue())
			})
		})

		When("get app/* return non-200 and non-404 status code", func() {
			BeforeEach(func() {
				fakeCC.AppendHandlers(ghttp.CombineHandlers(
					ghttp.VerifyRequest("GET", "/v3/apps/500"),
					ghttp.RespondWithJSONEncoded(http.StatusInternalServerError, models.CfInternalServerError)))
			})

			It("should error", func() {
				app, err := cfc.GetApp("500")
				Expect(app).To(BeNil())
				Expect(err).To(MatchError(MatchRegexp("failed getting app information for '500':.*'UnknownError'")))
			})
		})

		When("get app/*  returns a non-200 and non-404 status code with non-JSON response", func() {
			BeforeEach(func() {
				fakeCC.AppendHandlers(
					ghttp.CombineHandlers(
						ghttp.VerifyRequest("GET", "/v3/apps/invalid_json"),
						ghttp.RespondWithJSONEncoded(http.StatusInternalServerError, ""),
					),
				)
			})

			It("should error", func() {
				app, err := cfc.GetApp("invalid_json")
				Expect(app).To(BeNil())
				Expect(err.Error()).To(MatchRegexp("failed getting app information for 'invalid_json': failed to unmarshal"))
			})
		})

		When("cloud controller is not reachable", func() {
			BeforeEach(func() {
				fakeCC.Close()
				fakeCC = nil
			})

			It("should error", func() {
				app, err := cfc.GetApp("something")
				Expect(app).To(BeNil())
				IsUrlNetOpError(err)
			})
		})

		When("cloud controller returns incorrect message body", func() {
			BeforeEach(func() {
				fakeCC.AppendHandlers(
					ghttp.CombineHandlers(
						ghttp.VerifyRequest("GET", "/v3/apps/incorrect_object"),
						ghttp.RespondWithJSONEncoded(http.StatusOK, `{"entity":{"instances:"abc"}}`),
					),
				)
			})

			It("should error", func() {
				app, err := cfc.GetApp("incorrect_object")
				Expect(app).To(BeNil())
				Expect(err).To(MatchError(MatchRegexp("failed unmarshalling app information for 'incorrect_object': .* cannot unmarshal string")))
				var errType *json.UnmarshalTypeError
				Expect(errors.As(err, &errType)).Should(BeTrue(), "Error was: %#v", interface{}(err))
			})

		})
	})

	Describe("GetAppProcesses", func() {

		When("get process succeeds", func() {
			BeforeEach(func() {
				fakeCC.AppendHandlers(
					ghttp.CombineHandlers(
						ghttp.VerifyRequest("GET", "/v3/processes/test-app-id"),
						ghttp.RespondWith(http.StatusOK, LoadFile("testdata/app_processes.json"), http.Header{"Content-Type": []string{"application/json"}}),
					),
				)
			})

			It("returns correct state", func() {
				process, err := cfc.GetAppProcesses("test-app-id")
				Expect(err).NotTo(HaveOccurred())
				created, err := time.Parse(time.RFC3339, "2016-03-23T18:48:22Z")
				Expect(err).NotTo(HaveOccurred())
				updated, err := time.Parse(time.RFC3339, "2016-03-23T18:48:42Z")
				Expect(err).NotTo(HaveOccurred())
				Expect(process).To(Equal(&cf.Processes{
					Guid:       "6a901b7c-9417-4dc1-8189-d3234aa0ab82",
					Type:       "web",
					Command:    "rackup",
					Instances:  5,
					MemoryInMb: 256,
					DiskInMb:   1024,
					CreatedAt:  created,
					UpdatedAt:  updated,
				}))
			})
		})

		When("get processes return 404 status code", func() {
			BeforeEach(func() {
				fakeCC.AppendHandlers(
					ghttp.CombineHandlers(
						ghttp.VerifyRequest("GET", "/v3/processes/404"),
						ghttp.RespondWithJSONEncoded(http.StatusNotFound, models.CfResourceNotFound),
					),
				)
			})

			It("should error", func() {
				process, err := cfc.GetAppProcesses("404")
				Expect(process).To(BeNil())
				var cfError *models.CfError
				Expect(errors.As(err, &cfError) && cfError.IsNotFound()).To(BeTrue())
				Expect(models.IsNotFound(err)).To(BeTrue())
			})
		})

		When("get processes/* return non-200 and non-404 status code", func() {
			BeforeEach(func() {
				fakeCC.AppendHandlers(ghttp.CombineHandlers(
					ghttp.VerifyRequest("GET", "/v3/processes/500"),
					ghttp.RespondWithJSONEncoded(http.StatusInternalServerError, models.CfInternalServerError)))
			})

			It("should error", func() {
				process, err := cfc.GetAppProcesses("500")
				Expect(process).To(BeNil())
				Expect(err).To(MatchError(MatchRegexp("failed getting processes information for '500':.*'UnknownError'")))
			})
		})

		When("get processes/*  returns a non-200 and non-404 status code with non-JSON response", func() {
			BeforeEach(func() {
				fakeCC.AppendHandlers(
					ghttp.CombineHandlers(
						ghttp.VerifyRequest("GET", "/v3/processes/invalid_json"),
						ghttp.RespondWithJSONEncoded(http.StatusInternalServerError, ""),
					),
				)
			})

			It("should error", func() {
				process, err := cfc.GetAppProcesses("invalid_json")
				Expect(process).To(BeNil())
				Expect(err.Error()).To(MatchRegexp("failed getting processes information for 'invalid_json': failed to unmarshal"))
			})
		})

		When(" get processes call and cloud controller is not reachable", func() {
			BeforeEach(func() {
				fakeCC.Close()
				fakeCC = nil
			})

			It("should error", func() {
				app, err := cfc.GetAppProcesses("something")
				Expect(app).To(BeNil())
				IsUrlNetOpError(err)
			})
		})

		When("get processes returns incorrect message body", func() {
			BeforeEach(func() {
				fakeCC.AppendHandlers(
					ghttp.CombineHandlers(
						ghttp.VerifyRequest("GET", "/v3/processes/incorrect_object"),
						ghttp.RespondWithJSONEncoded(http.StatusOK, `{"entity":{"instances:"abc"}}`),
					),
				)
			})

			It("should error", func() {
				process, err := cfc.GetAppProcesses("incorrect_object")
				Expect(process).To(BeNil())
				Expect(err).To(MatchError(MatchRegexp("failed unmarshalling processes information for 'incorrect_object': .* cannot unmarshal string")))
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
