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

var usageExample = `{
  "guid": "a595fe2f-01ff-4965-a50c-290258ab8582",
  "created_at": "2020-05-28T16:41:23Z",
  "updated_at": "2020-05-28T16:41:26Z",
  "state": {
    "current": "STARTED",
    "previous": "STOPPED"
  },
  "app": {
    "guid": "guid-f93250f7-7ef5-4b02-8d33-353919ce8358",
    "name": "name-1982"
  },
  "process": {
    "guid": "guid-e9d2d5a0-69a6-46ef-bac5-43f3ed177614",
    "type": "type-1983"
  },
  "space": {
    "guid": "guid-5e28f12f-9d80-473e-b826-537b148eb338",
    "name": "name-1664"
  },
  "organization": {
    "guid": "guid-036444f4-f2f5-4ea8-a353-e73330ca0f0a"
  },
  "buildpack": {
    "guid": "guid-34916716-31d7-40c1-9afd-f312996c9654",
    "name": "label-64"
  },
  "task": {
    "guid": "guid-7cc11646-bf38-4f4e-b6e0-9581916a74d9",
    "name": "name-2929"
  },
  "memory_in_mb_per_instance": {
    "current": 512,
    "previous": 256
  },
  "instance_count": {
    "current": 6,
    "previous": 5
  },
  "links": {
    "self": {
      "href": "https://api.example.org/v3/app_usage_events/a595fe2f-01ff-4965-a50c-290258ab8582"
    }
  }
}
`

var _ = Describe("App", func() {

	var (
		conf            *CFConfig
		cfc             CFClient
		fakeCC          *ghttp.Server
		fakeLoginServer *ghttp.Server
		err             error
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
			BeforeEach(func() {
				fakeCC.AppendHandlers(
					ghttp.CombineHandlers(
						ghttp.VerifyRequest("GET", "/v3/app_usage_events/test-app-id"),
						ghttp.RespondWith(http.StatusOK, usageExample, http.Header{"Content-Type": []string{"application/json"}}),
					),
				)
			})

			It("returns correct instance number", func() {
				Expect(err).NotTo(HaveOccurred())
				Expect(appEntity.Instances).To(Equal(6))
				Expect(*appEntity.State).To(Equal("STARTED"))
			})
		})

		Context("when get app summary return 404 status code", func() {
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
				Expect(appEntity).To(BeNil())
				var cfError *models.CfError
				Expect(errors.As(err, &cfError) && cfError.IsNotFound()).To(BeTrue())
			})
		})

		Context("when get app usage return non-200 and non-404 status code", func() {
			BeforeEach(func() {
				fakeCC.AppendHandlers(
					ghttp.CombineHandlers(
						ghttp.RespondWithJSONEncoded(http.StatusInternalServerError, models.CfInternalServerError),
					),
				)
			})

			It("should error", func() {
				Expect(appEntity).To(BeNil())
				Expect(err).To(MatchError(MatchRegexp("failed getting application usage events: *")))
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
				Expect(err.Error()).To(MatchRegexp("failed getting application usage events:"))
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
