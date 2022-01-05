package metricsgateway_test

import (
	"errors"
	"time"

	"code.cloudfoundry.org/app-autoscaler/src/autoscaler/fakes"
	. "code.cloudfoundry.org/app-autoscaler/src/autoscaler/metricsgateway"

	"code.cloudfoundry.org/clock/fakeclock"
	"code.cloudfoundry.org/lager"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("AppManager", func() {
	var (
		policyDB                  *fakes.FakePolicyDB
		clock                     *fakeclock.FakeClock
		appManager                *AppManager
		logger                    lager.Logger
		testAppIDRetrieveInterval = 5 * time.Second
	)

	BeforeEach(func() {
		policyDB = &fakes.FakePolicyDB{}
		clock = fakeclock.NewFakeClock(time.Now())
		logger = lager.NewLogger("AppManager-test")
	})
	Context("Start", func() {
		JustBeforeEach(func() {
			appManager = NewAppManager(logger, clock, testAppIDRetrieveInterval, policyDB)
			appManager.Start()

		})

		AfterEach(func() {
			appManager.Stop()
		})

		Context("when the AppManager is started", func() {
			BeforeEach(func() {
				i := 0
				policyDB.GetAppIdsStub = func() (map[string]bool, error) {
					if i == 0 {
						i++
						return map[string]bool{"testAppId-1": true, "testAppId-2": true}, nil
					} else {
						return map[string]bool{"testAppId-3": true, "testAppId-4": true}, nil
					}
				}
			})
			It("should retrieve app ids for every interval", func() {
				Eventually(policyDB.GetAppIdsCallCount).Should(Equal(1))
				Eventually(appManager.GetAppIDs).Should(Equal(map[string]bool{"testAppId-1": true, "testAppId-2": true}))
				By("app ids in policy changes")
				clock.Increment(1 * testAppIDRetrieveInterval)
				Eventually(policyDB.GetAppIdsCallCount).Should(Equal(2))
				Eventually(appManager.GetAppIDs).Should(Equal(map[string]bool{"testAppId-3": true, "testAppId-4": true}))
			})

			Context("when retrieving policies from policyDB fails", func() {
				BeforeEach(func() {
					policyDB.GetAppIdsStub = func() (map[string]bool, error) {
						return nil, errors.New("error when retrieve app ids from policyDB")
					}
				})
				It("should return an empty app id map", func() {
					clock.Increment(2 * testAppIDRetrieveInterval)
					appIDMap := appManager.GetAppIDs()
					Expect(len(appIDMap)).To(Equal(0))
				})
			})
		})
	})

	Context("Stop", func() {
		BeforeEach(func() {
			appManager = NewAppManager(logger, clock, testAppIDRetrieveInterval, policyDB)
			appManager.Start()
			Eventually(policyDB.GetAppIdsCallCount).Should(Equal(1))

			appManager.Stop()
		})

		It("stops the polling", func() {
			clock.Increment(5 * testAppIDRetrieveInterval)
			Consistently(policyDB.GetAppIdsCallCount).Should(Or(Equal(1), Equal(2)))
		})
	})
})
