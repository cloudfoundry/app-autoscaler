package sqldb_test

import (
	"autoscaler/db"
	. "autoscaler/db/sqldb"
	"autoscaler/models"

	"code.cloudfoundry.org/lager"
	"github.com/lib/pq"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"golang.org/x/crypto/bcrypt"

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
		appId             string
		policies          []*models.PolicyJson
		testMetricName    string = "TestMetricName"
		username          string
		password          string
		encryptedPassword []byte
		encryptedUsername []byte
		isValid           bool
		isMetricTypeValid bool
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
			It("should error", func() {
				Expect(err).To(BeAssignableToTypeOf(&pq.Error{}))
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

	Describe("GetCustomMetricsCreds", func() {
		BeforeEach(func() {
			pdb, err = NewPolicySQLDB(dbConfig, logger)
			Expect(err).NotTo(HaveOccurred())

			cleanCredentialsTable()
		})

		AfterEach(func() {
			err = pdb.Close()
			Expect(err).NotTo(HaveOccurred())
		})

		JustBeforeEach(func() {
			username, password, err = pdb.GetCustomMetricsCreds("an-app-id")
		})

		Context("when credentials table is empty", func() {
			It("should not return any credentials", func() {
				Expect(err).To(HaveOccurred())
				Expect(password).To(BeEmpty())
				Expect(username).To(BeEmpty())
			})
		})

		Context("when policy table is not empty", func() {
			BeforeEach(func() {
				insertCustomMetricsBindingCredentials("an-app-id", "username", "password")
			})

			It("Should get the password", func() {
				Expect(err).NotTo(HaveOccurred())
				Expect(password).To(Equal("password"))
				Expect(username).To(Equal("username"))
			})

		})
	})

	Describe("ValidateCustomMetricsCreds", func() {
		BeforeEach(func() {
			pdb, err = NewPolicySQLDB(dbConfig, logger)
			Expect(err).NotTo(HaveOccurred())

			cleanCredentialsTable()
			encryptedUsername, _ = bcrypt.GenerateFromPassword([]byte("username"), 8)
			encryptedPassword, _ = bcrypt.GenerateFromPassword([]byte("password"), 8)
		})

		AfterEach(func() {
			err = pdb.Close()
			Expect(err).NotTo(HaveOccurred())
		})

		JustBeforeEach(func() {
			isValid = pdb.ValidateCustomMetricsCreds("an-app-id", "username", "password")
		})

		Context("when policy table is not empty", func() {
			BeforeEach(func() {
				insertCustomMetricsBindingCredentials("an-app-id", string(encryptedUsername[:]), string(encryptedPassword[:]))
			})

			It("Should get the password", func() {
				Expect(err).NotTo(HaveOccurred())
				Expect(isValid).To(BeTrue())
			})

		})
	})

	Describe("ValidateCustomMetricTypes", func() {
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
			metricsConsumer := &models.MetricsConsumer{
				AppGUID:       "an-app-id",
				InstanceIndex: 1,
				CustomMetrics: []*models.CustomMetric{{
					Name:          "testMetricName",
					Value:         12,
					Unit:          "unit",
					InstanceIndex: 1,
					AppGUID:       "an-app-id",
				}},
			}
			isMetricTypeValid, err = pdb.ValidateCustomMetricTypes("an-app-id", metricsConsumer)
		})

		Context("when policy table does not contain policy", func() {

			It("Should validate fail to validate metrictypes", func() {
				Expect(isMetricTypeValid).To(BeFalse())
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(Equal("no policy found"))
			})

		})

		Context("when policy table contains policy with matched metrictypes", func() {
			BeforeEach(func() {
				insertPolicy("an-app-id", &models.ScalingPolicy{
					InstanceMin: 1,
					InstanceMax: 6,
					ScalingRules: []*models.ScalingRule{{
						MetricType:            "testMetricName",
						BreachDurationSeconds: 180,
						Threshold:             1048,
						Operator:              ">",
						CoolDownSeconds:       300,
						Adjustment:            "+1"}},
				})
			})

			It("Should validate the metrictypes successfully", func() {
				Expect(isMetricTypeValid).To(BeTrue())
				Expect(err).NotTo(HaveOccurred())
			})

		})

		Context("when policy table contains policy with unmatched metrictypes", func() {
			BeforeEach(func() {
				insertPolicy("an-app-id", &models.ScalingPolicy{
					InstanceMin: 1,
					InstanceMax: 6,
					ScalingRules: []*models.ScalingRule{{
						MetricType:            "testMetricNameUnMatched",
						BreachDurationSeconds: 180,
						Threshold:             1048,
						Operator:              ">",
						CoolDownSeconds:       300,
						Adjustment:            "+1"}},
				})
			})

			It("Should validate the metrictypes", func() {
				Expect(isMetricTypeValid).To(BeFalse())
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(Equal("CustomMetric name does not match with metric defined in policy"))
			})

		})

	})

})
