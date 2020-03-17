package sqldb_test

import (
	"autoscaler/db"
	. "autoscaler/db/sqldb"
	"autoscaler/models"
	"database/sql"
	"github.com/lib/pq"
	"github.com/go-sql-driver/mysql"
	"code.cloudfoundry.org/lager"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"encoding/json"
	"os"
	"time"
)

var _ = Describe("PolicySQLDB", func() {
	var (
		pdb             *PolicySQLDB
		dbConfig        db.DatabaseConfig
		logger          lager.Logger
		err             error
		appIds          map[string]bool
		scalingPolicy   *models.ScalingPolicy
		policyJson      []byte
		policyJsonStr   string
		appId           string
		policies        []*models.PolicyJson
		testMetricName  string = "TestMetricName"
		username        string
		password        string
		anotherUsername string
		antherPassword  string
		credential      *models.Credential
	)

	BeforeEach(func() {
		logger = lager.NewLogger("policy-sqldb-test")
		dbConfig = db.DatabaseConfig{
			URL:                   os.Getenv("DBURL"),
			MaxOpenConnections:    10,
			MaxIdleConnections:    5,
			ConnectionMaxLifetime: 10 * time.Second,
		}
	})

	Describe("NewPolicySQLDB", func() {
		JustBeforeEach(func() {
			pdb, err = NewPolicySQLDB(dbConfig, logger)
		})

		AfterEach(func() {
			if pdb != nil {
				err = pdb.Close()
				Expect(err).NotTo(HaveOccurred())
			}
		})

		Context("when db url is not correct", func() {
			BeforeEach(func() {
				dbConfig.URL = "postgres://not-exist-user:not-exist-password@localhost/autoscaler?sslmode=disable"
			})
			It("should throw an error", func() {
				Expect(err).To(BeAssignableToTypeOf(&pq.Error{}))
			})
		})

		Context("when mysql db url is not correct", func() {
			BeforeEach(func() {
				dbConfig.URL = "not-exist-user:not-exist-password@tcp(localhost)/autoscaler?tls=false"
			})
			It("should throw an error", func() {
				Expect(err).To(BeAssignableToTypeOf(&mysql.MySQLError{}))
			})
		})
		
		Context("when db url is correct", func() {
			It("should not error", func() {
				Expect(err).NotTo(HaveOccurred())
				Expect(pdb).NotTo(BeNil())
			})
		})
	})

	Describe("GetAppIds", func() {
		BeforeEach(func() {
			pdb, err = NewPolicySQLDB(dbConfig, logger)
			Expect(err).NotTo(HaveOccurred())

			cleanPolicyTable()
		})

		AfterEach(func() {
			err = pdb.Close()
			Expect(err).NotTo(HaveOccurred())
		})

		JustBeforeEach(func() {
			appIds, err = pdb.GetAppIds()
		})

		Context("when policy table is empty", func() {
			It("returns no app ids", func() {
				Expect(err).NotTo(HaveOccurred())
				Expect(appIds).To(BeEmpty())
			})
		})

		Context("when policy table is not empty", func() {
			BeforeEach(func() {
				scalingPolicy = &models.ScalingPolicy{InstanceMax: 1, InstanceMin: 6}
				insertPolicy("first-app-id", scalingPolicy)
				insertPolicy("second-app-id", scalingPolicy)
				insertPolicy("third-app-id", scalingPolicy)
			})

			It("returns all app ids", func() {
				Expect(err).NotTo(HaveOccurred())
				Expect(appIds).To(HaveKey("first-app-id"))
				Expect(appIds).To(HaveKey("second-app-id"))
				Expect(appIds).To(HaveKey("third-app-id"))
			})
		})
	})

	Describe("GetAppPolicy", func() {
		BeforeEach(func() {
			pdb, err = NewPolicySQLDB(dbConfig, logger)
			Expect(err).NotTo(HaveOccurred())

			cleanPolicyTable()

			insertPolicy("an-app-id", &models.ScalingPolicy{
				InstanceMin: 1,
				InstanceMax: 6,
				ScalingRules: []*models.ScalingRule{{
					MetricType:            testMetricName,
					BreachDurationSeconds: 180,
					Threshold:             1048576000,
					Operator:              ">",
					CoolDownSeconds:       300,
					Adjustment:            "+10%"}}})
			insertPolicy("another-app-id", &models.ScalingPolicy{
				InstanceMin: 2,
				InstanceMax: 8,
				ScalingRules: []*models.ScalingRule{{
					MetricType:            testMetricName,
					BreachDurationSeconds: 300,
					Threshold:             104857600,
					Operator:              "<",
					CoolDownSeconds:       120,
					Adjustment:            "-2"}}})
		})

		AfterEach(func() {
			err = pdb.Close()
			Expect(err).NotTo(HaveOccurred())
		})

		JustBeforeEach(func() {
			scalingPolicy, err = pdb.GetAppPolicy(appId)
		})

		Context("when policy table has the app", func() {
			BeforeEach(func() {
				appId = "an-app-id"
			})

			It("returns the policy", func() {
				Expect(err).NotTo(HaveOccurred())
				Expect(*scalingPolicy).To(Equal(models.ScalingPolicy{
					InstanceMin: 1,
					InstanceMax: 6,
					ScalingRules: []*models.ScalingRule{{
						MetricType:            testMetricName,
						BreachDurationSeconds: 180,
						Threshold:             1048576000,
						Operator:              ">",
						CoolDownSeconds:       300,
						Adjustment:            "+10%"}}}))
			})

		})

		Context("when policy table does not have the app", func() {
			BeforeEach(func() {
				appId = "non-existent-app"
			})

			It("should return nil", func() {
				Expect(err).NotTo(HaveOccurred())
				Expect(scalingPolicy).To(BeNil())
			})
		})
	})

	Describe("RetrievePolicies", func() {
		BeforeEach(func() {
			pdb, err = NewPolicySQLDB(dbConfig, logger)
			Expect(err).NotTo(HaveOccurred())

			cleanPolicyTable()
		})

		AfterEach(func() {
			err = pdb.Close()
			Expect(err).NotTo(HaveOccurred())
		})

		JustBeforeEach(func() {
			scalingPolicy = &models.ScalingPolicy{}
			insertPolicy("first-app-id", scalingPolicy)
			insertPolicy("second-app-id", scalingPolicy)
			insertPolicy("third-app-id", scalingPolicy)
			policies, err = pdb.RetrievePolicies()
			for i, policy := range policies {
				policy.PolicyStr = formatPolicyString(policy.PolicyStr)
				policies[i] = policy
			}
		})

		Context("when retriving all the policies", func() {
			It("returns all the policies", func() {
				Expect(err).NotTo(HaveOccurred())

				policyJson, err = json.Marshal(models.ScalingPolicy{})
				Expect(err).NotTo(HaveOccurred())

				Expect(policies).To(ConsistOf(
					&models.PolicyJson{
						AppId:     "first-app-id",
						PolicyStr: string(policyJson),
					},
					&models.PolicyJson{
						AppId:     "second-app-id",
						PolicyStr: string(policyJson),
					},
					&models.PolicyJson{
						AppId:     "third-app-id",
						PolicyStr: string(policyJson),
					},
				))
			})
		})
	})

	Describe("SavePolicy", func() {
		BeforeEach(func() {
			pdb, err = NewPolicySQLDB(dbConfig, logger)
			Expect(err).NotTo(HaveOccurred())

			cleanPolicyTable()
		})

		AfterEach(func() {
			err = pdb.Close()
			Expect(err).NotTo(HaveOccurred())
		})
		Context("when no policy is present for the app_id", func() {
			JustBeforeEach(func() {
				policyJsonStr = `{
					"instance_max_count":4,
					"instance_min_count":1,
					"scaling_rules":[
					{
						"metric_type":"memoryutil",
						"breach_duration_secs":600,
						"threshold":90,
						"operator":">=",
						"cool_down_secs":300,
						"adjustment":"+1"
					}]
				}`
				err = pdb.SaveAppPolicy("an-app-id", policyJsonStr, "1234")
			})
			It("saves the policy", func() {
				Expect(err).NotTo(HaveOccurred())
				Expect(formatPolicyString(getAppPolicy("an-app-id"))).To(Equal(formatPolicyString(policyJsonStr)))
			})
		})

		Context("when a policy is already present for the app_id", func() {
			JustBeforeEach(func() {
				insertPolicy("an-app-id", &models.ScalingPolicy{
					InstanceMin: 1,
					InstanceMax: 6,
					ScalingRules: []*models.ScalingRule{{
						MetricType:            testMetricName,
						BreachDurationSeconds: 180,
						Threshold:             1048576000,
						Operator:              ">",
						CoolDownSeconds:       300,
						Adjustment:            "+10%"}}})
				policyJsonStr = `{
					"instance_max_count":4,
					"instance_min_count":1,
					"scaling_rules":[
					{
						"metric_type":"memoryutil",
						"breach_duration_secs":600,
						"threshold":90,
						"operator":">=",
						"cool_down_secs":300,
						"adjustment":"+1"
					}]
				}`
				err = pdb.SaveAppPolicy("an-app-id", policyJsonStr, "1234")
			})
			It("updates the policy", func() {
				Expect(err).NotTo(HaveOccurred())
				Expect(formatPolicyString(getAppPolicy("an-app-id"))).To(Equal(formatPolicyString(policyJsonStr)))
			})
		})
	})

	Describe("DeletePolicy", func() {
		BeforeEach(func() {
			pdb, err = NewPolicySQLDB(dbConfig, logger)
			Expect(err).NotTo(HaveOccurred())

			cleanPolicyTable()
		})

		AfterEach(func() {
			err = pdb.Close()
			Expect(err).NotTo(HaveOccurred())
		})

		JustBeforeEach(func() {
			err = pdb.DeletePolicy("an-app-id")
		})

		Context("when there is no policy in the table", func() {
			It("should not error", func() {
				Expect(err).NotTo(HaveOccurred())
			})
		})

		Context("when policy table is not empty", func() {
			BeforeEach(func() {
				scalingPolicy = &models.ScalingPolicy{InstanceMax: 1, InstanceMin: 6}
				insertPolicy("an-app-id", scalingPolicy)
			})

			It("should delete the policy", func() {
				Expect(err).NotTo(HaveOccurred())
				policy, err := pdb.GetAppPolicy("an-app-id")
				Expect(err).NotTo(HaveOccurred())
				Expect(policy).To(BeNil())
			})
		})

		Context("when there is database error", func() {
			BeforeEach(func() {
				pdb.Close()
			})
			It("should error", func() {
				Expect(err).To(HaveOccurred())
			})
		})
	})

	Describe("GetCredential", func() {
		BeforeEach(func() {
			pdb, err = NewPolicySQLDB(dbConfig, logger)
			Expect(err).NotTo(HaveOccurred())

			cleanCredentialTable()
		})

		AfterEach(func() {
			err = pdb.Close()
			Expect(err).NotTo(HaveOccurred())
		})

		JustBeforeEach(func() {
			credential, err = pdb.GetCredential("an-app-id")
		})

		Context("when there is no credential for target application", func() {
			It("should not return any credentials", func() {
				Expect(err).To(Equal(sql.ErrNoRows))
				Expect(credential).To(BeNil())
			})
		})

		Context("when there is credential for target application", func() {
			BeforeEach(func() {
				insertCredential("an-app-id", "username", "password")
			})

			It("Should get the credentials", func() {
				Expect(err).NotTo(HaveOccurred())
				Expect(credential.Password).To(Equal("password"))
				Expect(credential.Username).To(Equal("username"))
			})

		})
	})
	Describe("SaveCredential", func() {
		BeforeEach(func() {
			pdb, err = NewPolicySQLDB(dbConfig, logger)
			Expect(err).NotTo(HaveOccurred())
			cleanCredentialTable()
			appId = "the-test-app-id"
			username = "the-user-name"
			password = "the-password"
			anotherUsername = "the-user-name"
			antherPassword = "the-password"
		})

		AfterEach(func() {
			err = pdb.Close()
			Expect(err).NotTo(HaveOccurred())
		})

		JustBeforeEach(func() {
			err = pdb.SaveCredential(appId, models.Credential{
				Username: username,
				Password: password,
			})
			Expect(err).NotTo(HaveOccurred())
		})
		Context("when no credential is present", func() {
			It("saves the credential", func() {
				usernameResult, passwordResult, err := getCredential(appId)
				Expect(err).NotTo(HaveOccurred())
				Expect(usernameResult).To(Equal(username))
				Expect(passwordResult).To(Equal(password))
			})
		})
		Context("when the credential is already present", func() {
			BeforeEach(func() {
				err = insertCredential(appId, anotherUsername, antherPassword)
				Expect(err).NotTo(HaveOccurred())
				usernameResult, passwordResult, err := getCredential(appId)
				Expect(err).NotTo(HaveOccurred())
				Expect(usernameResult).To(Equal(anotherUsername))
				Expect(passwordResult).To(Equal(antherPassword))

			})
			It("updates the credential", func() {
				usernameResult, passwordResult, err := getCredential(appId)
				Expect(err).NotTo(HaveOccurred())
				Expect(usernameResult).To(Equal(username))
				Expect(passwordResult).To(Equal(password))
			})
		})
	})
	Describe("DeleteCred", func() {
		BeforeEach(func() {
			pdb, err = NewPolicySQLDB(dbConfig, logger)
			Expect(err).NotTo(HaveOccurred())
			cleanCredentialTable()
			appId = "the-test-app-id"
			username = "the-user-name"
			password = "the-password"
		})

		AfterEach(func() {
			err = pdb.Close()
			Expect(err).NotTo(HaveOccurred())
		})

		JustBeforeEach(func() {
			err = pdb.DeleteCredential(appId)
		})
		Context("when there is no credential in the table", func() {
			It("should not error", func() {
				Expect(err).NotTo(HaveOccurred())
			})
		})

		Context("when policy table is not empty", func() {
			BeforeEach(func() {
				err = insertCredential(appId, username, password)
				Expect(err).NotTo(HaveOccurred())
			})

			It("should delete the credential", func() {
				Expect(err).NotTo(HaveOccurred())
				hasCredential := hasCredential(appId)
				Expect(hasCredential).To(BeFalse())
			})
		})
		Context("when there is database error", func() {
			BeforeEach(func() {
				pdb.Close()
			})
			It("should error", func() {
				Expect(err).To(HaveOccurred())
			})
		})
	})

})
