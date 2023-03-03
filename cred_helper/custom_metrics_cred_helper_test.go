package cred_helper_test

import (
	"context"
	"database/sql"
	"errors"
	"time"

	"code.cloudfoundry.org/lager/v3"
	"github.com/patrickmn/go-cache"

	"code.cloudfoundry.org/app-autoscaler/src/autoscaler/cred_helper"

	"code.cloudfoundry.org/app-autoscaler/src/autoscaler/fakes"
	"code.cloudfoundry.org/app-autoscaler/src/autoscaler/models"

	. "github.com/onsi/ginkgo/v2"
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
		creds                  cred_helper.Credentials
		credentialCache        cache.Cache
	)

	BeforeEach(func() {
		policyDB = &fakes.FakePolicyDB{}
		credentialCache = *cache.New(10*time.Minute, -1)

		logger := lager.NewLogger("custom_metrics_helper_test")
		logger.RegisterSink(lager.NewWriterSink(GinkgoWriter, lager.DEBUG))

		creds = cred_helper.NewCustomMetricsCredHelperWithCache(policyDB, cred_helper.MaxRetry, credentialCache, 10*time.Minute, logger)

	})
	Context("CreateCredential", func() {
		var err error
		JustBeforeEach(func() {
			credResult, err = creds.Create(context.Background(), appId, userProvidedCredential)
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
		Context("when there is continuous error when calling policydb", func() {
			BeforeEach(func() {
				policyDB.GetCredentialReturns(nil, sql.ErrNoRows)
				policyDB.SaveCredentialReturns(errors.New("dberror"))

			})
			It("should try MaxRetry times and return error", func() {
				Expect(policyDB.SaveCredentialCallCount()).To(Equal(cred_helper.MaxRetry))
				Expect(credResult).To(BeNil())
				Expect(err).To(HaveOccurred())
			})
		})
	})
	Context("DeleteCredential", func() {
		var err error
		JustBeforeEach(func() {
			err = creds.Delete(context.Background(), appId)
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
		Context("when there is continuous error when calling policydb", func() {
			BeforeEach(func() {
				policyDB.DeleteCredentialReturns(errors.New("dberror"))
			})
			It("should try MaxRetry times and return error", func() {
				Expect(policyDB.DeleteCredentialCallCount()).To(Equal(cred_helper.MaxRetry))
				Expect(err).To(HaveOccurred())
			})
		})
	})

	Context("ValidateCredentials", func() {
		Context("credentials exists in the cache", func() {
			It("should get the credentials from cache without searching from database", func() {
				storedCredentials := &models.Credential{
					Username: "$2a$10$YnQNQYcvl/Q2BKtThOKFZ.KB0nTIZwhKr5q1pWTTwC/PUAHsbcpFu",
					Password: "$2a$10$6nZ73cm7IV26wxRnmm5E1.nbk9G.0a4MrbzBFPChkm5fPftsUwj9G",
				}
				credentialCache.Set("an-app-id", storedCredentials, 10*time.Minute)

				valid, err := creds.Validate(context.TODO(), "an-app-id", models.Credential{Username: "username", Password: "password"})
				Expect(err).ShouldNot(HaveOccurred())
				Expect(valid).To(Equal(true))

				Expect(policyDB.GetCredentialCallCount()).To(Equal(0))
			})

		})

		Context("credentials do not exists in the cache but exist in the database", func() {
			It("should get the credentials from database, add it to the cache", func() {
				storedCredentials := &models.Credential{
					Username: "$2a$10$YnQNQYcvl/Q2BKtThOKFZ.KB0nTIZwhKr5q1pWTTwC/PUAHsbcpFu",
					Password: "$2a$10$6nZ73cm7IV26wxRnmm5E1.nbk9G.0a4MrbzBFPChkm5fPftsUwj9G",
				}

				policyDB.GetCredentialReturns(storedCredentials, nil)

				valid, err := creds.Validate(context.TODO(), "an-app-id", models.Credential{Username: "username", Password: "password"})
				Expect(err).ShouldNot(HaveOccurred())
				Expect(valid).To(Equal(true))

				Expect(policyDB.GetCredentialCallCount()).To(Equal(1))

				//fills the cache
				_, found := credentialCache.Get("an-app-id")
				Expect(found).To(Equal(true))
			})
		})

		Context("when credentials neither exists in the cache nor exist in the database", func() {
			It("should search in both cache & database and returns an error", func() {
				policyDB.GetCredentialReturns(nil, errors.New("some error"))

				valid, err := creds.Validate(context.TODO(), "an-app-id", models.Credential{Username: "username", Password: "password"})
				Expect(err).Should(HaveOccurred())
				Expect(valid).To(Equal(false))

				Expect(policyDB.GetCredentialCallCount()).To(Equal(1))

				//fills the cache
				_, found := credentialCache.Get("an-app-id")
				Expect(found).To(Equal(false))
			})
		})

		Context("when a stale credentials exists in the cache", func() {
			It("should search in the database", func() {
				credentialCache.Set("an-app-id", &models.Credential{Username: "some-stale-hashed-username", Password: "some-stale-hashed-password"}, 10*time.Minute)

				storedCredentials := &models.Credential{
					Username: "$2a$10$YnQNQYcvl/Q2BKtThOKFZ.KB0nTIZwhKr5q1pWTTwC/PUAHsbcpFu",
					Password: "$2a$10$6nZ73cm7IV26wxRnmm5E1.nbk9G.0a4MrbzBFPChkm5fPftsUwj9G",
				}

				policyDB.GetCredentialReturns(storedCredentials, nil)

				valid, err := creds.Validate(context.TODO(), "an-app-id", models.Credential{Username: "username", Password: "password"})
				Expect(err).ShouldNot(HaveOccurred())
				Expect(valid).To(Equal(true))

				Expect(policyDB.GetCredentialCallCount()).To(Equal(1))

				//fills the cache
				_, found := credentialCache.Get("an-app-id")
				Expect(found).To(Equal(true))
			})
		})
	})
})
