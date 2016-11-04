package server_test

import (
	"autoscaler/models"
	"autoscaler/scalingengine"
	"autoscaler/scalingengine/fakes"
	. "autoscaler/scalingengine/server"

	"code.cloudfoundry.org/lager/lagertest"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/gbytes"

	"bytes"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
)

const testUrlScalingHistories = "http://localhost/v1/apps/an-app-id/scaling_histories"
const testUrlActiveSchedules = "http://localhost/v1/apps/an-app-id/active_schedules/a-schedule-id"

var _ = Describe("ScalingHandler", func() {
	var (
		scalingEngineDB    *fakes.FakeScalingEngineDB
		scalingEngine      *fakes.FakeScalingEngine
		handler            *ScalingHandler
		resp               *httptest.ResponseRecorder
		req                *http.Request
		body               []byte
		err                error
		trigger            *models.Trigger
		buffer             *gbytes.Buffer
		history1, history2 *models.AppScalingHistory
	)

	BeforeEach(func() {
		logger := lagertest.NewTestLogger("scaling-handler-test")
		buffer = logger.Buffer()
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
				scalingEngine.ScaleReturns(3, nil)

				trigger = &models.Trigger{
					MetricType: models.MetricNameMemory,
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

				props := &models.AppEntity{}
				err = json.Unmarshal(resp.Body.Bytes(), props)
				Expect(err).NotTo(HaveOccurred())
				Expect(props.Instances).To(Equal(3))
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
				scalingEngine.ScaleReturns(0, errors.New("an error"))

				trigger = &models.Trigger{
					MetricType: models.MetricNameMemory,
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
		})

		Context("when request query string is valid", func() {
			Context("when there are both start and end time in query string", func() {
				BeforeEach(func() {
					req, err = http.NewRequest(http.MethodGet, testUrlScalingHistories+"?start=123&end=567", nil)
					Expect(err).ToNot(HaveOccurred())
				})

				It("retrieves scaling histories from database with the given start and end time ", func() {
					appid, start, end := scalingEngineDB.RetrieveScalingHistoriesArgsForCall(0)
					Expect(appid).To(Equal("an-app-id"))
					Expect(start).To(Equal(int64(123)))
					Expect(end).To(Equal(int64(567)))
				})
			})

			Context("when there is no start time in query string", func() {
				BeforeEach(func() {
					req, err = http.NewRequest(http.MethodGet, testUrlScalingHistories+"?end=123", nil)
					Expect(err).ToNot(HaveOccurred())
				})

				It("queries metrics from database with start time  0", func() {
					_, start, _ := scalingEngineDB.RetrieveScalingHistoriesArgsForCall(0)
					Expect(start).To(Equal(int64(0)))
				})
			})

			Context("when there is no end time in query string", func() {
				BeforeEach(func() {
					req, err = http.NewRequest(http.MethodGet, testUrlScalingHistories+"?start=123", nil)
					Expect(err).ToNot(HaveOccurred())
				})

				It("queries metrics from database with end time -1 ", func() {
					_, _, end := scalingEngineDB.RetrieveScalingHistoriesArgsForCall(0)
					Expect(end).To(Equal(int64(-1)))
				})
			})

			Context("when query database succeeds", func() {
				BeforeEach(func() {
					req, err = http.NewRequest(http.MethodGet, testUrlScalingHistories+"?start=123&end=567", nil)
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

					scalingEngineDB.RetrieveScalingHistoriesReturns([]*models.AppScalingHistory{history1, history2}, nil)
				})

				It("returns 200 with scaling histories in message body", func() {
					Expect(resp.Code).To(Equal(http.StatusOK))

					histories := &[]models.AppScalingHistory{}
					err = json.Unmarshal(resp.Body.Bytes(), histories)

					Expect(err).ToNot(HaveOccurred())
					Expect(*histories).To(Equal([]models.AppScalingHistory{*history1, *history2}))
				})
			})

			Context("when query database fails", func() {
				BeforeEach(func() {
					req, err = http.NewRequest(http.MethodGet, testUrlScalingHistories+"?start=123&end=567", nil)
					Expect(err).ToNot(HaveOccurred())
					scalingEngineDB.RetrieveScalingHistoriesReturns(nil, errors.New("database error"))
				})

				It("returns 500", func() {
					Expect(resp.Code).To(Equal(http.StatusInternalServerError))

					errJson := &models.ErrorResponse{}
					err = json.Unmarshal(resp.Body.Bytes(), errJson)

					Expect(err).ToNot(HaveOccurred())
					Expect(errJson).To(Equal(&models.ErrorResponse{
						Code:    "Interal-Server-Error",
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
					Code:    "Interal-Server-Error",
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
			It("returns 204", func() {
				Expect(resp.Code).To(Equal(http.StatusNoContent))
			})
		})

		Context("when active schedule is not found", func() {
			BeforeEach(func() {
				scalingEngine.RemoveActiveScheduleReturns(&scalingengine.ActiveScheduleNotFoundError{})
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
					Code:    "Interal-Server-Error",
					Message: "Error removing active schedule",
				}))
			})
		})
	})
})
