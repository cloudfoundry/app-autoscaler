package server_test

import (
	"fmt"

	"code.cloudfoundry.org/app-autoscaler/src/autoscaler/cf"
	"code.cloudfoundry.org/app-autoscaler/src/autoscaler/fakes"
	"code.cloudfoundry.org/app-autoscaler/src/autoscaler/models"
	. "code.cloudfoundry.org/app-autoscaler/src/autoscaler/scalingengine/server"
	"code.cloudfoundry.org/lager/v3/lagertest"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"bytes"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
)

const testUrlActiveSchedules = "http://localhost/v1/apps/an-app-id/active_schedules/a-schedule-id"
const testUrlAppActiveSchedule = "http://localhost/v1/apps/an-app-id/active_schedules"

var _ = Describe("ScalingHandler", func() {
	var (
		scalingEngineDB *fakes.FakeScalingEngineDB
		scalingEngine   *fakes.FakeScalingEngine
		handler         *ScalingHandler
		resp            *httptest.ResponseRecorder
		req             *http.Request
		body            []byte
		err             error
		trigger         *models.Trigger
		activeSchedule  *models.ActiveSchedule
		testMetricName  = "Test-Metric-Name"
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

		Context("when an internal cf-call fails", func() {
			BeforeEach(func() {
				cfAPIError := cf.CfError{
					Errors: cf.ErrorItems([]cf.CfErrorItem{{
						Code:   http.StatusNotFound,
						Title:  "Some title",
						Detail: "Something went wrong.",
					}}),
					StatusCode: http.StatusNotFound, ResourceId: "unknown resource", Url: "https://some.url",
				}
				cfAPIErrorJson, err := json.Marshal(cfAPIError)
				Expect(err).NotTo(HaveOccurred()) // Test implementation wrong: Object not json-serializable!"
				requestError := cf.NewCfError(
					"A URL for an cloud-controller", "resourceID", cfAPIError.StatusCode, cfAPIErrorJson)
				clientError := fmt.Errorf("Error doing a request: %w", requestError)

				scalingEngine.ScaleReturns(&models.AppScalingResult{
					AppId:             "an-app-id",
					Status:            models.ScalingStatusFailed,
					Adjustment:        0,
					CooldownExpiredAt: 0,
				}, clientError)

				trigger = &models.Trigger{
					MetricType: testMetricName,
					Adjustment: "+1",
				}
				body, err = json.Marshal(trigger)
				Expect(err).NotTo(HaveOccurred())

				req, err = http.NewRequest("POST", "", bytes.NewReader(body))
				Expect(err).NotTo(HaveOccurred())
			})

			It("returns the status code of the cloud api", func() {
				Expect(resp.Code).To(Equal(http.StatusNotFound))

				errJson := &models.ErrorResponse{}
				err = json.Unmarshal(resp.Body.Bytes(), errJson)

				Expect(err).ToNot(HaveOccurred())
				Expect(errJson.Code).To(Equal("Error on request to the cloud-controller via a cf-client"))
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
