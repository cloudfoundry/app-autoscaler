package sqldb_test

import (
	"autoscaler/db"
	. "autoscaler/db/sqldb"
	"autoscaler/models"

	"code.cloudfoundry.org/lager"
	"github.com/lib/pq"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"os"
	"time"
)

var _ = Describe("SbssSQLDb", func() {
	var (
		dbConfig           db.DatabaseConfig
		logger             lager.Logger
		sdb                *SbssSQLDb
		err                error
		credential         *models.Credential
		credentialsOptions models.CredentialsOptions
	)

	BeforeEach(func() {
		logger = lager.NewLogger("sbss-sqldb-test")
		dbConfig = db.DatabaseConfig{
			URL:                   os.Getenv("SBSS_DBURL"),
			MaxOpenConnections:    10,
			MaxIdleConnections:    5,
			ConnectionMaxLifetime: 10 * time.Second,
			ConnectionMaxIdleTime: 10 * time.Second,
		}
		credentialsOptions = models.CredentialsOptions{
			InstanceId: "app-id-1",
			BindingId:  "app-id-1",
		}
	})

	Describe("NewSchedulerSQLDB", func() {
		JustBeforeEach(func() {
			sdb, err = NewSbssSQLDb(dbConfig, logger)
		})

		AfterEach(func() {
			if sdb != nil {
				err = sdb.Close()
				Expect(err).NotTo(HaveOccurred())
			}
		})

		Context("when db url is not correct", func() {
			BeforeEach(func() {
				dbConfig.URL = "postgres://not-exist-user:not-exist-password@localhost/autoscaler?sslmode=disable"
			})
			It("should error", func() {
				Expect(err).To(BeAssignableToTypeOf(&pq.Error{}))
			})

		})

		Context("when url is correct", func() {
			It("should not error", func() {
				Expect(err).NotTo(HaveOccurred())
				Expect(sdb).NotTo(BeNil())
			})
		})
	})

	Describe("CreateCredentials", func() {
		BeforeEach(func() {
			sdb, err = NewSbssSQLDb(dbConfig, logger)
			Expect(err).NotTo(HaveOccurred())
		})

		AfterEach(func() {
			err = sdb.DeleteCredentials(credentialsOptions)
			Expect(err).NotTo(HaveOccurred())
			err = sdb.Close()
			Expect(err).NotTo(HaveOccurred())
		})

		JustBeforeEach(func() {
			credential, err = sdb.CreateCredentials(credentialsOptions)
		})

		Context("when no credentials exist", func() {
			It("returns new credentials", func() {
				Expect(err).NotTo(HaveOccurred())
				Expect(credential).NotTo(BeNil())
				Expect(credential.Username).NotTo(BeNil())
				Expect(credential.Username).NotTo(HaveLen(0))
				Expect(credential.Password).NotTo(BeNil())
				Expect(credential.Password).NotTo(HaveLen(0))
			})
		})

		Context("when the credentials have already been created", func() {
			BeforeEach(func() {
				_, err = sdb.CreateCredentials(credentialsOptions)
				Expect(err).NotTo(HaveOccurred())
			})
			It("returns an error", func() {
				Expect(err).To(HaveOccurred())
			})
		})

	})

	Describe("DeleteCredentials", func() {
		BeforeEach(func() {
			sdb, err = NewSbssSQLDb(dbConfig, logger)
			Expect(err).NotTo(HaveOccurred())
		})

		AfterEach(func() {
			err = sdb.Close()
			Expect(err).NotTo(HaveOccurred())
		})

		JustBeforeEach(func() {
			err = sdb.DeleteCredentials(credentialsOptions)
		})

		Context("when no credentials exist", func() {
			It("returns no error", func() {
				Expect(err).NotTo(HaveOccurred())
			})
		})

		Context("when the credentials have already been created", func() {
			BeforeEach(func() {
				_, err = sdb.CreateCredentials(credentialsOptions)
				Expect(err).NotTo(HaveOccurred())
			})
			It("returns no error", func() {
				Expect(err).NotTo(HaveOccurred())
			})
		})

	})

	Describe("ValidateCredentials", func() {
		var returnedCredentialOptions *models.CredentialsOptions
		BeforeEach(func() {
			sdb, err = NewSbssSQLDb(dbConfig, logger)
			Expect(err).NotTo(HaveOccurred())
			credential, err = sdb.CreateCredentials(credentialsOptions)
			Expect(err).NotTo(HaveOccurred())
		})

		AfterEach(func() {
			err = sdb.DeleteCredentials(credentialsOptions)
			Expect(err).NotTo(HaveOccurred())
			err = sdb.Close()
			Expect(err).NotTo(HaveOccurred())
		})

		Context("when the right credentials are validated", func() {
			It("returns the credentials options", func() {
				returnedCredentialOptions, err = sdb.ValidateCredentials(*credential)
				Expect(err).NotTo(HaveOccurred())
				Expect(returnedCredentialOptions).To(Equal(&credentialsOptions))
			})
		})

		Context("when the credentials are wrong", func() {
			It("returns an error", func() {
				returnedCredentialOptions, err = sdb.ValidateCredentials(models.Credential{
					Username: "wrong",
					Password: "credentials",
				})
				Expect(err).To(HaveOccurred())
			})
		})

	})

})
