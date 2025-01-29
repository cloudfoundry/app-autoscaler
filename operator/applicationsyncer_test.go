package operator_test

import (
	"context"
	"errors"

	"code.cloudfoundry.org/app-autoscaler/src/autoscaler/cf"
	"code.cloudfoundry.org/app-autoscaler/src/autoscaler/fakes"
	"code.cloudfoundry.org/app-autoscaler/src/autoscaler/operator"

	"code.cloudfoundry.org/lager/v3/lagertest"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("AppSynchronizer", func() {
	var (
		appSynchronizer *operator.ApplicationSynchronizer
		cfc             *fakes.FakeContextClient
		policyDB        *fakes.FakePolicyDB
		appDetails      map[string]bool
	)

	BeforeEach(func() {
		logger := lagertest.NewTestLogger("application-synchoronizer-test")
		cfc = &fakes.FakeContextClient{}
		policyDB = &fakes.FakePolicyDB{}
		appSynchronizer = operator.NewApplicationSynchronizer(cfc, policyDB, logger)
	})

	Describe("Sync", func() {
		JustBeforeEach(func() {
			policyDB.GetAppIdsReturns(appDetails, nil)
			appSynchronizer.Operate(context.Background())
		})

		BeforeEach(func() {
			appDetails = make(map[string]bool)
			appDetails["an-app-id"] = true
		})

		When("trying to delete existing application records from policy db", func() {
			BeforeEach(func() {
				cfc.GetAppReturns(&cf.App{}, nil)
			})
			It("should not delete", func() {
				Eventually(policyDB.GetAppIdsCallCount).Should(Equal(1))
				Eventually(cfc.GetAppCallCount).Should(Equal(1))
				Consistently(policyDB.DeletePolicyCallCount).Should(Equal(0))
			})
		})

		When("trying to delete non-existent application records from policy db", func() {
			BeforeEach(func() {
				cfc.GetAppReturns(nil, cf.CfResourceNotFound)
			})
			It("should successfully delete", func() {
				Eventually(policyDB.GetAppIdsCallCount).Should(Equal(1))
				Eventually(cfc.GetAppCallCount).Should(Equal(1))
				Eventually(policyDB.DeletePolicyCallCount).Should(Equal(1))
			})

			When("deleting non-existent application records from policy db fails", func() {
				BeforeEach(func() {
					appDetails["a-second-id"] = true
					policyDB.DeletePolicyReturns(errors.New("some error"))
				})
				It("it should continue and try to delete the others", func() {
					Eventually(policyDB.GetAppIdsCallCount).Should(Equal(1))
					Eventually(cfc.GetAppCallCount).Should(Equal(2))
					Eventually(policyDB.DeletePolicyCallCount).Should(Equal(2))
				})
			})
		})
	})
})
