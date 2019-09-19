package sbss_cred_helper

import (
	"errors"

	"code.cloudfoundry.org/lager/lagertest"

	"autoscaler/fakes"
	"autoscaler/models"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("CustomMetricCredHelper", func() {
	var (
		sbssDB     *fakes.FakeSbssDB
		appId      = "testAppId"
		credResult *models.Credential
		logger     *lagertest.TestLogger
	)
	BeforeEach(func() {
		sbssDB = &fakes.FakeSbssDB{}
		logger = lagertest.NewTestLogger("oauth")
	})
	Context("CreateCredential", func() {
		var err error
		JustBeforeEach(func() {
			credResult, err = CreateCredential(appId, sbssDB, MaxRetry, logger)
		})
		Context("when there is no error when calling sbssDB", func() {
			BeforeEach(func() {
				sbssDB.CreateCredentialsReturns(&models.Credential{
					Username: "secure-username",
					Password: "secure-password",
				}, nil)
			})
			Context("when credential does not exist", func() {
				It("should try saving only once and succeed", func() {
					Expect(sbssDB.CreateCredentialsCallCount()).To(Equal(1))
					Expect(credResult).NotTo(BeNil())
					Expect(err).NotTo(HaveOccurred())
				})
			})

		})
		Context("when there is continuous error when calling policydb", func() {
			BeforeEach(func() {
				sbssDB.CreateCredentialsReturns(nil, errors.New("dberror"))
				sbssDB.DeleteCredentialsReturns(errors.New("dberror"))

			})
			It("should try MaxRetry times and return error", func() {
				Expect(sbssDB.CreateCredentialsCallCount()).To(Equal(MaxRetry))
				Expect(sbssDB.DeleteCredentialsCallCount()).To(Equal(MaxRetry))
				Expect(credResult).To(BeNil())
				Expect(err).To(HaveOccurred())
			})
		})
	})
	Context("DeleteCredential", func() {
		var err error
		JustBeforeEach(func() {
			err = DeleteCredential(appId, sbssDB, MaxRetry, logger)
		})
		Context("when there is no error when calling policydb", func() {
			BeforeEach(func() {
				sbssDB.DeleteCredentialsReturns(nil)
			})
			It("should try only once and succeed", func() {
				Expect(sbssDB.DeleteCredentialsCallCount()).To(Equal(1))
				Expect(err).NotTo(HaveOccurred())
			})
		})
		Context("when there is continous error when calling policydb", func() {
			BeforeEach(func() {
				sbssDB.DeleteCredentialsReturns(errors.New("dberror"))
			})
			It("should try MaxRetry times and return error", func() {
				Expect(sbssDB.DeleteCredentialsCallCount()).To(Equal(MaxRetry))
				Expect(err).To(HaveOccurred())
			})
		})
	})
})
