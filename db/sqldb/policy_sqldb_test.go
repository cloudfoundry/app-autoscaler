package sqldb_test

import (
	"database/sql"
	"fmt"
	"strings"

	. "code.cloudfoundry.org/app-autoscaler/src/autoscaler/testhelpers"

	"code.cloudfoundry.org/app-autoscaler/src/autoscaler/db"
	. "code.cloudfoundry.org/app-autoscaler/src/autoscaler/db/sqldb"
	"code.cloudfoundry.org/app-autoscaler/src/autoscaler/models"

	"code.cloudfoundry.org/lager"
	"github.com/go-sql-driver/mysql"
	"github.com/lib/pq"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"encoding/json"
	"os"
	"time"
)

var _ = Describe("PolicySQLDB", func() {
	var (
		pdb               *PolicySQLDB
		dbConfig          db.DatabaseConfig
		logger            lager.Logger
		err               error
		appIds            map[string]bool
		scalingPolicy     *models.ScalingPolicy
		policyJson        []byte
		policyJsonStr     string
		policies          []*models.PolicyJson
		testMetricName    = "TestMetricName"
		username          string
		password          string
		anotherUsername   string
		antherPassword    string
		credential        *models.Credential
		policyGuid        string
		policyGuid2       string
		policyGuid3       string
		anotherPolicyGuid string
		appId             string
		appId2            string
		appId3            string
	)

	BeforeEach(func() {
		logger = lager.NewLogger("policy-sqldb-test")
		logger.RegisterSink(lager.NewWriterSink(GinkgoWriter, lager.DEBUG))
		dbConfig = db.DatabaseConfig{
			URL:                   os.Getenv("DBURL"),
			MaxOpenConnections:    10,
			MaxIdleConnections:    5,
			ConnectionMaxLifetime: 10 * time.Second,
			ConnectionMaxIdleTime: 10 * time.Second,
		}
		pdb, err = NewPolicySQLDB(dbConfig, logger)
		FailOnError("unable to connect policy-db: ", err)
		DeferCleanup(func() error {
			if pdb != nil {
				return pdb.Close()
			} else {
				return nil
			}
		})
		policyGuid = addProcessIdTo("a-policy-guid")
		policyGuid2 = addProcessIdTo("a-policy-guid-2")
		policyGuid3 = addProcessIdTo("a-policy-guid-3")
		anotherPolicyGuid = addProcessIdTo("another-policy-guid")
		appId = addProcessIdTo("first-bound-app-id")
		appId2 = addProcessIdTo("second-bound-app-id")
		appId3 = addProcessIdTo("third-bound-app-id")
		username = addProcessIdTo("the-user-name")
		password = addProcessIdTo("the-password")
		anotherUsername = addProcessIdTo("another-user-name")
		antherPassword = addProcessIdTo("another-password")

		deletePolicies(pdb, logger, policyGuid, policyGuid2, policyGuid3)
		deleteApps(pdb, logger, appId, appId2, appId3)
		deleteCredentials(pdb, logger, appId, appId2, appId3)
	})

	Describe("NewPolicySQLDB", func() {
		JustBeforeEach(func() {
			if pdb != nil {
				_ = pdb.Close()
			}
			pdb, err = NewPolicySQLDB(dbConfig, logger)
		})
		Context("when db url is not correct", func() {
			BeforeEach(func() {
				if !strings.Contains(os.Getenv("DBURL"), "postgres") {
					Skip("Mysql test")
				}
				dbConfig.URL = "postgres://not-exist-user:not-exist-password@localhost/autoscaler?sslmode=disable"
			})
			It("should throw an error", func() {
				Expect(err).To(BeAssignableToTypeOf(&pq.Error{}))
			})
		})

		Context("when mysql db url is not correct", func() {
			BeforeEach(func() {
				if strings.Contains(os.Getenv("DBURL"), "postgres") {
					Skip("Not configured for mysql")
				}
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

	Describe("GetAppIds", Serial, func() {
		BeforeEach(func() {
			cleanPolicyTable()
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
				insertPolicy(appId, scalingPolicy, policyGuid)
				insertPolicy(appId2, scalingPolicy, policyGuid)
				insertPolicy(appId3, scalingPolicy, policyGuid)
			})

			It("returns all app ids", func() {
				Expect(err).NotTo(HaveOccurred())
				Expect(appIds).To(HaveKey(appId))
				Expect(appIds).To(HaveKey(appId2))
				Expect(appIds).To(HaveKey(appId3))
			})
		})
	})

	Describe("GetAppPolicy", func() {
		BeforeEach(func() {
			insertPolicy(appId, &models.ScalingPolicy{
				InstanceMin: 1,
				InstanceMax: 6,
				ScalingRules: []*models.ScalingRule{{
					MetricType:            testMetricName,
					BreachDurationSeconds: 180,
					Threshold:             1048576000,
					Operator:              ">",
					CoolDownSeconds:       300,
					Adjustment:            "+10%"}}}, policyGuid)
			insertPolicy(appId2, &models.ScalingPolicy{
				InstanceMin: 2,
				InstanceMax: 8,
				ScalingRules: []*models.ScalingRule{{
					MetricType:            testMetricName,
					BreachDurationSeconds: 300,
					Threshold:             104857600,
					Operator:              "<",
					CoolDownSeconds:       120,
					Adjustment:            "-2"}}}, policyGuid)
		})

		JustBeforeEach(func() {
			scalingPolicy, err = pdb.GetAppPolicy(appId)
		})

		Context("when policy table has the app", func() {
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
				appId = addProcessIdTo("non-existent-app")
			})

			It("should return nil", func() {
				Expect(err).NotTo(HaveOccurred())
				Expect(scalingPolicy).To(BeNil())
			})
		})
	})

	Describe("retrieve all policies", Serial, func() {

		JustBeforeEach(func() {
			cleanPolicyTable()
			scalingPolicy = &models.ScalingPolicy{}
			insertPolicy(appId, scalingPolicy, policyGuid)
			insertPolicy(appId2, scalingPolicy, policyGuid)
			insertPolicy(appId3, scalingPolicy, policyGuid)
			policies, err = pdb.RetrievePolicies()
			for i, policy := range policies {
				policy.PolicyStr, err = formatPolicyString(policy.PolicyStr)
				Expect(err).NotTo(HaveOccurred())
				policies[i] = policy
			}
		})

		Context("when retrieving all the policies", func() {
			It("returns all the policies", func() {
				Expect(err).NotTo(HaveOccurred())

				policyJson, err = json.Marshal(models.ScalingPolicy{})
				Expect(err).NotTo(HaveOccurred())

				Expect(policies).To(ConsistOf(
					&models.PolicyJson{
						AppId:     appId,
						PolicyStr: string(policyJson),
					},
					&models.PolicyJson{
						AppId:     appId2,
						PolicyStr: string(policyJson),
					},
					&models.PolicyJson{
						AppId:     appId3,
						PolicyStr: string(policyJson),
					},
				))
			})
		})
	})

	Describe("SavePolicy", func() {
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
				err = pdb.SaveAppPolicy(appId, policyJsonStr, policyGuid)
			})
			It("saves the policy", func() {
				Expect(err).NotTo(HaveOccurred())
				policyString, err := formatPolicyString(policyJsonStr)
				Expect(err).NotTo(HaveOccurred())
				Expect(formatPolicyString(getAppPolicy(appId))).To(Equal(policyString))
			})
		})

		Context("when a policy is already present for the app_id", func() {
			JustBeforeEach(func() {
				insertPolicy(appId, &models.ScalingPolicy{
					InstanceMin: 1,
					InstanceMax: 6,
					ScalingRules: []*models.ScalingRule{{
						MetricType:            testMetricName,
						BreachDurationSeconds: 180,
						Threshold:             1048576000,
						Operator:              ">",
						CoolDownSeconds:       300,
						Adjustment:            "+10%"}}}, policyGuid)

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

				err = pdb.SaveAppPolicy(appId, policyJsonStr, policyGuid)
			})
			It("updates the policy", func() {
				Expect(err).NotTo(HaveOccurred())
				policyString, err := formatPolicyString(policyJsonStr)
				Expect(err).NotTo(HaveOccurred())
				Expect(formatPolicyString(getAppPolicy(appId))).To(Equal(policyString))
			})
		})
	})

	Describe("DeletePolicy", func() {
		JustBeforeEach(func() {
			err = pdb.DeletePolicy(appId)
		})

		Context("when there is no policy in the table", func() {
			It("should not error", func() {
				Expect(err).NotTo(HaveOccurred())
			})
		})

		Context("when policy table is not empty", func() {
			BeforeEach(func() {
				scalingPolicy = &models.ScalingPolicy{InstanceMax: 1, InstanceMin: 6}
				insertPolicy(appId, scalingPolicy, policyGuid)
			})

			It("should delete the policy", func() {
				Expect(err).NotTo(HaveOccurred())
				policy, err := pdb.GetAppPolicy(appId)
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

	Describe("DeletePoliciesByPolicyGuid", func() {
		var updatedApps []string

		JustBeforeEach(func() {
			updatedApps, err = pdb.DeletePoliciesByPolicyGuid(policyGuid)
		})

		Context("when there is no policy in the table", func() {
			It("should not error", func() {
				Expect(err).NotTo(HaveOccurred())
				Expect(updatedApps).To(BeEmpty())
			})
		})

		Context("when policy table is not empty", func() {
			BeforeEach(func() {
				scalingPolicy = &models.ScalingPolicy{InstanceMax: 1, InstanceMin: 6}
				insertPolicyWithGuid(appId, scalingPolicy, policyGuid)
				insertPolicyWithGuid(appId2, scalingPolicy, anotherPolicyGuid)
			})

			It("should delete the policy with the specified policy guid", func() {
				Expect(err).NotTo(HaveOccurred())
				Expect(updatedApps).To(ConsistOf(appId))

				policy, err := pdb.GetAppPolicy(appId)
				Expect(err).NotTo(HaveOccurred())
				Expect(policy).To(BeNil())
				policy, err = pdb.GetAppPolicy(appId2)
				Expect(err).NotTo(HaveOccurred())
				Expect(policy).NotTo(BeNil())
			})
		})

		Context("when there is database error", func() {
			BeforeEach(func() {
				_ = pdb.Close()
			})
			It("should error", func() {
				Expect(err).To(HaveOccurred())
			})
		})
	})

	Describe("SetOrUpdateDefaultAppPolicy", func() {
		newPolicy := `{"new": "policy"}`
		var modifiedApps []string
		JustBeforeEach(func() {
			modifiedApps, err = pdb.SetOrUpdateDefaultAppPolicy([]string{appId, appId2, appId3}, policyGuid, newPolicy, policyGuid2)
		})

		Context("when policy table is empty", func() {
			It("inserts the policies for the apps", func() {
				Expect(err).NotTo(HaveOccurred())
				Expect(modifiedApps).To(ConsistOf(appId, appId2, appId3))
				Expect(getAppPolicy(appId)).To(MatchJSON(newPolicy))
				Expect(getAppPolicy(appId2)).To(MatchJSON(newPolicy))
				Expect(getAppPolicy(appId3)).To(MatchJSON(newPolicy))
			})
		})

		Context("when policy table is not empty", func() {
			BeforeEach(func() {
				scalingPolicy = &models.ScalingPolicy{InstanceMax: 1, InstanceMin: 6}
				insertPolicyWithGuid(appId, scalingPolicy, policyGuid)
				insertPolicyWithGuid(appId2, scalingPolicy, policyGuid2)
				insertPolicyWithGuid("unrelated-app-id", scalingPolicy, policyGuid3)
			})

			It("sets the default app policy on apps without a scaling policy and apps with the old policy", func() {
				Expect(modifiedApps).To(ConsistOf(appId, appId3))
				Expect(getAppPolicy(appId)).To(MatchJSON(newPolicy))
				Expect(getAppPolicy(appId2)).ToNot(MatchJSON(newPolicy))
				Expect(getAppPolicy(appId3)).To(MatchJSON(newPolicy))
				Expect(getAppPolicy("unrelated-app-id")).NotTo(MatchJSON(newPolicy))
			})
		})

		Context("when there is database error", func() {
			BeforeEach(func() {
				_ = pdb.Close()
			})
			It("should error", func() {
				Expect(err).To(HaveOccurred())
			})
		})
	})

	Describe("GetCredential", func() {

		JustBeforeEach(func() {
			credential, err = pdb.GetCredential(appId)
		})

		Context("when there is no credential for target application", func() {
			It("should not return any credentials", func() {
				Expect(err).To(Equal(sql.ErrNoRows))
				Expect(credential).To(BeNil())
			})
		})

		Context("when there is credential for target application", func() {
			BeforeEach(func() {
				err = insertCredential(appId, "username", "password")
			})

			It("Should get the credentials", func() {
				Expect(err).NotTo(HaveOccurred())
				Expect(credential.Password).To(Equal("password"))
				Expect(credential.Username).To(Equal("username"))
			})

		})
	})
	Describe("SaveCredential", func() {

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
				_ = pdb.Close()
			})
			It("should error", func() {
				Expect(err).To(HaveOccurred())
			})
		})
	})

})

func deleteCredentials(pdb *PolicySQLDB, logger lager.Logger, appIds ...string) {
	for _, appId := range appIds {
		err := pdb.DeleteCredential(appId)
		if err != nil {
			logger.Error(fmt.Sprintf("DeleteCredential app: %s", appId), err)
		}
	}
}

func deletePolicies(pdb *PolicySQLDB, logger lager.Logger, policyGuids ...string) {
	for _, policyGuid := range policyGuids {
		_, err := pdb.DeletePoliciesByPolicyGuid(policyGuid)
		if err != nil {
			logger.Error(fmt.Sprintf("DeletePoliciesByPolicyGuid policy: %s", policyGuid), err)
		}
	}
}

func deleteApps(pdb *PolicySQLDB, logger lager.Logger, appIds ...string) {
	for _, appId := range appIds {
		err := pdb.DeletePolicy(appId)
		if err != nil {
			logger.Error(fmt.Sprintf("DeletePolicy app: %s", appId), err)
		}
	}
}
