package cred_helper_test

import (
	"context"
	"errors"

	"code.cloudfoundry.org/app-autoscaler/src/autoscaler/cred_helper"
	"code.cloudfoundry.org/app-autoscaler/src/autoscaler/fakes"
	"code.cloudfoundry.org/app-autoscaler/src/autoscaler/models"
	"code.cloudfoundry.org/lager/v3"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("StoredProcedureCredHelper", func() {
	var (
		storedProcedureDb *fakes.FakeStoredProcedureDB
		appId             = "testAppId"
		credResult        *models.Credential
		credentials       cred_helper.Credentials
	)
	BeforeEach(func() {
		storedProcedureDb = &fakes.FakeStoredProcedureDB{}

		logger := lager.NewLogger("stored_procedure_helper_test")
		logger.RegisterSink(lager.NewWriterSink(GinkgoWriter, lager.DEBUG))

		credentials = cred_helper.NewStoredProcedureCredHelper(storedProcedureDb, cred_helper.MaxRetry, logger)

	})
	Context("CreateCredential", func() {
		var err error
		JustBeforeEach(func() {
			credResult, err = credentials.Create(context.Background(), appId, nil)
		})
		Context("when there is no error when calling storedProcedureDb", func() {
			BeforeEach(func() {
				storedProcedureDb.CreateCredentialsReturns(&models.Credential{
					Username: "secure-username",
					Password: "secure-password",
				}, nil)
			})
			Context("when credential does not exist", func() {
				It("should try saving only once and succeed", func() {
					Expect(storedProcedureDb.CreateCredentialsCallCount()).To(Equal(1))
					Expect(credResult).NotTo(BeNil())
					Expect(err).NotTo(HaveOccurred())
				})
			})

		})
		Context("when there is continuous error when calling storedProcedureDb", func() {
			BeforeEach(func() {
				storedProcedureDb.CreateCredentialsReturns(nil, errors.New("dberror"))
				storedProcedureDb.DeleteCredentialsReturns(errors.New("dberror"))

			})
			It("should try MaxRetry times and return error", func() {
				Expect(storedProcedureDb.CreateCredentialsCallCount()).To(Equal(cred_helper.MaxRetry))
				Expect(storedProcedureDb.DeleteCredentialsCallCount()).To(Equal(cred_helper.MaxRetry))
				Expect(credResult).To(BeNil())
				Expect(err).To(HaveOccurred())
			})
		})
	})
	Context("DeleteCredential", func() {
		var err error
		JustBeforeEach(func() {
			err = credentials.Delete(context.Background(), appId)
		})
		Context("when there is no error when calling storedProcedureDb", func() {
			BeforeEach(func() {
				storedProcedureDb.DeleteCredentialsReturns(nil)
			})
			It("should try only once and succeed", func() {
				Expect(storedProcedureDb.DeleteCredentialsCallCount()).To(Equal(1))
				Expect(err).NotTo(HaveOccurred())
			})
		})
		Context("when there is continuous error when calling storedProcedureDb", func() {
			BeforeEach(func() {
				storedProcedureDb.DeleteCredentialsReturns(errors.New("dberror"))
			})
			It("should try MaxRetry times and return error", func() {
				Expect(storedProcedureDb.DeleteCredentialsCallCount()).To(Equal(cred_helper.MaxRetry))
				Expect(err).To(HaveOccurred())
			})
		})
	})
})
