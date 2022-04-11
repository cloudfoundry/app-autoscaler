package server_test

import (
	"code.cloudfoundry.org/app-autoscaler/src/autoscaler/db"
	"code.cloudfoundry.org/app-autoscaler/src/autoscaler/fakes"
	"code.cloudfoundry.org/app-autoscaler/src/autoscaler/models"
	. "code.cloudfoundry.org/app-autoscaler/src/autoscaler/scalingengine/server"

	"code.cloudfoundry.org/lager/lagertest"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"bytes"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
)

const testUrlScalingHistories = "http://localhost/v1/apps/an-app-id/scaling_histories"
const testUrlActiveSchedules = "http://localhost/v1/apps/an-app-id/active_schedules/a-schedule-id"
const testUrlAppActiveSchedule = "http://localhost/v1/apps/an-app-id/active_schedules"

var _ = Describe("ScalingHandler", func() {
	var (
		scalingEngineDB              *fakes.FakeScalingEngineDB
		scalingEngine                *fakes.FakeScalingEngine
		handler                      *ScalingHandler
		resp                         *httptest.ResponseRecorder
		req                          *http.Request
		body                         []byte
		err                          error
		trigger                      *models.Trigger
		history1, history2, history3 *models.AppScalingHistory
		activeSchedule               *models.ActiveSchedule
		testMetricName               = "Test-Metric-Name"
	)

	BeforeEach(func() {
		logger := lagertest.NewTestLogger("scaling-handler-test")
		scalingEngineDB = &fakes.FakeScalingEngineDB{}
		scalingEngine = &fakes.FakeScalingEngine{}
		handler = NewScalingHandler(logger, scalingEngineDB, scalingEngine)
		resp = httptest.NewRecorder()
	})

	Describe("Scale", func() {
		JustBeforeEach(func() {
			handler.Scale(resp, req, map[string]string{"appid": "an-app-id"})
		})

		Context("when scaling app succeeds", func() {
			BeforeEach(func() {
				scalingEngine.ScaleReturns(&models.AppScalingResult{
					AppId:             "an-app-id",
					Status:            models.ScalingStatusSucceeded,
					Adjustment:        1,
					CooldownExpiredAt: 10000,
				}, nil)

				trigger = &models.Trigger{
					MetricType: testMetricName,
					Adjustment: "+1",
				}
				body, err = json.Marshal(trigger)
				Expect(err).NotTo(HaveOccurred())

				req, err = http.NewRequest("POST", "", bytes.NewReader(body))
				Expect(err).NotTo(HaveOccurred())
			})

			It("returns 200 with new instances number", func() {
				Expect(resp.Code).To(Equal(http.StatusOK))

				Expect(scalingEngine.ScaleCallCount()).To(Equal(1))
				appId, scaleTrigger := scalingEngine.ScaleArgsForCall(0)
				Expect(appId).To(Equal("an-app-id"))
				Expect(scaleTrigger).To(Equal(trigger))

				props := &models.AppScalingResult{}
				err = json.Unmarshal(resp.Body.Bytes(), props)
				Expect(err).NotTo(HaveOccurred())
				Expect(props.Adjustment).To(Equal(1))
				Expect(props.AppId).To(Equal("an-app-id"))
				Expect(props.Status).To(Equal(models.ScalingStatusSucceeded))
				Expect(props.CooldownExpiredAt).To(Equal(int64(10000)))
			})
		})

		Context("when request body is not valid", func() {
			BeforeEach(func() {
				req, err = http.NewRequest("POST", "", bytes.NewReader([]byte(`bad body`)))
				Expect(err).NotTo(HaveOccurred())
			})

			It("returns 400", func() {
				Expect(resp.Code).To(Equal(http.StatusBadRequest))

				errJson := &models.ErrorResponse{}
				err = json.Unmarshal(resp.Body.Bytes(), errJson)
				Expect(err).ToNot(HaveOccurred())
				Expect(errJson).To(Equal(&models.ErrorResponse{
					Code:    "Bad-Request",
					Message: "Incorrect trigger in request body",
				}))
			})
		})

		Context("when scaling app fails", func() {
			BeforeEach(func() {
				scalingEngine.ScaleReturns(&models.AppScalingResult{
					AppId:             "an-app-id",
					Status:            models.ScalingStatusFailed,
					Adjustment:        0,
					CooldownExpiredAt: 0,
				}, errors.New("an error"))

				trigger = &models.Trigger{
					MetricType: testMetricName,
					Adjustment: "+1",
				}
				body, err = json.Marshal(trigger)
				Expect(err).NotTo(HaveOccurred())

				req, err = http.NewRequest("POST", "", bytes.NewReader(body))
				Expect(err).NotTo(HaveOccurred())
			})

			It("returns 500", func() {
				Expect(resp.Code).To(Equal(http.StatusInternalServerError))

				errJson := &models.ErrorResponse{}
				err = json.Unmarshal(resp.Body.Bytes(), errJson)
				Expect(err).ToNot(HaveOccurred())
				Expect(errJson).To(Equal(&models.ErrorResponse{
					Code:    "Internal-server-error",
					Message: "Error taking scaling action",
				}))
			})
		})
	})

	Describe("GetScalingHistories", func() {
		JustBeforeEach(func() {
			handler.GetScalingHistories(resp, req, map[string]string{"appid": "an-app-id"})
		})

		Context("when request query string is invalid", func() {
			Context("when there are multiple start pararmeters in query string", func() {
				BeforeEach(func() {
					req, err = http.NewRequest(http.MethodGet, testUrlScalingHistories+"?start=123&start=231", nil)
					Expect(err).ToNot(HaveOccurred())
				})

				It("returns 400", func() {
					Expect(resp.Code).To(Equal(http.StatusBadRequest))

					errJson := &models.ErrorResponse{}
					err = json.Unmarshal(resp.Body.Bytes(), errJson)

					Expect(err).ToNot(HaveOccurred())
					Expect(errJson).To(Equal(&models.ErrorResponse{
						Code:    "Bad-Request",
						Message: "Incorrect start parameter in query string",
					}))
				})
			})

			Context("when start time is not a number", func() {
				BeforeEach(func() {
					req, err = http.NewRequest(http.MethodGet, testUrlScalingHistories+"?start=abc", nil)
					Expect(err).ToNot(HaveOccurred())
				})

				It("returns 400", func() {
					Expect(resp.Code).To(Equal(http.StatusBadRequest))

					errJson := &models.ErrorResponse{}
					err = json.Unmarshal(resp.Body.Bytes(), errJson)

					Expect(err).ToNot(HaveOccurred())
					Expect(errJson).To(Equal(&models.ErrorResponse{
						Code:    "Bad-Request",
						Message: "Error parsing start time",
					}))
				})
			})

			Context("when there are multiple end parameters in query string", func() {
				BeforeEach(func() {
					req, err = http.NewRequest(http.MethodGet, testUrlScalingHistories+"?end=123&end=231", nil)
					Expect(err).ToNot(HaveOccurred())
				})

				It("returns 400", func() {
					Expect(resp.Code).To(Equal(http.StatusBadRequest))

					errJson := &models.ErrorResponse{}
					err = json.Unmarshal(resp.Body.Bytes(), errJson)

					Expect(err).ToNot(HaveOccurred())
					Expect(errJson).To(Equal(&models.ErrorResponse{
						Code:    "Bad-Request",
						Message: "Incorrect end parameter in query string",
					}))
				})
			})

			Context("when end time is not a number", func() {
				BeforeEach(func() {
					req, err = http.NewRequest(http.MethodGet, testUrlScalingHistories+"?end=abc", nil)
					Expect(err).ToNot(HaveOccurred())
				})

				It("returns 400", func() {
					Expect(resp.Code).To(Equal(http.StatusBadRequest))

					errJson := &models.ErrorResponse{}
					err = json.Unmarshal(resp.Body.Bytes(), errJson)

					Expect(err).ToNot(HaveOccurred())
					Expect(errJson).To(Equal(&models.ErrorResponse{
						Code:    "Bad-Request",
						Message: "Error parsing end time",
					}))
				})
			})

			Context("when there are multiple order pararmeters in query string", func() {
				BeforeEach(func() {
					req, err = http.NewRequest(http.MethodGet, testUrlScalingHistories+"?order=asc&order=asc", nil)
					Expect(err).ToNot(HaveOccurred())
				})

				It("returns 400", func() {
					Expect(resp.Code).To(Equal(http.StatusBadRequest))

					errJson := &models.ErrorResponse{}
					err = json.Unmarshal(resp.Body.Bytes(), errJson)

					Expect(err).ToNot(HaveOccurred())
					Expect(errJson).To(Equal(&models.ErrorResponse{
						Code:    "Bad-Request",
						Message: "Incorrect order parameter in query string",
					}))
				})
			})

			Context("when order value is invalid", func() {
				BeforeEach(func() {
					req, err = http.NewRequest(http.MethodGet, testUrlScalingHistories+"?order=invalid-order", nil)
					Expect(err).ToNot(HaveOccurred())
				})

				It("returns 400", func() {
					Expect(resp.Code).To(Equal(http.StatusBadRequest))

					errJson := &models.ErrorResponse{}
					err = json.Unmarshal(resp.Body.Bytes(), errJson)

					Expect(err).ToNot(HaveOccurred())
					Expect(errJson).To(Equal(&models.ErrorResponse{
						Code:    "Bad-Request",
						Message: "Incorrect order parameter in query string, the value can only be 'ASC' or 'DESC'",
					}))
				})
			})

			Context("when there are multiple include pararmeters in query string", func() {
				BeforeEach(func() {
					req, err = http.NewRequest(http.MethodGet, testUrlScalingHistories+"?include=all&include=all", nil)
					Expect(err).ToNot(HaveOccurred())
				})

				It("returns 400", func() {
					Expect(resp.Code).To(Equal(http.StatusBadRequest))

					errJson := &models.ErrorResponse{}
					err = json.Unmarshal(resp.Body.Bytes(), errJson)

					Expect(err).ToNot(HaveOccurred())
					Expect(errJson).To(Equal(&models.ErrorResponse{
						Code:    "Bad-Request",
						Message: "Incorrect include parameter in query string",
					}))
				})
			})

			Context("when include value is invalid", func() {
				BeforeEach(func() {
					req, err = http.NewRequest(http.MethodGet, testUrlScalingHistories+"?include=invalid-include-value", nil)
					Expect(err).ToNot(HaveOccurred())
				})

				It("returns 400", func() {
					Expect(resp.Code).To(Equal(http.StatusBadRequest))

					errJson := &models.ErrorResponse{}
					err = json.Unmarshal(resp.Body.Bytes(), errJson)

					Expect(err).ToNot(HaveOccurred())
					Expect(errJson).To(Equal(&models.ErrorResponse{
						Code:    "Bad-Request",
						Message: "Incorrect include parameter in query string, the value can only be 'all'",
					}))
				})
			})
		})

		Context("when request query string is valid", func() {
			Context("when start, end, order and include parameter are all in query string", func() {
				BeforeEach(func() {
					req, err = http.NewRequest(http.MethodGet, testUrlScalingHistories+"?start=123&end=567&order=desc&include=all", nil)
					Expect(err).ToNot(HaveOccurred())
				})

				It("retrieves scaling histories from database with the given start and end time and order ", func() {
					appid, start, end, order, includeAll := scalingEngineDB.RetrieveScalingHistoriesArgsForCall(0)
					Expect(appid).To(Equal("an-app-id"))
					Expect(start).To(Equal(int64(123)))
					Expect(end).To(Equal(int64(567)))
					Expect(order).To(Equal(db.DESC))
					Expect(includeAll).To(BeTrue())
				})
			})

			Context("when there is no start time parameter in query string", func() {
				BeforeEach(func() {
					req, err = http.NewRequest(http.MethodGet, testUrlScalingHistories+"?end=123&order=desc", nil)
					Expect(err).ToNot(HaveOccurred())
				})

				It("queries scaling histories from database with start time  0", func() {
					_, start, _, _, _ := scalingEngineDB.RetrieveScalingHistoriesArgsForCall(0)
					Expect(start).To(Equal(int64(0)))
				})
			})

			Context("when there is no end time patameter in query string", func() {
				BeforeEach(func() {
					req, err = http.NewRequest(http.MethodGet, testUrlScalingHistories+"?start=123&order=desc", nil)
					Expect(err).ToNot(HaveOccurred())
				})

				It("queries scaling histories from database with end time -1 ", func() {
					_, _, end, _, _ := scalingEngineDB.RetrieveScalingHistoriesArgsForCall(0)
					Expect(end).To(Equal(int64(-1)))
				})
			})

			Context("when there is no order parameter in query string", func() {
				BeforeEach(func() {
					req, err = http.NewRequest(http.MethodGet, testUrlScalingHistories+"?start=123&end=567", nil)
					Expect(err).ToNot(HaveOccurred())
				})

				It("queries scaling histories from database with order desc", func() {
					_, _, _, order, _ := scalingEngineDB.RetrieveScalingHistoriesArgsForCall(0)
					Expect(order).To(Equal(db.DESC))
				})
			})

			Context("when there is no include parameter in query string", func() {
				BeforeEach(func() {
					req, err = http.NewRequest(http.MethodGet, testUrlScalingHistories+"?start=123&end=567", nil)
					Expect(err).ToNot(HaveOccurred())
				})

				It("queries all scaling histories from database", func() {
					_, _, _, _, includeAll := scalingEngineDB.RetrieveScalingHistoriesArgsForCall(0)
					Expect(includeAll).To(BeFalse())
				})
			})

			Context("when query database succeeds", func() {
				BeforeEach(func() {
					req, err = http.NewRequest(http.MethodGet, testUrlScalingHistories+"?start=123&end=567&order=desc&include=all", nil)
					Expect(err).ToNot(HaveOccurred())

					history1 = &models.AppScalingHistory{
						AppId:        "an-app-id",
						Timestamp:    222,
						ScalingType:  models.ScalingTypeDynamic,
						Status:       models.ScalingStatusSucceeded,
						OldInstances: 2,
						NewInstances: 4,
						Reason:       "a reason",
					}

					history2 = &models.AppScalingHistory{
						AppId:        "an-app-id",
						Timestamp:    333,
						ScalingType:  models.ScalingTypeSchedule,
						Status:       models.ScalingStatusFailed,
						OldInstances: 2,
						NewInstances: 4,
						Reason:       "a reason",
						Message:      "a message",
						Error:        "an error",
					}

					history3 = &models.AppScalingHistory{
						AppId:        "an-app-id",
						Timestamp:    444,
						ScalingType:  models.ScalingTypeDynamic,
						Status:       models.ScalingStatusIgnored,
						OldInstances: 2,
						NewInstances: 4,
						Reason:       "a reason",
						Message:      "a message",
					}

					scalingEngineDB.RetrieveScalingHistoriesReturns([]*models.AppScalingHistory{history3, history2, history1}, nil)
				})

				It("returns 200 with scaling histories in message body", func() {
					Expect(resp.Code).To(Equal(http.StatusOK))

					histories := &[]models.AppScalingHistory{}
					err = json.Unmarshal(resp.Body.Bytes(), histories)

					Expect(err).ToNot(HaveOccurred())
					Expect(*histories).To(Equal([]models.AppScalingHistory{*history3, *history2, *history1}))
				})
			})

			Context("when query database fails", func() {
				BeforeEach(func() {
					req, err = http.NewRequest(http.MethodGet, testUrlScalingHistories+"?start=123&end=567&order=desc", nil)
					Expect(err).ToNot(HaveOccurred())
					scalingEngineDB.RetrieveScalingHistoriesReturns(nil, errors.New("database error"))
				})

				It("returns 500", func() {
					Expect(resp.Code).To(Equal(http.StatusInternalServerError))

					errJson := &models.ErrorResponse{}
					err = json.Unmarshal(resp.Body.Bytes(), errJson)

					Expect(err).ToNot(HaveOccurred())
					Expect(errJson).To(Equal(&models.ErrorResponse{
						Code:    "Internal-Server-Error",
						Message: "Error getting scaling histories from database",
					}))
				})
			})
		})
	})

	Describe("StartActiveSchedule", func() {
		JustBeforeEach(func() {
			req, err = http.NewRequest(http.MethodPut, testUrlActiveSchedules, bytes.NewReader(body))
			Expect(err).ToNot(HaveOccurred())

			handler.StartActiveSchedule(resp, req, map[string]string{"appid": "an-app-id", "schduleid": "a-schdule-id"})
		})

		Context("when active schedule is valid", func() {
			BeforeEach(func() {
				body = []byte(`{"instance_min_count":1, "instance_max_count":5, "initial_min_instance_count":3}`)
			})

			It("returns 200", func() {
				Expect(resp.Code).To(Equal(http.StatusOK))
			})
		})

		Context("when active schedule is invalid", func() {
			BeforeEach(func() {
				body = []byte(`{"instance_min_count":"a"}`)
			})

			It("returns 400", func() {
				Expect(resp.Code).To(Equal(http.StatusBadRequest))

				errJson := &models.ErrorResponse{}
				err = json.Unmarshal(resp.Body.Bytes(), errJson)
				Expect(err).ToNot(HaveOccurred())
				Expect(errJson).To(Equal(&models.ErrorResponse{
					Code:    "Bad-Request",
					Message: "Incorrect active schedule in request body",
				}))
			})
		})

		Context("when setting active schedule fails", func() {
			BeforeEach(func() {
				body = []byte(`{"instance_min_count":1, "instance_max_count":5, "initial_min_instance_count":3}`)
				scalingEngine.SetActiveScheduleReturns(errors.New("an error"))
			})

			It("returns 500", func() {
				Expect(resp.Code).To(Equal(http.StatusInternalServerError))

				errJson := &models.ErrorResponse{}
				err = json.Unmarshal(resp.Body.Bytes(), errJson)
				Expect(err).ToNot(HaveOccurred())
				Expect(errJson).To(Equal(&models.ErrorResponse{
					Code:    "Internal-Server-Error",
					Message: "Error setting active schedule",
				}))
			})
		})
	})

	Describe("RemoveActiveSchedule", func() {
		JustBeforeEach(func() {
			req, err = http.NewRequest(http.MethodDelete, testUrlActiveSchedules, nil)
			Expect(err).ToNot(HaveOccurred())

			handler.RemoveActiveSchedule(resp, req, map[string]string{"appid": "an-app-id", "schduleid": "a-schdule-id"})
		})

		Context("when removing active schedule succeeds", func() {
			It("returns 200", func() {
				Expect(resp.Code).To(Equal(http.StatusOK))
			})
		})

		Context("when removing active schedule fails", func() {
			BeforeEach(func() {
				scalingEngine.RemoveActiveScheduleReturns(errors.New("an error"))
			})

			It("returns 500", func() {
				Expect(resp.Code).To(Equal(http.StatusInternalServerError))

				errJson := &models.ErrorResponse{}
				err = json.Unmarshal(resp.Body.Bytes(), errJson)
				Expect(err).ToNot(HaveOccurred())
				Expect(errJson).To(Equal(&models.ErrorResponse{
					Code:    "Internal-Server-Error",
					Message: "Error removing active schedule",
				}))
			})
		})
	})

	Describe("GetActiveSchedule", func() {
		JustBeforeEach(func() {
			handler.GetActiveSchedule(resp, req, map[string]string{"appid": "invalid-app-id"})
		})

		Context("when app id is invalid", func() {
			BeforeEach(func() {
				req, err = http.NewRequest(http.MethodGet, testUrlAppActiveSchedule, nil)
				Expect(err).ToNot(HaveOccurred())
				scalingEngineDB.GetActiveScheduleReturns(nil, nil)
			})

			It("returns 404", func() {
				Expect(resp.Code).To(Equal(http.StatusNotFound))

				errJson := &models.ErrorResponse{}
				err = json.Unmarshal(resp.Body.Bytes(), errJson)
				Expect(err).ToNot(HaveOccurred())
				Expect(errJson).To(Equal(&models.ErrorResponse{
					Code:    "Not-Found",
					Message: "Active schedule not found",
				}))
			})
		})

		Context("when query database succeeds", func() {
			BeforeEach(func() {
				req, err = http.NewRequest(http.MethodGet, testUrlAppActiveSchedule, nil)
				Expect(err).ToNot(HaveOccurred())

				activeSchedule = &models.ActiveSchedule{
					ScheduleId:         "a-schedule-id",
					InstanceMin:        1,
					InstanceMax:        5,
					InstanceMinInitial: 3,
				}

				scalingEngineDB.GetActiveScheduleReturns(activeSchedule, nil)
			})

			It("returns 200 with active schedule in message body", func() {
				Expect(resp.Code).To(Equal(http.StatusOK))

				actualActiveSchedule := &models.ActiveSchedule{}
				err = json.Unmarshal(resp.Body.Bytes(), actualActiveSchedule)

				Expect(err).ToNot(HaveOccurred())
				Expect(actualActiveSchedule).To(Equal(activeSchedule))
			})
		})

		Context("when query database fails", func() {
			BeforeEach(func() {
				scalingEngineDB.GetActiveScheduleReturns(nil, errors.New("database error"))
			})

			It("returns 500", func() {
				Expect(resp.Code).To(Equal(http.StatusInternalServerError))

				errJson := &models.ErrorResponse{}
				err = json.Unmarshal(resp.Body.Bytes(), errJson)

				Expect(err).ToNot(HaveOccurred())
				Expect(errJson).To(Equal(&models.ErrorResponse{
					Code:    "Internal-Server-Error",
					Message: "Error getting active schedule from database",
				}))
			})
		})
	})
})
