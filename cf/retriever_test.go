package cf_test

import (
	"errors"
	"net/url"
	"regexp"

	"code.cloudfoundry.org/clock"
	"code.cloudfoundry.org/lager"

	"code.cloudfoundry.org/app-autoscaler/src/autoscaler/cf"
	"code.cloudfoundry.org/app-autoscaler/src/autoscaler/models"
	. "code.cloudfoundry.org/app-autoscaler/src/autoscaler/testhelpers"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	. "github.com/onsi/gomega/ghttp"

	"encoding/json"
	"net/http"
)

var _ = Describe("Cf client Retriever", func() {

	var (
		conf            *cf.Config
		cfc             *cf.Client
		fakeCC          *MockServer
		fakeLoginServer *Server
		err             error
		logger          lager.Logger
	)

	var setCfcClient = func(maxRetries int) {
		conf = &cf.Config{}
		conf.API = fakeCC.URL()
		conf.MaxRetries = maxRetries
		conf.MaxRetryWaitMs = 1
		cfc = cf.NewCFClient(conf, logger, clock.NewClock())
		err = cfc.Login()
		Expect(err).NotTo(HaveOccurred())
	}

	BeforeEach(func() {
		fakeCC = NewMockServer()
		fakeLoginServer = NewServer()
		fakeCC.Add().Info(fakeLoginServer.URL())
		fakeLoginServer.RouteToHandler("POST", cf.PathCFAuth, RespondWithJSONEncoded(http.StatusOK, cf.Tokens{
			AccessToken: "test-access-token",
			ExpiresIn:   12000,
		}))
		logger = lager.NewLogger("cf")
		logger.RegisterSink(lager.NewWriterSink(GinkgoWriter, lager.DEBUG))
		setCfcClient(0)
	})

	AfterEach(func() {
		if fakeCC != nil {
			fakeCC.Close()
		}
		if fakeLoginServer != nil {
			fakeLoginServer.Close()
		}
	})

	Describe("Client.Get", func() {
		When("has an invalid url", func() {

			It("should return error", func() {
				app, err := cfc.Get("v3/invalid")
				Expect(app).To(BeNil())
				var urlErr *url.Error
				Expect(err).To(HaveOccurred())
				Expect(errors.As(err, &urlErr) && urlErr.Op == "parse").To(BeTrue())
			})
		})

		When("getToken Fails", func() {
			It("should return error", func() {
				//TODO extract out a mock for token getting.
			})
		})

		When("get returns 404", func() {
			BeforeEach(func() {
				fakeCC.AppendHandlers(
					CombineHandlers(
						VerifyRequest("GET", "/v3/something/404"),
						RespondWithJSONEncoded(http.StatusNotFound, models.CfResourceNotFound),
					),
				)
			})

			It("should return IsNotFound error", func() {
				_, err := cfc.Get("/v3/something/404")
				var cfError *models.CfError
				Expect(err).To(MatchError(MatchRegexp(`GET request failed: cf api Error url='', resourceId='': \['CF-ResourceNotFound' code: 10010.*\]`)))
				Expect(errors.As(err, &cfError) && cfError.IsNotFound()).To(BeTrue())
				Expect(models.IsNotFound(err)).To(BeTrue())
			})
			It("should close the response body", func() {
				resp, err := cfc.Get("/v3/something/404")
				Expect(err).ToNot(BeNil())
				_, err = resp.Body.Read([]byte{})
				Expect(err).ToNot(BeNil())
				Expect(err).To(MatchError(MatchRegexp(`closed response body`)))
			})
		})

		When("returns 500 status code", func() {
			BeforeEach(func() {
				setCfcClient(3)
			})
			When("it never recovers", func() {

				BeforeEach(func() {
					fakeCC.RouteToHandler("GET", regexp.MustCompile(`^/v3/some/url$`),
						RespondWithJSONEncoded(http.StatusInternalServerError, models.CfInternalServerError),
					)
				})

				It("should error", func() {
					_, err := cfc.Get("/v3/some/url")
					Expect(fakeCC.Count().Requests(`^/v3/some/url$`)).To(Equal(4))
					Expect(err).To(MatchError(MatchRegexp("GET request failed:.*'UnknownError'")))
				})
			})
			When("it recovers after 3 retries", func() {
				BeforeEach(func() {
					fakeCC.RouteToHandler("GET", regexp.MustCompile(`^/v3/some/url$`),
						RespondWithMultiple(
							RespondWithJSONEncoded(http.StatusInternalServerError, models.CfInternalServerError),
							RespondWithJSONEncoded(http.StatusInternalServerError, models.CfInternalServerError),
							RespondWithJSONEncoded(http.StatusInternalServerError, models.CfInternalServerError),
							RespondWith(http.StatusOK, LoadFile("testdata/app.json"), http.Header{"Content-Type": []string{"application/json"}}),
						))
				})

				It("should return success", func() {
					app, err := cfc.Get("/v3/some/url")
					Expect(err).NotTo(HaveOccurred())
					Expect(app).ToNot(BeNil())
					Expect(fakeCC.Count().Requests(`^/v3/some/url$`)).To(Equal(4))
				})
			})

			When("cloud controller is not reachable", func() {
				BeforeEach(func() {
					fakeCC.Close()
					fakeCC = nil
				})

				It("should error", func() {
					app, err := cfc.Get("/something")
					Expect(app).To(BeNil())
					Expect(err).To(MatchError(MatchRegexp(`connection refused`)))
					IsUrlNetOpError(err)
				})
			})
		})
	})

	Describe("Client.Post", func() {
		When("has an invalid url", func() {
			It("should return error", func() {
				app, err := cfc.Post("v3/invalid", nil)
				Expect(app).To(BeNil())
				var urlErr *url.Error
				Expect(err).To(HaveOccurred())
				Expect(errors.As(err, &urlErr) && urlErr.Op == "parse").To(BeTrue())
			})
		})
		When("passed valid struct", func() {
			It("should return error", func() {
				app, err := cfc.Post("/v3/invalid", make(chan int))
				Expect(app).To(BeNil())
				Expect(err).To(HaveOccurred())
				Expect(err).To(MatchError(MatchRegexp(`failed post:.*`)))
				var typeErr *json.UnsupportedTypeError
				Expect(err).To(HaveOccurred())
				Expect(errors.As(err, &typeErr)).To(BeTrue())
			})
		})
		When("post is called", func() {
			BeforeEach(func() {
				fakeCC.AppendHandlers(
					CombineHandlers(
						VerifyRequest("POST", "/v3/post"),
						VerifyContentType("application/json"),
						VerifyBody([]byte(`{"name":"monty"}`)),
						RespondWithJSONEncoded(http.StatusOK, `{}`),
					),
				)
			})
			It("should return response", func() {
				_, err := cfc.Post("/v3/post", struct {
					Name string `json:"name"`
				}{Name: "monty"})
				Expect(err).ToNot(HaveOccurred())
			})
		})
	})

	Describe("ResourceRetriever T", func() {
		type TestItem struct {
			Name string `json:"name"`
			Type string `json:"type"`
		}
		When("A successful call is made", func() {
			BeforeEach(func() {
				fakeCC.AppendHandlers(
					CombineHandlers(
						VerifyRequest("GET", "/v3/something"),
						RespondWithJSONEncoded(http.StatusOK, TestItem{Name: "test_name", Type: "test_type"}),
					),
				)
			})

			It("should return correct test item", func() {
				item, err := cf.ResourceRetriever[TestItem]{cfc}.Get("/v3/something")
				Expect(err).ToNot(HaveOccurred())
				Expect(item).To(Equal(TestItem{Name: "test_name", Type: "test_type"}))
			})
		})
	})

	Describe("PagedResourceRetriever T", func() {
		Context("GetPage", func() {
			When("response has invalid json", func() {
				BeforeEach(func() {
					fakeCC.RouteToHandler("GET", "/v3/apps/invalid_json", RespondWith(http.StatusOK, "{"))
				})

				It("should error", func() {
					_, err := cf.PagedResourceRetriever[cf.Process]{cfc}.GetPage("/v3/apps/invalid_json")
					Expect(err).To(HaveOccurred())
					Expect(err.Error()).To(MatchRegexp(`failed unmarshalling cf.Response\[.*cf.Process\]: unexpected EOF`))
				})
			})

			When("response has incorrect message body", func() {
				BeforeEach(func() {
					fakeCC.AppendHandlers(
						CombineHandlers(
							VerifyRequest("GET", "/v3/incorrect_object"),
							RespondWithJSONEncoded(http.StatusOK, `{"entity":{"instances:"abc"}}`),
						),
					)
				})

				It("should error", func() {
					_, err := cf.PagedResourceRetriever[cf.Process]{cfc}.GetPage("/v3/incorrect_object")
					Expect(err).To(MatchError(MatchRegexp(`failed unmarshalling cf.Response\[.*cf.Process\]: json: cannot unmarshal string into Go value.*`)))
					var errType *json.UnmarshalTypeError
					Expect(errors.As(err, &errType)).Should(BeTrue(), "Error was: %#v", interface{}(err))
				})

			})
		})

		Context("GetAllPages", func() {

			When("there are 3 pages with null terminated pagination", func() {

				BeforeEach(func() {
					fakeCC.AppendHandlers(
						CombineHandlers(
							VerifyRequest("GET", "/v3/items"),
							RespondWithJSONEncoded(http.StatusOK,
								cf.Response[cf.Process]{
									Resources:  cf.Processes{{Instances: 1}, {Instances: 1}},
									Pagination: cf.Pagination{Next: cf.Href{Url: fakeCC.URL() + "/v3/items/1"}},
								}),
						),
						CombineHandlers(
							VerifyRequest("GET", "/v3/items/1"),
							RespondWithJSONEncoded(http.StatusOK,
								cf.Response[cf.Process]{
									Resources:  cf.Processes{{Instances: 1}, {Instances: 1}},
									Pagination: cf.Pagination{Next: cf.Href{Url: fakeCC.URL() + "/v3/items/2"}},
								}),
						),
						CombineHandlers(
							VerifyRequest("GET", "/v3/items/2"),
							RespondWith(
								http.StatusOK,
								`{"pagination":{ "next": null }, "resources":[{ "instances": 1 },{ "instances": 1 }] }`,
								http.Header{"Content-Type": []string{"application/json"}}),
						),
					)
				})

				It("counts all processes", func() {
					resp, err := cf.PagedResourceRetriever[cf.Process]{cfc}.GetAllPages("/v3/items")
					Expect(err).ToNot(HaveOccurred())
					Expect(len(resp)).To(Equal(6))
				})
			})
			When("the second page fails", func() {
				BeforeEach(func() {
					fakeCC.AppendHandlers(
						CombineHandlers(
							VerifyRequest("GET", "/v3/items"),
							RespondWithJSONEncoded(http.StatusOK,
								cf.Response[cf.Process]{
									Resources:  cf.Processes{{Instances: 1}, {Instances: 1}},
									Pagination: cf.Pagination{Next: cf.Href{Url: fakeCC.URL() + "/v3/items/1"}},
								}),
						),
						CombineHandlers(
							VerifyRequest("GET", "/v3/items/1"),
							RespondWithJSONEncoded(http.StatusOK,
								cf.Response[cf.Process]{
									Resources:  cf.Processes{{Instances: 1}, {Instances: 1}},
									Pagination: cf.Pagination{Next: cf.Href{Url: fakeCC.URL() + "/v3/items/2"}},
								}),
						),
					)
					fakeCC.RouteToHandler("GET", "/v3/items/2", RespondWithJSONEncoded(http.StatusInternalServerError, models.CfInternalServerError))
				})

				It("returns correct state", func() {
					_, err := cf.PagedResourceRetriever[cf.Process]{cfc}.GetAllPages("/v3/items")
					Expect(err).To(HaveOccurred())
					Expect(err).To(MatchError(MatchRegexp(`failed getting page 3: failed getting cf.Response\[.*cf.Process\]:.*'UnknownError'.*`)))
				})
			})
		})
	})

})
