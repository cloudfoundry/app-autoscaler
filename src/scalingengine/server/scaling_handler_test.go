package server_test

import (
	"models"
	"scalingengine/fakes"
	. "scalingengine/server"

	"code.cloudfoundry.org/clock"
	"code.cloudfoundry.org/clock/fakeclock"

	"code.cloudfoundry.org/lager/lagertest"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/gbytes"

	"bytes"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strconv"
	"time"
)

var testUrlScalingHistories = "http://localhost/v1/apps/an-app-id/scaling_histories"

var _ = Describe("ScalingHandler", func() {
	var (
		cfc                *fakes.FakeCfClient
		policyDB           *fakes.FakePolicyDB
		historyDB          *fakes.FakeHistoryDB
		hClock             clock.Clock
		handler            *ScalingHandler
		resp               *httptest.ResponseRecorder
		req                *http.Request
		body               []byte
		err                error
		instances          int
		newInstances       int
		adjustment         string
		trigger            *models.Trigger
		buffer             *gbytes.Buffer
		history1, history2 *models.AppScalingHistory
	)

	BeforeEach(func() {
		cfc = &fakes.FakeCfClient{}
		logger := lagertest.NewTestLogger("scaling-handler-test")
		buffer = logger.Buffer()
		policyDB = &fakes.FakePolicyDB{}
		historyDB = &fakes.FakeHistoryDB{}
		hClock = fakeclock.NewFakeClock(time.Now())
		handler = NewScalingHandler(logger, cfc, policyDB, historyDB, hClock)
		resp = httptest.NewRecorder()
	})

	Describe("ComputeNewInstances", func() {
		BeforeEach(func() {
			instances = 3
		})

		JustBeforeEach(func() {
			newInstances, err = handler.ComputeNewInstances(instances, adjustment)
		})

		Context("when adjustment is not valid", func() {
			Context("when adjustment is not a valid percentage", func() {
				BeforeEach(func() {
					adjustment = "10.5a%"
				})

				It("should error", func() {
					Expect(err).To(BeAssignableToTypeOf(&strconv.NumError{}))
				})
			})
			Context("when adjustment is not a valid step", func() {
				BeforeEach(func() {
					adjustment = "#1"
				})

				It("should error", func() {
					Expect(err).To(BeAssignableToTypeOf(&strconv.NumError{}))
				})
			})
		})

		Context("when adjustment is valid", func() {
			Context("when adjustment is by percentage", func() {
				BeforeEach(func() {
					adjustment = "50%"
				})
				It("returns correct new instance number", func() {
					Expect(err).NotTo(HaveOccurred())
					Expect(newInstances).To(Equal(5))
				})
			})

			Context("when adjustment is by step", func() {
				BeforeEach(func() {
					adjustment = "-2"
				})
				It("returns correct new instance number", func() {
					Expect(err).NotTo(HaveOccurred())
					Expect(newInstances).To(Equal(1))
				})
			})
		})
	})

	Describe("Scale", func() {
		BeforeEach(func() {
			trigger = &models.Trigger{
				MetricType:            models.MetricNameMemory,
				BreachDurationSeconds: 100,
				Threshold:             222222,
				Operator:              ">",
				Adjustment:            "+1",
			}
		})
		JustBeforeEach(func() {
			newInstances, err = handler.Scale("an-app-id", trigger)
		})

		Context("when scaling succeeds", func() {
			BeforeEach(func() {
				policyDB.GetAppPolicyReturns(&models.ScalingPolicy{InstanceMin: 1, InstanceMax: 6}, nil)
				cfc.GetAppInstancesReturns(2, nil)
			})

			It("sets the new app instance number and stores the succeeded scaling history", func() {
				Expect(err).NotTo(HaveOccurred())
				id, num := cfc.SetAppInstancesArgsForCall(0)
				Expect(id).To(Equal("an-app-id"))
				Expect(num).To(Equal(3))
				Expect(newInstances).To(Equal(3))

				Expect(historyDB.SaveScalingHistoryArgsForCall(0)).To(Equal(&models.AppScalingHistory{
					AppId:        "an-app-id",
					Timestamp:    hClock.Now().UnixNano(),
					ScalingType:  models.ScalingTypeDynamic,
					Status:       models.ScalingStatusSucceeded,
					OldInstances: 2,
					NewInstances: 3,
					Reason:       "+1 instance(s) because memorybytes > 222222 for 100 seconds",
				}))

			})
		})

		Context("when app instances not changed", func() {
			BeforeEach(func() {
				trigger.Adjustment = "+20%"
				policyDB.GetAppPolicyReturns(&models.ScalingPolicy{InstanceMin: 1, InstanceMax: 6}, nil)
				cfc.GetAppInstancesReturns(2, nil)
			})

			It("does not update the app and stores the ignored scaling history", func() {
				Expect(err).NotTo(HaveOccurred())
				Expect(cfc.SetAppInstancesCallCount()).To(BeZero())
				Expect(newInstances).To(Equal(2))

				Expect(historyDB.SaveScalingHistoryArgsForCall(0)).To(Equal(&models.AppScalingHistory{
					AppId:        "an-app-id",
					Timestamp:    hClock.Now().UnixNano(),
					ScalingType:  models.ScalingTypeDynamic,
					Status:       models.ScalingStatusIgnored,
					OldInstances: 2,
					NewInstances: 2,
					Reason:       "+20% instance(s) because memorybytes > 222222 for 100 seconds",
				}))

			})
		})

		Context("when it exceeds max instances limit", func() {
			BeforeEach(func() {
				trigger.Adjustment = "+2"
				policyDB.GetAppPolicyReturns(&models.ScalingPolicy{InstanceMin: 1, InstanceMax: 6}, nil)
				cfc.GetAppInstancesReturns(5, nil)
			})

			It("does upate the app instance with max instances and stores the succeeded scaling history", func() {
				Expect(err).NotTo(HaveOccurred())

				id, num := cfc.SetAppInstancesArgsForCall(0)
				Expect(id).To(Equal("an-app-id"))
				Expect(num).To(Equal(6))
				Expect(newInstances).To(Equal(6))

				Expect(historyDB.SaveScalingHistoryArgsForCall(0)).To(Equal(&models.AppScalingHistory{
					AppId:        "an-app-id",
					Timestamp:    hClock.Now().UnixNano(),
					ScalingType:  models.ScalingTypeDynamic,
					Status:       models.ScalingStatusSucceeded,
					OldInstances: 5,
					NewInstances: 6,
					Reason:       "+2 instance(s) because memorybytes > 222222 for 100 seconds",
					Message:      "limit to max instances 6",
				}))

			})
		})

		Context("when it exceeds min instances limit", func() {
			BeforeEach(func() {
				trigger.Adjustment = "-60%"
				policyDB.GetAppPolicyReturns(&models.ScalingPolicy{InstanceMin: 2, InstanceMax: 6}, nil)
				cfc.GetAppInstancesReturns(3, nil)
			})

			It("does upate the app instance with max instances and stores the succeeded scaling history", func() {
				Expect(err).NotTo(HaveOccurred())

				id, num := cfc.SetAppInstancesArgsForCall(0)
				Expect(id).To(Equal("an-app-id"))
				Expect(num).To(Equal(2))
				Expect(newInstances).To(Equal(2))

				Expect(historyDB.SaveScalingHistoryArgsForCall(0)).To(Equal(&models.AppScalingHistory{
					AppId:        "an-app-id",
					Timestamp:    hClock.Now().UnixNano(),
					ScalingType:  models.ScalingTypeDynamic,
					Status:       models.ScalingStatusSucceeded,
					OldInstances: 3,
					NewInstances: 2,
					Reason:       "-60% instance(s) because memorybytes > 222222 for 100 seconds",
					Message:      "limit to min instances 2",
				}))

			})
		})

		Context("when getting policy fails", func() {
			BeforeEach(func() {
				policyDB.GetAppPolicyReturns(nil, errors.New("test error"))
			})

			It("should error and store the failed scaling history", func() {
				Expect(err).To(HaveOccurred())
				Eventually(buffer).Should(gbytes.Say("scale-get-app-policy"))
				Eventually(buffer).Should(gbytes.Say("test error"))

				Expect(historyDB.SaveScalingHistoryArgsForCall(0)).To(Equal(&models.AppScalingHistory{
					AppId:        "an-app-id",
					Timestamp:    hClock.Now().UnixNano(),
					ScalingType:  models.ScalingTypeDynamic,
					Status:       models.ScalingStatusFailed,
					OldInstances: -1,
					NewInstances: -1,
					Reason:       "+1 instance(s) because memorybytes > 222222 for 100 seconds",
					Error:        "failed to get scaling policy",
				}))

			})
		})

		Context("when getting app instances from cloud controller fails", func() {
			BeforeEach(func() {
				cfc.GetAppInstancesReturns(-1, errors.New("test error"))
			})

			It("should error and store the failed scaling history", func() {
				Expect(err).To(HaveOccurred())
				Eventually(buffer).Should(gbytes.Say("scale-get-app-instances"))
				Eventually(buffer).Should(gbytes.Say("test error"))

				Expect(historyDB.SaveScalingHistoryArgsForCall(0)).To(Equal(&models.AppScalingHistory{
					AppId:        "an-app-id",
					Timestamp:    hClock.Now().UnixNano(),
					ScalingType:  models.ScalingTypeDynamic,
					Status:       models.ScalingStatusFailed,
					OldInstances: -1,
					NewInstances: -1,
					Reason:       "+1 instance(s) because memorybytes > 222222 for 100 seconds",
					Error:        "failed to get app instances",
				}))

			})
		})

		Context("when computing new app instances fails", func() {
			BeforeEach(func() {
				trigger.Adjustment = "+a"
				policyDB.GetAppPolicyReturns(&models.ScalingPolicy{}, nil)
				cfc.GetAppInstancesReturns(2, nil)
			})

			It("should error and store failed scaling history", func() {
				Expect(err).To(HaveOccurred())
				Eventually(buffer).Should(gbytes.Say("scale-compute-new-instance"))

				Expect(historyDB.SaveScalingHistoryArgsForCall(0)).To(Equal(&models.AppScalingHistory{
					AppId:        "an-app-id",
					Timestamp:    hClock.Now().UnixNano(),
					ScalingType:  models.ScalingTypeDynamic,
					Status:       models.ScalingStatusFailed,
					OldInstances: 2,
					NewInstances: -1,
					Reason:       "+a instance(s) because memorybytes > 222222 for 100 seconds",
					Error:        "failed to compute new app instances",
				}))
			})
		})

		Context("when set new instances fails", func() {
			BeforeEach(func() {
				policyDB.GetAppPolicyReturns(&models.ScalingPolicy{InstanceMin: 1, InstanceMax: 6}, nil)
				cfc.GetAppInstancesReturns(2, nil)
				cfc.SetAppInstancesReturns(errors.New("test error"))
			})

			It("should error and store failed scaling history", func() {
				Expect(err).To(HaveOccurred())
				Eventually(buffer).Should(gbytes.Say("scale-set-app-instances"))
				Eventually(buffer).Should(gbytes.Say("test error"))

				Expect(historyDB.SaveScalingHistoryArgsForCall(0)).To(Equal(&models.AppScalingHistory{
					AppId:        "an-app-id",
					Timestamp:    hClock.Now().UnixNano(),
					ScalingType:  models.ScalingTypeDynamic,
					Status:       models.ScalingStatusFailed,
					OldInstances: 2,
					NewInstances: 3,
					Reason:       "+1 instance(s) because memorybytes > 222222 for 100 seconds",
					Error:        "failed to set app instances",
				}))

			})
		})

	})

	Describe("HandleScale", func() {
		JustBeforeEach(func() {
			handler.HandleScale(resp, req, map[string]string{"appid": "an-app-id"})
		})

		Context("when scaling app succeeds", func() {
			BeforeEach(func() {
				policyDB.GetAppPolicyReturns(&models.ScalingPolicy{InstanceMin: 1, InstanceMax: 6}, nil)
				cfc.GetAppInstancesReturns(2, nil)

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
				policyDB.GetAppPolicyReturns(nil, errors.New("an error"))

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
					appid, start, end := historyDB.RetrieveScalingHistoriesArgsForCall(0)
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
					_, start, _ := historyDB.RetrieveScalingHistoriesArgsForCall(0)
					Expect(start).To(Equal(int64(0)))
				})

			})

			Context("when there is no end time in query string", func() {
				BeforeEach(func() {
					req, err = http.NewRequest(http.MethodGet, testUrlScalingHistories+"?start=123", nil)
					Expect(err).ToNot(HaveOccurred())
				})

				It("queries metrics from database with end time -1 ", func() {
					_, _, end := historyDB.RetrieveScalingHistoriesArgsForCall(0)
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

					historyDB.RetrieveScalingHistoriesReturns([]*models.AppScalingHistory{history1, history2}, nil)
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
					historyDB.RetrieveScalingHistoriesReturns(nil, errors.New("database error"))
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
})
