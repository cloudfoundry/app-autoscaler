package custom_metrics_cred_helper_test

import (
	"database/sql"
	"errors"

	. "autoscaler/api/custom_metrics_cred_helper"
	"autoscaler/fakes"
	"autoscaler/models"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("CustomMetricCredHelper", func() {
	var (
		policyDB               *fakes.FakePolicyDB
		appId                  = "testAppId"
		testUserName           = "the-user-name"
		testPassword           = "the-password"
		userProvidedCredential *models.Credential
		credResult             *models.Credential
	)
	BeforeEach(func() {
		policyDB = &fakes.FakePolicyDB{}
	})
	Context("CreateCredential", func() {
		var err error
		JustBeforeEach(func() {
			credResult, err = CreateCredential(appId, userProvidedCredential, policyDB, MaxRetry)
		})
		Context("when userProvideCredential is not nil", func() {
			BeforeEach(func() {
				userProvidedCredential = &models.Credential{
					Username: testUserName,
					Password: testPassword,
				}
				policyDB.SaveCredentialReturns(nil)
			})
			It("saves the credential user provided", func() {

				Expect(credResult.Username).To(Equal(testUserName))
				Expect(credResult.Password).To(Equal(testPassword))
			})
		})
		Context("when there is no error when calling policydb", func() {
			BeforeEach(func() {
				policyDB.SaveCredentialReturns(nil)
			})
			Context("when credential does not exist", func() {
				It("should try saving only once and succeed", func() {
					Expect(policyDB.SaveCredentialCallCount()).To(Equal(1))
					Expect(credResult).NotTo(BeNil())
					Expect(err).NotTo(HaveOccurred())
				})
			})

		})
		Context("when there is continous error when calling policydb", func() {
			BeforeEach(func() {
				policyDB.GetCredentialReturns(nil, sql.ErrNoRows)
				policyDB.SaveCredentialReturns(errors.New("dberror"))

			})
			It("should try MaxRetry times and return error", func() {
				Expect(policyDB.SaveCredentialCallCount()).To(Equal(MaxRetry))
				Expect(credResult).To(BeNil())
				Expect(err).To(HaveOccurred())
			})
		})
	})
	Context("DeleteCredential", func() {
		var err error
		JustBeforeEach(func() {
			err = DeleteCredential(appId, policyDB, MaxRetry)
		})
		Context("when there is no error when calling policydb", func() {
			BeforeEach(func() {
				policyDB.DeleteCredentialReturns(nil)
			})
			It("should try only once and succeed", func() {
				Expect(policyDB.DeleteCredentialCallCount()).To(Equal(1))
				Expect(err).NotTo(HaveOccurred())
			})
		})
		Context("when there is continous error when calling policydb", func() {
			BeforeEach(func() {
				policyDB.DeleteCredentialReturns(errors.New("dberror"))
			})
			It("should try MaxRetry times and return error", func() {
				Expect(policyDB.DeleteCredentialCallCount()).To(Equal(MaxRetry))
				Expect(err).To(HaveOccurred())
			})
		})
	})
})
