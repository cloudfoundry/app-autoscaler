package app_test

import (
	"net/http"
	"time"

	"code.cloudfoundry.org/app-autoscaler-release/src/acceptance/assets/app/go_app/internal/app/appfakes"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("Responsetime tests", func() {

	Context("Responsetime tests", func() {
		fakeTimeWaster := &appfakes.FakeTimeWaster{}
		It("should err if delayInMS not an int64", func() {
			apiTest(fakeTimeWaster, nil, nil, nil).
				Get("/responsetime/slow/yes").
				Expect(GinkgoT()).
				Status(http.StatusBadRequest).
				Body(`{"error":{"description":"invalid milliseconds: strconv.ParseInt: parsing \"yes\": invalid syntax"}}`).
				End()
		})
		It("should err if memory out of bounds", func() {
			apiTest(fakeTimeWaster, nil, nil, nil).
				Get("/responsetime/slow/100001010101010249032897287298719874687936483275648273632429479827398798271").
				Expect(GinkgoT()).
				Status(http.StatusBadRequest).
				Body(`{"error":{"description":"invalid milliseconds: strconv.ParseInt: parsing \"100001010101010249032897287298719874687936483275648273632429479827398798271\": value out of range"}}`).
				End()
		})

		It("should return ok and sleep correctDuration", func() {
			apiTest(fakeTimeWaster, nil, nil, nil).
				Get("/responsetime/slow/4000").
				Expect(GinkgoT()).
				Status(http.StatusOK).
				Body(`{"duration":"4s"}`).
				End()
			Expect(fakeTimeWaster.SleepCallCount()).To(Equal(1))
			Expect(fakeTimeWaster.SleepArgsForCall(0)).Should(Equal(4000 * time.Millisecond))
		})
	})
})
