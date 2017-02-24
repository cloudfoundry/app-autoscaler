package syncer_test

import (
	efakes "autoscaler/eventgenerator/aggregator/fakes"
	sfakes "autoscaler/scalingengine/fakes"
	"autoscaler/syncer"
	"errors"

	"code.cloudfoundry.org/lager/lagertest"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/gbytes"
)

var _ = Describe("Activescheduledb synchronize", func() {

	var (
		policyDb             *efakes.FakePolicyDB
		schedulerDb          *sfakes.FakeSchedulerDB
		buffer               *gbytes.Buffer
		activeScheduleSyncer *syncer.ActiveScheduleSyncer
		err                  error
	)
	BeforeEach(func() {

		logger := lagertest.NewTestLogger("ActiveScheduleSyncer-test")
		buffer = logger.Buffer()

		policyDb = &efakes.FakePolicyDB{}
		schedulerDb = &sfakes.FakeSchedulerDB{}
		activeScheduleSyncer = syncer.NewActiveScheduleSyncer(policyDb, schedulerDb, logger)

	})
	Describe("Synchronize", func() {
		JustBeforeEach(func() {
			err = activeScheduleSyncer.Synchronize()
		})
		Context("When failed to get app ids from policy db", func() {
			BeforeEach(func() {
				policyDb.GetAppIdsReturns(nil, errors.New("get-policy-error"))
			})
			It("should not do synchronization", func() {
				Expect(policyDb.GetAppIdsCallCount()).To(Equal(1))
				Expect(schedulerDb.SynchronizeActiveSchedulesCallCount()).To(Equal(0))
				Eventually(buffer).Should(gbytes.Say("get-policy-error"))
				Expect(err).To(HaveOccurred())
			})
		})
		Context("When there is no data in policy db", func() {
			BeforeEach(func() {
				policyDb.GetAppIdsReturns(map[string]bool{}, nil)
			})
			It("should not do synchronization", func() {
				Expect(policyDb.GetAppIdsCallCount()).To(Equal(1))
				Expect(schedulerDb.SynchronizeActiveSchedulesCallCount()).To(Equal(0))
				Eventually(buffer).Should(gbytes.Say("No application found in policy database"))
				Expect(err).NotTo(HaveOccurred())
			})
		})
		Context("When synchronization fails", func() {
			BeforeEach(func() {
				policyDb.GetAppIdsReturns(map[string]bool{"appId1": true, "appId2": true}, nil)
				schedulerDb.SynchronizeActiveSchedulesReturns(errors.New("delete-active-schedule-error"))
			})
			It("should error", func() {
				Expect(policyDb.GetAppIdsCallCount()).To(Equal(1))
				Expect(schedulerDb.SynchronizeActiveSchedulesCallCount()).To(Equal(1))
				Eventually(buffer).Should(gbytes.Say("delete-active-schedule-error"))
				Expect(err).To(HaveOccurred())
			})
		})
		Context("When synchronization succeeds", func() {
			BeforeEach(func() {
				policyDb.GetAppIdsReturns(map[string]bool{"appId1": true, "appId2": true}, nil)
				schedulerDb.SynchronizeActiveSchedulesReturns(nil)
			})
			It("should synchronize", func() {
				Expect(policyDb.GetAppIdsCallCount()).To(Equal(1))
				Expect(schedulerDb.SynchronizeActiveSchedulesCallCount()).To(Equal(1))
				Eventually(buffer).Should(gbytes.Say("Synchronize active schedules with policies successfully"))
				Expect(err).NotTo(HaveOccurred())
			})
		})
	})

})
