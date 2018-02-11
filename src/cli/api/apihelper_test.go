package api_test

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"strconv"
	"time"

	. "autoscaler/models"
	. "cli/api"
	. "cli/models"
	"cli/ui"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/ghttp"
	. "github.com/onsi/gomega/gstruct"
)

var _ = Describe("API Helper Test", func() {

	const (
		fakeAppId       string = "fakeAppId"
		fakeAccessToken string = "fakeAccessToken"
	)

	var (
		err        error
		apiServer  *ghttp.Server
		apihelper  *APIHelper
		fakePolicy ScalingPolicy = ScalingPolicy{
			InstanceMin: 1,
			InstanceMax: 2,
			ScalingRules: []*ScalingRule{
				{
					MetricType:            "memoryused",
					StatWindowSeconds:     300,
					BreachDurationSeconds: 600,
					Threshold:             30,
					Operator:              "<=",
					CoolDownSeconds:       300,
					Adjustment:            "-1",
				},
			},
		}
	)

	BeforeEach(func() {

		os.Unsetenv("CF_TRACE")
		apiServer = ghttp.NewServer()
		apihelper = NewAPIHelper(
			&APIEndpoint{
				URL:               apiServer.URL(),
				SkipSSLValidation: false,
			},
			&CFClient{
				CCAPIEndpoint: "fakeCCAPI",
				AuthToken:     fakeAccessToken,
				AppId:         fakeAppId,
				AppName:       "fakeAppName",
			},
			"false",
		)

	})

	AfterEach(func() {
		apiServer.Close()
	})

	Context("Common invalid access errors", func() {

		Context("Server not started", func() {
			BeforeEach(func() {
				apiServer.Close()
			})

			It("Fail to check health", func() {
				err = apihelper.CheckHealth()
				Expect(err).Should(HaveOccurred())
			})

			It("Fail to get policy", func() {
				_, err = apihelper.GetPolicy()
				Expect(err).Should(HaveOccurred())
			})

		})

		Context("No /health endpoint", func() {

			BeforeEach(func() {
				apiServer.RouteToHandler("GET", "/health",
					ghttp.RespondWith(http.StatusNotFound, ""),
				)
			})

			It("Fail to check health", func() {
				err = apihelper.CheckHealth()
				Expect(err).Should(HaveOccurred())
				Expect(err).Should(MatchError(fmt.Sprintf(ui.InvalidAPIEndpoint, apihelper.Endpoint.URL)))
			})

			It("Fail to get policy", func() {
				_, err = apihelper.GetPolicy()
				Expect(err).Should(HaveOccurred())
			})

		})

		Context("Connecting to self-signed TLS server with --skip-ssl-validation = false", func() {
			var err error
			var apiTLSServer *ghttp.Server
			var apiTLSHelper *APIHelper

			BeforeEach(func() {
				apiTLSServer = ghttp.NewTLSServer()
				apiTLSServer.RouteToHandler("GET", "/health",
					ghttp.RespondWith(http.StatusOK, ""),
				)
				apiTLSServer.RouteToHandler("GET", "/v1/apps/"+fakeAppId+"/policy",
					ghttp.RespondWith(http.StatusOK, &fakePolicy),
				)
				apiTLSHelper = NewAPIHelper(
					&APIEndpoint{
						URL: apiTLSServer.URL(),
					},
					&CFClient{
						CCAPIEndpoint: "fakeCCAPI",
						AuthToken:     fakeAccessToken,
						AppId:         fakeAppId,
						AppName:       "fakeAppName",
					},
					"false",
				)

			})

			AfterEach(func() {
				apiTLSServer.Close()
			})

			It("Fail to check health", func() {
				err = apiTLSHelper.CheckHealth()
				Expect(err).Should(HaveOccurred())
				Expect(err).Should(MatchError(fmt.Sprintf(ui.InvalidSSLCerts, apiTLSHelper.Endpoint.URL)))
			})

			It("Fail to Get policy", func() {
				_, err = apiTLSHelper.GetPolicy()
				Expect(err).Should(HaveOccurred())
				Expect(err).Should(MatchError(fmt.Sprintf(ui.InvalidSSLCerts, apiTLSHelper.Endpoint.URL)))
			})

		})

	})

	Context("When API Server is valid", func() {

		BeforeEach(func() {
			apiServer.RouteToHandler("GET", "/health",
				ghttp.RespondWith(http.StatusOK, ""),
			)
		})

		Context("Check Health", func() {
			It("succeed", func() {
				err = apihelper.CheckHealth()
				Expect(err).NotTo(HaveOccurred())
			})
		})

		Context("Get policy", func() {
			var urlpath string = "/v1/apps/" + fakeAppId + "/policy"

			Context("Succeed with valid auth token", func() {
				BeforeEach(func() {
					apiServer.RouteToHandler("GET", urlpath,
						ghttp.CombineHandlers(
							ghttp.RespondWithJSONEncoded(http.StatusOK, &fakePolicy),
							ghttp.VerifyHeaderKV("Authorization", fakeAccessToken),
						),
					)
				})

				It("succeed", func() {
					response, err := apihelper.GetPolicy()
					Expect(err).NotTo(HaveOccurred())

					var actualPolicy ScalingPolicy
					_ = json.Unmarshal([]byte(response), &actualPolicy)
					Expect(actualPolicy).To(MatchFields(IgnoreExtras, Fields{
						"InstanceMin": BeNumerically("==", fakePolicy.InstanceMin),
						"InstanceMax": BeNumerically("==", fakePolicy.InstanceMax),
					}))

					Expect(*actualPolicy.ScalingRules[0]).To(MatchFields(IgnoreExtras, Fields{
						"MetricType":            Equal(fakePolicy.ScalingRules[0].MetricType),
						"StatWindowSeconds":     BeNumerically("==", fakePolicy.ScalingRules[0].StatWindowSeconds),
						"BreachDurationSeconds": BeNumerically("==", fakePolicy.ScalingRules[0].BreachDurationSeconds),
						"Threshold":             BeNumerically("==", fakePolicy.ScalingRules[0].Threshold),
						"Operator":              Equal(fakePolicy.ScalingRules[0].Operator),
						"CoolDownSeconds":       BeNumerically("==", fakePolicy.ScalingRules[0].CoolDownSeconds),
						"Adjustment":            Equal(fakePolicy.ScalingRules[0].Adjustment),
					}))
				})
			})

			Context("Unauthorized Access", func() {
				BeforeEach(func() {
					apiServer.RouteToHandler("GET", urlpath,
						ghttp.RespondWith(http.StatusUnauthorized, ""),
					)
				})

				It("Fail with 401 error", func() {
					_, err = apihelper.GetPolicy()
					Expect(err).Should(HaveOccurred())
					Expect(err).Should(MatchError(fmt.Sprintf(ui.Unauthorized, apihelper.Endpoint.URL, apihelper.Client.CCAPIEndpoint)))
				})
			})

			Context("Policy Not Found", func() {
				BeforeEach(func() {
					apiServer.RouteToHandler("GET", urlpath,
						ghttp.RespondWith(http.StatusNotFound, ""),
					)
				})

				It("Fail with 404 error", func() {
					_, err = apihelper.GetPolicy()
					Expect(err).Should(HaveOccurred())
					Expect(err).Should(MatchError(fmt.Sprintf(ui.PolicyNotFound, apihelper.Client.AppName)))
				})
			})
			Context("Default error handling", func() {
				BeforeEach(func() {
					apiServer.RouteToHandler("GET", urlpath,
						ghttp.RespondWith(http.StatusInternalServerError, `{"success":false,"error":{"message":"Internal error","statusCode":500},"result":null}`),
					)
				})

				It("Fail with 500 error", func() {
					_, err = apihelper.GetPolicy()
					Expect(err).Should(HaveOccurred())
					Expect(err).Should(MatchError("Internal error"))
				})
			})

		})

		Context("Create Policy", func() {
			var urlpath string = "/v1/apps/" + fakeAppId + "/policy"

			Context("201 Created with valid auth token", func() {
				BeforeEach(func() {
					apiServer.RouteToHandler("PUT", urlpath,
						ghttp.CombineHandlers(
							ghttp.RespondWith(http.StatusCreated, ""),
							ghttp.VerifyHeaderKV("Authorization", fakeAccessToken),
						),
					)
				})

				It("succeed", func() {
					err = apihelper.CreatePolicy(fakePolicy)
					Expect(err).NotTo(HaveOccurred())
				})
			})

			Context("200 OK with valid auth token", func() {
				BeforeEach(func() {
					apiServer.RouteToHandler("PUT", urlpath,
						ghttp.CombineHandlers(
							ghttp.RespondWith(http.StatusOK, ""),
							ghttp.VerifyHeaderKV("Authorization", fakeAccessToken),
						),
					)
				})

				It("succeed", func() {
					err = apihelper.CreatePolicy(fakePolicy)
					Expect(err).NotTo(HaveOccurred())
				})
			})

			Context("Unauthorized Access", func() {
				BeforeEach(func() {
					apiServer.RouteToHandler("PUT", urlpath,
						ghttp.RespondWith(http.StatusUnauthorized, ""),
					)
				})

				It("Fail with 401 error", func() {
					err = apihelper.CreatePolicy(fakePolicy)
					Expect(err).Should(HaveOccurred())
					Expect(err).Should(MatchError(fmt.Sprintf(ui.Unauthorized, apihelper.Endpoint.URL, apihelper.Client.CCAPIEndpoint)))
				})
			})

			Context("Invalid Policy Format", func() {
				BeforeEach(func() {
					apiServer.RouteToHandler("PUT", urlpath,
						ghttp.RespondWith(http.StatusBadRequest, `{"success":false,"error":[{"property":"instance_min_count","message":"instance_min_count and instance_max_count values are not compatible","instance":{"instance_max_count":2,"instance_min_count":10,"scaling_rules":[{"adjustment":"+1","breach_duration_secs":600,"cool_down_secs":300,"metric_type":"memoryused","operator":">","stat_window_secs":300,"threshold":100},{"adjustment":"-1","breach_duration_secs":600,"cool_down_secs":300,"metric_type":"memoryused","operator":"<=","stat_window_secs":300,"threshold":5}]},"stack":"instance_min_count 10 is higher or equal to instance_max_count 2 in policy_json"}],"result":null}`),
					)
				})

				It("Fail with 400 error", func() {
					err = apihelper.CreatePolicy(fakePolicy)
					Expect(err).Should(HaveOccurred())
					Expect(err).Should(MatchError(fmt.Sprintf(ui.InvalidPolicy, "instance_min_count 10 is higher or equal to instance_max_count 2 in policy_json")))
				})
			})

			Context("Default error handling", func() {
				BeforeEach(func() {
					apiServer.RouteToHandler("PUT", urlpath,
						ghttp.RespondWith(http.StatusInternalServerError, `{"success":false,"error":{"message":"Internal error","statusCode":500},"result":null}`),
					)
				})

				It("Fail with 500 error", func() {
					err = apihelper.CreatePolicy(fakePolicy)
					Expect(err).Should(HaveOccurred())
					Expect(err).Should(MatchError("Internal error"))
				})
			})

		})

		Context("Delete Policy", func() {
			var urlpath string = "/v1/apps/" + fakeAppId + "/policy"

			Context("Succeed with valid auth token", func() {
				BeforeEach(func() {
					apiServer.RouteToHandler("DELETE", urlpath,
						ghttp.CombineHandlers(
							ghttp.RespondWith(http.StatusOK, ""),
							ghttp.VerifyHeaderKV("Authorization", fakeAccessToken),
						),
					)
				})

				It("succeed", func() {
					err = apihelper.DeletePolicy()
					Expect(err).NotTo(HaveOccurred())
				})
			})

			Context("Unauthorized Access", func() {
				BeforeEach(func() {
					apiServer.RouteToHandler("DELETE", urlpath,
						ghttp.RespondWith(http.StatusUnauthorized, ""),
					)
				})

				It("Fail with 401 error", func() {
					err = apihelper.DeletePolicy()
					Expect(err).Should(HaveOccurred())
					Expect(err).Should(MatchError(fmt.Sprintf(ui.Unauthorized, apihelper.Endpoint.URL, apihelper.Client.CCAPIEndpoint)))
				})
			})

			Context("Policy not found", func() {
				BeforeEach(func() {
					apiServer.RouteToHandler("DELETE", urlpath,
						ghttp.RespondWith(http.StatusNotFound, `{"success":false,"error":{"message":"No policy bound with application","statusCode":404},"result":null}`),
					)
				})

				It("Fail with 404 error", func() {
					err = apihelper.DeletePolicy()
					Expect(err).Should(HaveOccurred())
					Expect(err).Should(MatchError(fmt.Sprintf(ui.PolicyNotFound, apihelper.Client.AppName)))
				})
			})

			Context("Default error handling", func() {
				BeforeEach(func() {
					apiServer.RouteToHandler("DELETE", urlpath,
						ghttp.RespondWith(http.StatusInternalServerError, `{"success":false,"error":{"message":"Internal error","statusCode":500},"result":null}`),
					)
				})

				It("Fail with 500 error", func() {
					err = apihelper.DeletePolicy()
					Expect(err).Should(HaveOccurred())
					Expect(err).Should(MatchError("Internal error"))
				})
			})

		})

		Context("Get Metrics", func() {
			var urlpath = "/v1/apps/" + fakeAppId + "/metric_histories/memoryused"
			var now int64
			var metrics, reversedMetrics []*AppInstanceMetric

			BeforeEach(func() {
				now = time.Now().UnixNano()

				for i := 0; i < 30; i++ {
					metrics = append(metrics, &AppInstanceMetric{
						AppId:         fakeAppId,
						InstanceIndex: 0,
						CollectedAt:   now + int64(i*30*1E9),
						Name:          "memoryused",
						Unit:          "MB",
						Value:         "100",
						Timestamp:     now + int64(i*30*1E9),
					})
				}

				for i := 0; i < 30; i++ {
					reversedMetrics = append(reversedMetrics, metrics[len(metrics)-1-i])
				}
			})

			Context("With valid auth token", func() {

				Context("Query multiple pages with order asc", func() {
					BeforeEach(func() {
						apiServer.AppendHandlers(
							ghttp.CombineHandlers(
								ghttp.RespondWithJSONEncoded(http.StatusOK, &MetricsResults{
									TotalResults: 30,
									TotalPages:   3,
									Page:         1,
									Metrics:      metrics[0:10],
								}),
								ghttp.VerifyHeaderKV("Authorization", fakeAccessToken),
								ghttp.VerifyRequest("GET", urlpath, "order=asc&page=1"),
							),
						)

						apiServer.AppendHandlers(
							ghttp.CombineHandlers(
								ghttp.RespondWithJSONEncoded(http.StatusOK, &MetricsResults{
									TotalResults: 30,
									TotalPages:   3,
									Page:         2,
									Metrics:      metrics[10:20],
								}),
								ghttp.VerifyHeaderKV("Authorization", fakeAccessToken),
								ghttp.VerifyRequest("GET", urlpath, "order=asc&page=2"),
							),
						)

						apiServer.AppendHandlers(
							ghttp.CombineHandlers(
								ghttp.RespondWithJSONEncoded(http.StatusOK, &MetricsResults{
									TotalResults: 30,
									TotalPages:   3,
									Page:         3,
									Metrics:      metrics[20:30],
								}),
								ghttp.VerifyHeaderKV("Authorization", fakeAccessToken),
								ghttp.VerifyRequest("GET", urlpath, "order=asc&page=3"),
							),
						)
					})

					It("succeed", func() {

						next, data, err := apihelper.GetMetrics("memoryused", 0, 0, false, uint64(1))
						Expect(err).NotTo(HaveOccurred())
						Expect(next).To(BeTrue())
						Expect(len(data)).To(Equal(10))

						for i, row := range data {
							Expect(row[0]).To(Equal("memoryused"))
							Expect(row[1]).To(Equal("0"))
							Expect(row[2]).To(Equal("100MB"))
							Expect(row[3]).To(Equal(time.Unix(0, now+int64(i*30*1E9)).Format(time.RFC3339)))
						}

						next, data, err = apihelper.GetMetrics("memoryused", 0, 0, false, uint64(2))
						Expect(err).NotTo(HaveOccurred())
						Expect(next).To(BeTrue())
						Expect(len(data)).To(Equal(10))

						for i, row := range data {
							Expect(row[0]).To(Equal("memoryused"))
							Expect(row[1]).To(Equal("0"))
							Expect(row[2]).To(Equal("100MB"))
							Expect(row[3]).To(Equal(time.Unix(0, now+int64((i+10)*30*1E9)).Format(time.RFC3339)))
						}

						next, data, err = apihelper.GetMetrics("memoryused", 0, 0, false, uint64(3))
						Expect(err).NotTo(HaveOccurred())
						Expect(next).To(BeFalse())
						Expect(len(data)).To(Equal(10))

						for i, row := range data {
							Expect(row[0]).To(Equal("memoryused"))
							Expect(row[1]).To(Equal("0"))
							Expect(row[2]).To(Equal("100MB"))
							Expect(row[3]).To(Equal(time.Unix(0, now+int64((i+20)*30*1E9)).Format(time.RFC3339)))
						}

					})
				})

				Context("Query multiple pages with order desc", func() {
					BeforeEach(func() {
						apiServer.AppendHandlers(
							ghttp.CombineHandlers(
								ghttp.RespondWithJSONEncoded(http.StatusOK, &MetricsResults{
									TotalResults: 30,
									TotalPages:   3,
									Page:         1,
									Metrics:      reversedMetrics[0:10],
								}),
								ghttp.VerifyHeaderKV("Authorization", fakeAccessToken),
								ghttp.VerifyRequest("GET", urlpath, "order=desc&page=1"),
							),
						)

						apiServer.AppendHandlers(
							ghttp.CombineHandlers(
								ghttp.RespondWithJSONEncoded(http.StatusOK, &MetricsResults{
									TotalResults: 30,
									TotalPages:   3,
									Page:         2,
									Metrics:      reversedMetrics[10:20],
								}),
								ghttp.VerifyHeaderKV("Authorization", fakeAccessToken),
								ghttp.VerifyRequest("GET", urlpath, "order=desc&page=2"),
							),
						)

						apiServer.AppendHandlers(
							ghttp.CombineHandlers(
								ghttp.RespondWithJSONEncoded(http.StatusOK, &MetricsResults{
									TotalResults: 30,
									TotalPages:   3,
									Page:         3,
									Metrics:      reversedMetrics[20:30],
								}),
								ghttp.VerifyHeaderKV("Authorization", fakeAccessToken),
								ghttp.VerifyRequest("GET", urlpath, "order=desc&page=3"),
							),
						)
					})

					It("succeed", func() {

						next, data, err := apihelper.GetMetrics("memoryused", 0, 0, true, uint64(1))
						Expect(err).NotTo(HaveOccurred())
						Expect(next).To(BeTrue())
						Expect(len(data)).To(Equal(10))

						for i, row := range data {
							Expect(row[0]).To(Equal("memoryused"))
							Expect(row[1]).To(Equal("0"))
							Expect(row[2]).To(Equal("100MB"))
							Expect(row[3]).To(Equal(time.Unix(0, now+int64((29-i)*30*1E9)).Format(time.RFC3339)))
						}

						next, data, err = apihelper.GetMetrics("memoryused", 0, 0, true, uint64(2))
						Expect(err).NotTo(HaveOccurred())
						Expect(next).To(BeTrue())
						Expect(len(data)).To(Equal(10))

						for i, row := range data {
							Expect(row[0]).To(Equal("memoryused"))
							Expect(row[1]).To(Equal("0"))
							Expect(row[2]).To(Equal("100MB"))
							Expect(row[3]).To(Equal(time.Unix(0, now+int64((19-i)*30*1E9)).Format(time.RFC3339)))
						}

						next, data, err = apihelper.GetMetrics("memoryused", 0, 0, true, uint64(3))
						Expect(err).NotTo(HaveOccurred())
						Expect(next).To(BeFalse())
						Expect(len(data)).To(Equal(10))

						for i, row := range data {
							Expect(row[0]).To(Equal("memoryused"))
							Expect(row[1]).To(Equal("0"))
							Expect(row[2]).To(Equal("100MB"))
							Expect(row[3]).To(Equal(time.Unix(0, now+int64((9-i)*30*1E9)).Format(time.RFC3339)))
						}

					})
				})

				Context("Query with asc & start time & end time ", func() {
					BeforeEach(func() {
						apiServer.AppendHandlers(
							ghttp.CombineHandlers(
								ghttp.RespondWithJSONEncoded(http.StatusOK, &MetricsResults{
									TotalResults: 10,
									TotalPages:   1,
									Page:         1,
									Metrics:      metrics[0:10],
								}),
								ghttp.VerifyHeaderKV("Authorization", fakeAccessToken),
								ghttp.VerifyRequest("GET", urlpath, fmt.Sprintf("order=asc&page=1&start-time=%v&end-time=%v", now, now+int64(9*30*1E9))),
							),
						)
					})

					It("succeed", func() {

						next, data, err := apihelper.GetMetrics("memoryused", now, now+int64(9*30*1E9), false, uint64(1))
						Expect(err).NotTo(HaveOccurred())
						Expect(next).To(BeFalse())
						Expect(len(data)).To(Equal(10))

						for i, row := range data {
							Expect(row[0]).To(Equal("memoryused"))
							Expect(row[1]).To(Equal("0"))
							Expect(row[2]).To(Equal("100MB"))
							Expect(row[3]).To(Equal(time.Unix(0, now+int64(i*30*1E9)).Format(time.RFC3339)))
						}

					})
				})

				Context("Query when no metrics avaialable in desired period", func() {
					BeforeEach(func() {
						apiServer.AppendHandlers(
							ghttp.CombineHandlers(
								ghttp.RespondWithJSONEncoded(http.StatusOK, &MetricsResults{
									TotalResults: 0,
									TotalPages:   0,
									Page:         1,
									Metrics:      []*AppInstanceMetric{},
								}),
								ghttp.VerifyHeaderKV("Authorization", fakeAccessToken),
								ghttp.VerifyRequest("GET", urlpath, fmt.Sprintf("order=asc&page=1&start-time=%v&end-time=%v", now, now+int64(9*30*1E9))),
							),
						)
					})

					It("succeed", func() {
						next, data, err := apihelper.GetMetrics("memoryused", now, now+int64(9*30*1E9), false, uint64(1))
						Expect(err).NotTo(HaveOccurred())
						Expect(next).To(BeFalse())
						Expect(len(data)).To(Equal(0))

					})
				})
			})

			Context("Unauthorized Access", func() {
				BeforeEach(func() {
					apiServer.AppendHandlers(
						ghttp.CombineHandlers(
							ghttp.RespondWith(http.StatusUnauthorized, ""),
							ghttp.VerifyRequest("GET", urlpath, "order=asc&page=1"),
						),
					)
				})

				It("Fail with 401 error", func() {
					_, _, err = apihelper.GetMetrics("memoryused", 0, 0, false, uint64(1))
					Expect(err).Should(HaveOccurred())
					Expect(err).Should(MatchError(fmt.Sprintf(ui.Unauthorized, apihelper.Endpoint.URL, apihelper.Client.CCAPIEndpoint)))
				})
			})

			Context("Default error handling", func() {
				BeforeEach(func() {
					apiServer.AppendHandlers(
						ghttp.CombineHandlers(
							ghttp.RespondWith(http.StatusInternalServerError, `{"success":false,"error":{"message":"Internal error","statusCode":500},"result":null}`),
							ghttp.VerifyRequest("GET", urlpath, "order=asc&page=1"),
						),
					)
				})

				It("Fail with 500 error", func() {
					_, _, err = apihelper.GetMetrics("memoryused", 0, 0, false, uint64(1))
					Expect(err).Should(HaveOccurred())
					Expect(err).Should(MatchError("Internal error"))
				})
			})

		})

		Context("Get Histrory", func() {
			var urlpath = "/v1/apps/" + fakeAppId + "/scaling_histories"
			var now int64
			var histories, reversedHistories []*AppScalingHistory

			BeforeEach(func() {
				now = time.Now().UnixNano()

				for i := 0; i < 10; i++ {
					histories = append(histories, &AppScalingHistory{
						AppId:        fakeAppId,
						Timestamp:    now + int64(i*120*1E9),
						ScalingType:  0, //dynamic
						Status:       0, //succeed
						OldInstances: i + 1,
						NewInstances: i + 2,
						Reason:       "fakeReason",
						Message:      "",
						Error:        "fakeError",
					})
				}

				for i := 10; i < 20; i++ {
					histories = append(histories, &AppScalingHistory{
						AppId:        fakeAppId,
						Timestamp:    now + int64(i*120*1E9),
						ScalingType:  1, //scheduled
						Status:       0, //succeed
						OldInstances: i + 1,
						NewInstances: i + 2,
						Reason:       "fakeReason",
						Message:      "",
						Error:        "fakeError",
					})
				}

				for i := 20; i < 30; i++ {
					histories = append(histories, &AppScalingHistory{
						AppId:        fakeAppId,
						Timestamp:    now + int64(i*120*1E9),
						ScalingType:  1, //scheduled
						Status:       1, //failed
						OldInstances: i + 1,
						NewInstances: i + 2,
						Reason:       "fakeReason",
						Message:      "",
						Error:        "fakeError",
					})
				}
				for i := 0; i < len(histories); i++ {
					reversedHistories = append(reversedHistories, histories[len(histories)-1-i])
				}
			})

			Context("With valid auth token", func() {

				var histories_ut []*AppScalingHistory
				Context("Query single pages wiht different history format", func() {
					BeforeEach(func() {

						histories_ut = append(histories_ut, &AppScalingHistory{
							AppId:        fakeAppId,
							Timestamp:    now,
							ScalingType:  0, //dynamic
							Status:       0, //succeed
							OldInstances: 10,
							NewInstances: 11,
							Reason:       "fakeReason",
							Message:      "",
							Error:        "fakeError",
						})
						histories_ut = append(histories_ut, &AppScalingHistory{
							AppId:        fakeAppId,
							Timestamp:    now + int64(120*1E9),
							ScalingType:  0, //dynamic
							Status:       0, //succeed
							OldInstances: 11,
							NewInstances: 2,
							Reason:       "fakeReason",
							Message:      "fakeMsg",
							Error:        "fakeError",
						})

						apiServer.AppendHandlers(
							ghttp.CombineHandlers(
								ghttp.RespondWithJSONEncoded(http.StatusOK, &HistoryResults{
									TotalResults: 2,
									TotalPages:   1,
									Page:         1,
									Histories:    histories_ut[0:2],
								}),
								ghttp.VerifyHeaderKV("Authorization", fakeAccessToken),
								ghttp.VerifyRequest("GET", urlpath, "order=asc&page=1"),
							),
						)

					})

					It("succeed", func() {
						next, data, err := apihelper.GetHistory(0, 0, false, uint64(1))
						Expect(err).NotTo(HaveOccurred())
						Expect(next).To(BeFalse())
						Expect(len(data)).To(Equal(2))

						Expect(data[0][0]).To(Equal("dynamic"))
						Expect(data[0][1]).To(Equal("succeed"))
						Expect(data[0][2]).To(Equal("10->11"))
						Expect(data[0][3]).To(Equal(time.Unix(0, now).Format(time.RFC3339)))
						Expect(data[0][4]).To(Equal("fakeReason"))
						Expect(data[0][5]).To(Equal("fakeError"))

						Expect(data[1][0]).To(Equal("dynamic"))
						Expect(data[1][1]).To(Equal("succeed"))
						Expect(data[1][2]).To(Equal("11->2"))
						Expect(data[1][3]).To(Equal(time.Unix(0, now+int64(120*1E9)).Format(time.RFC3339)))
						Expect(data[1][4]).To(Equal("-9 instance(s) because fakeMsg"))
						Expect(data[1][5]).To(Equal("fakeError"))

					})
				})

				Context("Query multiple pages with order asc", func() {
					BeforeEach(func() {
						apiServer.AppendHandlers(
							ghttp.CombineHandlers(
								ghttp.RespondWithJSONEncoded(http.StatusOK, &HistoryResults{
									TotalResults: 30,
									TotalPages:   4,
									Page:         1,
									Histories:    histories[0:10],
								}),
								ghttp.VerifyHeaderKV("Authorization", fakeAccessToken),
								ghttp.VerifyRequest("GET", urlpath, "order=asc&page=1"),
							),
						)

						apiServer.AppendHandlers(
							ghttp.CombineHandlers(
								ghttp.RespondWithJSONEncoded(http.StatusOK, &HistoryResults{
									TotalResults: 30,
									TotalPages:   3,
									Page:         2,
									Histories:    histories[10:20],
								}),
								ghttp.VerifyHeaderKV("Authorization", fakeAccessToken),
								ghttp.VerifyRequest("GET", urlpath, "order=asc&page=2"),
							),
						)

						apiServer.AppendHandlers(
							ghttp.CombineHandlers(
								ghttp.RespondWithJSONEncoded(http.StatusOK, &HistoryResults{
									TotalResults: 30,
									TotalPages:   3,
									Page:         3,
									Histories:    histories[20:30],
								}),
								ghttp.VerifyHeaderKV("Authorization", fakeAccessToken),
								ghttp.VerifyRequest("GET", urlpath, "order=asc&page=3"),
							),
						)

					})

					It("succeed", func() {

						next, data, err := apihelper.GetHistory(0, 0, false, uint64(1))
						Expect(err).NotTo(HaveOccurred())
						Expect(next).To(BeTrue())
						Expect(len(data)).To(Equal(10))

						for i, row := range data {
							Expect(row[0]).To(Equal("dynamic"))
							Expect(row[1]).To(Equal("succeed"))
							Expect(row[2]).To(Equal(strconv.Itoa(i+1) + "->" + strconv.Itoa(i+2)))
							Expect(row[3]).To(Equal(time.Unix(0, now+int64(i*120*1E9)).Format(time.RFC3339)))
							Expect(row[4]).To(Equal("fakeReason"))
							Expect(row[5]).To(Equal("fakeError"))
						}

						next, data, err = apihelper.GetHistory(0, 0, false, uint64(2))
						Expect(err).NotTo(HaveOccurred())
						Expect(next).To(BeTrue())
						Expect(len(data)).To(Equal(10))

						for i, row := range data {
							Expect(row[0]).To(Equal("scheduled"))
							Expect(row[1]).To(Equal("succeed"))
							Expect(row[2]).To(Equal(strconv.Itoa(i+10+1) + "->" + strconv.Itoa(i+10+2)))
							Expect(row[3]).To(Equal(time.Unix(0, now+int64((i+10)*120*1E9)).Format(time.RFC3339)))
							Expect(row[4]).To(Equal("fakeReason"))
							Expect(row[5]).To(Equal("fakeError"))
						}

						next, data, err = apihelper.GetHistory(0, 0, false, uint64(3))
						Expect(err).NotTo(HaveOccurred())
						Expect(next).To(BeFalse())
						Expect(len(data)).To(Equal(10))

						for i, row := range data {
							Expect(row[0]).To(Equal("scheduled"))
							Expect(row[1]).To(Equal("failed"))
							Expect(row[2]).To(Equal(strconv.Itoa(i+20+1) + "->" + strconv.Itoa(i+20+2)))
							Expect(row[3]).To(Equal(time.Unix(0, now+int64((i+20)*120*1E9)).Format(time.RFC3339)))
							Expect(row[4]).To(Equal("fakeReason"))
							Expect(row[5]).To(Equal("fakeError"))
						}

					})
				})

				Context("Query multiple pages with order desc", func() {
					BeforeEach(func() {
						apiServer.AppendHandlers(
							ghttp.CombineHandlers(
								ghttp.RespondWithJSONEncoded(http.StatusOK, &HistoryResults{
									TotalResults: 30,
									TotalPages:   3,
									Page:         1,
									Histories:    reversedHistories[0:10],
								}),
								ghttp.VerifyHeaderKV("Authorization", fakeAccessToken),
								ghttp.VerifyRequest("GET", urlpath, "order=desc&page=1"),
							),
						)

						apiServer.AppendHandlers(
							ghttp.CombineHandlers(
								ghttp.RespondWithJSONEncoded(http.StatusOK, &HistoryResults{
									TotalResults: 30,
									TotalPages:   3,
									Page:         2,
									Histories:    reversedHistories[10:20],
								}),
								ghttp.VerifyHeaderKV("Authorization", fakeAccessToken),
								ghttp.VerifyRequest("GET", urlpath, "order=desc&page=2"),
							),
						)

						apiServer.AppendHandlers(
							ghttp.CombineHandlers(
								ghttp.RespondWithJSONEncoded(http.StatusOK, &HistoryResults{
									TotalResults: 30,
									TotalPages:   3,
									Page:         3,
									Histories:    reversedHistories[20:30],
								}),
								ghttp.VerifyHeaderKV("Authorization", fakeAccessToken),
								ghttp.VerifyRequest("GET", urlpath, "order=desc&page=3"),
							),
						)
					})

					It("succeed", func() {

						next, data, err := apihelper.GetHistory(0, 0, true, uint64(1))
						Expect(err).NotTo(HaveOccurred())
						Expect(next).To(BeTrue())
						Expect(len(data)).To(Equal(10))

						for i, row := range data {
							Expect(row[0]).To(Equal("scheduled"))
							Expect(row[1]).To(Equal("failed"))
							Expect(row[2]).To(Equal(strconv.Itoa(29-i+1) + "->" + strconv.Itoa(29-i+2)))
							Expect(row[3]).To(Equal(time.Unix(0, now+int64((29-i)*120*1E9)).Format(time.RFC3339)))
							Expect(row[4]).To(Equal("fakeReason"))
							Expect(row[5]).To(Equal("fakeError"))
						}

						next, data, err = apihelper.GetHistory(0, 0, true, uint64(2))
						Expect(err).NotTo(HaveOccurred())
						Expect(next).To(BeTrue())
						Expect(len(data)).To(Equal(10))

						for i, row := range data {
							Expect(row[0]).To(Equal("scheduled"))
							Expect(row[1]).To(Equal("succeed"))
							Expect(row[2]).To(Equal(strconv.Itoa(19-i+1) + "->" + strconv.Itoa(19-i+2)))
							Expect(row[3]).To(Equal(time.Unix(0, now+int64((19-i)*120*1E9)).Format(time.RFC3339)))
							Expect(row[4]).To(Equal("fakeReason"))
							Expect(row[5]).To(Equal("fakeError"))
						}

						next, data, err = apihelper.GetHistory(0, 0, true, uint64(3))
						Expect(err).NotTo(HaveOccurred())
						Expect(next).To(BeFalse())
						Expect(len(data)).To(Equal(10))

						for i, row := range data {
							Expect(row[0]).To(Equal("dynamic"))
							Expect(row[1]).To(Equal("succeed"))
							Expect(row[2]).To(Equal(strconv.Itoa(9-i+1) + "->" + strconv.Itoa(9-i+2)))
							Expect(row[3]).To(Equal(time.Unix(0, now+int64((9-i)*120*1E9)).Format(time.RFC3339)))
							Expect(row[4]).To(Equal("fakeReason"))
							Expect(row[5]).To(Equal("fakeError"))
						}

					})
				})

				Context("Query with asc & start time & end time ", func() {
					BeforeEach(func() {
						apiServer.AppendHandlers(
							ghttp.CombineHandlers(
								ghttp.RespondWithJSONEncoded(http.StatusOK, &HistoryResults{
									TotalResults: 10,
									TotalPages:   1,
									Page:         1,
									Histories:    histories[0:10],
								}),
								ghttp.VerifyHeaderKV("Authorization", fakeAccessToken),
								ghttp.VerifyRequest("GET", urlpath, fmt.Sprintf("order=asc&page=1&start-time=%v&end-time=%v", now, now+int64(9*120*1E9))),
							),
						)
					})

					It("succeed", func() {

						next, data, err := apihelper.GetHistory(now, now+int64(9*120*1E9), false, uint64(1))
						Expect(err).NotTo(HaveOccurred())
						Expect(next).To(BeFalse())
						Expect(len(data)).To(Equal(10))

						for i, row := range data {
							Expect(row[0]).To(Equal("dynamic"))
							Expect(row[1]).To(Equal("succeed"))
							Expect(row[2]).To(Equal(strconv.Itoa(i+1) + "->" + strconv.Itoa(i+2)))
							Expect(row[3]).To(Equal(time.Unix(0, now+int64(i*120*1E9)).Format(time.RFC3339)))
							Expect(row[4]).To(Equal("fakeReason"))
							Expect(row[5]).To(Equal("fakeError"))
						}

					})
				})

				Context("Query when no history avaialable in desired period", func() {
					BeforeEach(func() {
						apiServer.AppendHandlers(
							ghttp.CombineHandlers(
								ghttp.RespondWithJSONEncoded(http.StatusOK, &HistoryResults{
									TotalResults: 0,
									TotalPages:   0,
									Page:         1,
									Histories:    []*AppScalingHistory{},
								}),
								ghttp.VerifyHeaderKV("Authorization", fakeAccessToken),
								ghttp.VerifyRequest("GET", urlpath, fmt.Sprintf("order=asc&page=1&start-time=%v&end-time=%v", now, now+int64(9*120*1E9))),
							),
						)
					})

					It("succeed", func() {
						next, data, err := apihelper.GetHistory(now, now+int64(9*120*1E9), false, uint64(1))
						Expect(err).NotTo(HaveOccurred())
						Expect(next).To(BeFalse())
						Expect(len(data)).To(Equal(0))

					})
				})
			})

			Context("Unauthorized Access", func() {
				BeforeEach(func() {
					apiServer.AppendHandlers(
						ghttp.CombineHandlers(
							ghttp.RespondWith(http.StatusUnauthorized, ""),
							ghttp.VerifyRequest("GET", urlpath, "order=asc&page=1"),
						),
					)
				})

				It("Fail with 401 error", func() {
					_, _, err := apihelper.GetHistory(0, 0, false, uint64(1))
					Expect(err).Should(HaveOccurred())
					Expect(err).Should(MatchError(fmt.Sprintf(ui.Unauthorized, apihelper.Endpoint.URL, apihelper.Client.CCAPIEndpoint)))
				})
			})

			Context("Default error handling", func() {
				BeforeEach(func() {
					apiServer.AppendHandlers(
						ghttp.CombineHandlers(
							ghttp.RespondWith(http.StatusInternalServerError, `{"success":false,"error":{"message":"Internal error","statusCode":500},"result":null}`),
							ghttp.VerifyRequest("GET", urlpath, "order=asc&page=1"),
						),
					)
				})

				It("Fail with 500 error", func() {
					_, _, err := apihelper.GetHistory(0, 0, false, uint64(1))
					Expect(err).Should(HaveOccurred())
					Expect(err).Should(MatchError("Internal error"))
				})
			})

		})
	})
})
