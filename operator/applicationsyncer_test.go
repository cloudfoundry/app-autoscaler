package operator_test

import (
	"code.cloudfoundry.org/app-autoscaler/src/autoscaler/fakes"
	"code.cloudfoundry.org/app-autoscaler/src/autoscaler/models"
	"code.cloudfoundry.org/app-autoscaler/src/autoscaler/operator"

	"code.cloudfoundry.org/lager/lagertest"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("AppSynchronizer", func() {
	var (
		appSynchronizer *operator.ApplicationSynchronizer
		cfc             *fakes.FakeCFClient
		policyDB        *fakes.FakePolicyDB
	)

	BeforeEach(func() {
		logger := lagertest.NewTestLogger("application-synchoronizer-test")
		cfc = &fakes.FakeCFClient{}
		policyDB = &fakes.FakePolicyDB{}
		appSynchronizer = operator.NewApplicationSynchronizer(cfc, policyDB, logger)
	})

	Describe("Sync", func() {
		JustBeforeEach(func() {
			appSynchronizer.Operate()
		})

		BeforeEach(func() {
			appDetails := make(map[string]bool)
			appDetails["an-app-id"] = true
			policyDB.GetAppIdsReturns(appDetails, nil)
		})

		Context("when trying to delete existing application records from policy db", func() {
			BeforeEach(func() {
				appStateStarted := models.AppStatusStarted
				existent_app_details := models.AppEntity{Instances: 1, State: &appStateStarted}
				cfc.GetAppReturns(&existent_app_details, nil)
			})
			It("should not delete", func() {
				Eventually(policyDB.GetAppIdsCallCount).Should(Equal(1))
				Eventually(cfc.GetAppCallCount).Should(Equal(1))
				Consistently(policyDB.DeletePolicyCallCount).Should(Equal(0))
			})
		})

		Context("when trying to delete non-existent application records from policy db", func() {
			BeforeEach(func() {
				err := models.NewAppNotFoundErr("The app could not be found")
				cfc.GetAppReturns(nil, err)
			})
			It("should successfully delete", func() {
				Eventually(policyDB.GetAppIdsCallCount).Should(Equal(1))
				Eventually(cfc.GetAppCallCount).Should(Equal(1))
				Eventually(policyDB.DeletePolicyCallCount).Should(Equal(1))
			})
		})
	})
})
