package server_test

import (
	"engine/fakes"
	. "engine/server"
	"models"

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
)

var _ = Describe("ScalingHandler", func() {
	var (
		cfc          *fakes.FakeCfClient
		database     *fakes.FakePolicyDB
		handler      *ScalingHandler
		resp         *httptest.ResponseRecorder
		req          *http.Request
		body         []byte
		err          error
		instanceMin  int
		instanceMax  int
		instances    int
		newInstances int
		adjustment   string
		trigger      *models.Trigger
		buffer       *gbytes.Buffer
	)

	BeforeEach(func() {
		cfc = &fakes.FakeCfClient{}

		logger := lagertest.NewTestLogger("scaling-handler-test")
		buffer = logger.Buffer()
		database = &fakes.FakePolicyDB{}
		resp = httptest.NewRecorder()
		handler = NewScalingHandler(logger, cfc, database)

	})

	Describe("ComputeNewInstances", func() {
		BeforeEach(func() {
			instances = 3
			instanceMin = 1
			instanceMax = 6
			adjustment = "+1"
		})

		JustBeforeEach(func() {
			newInstances, err = handler.ComputeNewInstances(instances, adjustment, instanceMin, instanceMax)
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

			Context("when adjustment is greater than instanceMax", func() {
				BeforeEach(func() {
					adjustment = "5"
				})
				It("returns instanceMax", func() {
					Expect(err).NotTo(HaveOccurred())
					Expect(newInstances).To(Equal(6))
				})
			})

			Context("when adjustment is less than instanceMin", func() {
				BeforeEach(func() {
					adjustment = "-90%"
				})
				It("returns instanceMin", func() {
					Expect(err).NotTo(HaveOccurred())
					Expect(newInstances).To(Equal(1))
				})
			})

		})

	})

	Describe("Scale", func() {
		BeforeEach(func() {
			trigger = &models.Trigger{
				MetricType: models.MetricNameMemory,
				Adjustment: "+1",
			}
		})
		JustBeforeEach(func() {
			newInstances, err = handler.Scale("an-app-id", trigger)
		})

		Context("when scaling succeeds", func() {
			BeforeEach(func() {
				database.GetAppPolicyReturns(&models.ScalingPolicy{InstanceMin: 1, InstanceMax: 6}, nil)
				cfc.GetAppInstancesReturns(2, nil)
			})

			It("should not error and set the new app instance number", func() {
				Expect(err).NotTo(HaveOccurred())
				id, num := cfc.SetAppInstancesArgsForCall(0)
				Expect(id).To(Equal("an-app-id"))
				Expect(num).To(Equal(3))
				Expect(newInstances).To(Equal(3))
			})
		})

		Context("when app instances not changed", func() {
			BeforeEach(func() {
				database.GetAppPolicyReturns(&models.ScalingPolicy{InstanceMin: 1, InstanceMax: 6}, nil)
				cfc.GetAppInstancesReturns(6, nil)
			})

			It("should not error and not update app", func() {
				Expect(err).NotTo(HaveOccurred())
				Expect(cfc.SetAppInstancesCallCount()).To(BeZero())
				Expect(newInstances).To(Equal(6))
			})
		})

		Context("when getting policy fails", func() {
			BeforeEach(func() {
				database.GetAppPolicyReturns(nil, errors.New("test error"))
			})

			It("should error", func() {
				Expect(err).To(HaveOccurred())
				Eventually(buffer).Should(gbytes.Say("scale-get-app-policy"))
				Eventually(buffer).Should(gbytes.Say("test error"))
			})
		})

		Context("when getting app instances from cloud controller fails", func() {
			BeforeEach(func() {
				cfc.GetAppInstancesReturns(-1, errors.New("test error"))
			})

			It("should error", func() {
				Expect(err).To(HaveOccurred())
				Eventually(buffer).Should(gbytes.Say("scale-get-app-instances"))
				Eventually(buffer).Should(gbytes.Say("test error"))
			})
		})

		Context("when computing new app instances fails", func() {
			BeforeEach(func() {
				trigger.Adjustment = "+a"
				database.GetAppPolicyReturns(&models.ScalingPolicy{}, nil)
			})

			It("should error", func() {
				Expect(err).To(HaveOccurred())
				Eventually(buffer).Should(gbytes.Say("scale-compute-new-instance"))
			})
		})

		Context("when set new instances fails", func() {
			BeforeEach(func() {
				database.GetAppPolicyReturns(&models.ScalingPolicy{InstanceMin: 1, InstanceMax: 6}, nil)
				cfc.GetAppInstancesReturns(2, nil)
				cfc.SetAppInstancesReturns(errors.New("test error"))
			})

			It("should error", func() {
				Expect(err).To(HaveOccurred())
				Eventually(buffer).Should(gbytes.Say("scale-set-app-instances"))
				Eventually(buffer).Should(gbytes.Say("test error"))
			})
		})

	})

	Describe("HandleScale", func() {
		JustBeforeEach(func() {
			handler.HandleScale(resp, req, map[string]string{"appid": "an-app-id"})
		})

		Context("when scaling app succeeds", func() {
			BeforeEach(func() {
				database.GetAppPolicyReturns(&models.ScalingPolicy{InstanceMin: 1, InstanceMax: 6}, nil)
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
				database.GetAppPolicyReturns(nil, errors.New("an error"))

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
})
