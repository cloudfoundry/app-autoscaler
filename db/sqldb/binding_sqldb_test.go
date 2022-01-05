package sqldb_test

import (
	"database/sql"
	"os"
	"strings"
	"time"

	"code.cloudfoundry.org/app-autoscaler/src/autoscaler/db"
	. "code.cloudfoundry.org/app-autoscaler/src/autoscaler/db/sqldb"
	"code.cloudfoundry.org/app-autoscaler/src/autoscaler/models"

	"code.cloudfoundry.org/lager"
	"github.com/go-sql-driver/mysql"
	"github.com/lib/pq"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("BindingSqldb", func() {
	var (
		bdb            *BindingSQLDB
		dbConfig       db.DatabaseConfig
		logger         lager.Logger
		err            error
		testInstanceId = "test-instance-id"
		testBindingId  = "test-binding-id"
		testAppId      = "test-app-id"
		testOrgGuid    = "test-org-guid"
		testOrgGuid2   = "test-org-guid-2"
		testSpaceGuid  = "test-space-guid"
		policyJsonStr  = `{
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
		policyGuid     = "test-policy-guid"
		policyJsonStr2 = `{
					"instance_max_count":4,
					"instance_min_count":1,
					"scaling_rules":[
					{
						"metric_type":"memoryutil",
						"breach_duration_secs":600,
						"threshold":10,
						"operator":">=",
						"cool_down_secs":300,
						"adjustment":"+1"
					}]
				}`
		policyGuid2 = "test-policy-guid-2"
	)

	BeforeEach(func() {
		logger = lager.NewLogger("binding-sqldb-test")
		dbConfig = db.DatabaseConfig{
			URL:                   os.Getenv("DBURL"),
			MaxOpenConnections:    10,
			MaxIdleConnections:    5,
			ConnectionMaxLifetime: 10 * time.Second,
			ConnectionMaxIdleTime: 10 * time.Second,
		}
	})

	Describe("NewBindingSQLDB", func() {
		JustBeforeEach(func() {
			bdb, err = NewBindingSQLDB(dbConfig, logger)
		})

		AfterEach(func() {
			if bdb != nil {
				err = bdb.Close()
				Expect(err).NotTo(HaveOccurred())
			}
		})

		Context("when db url is not correct", func() {
			BeforeEach(func() {
				if !strings.Contains(os.Getenv("DBURL"), "postgres") {
					Skip("Not configured for postgres")
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
				Expect(bdb).NotTo(BeNil())
			})
		})
	})

	Describe("CreateServiceInstance", func() {
		var (
			createdPolicyJsonStr string
			createdPolicyGuid    string
		)

		BeforeEach(func() {
			bdb, err = NewBindingSQLDB(dbConfig, logger)
			Expect(err).NotTo(HaveOccurred())

			createdPolicyGuid = policyJsonStr
			createdPolicyGuid = policyGuid

			cleanServiceBindingTable()
			cleanServiceInstanceTable()
		})
		AfterEach(func() {
			err = bdb.Close()
			Expect(err).NotTo(HaveOccurred())
		})
		JustBeforeEach(func() {
			err = bdb.CreateServiceInstance(models.ServiceInstance{ServiceInstanceId: testInstanceId, OrgId: testOrgGuid, SpaceId: testSpaceGuid, DefaultPolicy: createdPolicyJsonStr, DefaultPolicyGuid: createdPolicyGuid})
		})
		Context("When instance is being created first time", func() {
			It("should succeed", func() {
				Expect(err).NotTo(HaveOccurred())
				Expect(hasServiceInstance(testInstanceId)).To(BeTrue())
			})
		})
		Context("When instance is being created with an empty default policy", func() {
			BeforeEach(func() {
				createdPolicyGuid = ""
				createdPolicyGuid = ""
			})
			It("should save a NULL value to the database", func() {
				Expect(err).NotTo(HaveOccurred())
				Expect(hasServiceInstanceWithNullDefaultPolicy(testInstanceId)).To(BeTrue())
			})
		})
		Context("When instance already exists", func() {
			BeforeEach(func() {
				err = bdb.CreateServiceInstance(models.ServiceInstance{ServiceInstanceId: testInstanceId, OrgId: testOrgGuid, SpaceId: testSpaceGuid, DefaultPolicy: createdPolicyJsonStr, DefaultPolicyGuid: createdPolicyGuid})
				Expect(err).NotTo(HaveOccurred())
			})
			It("should error", func() {
				Expect(err).To(HaveOccurred())
				Expect(err).To(Equal(db.ErrAlreadyExists))
			})
		})
		Context("When a conflicting instance exists", func() {
			BeforeEach(func() {
				err = bdb.CreateServiceInstance(models.ServiceInstance{ServiceInstanceId: testInstanceId, OrgId: testOrgGuid2, SpaceId: testSpaceGuid, DefaultPolicy: createdPolicyJsonStr, DefaultPolicyGuid: createdPolicyGuid})
				Expect(err).NotTo(HaveOccurred())
			})
			It("should error", func() {
				Expect(err).To(HaveOccurred())
				Expect(err).To(Equal(db.ErrConflict))
			})
		})

	})

	Describe("UpdateServiceInstance", func() {
		var (
			updatedPolicyJsonStr string
			updatedPolicyGuid    string
		)

		BeforeEach(func() {
			bdb, err = NewBindingSQLDB(dbConfig, logger)
			Expect(err).NotTo(HaveOccurred())
			updatedPolicyJsonStr = policyJsonStr2
			updatedPolicyGuid = policyGuid2

			cleanServiceBindingTable()
			cleanServiceInstanceTable()
		})
		AfterEach(func() {
			err = bdb.Close()
			Expect(err).NotTo(HaveOccurred())
		})
		JustBeforeEach(func() {
			err = bdb.UpdateServiceInstance(models.ServiceInstance{ServiceInstanceId: testInstanceId, OrgId: testOrgGuid, SpaceId: testSpaceGuid, DefaultPolicy: updatedPolicyJsonStr, DefaultPolicyGuid: updatedPolicyGuid})
		})
		Context("when the instance to be updated does not exist", func() {
			It("should error", func() {
				Expect(hasServiceInstance(testInstanceId)).To(BeFalse())
				Expect(err).To(HaveOccurred())
				Expect(err).To(Equal(db.ErrDoesNotExist))
			})
		})
		Context("when the instances to be update exists", func() {
			BeforeEach(func() {
				err = bdb.CreateServiceInstance(models.ServiceInstance{ServiceInstanceId: testInstanceId, OrgId: testOrgGuid, SpaceId: testSpaceGuid, DefaultPolicy: policyJsonStr, DefaultPolicyGuid: policyGuid})
				Expect(err).NotTo(HaveOccurred())
			})
			Context("and the default policy is updated", func() {
				It("should save the update to the database", func() {
					Expect(err).NotTo(HaveOccurred())
					serviceInstance, err := bdb.GetServiceInstance(testInstanceId)
					Expect(err).NotTo(HaveOccurred())
					Expect(serviceInstance.ServiceInstanceId).To(Equal(testInstanceId))
					expectServiceInstancesToEqual(serviceInstance, &models.ServiceInstance{ServiceInstanceId: testInstanceId, OrgId: testOrgGuid, SpaceId: testSpaceGuid, DefaultPolicy: updatedPolicyJsonStr, DefaultPolicyGuid: updatedPolicyGuid})
				})
			})
			Context("and it is being updated with an empty default policy", func() {
				BeforeEach(func() {
					updatedPolicyJsonStr = ""
					updatedPolicyGuid = ""
				})
				It("should save a NULL value to the database", func() {
					Expect(err).NotTo(HaveOccurred())
					Expect(hasServiceInstanceWithNullDefaultPolicy(testInstanceId)).To(BeTrue())
				})
			})
			Context("when another instance exists", func() {
				BeforeEach(func() {
					err = bdb.CreateServiceInstance(models.ServiceInstance{ServiceInstanceId: testInstanceId + "2", OrgId: testOrgGuid2, SpaceId: testSpaceGuid, DefaultPolicy: policyJsonStr, DefaultPolicyGuid: policyGuid + "-other"})
					Expect(err).NotTo(HaveOccurred())
				})
				It("should not change the other service instance", func() {
					Expect(err).NotTo(HaveOccurred())
					serviceInstance, err := bdb.GetServiceInstance(testInstanceId + "2")
					Expect(err).NotTo(HaveOccurred())
					expectServiceInstancesToEqual(serviceInstance, &models.ServiceInstance{ServiceInstanceId: testInstanceId + "2", OrgId: testOrgGuid2, SpaceId: testSpaceGuid, DefaultPolicy: policyJsonStr, DefaultPolicyGuid: policyGuid + "-other"})
				})
			})
		})

	})

	Describe("DeleteServiceInstance", func() {
		BeforeEach(func() {
			bdb, err = NewBindingSQLDB(dbConfig, logger)
			Expect(err).NotTo(HaveOccurred())

			cleanServiceBindingTable()
			cleanServiceInstanceTable()
		})
		AfterEach(func() {
			err = bdb.Close()
			Expect(err).NotTo(HaveOccurred())
		})
		JustBeforeEach(func() {
			err = bdb.DeleteServiceInstance(testInstanceId)
		})
		Context("When instance doesn't exists", func() {
			It("should error", func() {
				Expect(err).To(HaveOccurred())
				Expect(err).To(Equal(db.ErrDoesNotExist))
			})
		})
		Context("When instance is present", func() {
			BeforeEach(func() {
				err = bdb.CreateServiceInstance(models.ServiceInstance{ServiceInstanceId: testInstanceId, OrgId: testOrgGuid, SpaceId: testSpaceGuid, DefaultPolicy: policyJsonStr, DefaultPolicyGuid: policyGuid})
				Expect(err).NotTo(HaveOccurred())
			})
			It("should succeed", func() {
				Expect(err).NotTo(HaveOccurred())
				Expect(hasServiceInstance(testInstanceId)).NotTo(BeTrue())
			})
		})
	})

	Describe("GetServiceInstance", func() {
		var retrievedServiceInstance *models.ServiceInstance
		BeforeEach(func() {
			bdb, err = NewBindingSQLDB(dbConfig, logger)
			Expect(err).NotTo(HaveOccurred())

			cleanServiceBindingTable()
			cleanServiceInstanceTable()
		})
		AfterEach(func() {
			err = bdb.Close()
			Expect(err).NotTo(HaveOccurred())
		})
		JustBeforeEach(func() {
			retrievedServiceInstance, err = bdb.GetServiceInstance(testInstanceId)
		})
		Context("when the service instance does not exist", func() {
			It("should return an error", func() {
				Expect(err).To(HaveOccurred())
				Expect(err).To(Equal(db.ErrDoesNotExist))
				Expect(retrievedServiceInstance).To(BeNil())
			})
		})

		Context("when the service instance exists", func() {
			BeforeEach(func() {
				err = bdb.CreateServiceInstance(models.ServiceInstance{ServiceInstanceId: testInstanceId, OrgId: testOrgGuid, SpaceId: testSpaceGuid, DefaultPolicy: policyJsonStr, DefaultPolicyGuid: policyGuid})
				Expect(err).NotTo(HaveOccurred())
			})
			It("should return what was created", func() {
				expectServiceInstancesToEqual(retrievedServiceInstance, &models.ServiceInstance{ServiceInstanceId: testInstanceId, OrgId: testOrgGuid, SpaceId: testSpaceGuid, DefaultPolicy: policyJsonStr, DefaultPolicyGuid: policyGuid})
			})
		})
		Context("when the service instance doesn't have a default policy", func() {
			BeforeEach(func() {
				err = bdb.CreateServiceInstance(models.ServiceInstance{ServiceInstanceId: testInstanceId, OrgId: testOrgGuid, SpaceId: testSpaceGuid})
				Expect(err).NotTo(HaveOccurred())
			})
			It("should return an empty default policy", func() {
				Expect(err).NotTo(HaveOccurred())
				Expect(retrievedServiceInstance.DefaultPolicy).To(BeEmpty())
				Expect(retrievedServiceInstance.DefaultPolicyGuid).To(BeEmpty())
			})
		})

	})

	Describe("GetServiceInstanceByAppId", func() {
		var retrievedServiceInstance *models.ServiceInstance
		BeforeEach(func() {
			bdb, err = NewBindingSQLDB(dbConfig, logger)
			Expect(err).NotTo(HaveOccurred())

			cleanServiceBindingTable()
			cleanServiceInstanceTable()
		})
		AfterEach(func() {
			err = bdb.Close()
			Expect(err).NotTo(HaveOccurred())
		})
		JustBeforeEach(func() {
			retrievedServiceInstance, err = bdb.GetServiceInstanceByAppId(testAppId)
		})
		Context("when the app is not bound to a service instance", func() {
			It("should return an error", func() {
				Expect(err).To(HaveOccurred())
				Expect(err).To(Equal(db.ErrDoesNotExist))
				Expect(retrievedServiceInstance).To(BeNil())
			})
		})

		Context("when the app is bound to a service instance", func() {
			BeforeEach(func() {
				err = bdb.CreateServiceInstance(models.ServiceInstance{ServiceInstanceId: testInstanceId, OrgId: testOrgGuid, SpaceId: testSpaceGuid, DefaultPolicy: policyJsonStr, DefaultPolicyGuid: policyGuid})
				Expect(err).NotTo(HaveOccurred())
				err = bdb.CreateServiceBinding(testBindingId, testInstanceId, testAppId)
			})
			It("should return what was created", func() {
				expectServiceInstancesToEqual(retrievedServiceInstance, &models.ServiceInstance{ServiceInstanceId: testInstanceId, OrgId: testOrgGuid, SpaceId: testSpaceGuid, DefaultPolicy: policyJsonStr, DefaultPolicyGuid: policyGuid})
			})
		})
	})

	Describe("CreateServiceBinding", func() {
		BeforeEach(func() {
			bdb, err = NewBindingSQLDB(dbConfig, logger)
			Expect(err).NotTo(HaveOccurred())

			cleanServiceBindingTable()
			cleanServiceInstanceTable()
		})
		AfterEach(func() {
			err = bdb.Close()
			Expect(err).NotTo(HaveOccurred())
		})
		JustBeforeEach(func() {
			err = bdb.CreateServiceBinding(testBindingId, testInstanceId, testAppId)
		})
		Context("When service instance doesn't exist", func() {
			It("should error", func() {
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("foreign key constraint"))
				Expect(hasServiceBinding(testBindingId, testInstanceId)).NotTo(BeTrue())
			})
		})
		Context("When service instance exists", func() {
			BeforeEach(func() {
				err = bdb.CreateServiceInstance(models.ServiceInstance{ServiceInstanceId: testInstanceId, OrgId: testOrgGuid, SpaceId: testSpaceGuid, DefaultPolicy: policyJsonStr, DefaultPolicyGuid: policyGuid})
				Expect(err).NotTo(HaveOccurred())
			})
			AfterEach(func() {
				cleanServiceBindingTable()
				cleanServiceInstanceTable()
			})
			Context("When service binding is being created first time", func() {
				It("should succeed", func() {
					Expect(err).NotTo(HaveOccurred())
					Expect(hasServiceBinding(testBindingId, testInstanceId)).To(BeTrue())
				})
			})
			Context("When service binding already exists", func() {
				It("should error", func() {
					err := bdb.CreateServiceBinding(testBindingId, testInstanceId, testAppId)
					Expect(err).To(HaveOccurred())
					Expect(err).To(Equal(db.ErrAlreadyExists))
				})
			})

		})
	})

	Describe("DeleteServiceBinding", func() {
		BeforeEach(func() {
			bdb, err = NewBindingSQLDB(dbConfig, logger)
			Expect(err).NotTo(HaveOccurred())

			cleanServiceBindingTable()
			cleanServiceInstanceTable()
		})
		AfterEach(func() {
			err = bdb.Close()
			Expect(err).NotTo(HaveOccurred())
		})
		JustBeforeEach(func() {
			err = bdb.DeleteServiceBinding(testBindingId)
		})
		Context("When service instance doesn't exist", func() {
			It("should error", func() {
				Expect(err).To(HaveOccurred())
				Expect(err).To(Equal(db.ErrDoesNotExist))
			})
		})
		Context("When service instance exists", func() {
			BeforeEach(func() {
				err = bdb.CreateServiceInstance(models.ServiceInstance{ServiceInstanceId: testInstanceId, OrgId: testOrgGuid, SpaceId: testSpaceGuid, DefaultPolicy: policyJsonStr, DefaultPolicyGuid: policyGuid})
				Expect(err).NotTo(HaveOccurred())
			})
			AfterEach(func() {
				cleanServiceBindingTable()
				cleanServiceInstanceTable()
			})
			Context("When service binding doesn't exists", func() {
				It("should error", func() {
					Expect(err).To(HaveOccurred())
					Expect(err).To(Equal(db.ErrDoesNotExist))
				})
			})
			Context("When service binding is present", func() {
				BeforeEach(func() {
					err = bdb.CreateServiceBinding(testBindingId, testInstanceId, testAppId)
					Expect(err).NotTo(HaveOccurred())
				})
				It("should succeed", func() {
					Expect(err).NotTo(HaveOccurred())
					Expect(hasServiceBinding(testBindingId, testInstanceId)).NotTo(BeTrue())
				})
			})

		})
	})

	Describe("DeleteServiceBindingByAppId", func() {
		BeforeEach(func() {
			bdb, err = NewBindingSQLDB(dbConfig, logger)
			Expect(err).NotTo(HaveOccurred())

			cleanServiceBindingTable()
			cleanServiceInstanceTable()

			err = bdb.CreateServiceInstance(models.ServiceInstance{ServiceInstanceId: testInstanceId, OrgId: testOrgGuid, SpaceId: testSpaceGuid, DefaultPolicy: policyJsonStr, DefaultPolicyGuid: policyGuid})
			Expect(err).NotTo(HaveOccurred())
			err = bdb.CreateServiceBinding(testBindingId, testInstanceId, testAppId)
			Expect(err).NotTo(HaveOccurred())
			err = bdb.DeleteServiceBindingByAppId(testAppId)
		})
		AfterEach(func() {
			cleanServiceBindingTable()
			cleanServiceInstanceTable()
			err = bdb.Close()
			Expect(err).NotTo(HaveOccurred())
		})
		It("should succeed", func() {
			Expect(err).NotTo(HaveOccurred())
			Expect(hasServiceBinding(testAppId, testInstanceId)).NotTo(BeTrue())
		})
	})

	Describe("CheckServiceBinding", func() {
		var bindingExists bool
		BeforeEach(func() {
			bdb, err = NewBindingSQLDB(dbConfig, logger)
			Expect(err).NotTo(HaveOccurred())

			cleanServiceBindingTable()
			cleanServiceInstanceTable()
		})
		JustBeforeEach(func() {
			bindingExists = bdb.CheckServiceBinding(testAppId)
		})
		AfterEach(func() {
			cleanServiceBindingTable()
			cleanServiceInstanceTable()
			err = bdb.Close()
			Expect(err).NotTo(HaveOccurred())
		})
		Context("when binding for bindingId exists", func() {
			BeforeEach(func() {
				err = bdb.CreateServiceInstance(models.ServiceInstance{ServiceInstanceId: testInstanceId, OrgId: testOrgGuid, SpaceId: testSpaceGuid, DefaultPolicy: policyJsonStr, DefaultPolicyGuid: policyGuid})
				Expect(err).NotTo(HaveOccurred())
				err = bdb.CreateServiceBinding(testBindingId, testInstanceId, testAppId)
				Expect(err).NotTo(HaveOccurred())
			})
			It("should return true", func() {
				Expect(err).NotTo(HaveOccurred())
				Expect(bindingExists).To(BeTrue())
			})
		})
		Context("when binding for bindingId does not exist", func() {
			It("should return error", func() {
				Expect(err).NotTo(HaveOccurred())
				Expect(bindingExists).To(BeFalse())
			})
		})

	})

	Describe("GetAppIdByBindingId", func() {
		var appIdResult string
		BeforeEach(func() {
			bdb, err = NewBindingSQLDB(dbConfig, logger)
			Expect(err).NotTo(HaveOccurred())

			cleanServiceBindingTable()
			cleanServiceInstanceTable()
		})
		JustBeforeEach(func() {

			appIdResult, err = bdb.GetAppIdByBindingId(testBindingId)
		})
		AfterEach(func() {
			cleanServiceBindingTable()
			cleanServiceInstanceTable()
			err = bdb.Close()
			Expect(err).NotTo(HaveOccurred())
		})
		Context("when binding for bindingId exists", func() {
			BeforeEach(func() {
				err = bdb.CreateServiceInstance(models.ServiceInstance{ServiceInstanceId: testInstanceId, OrgId: testOrgGuid, SpaceId: testSpaceGuid, DefaultPolicy: policyJsonStr, DefaultPolicyGuid: policyGuid})
				Expect(err).NotTo(HaveOccurred())
				err = bdb.CreateServiceBinding(testBindingId, testInstanceId, testAppId)
				Expect(err).NotTo(HaveOccurred())
			})
			It("should succeed", func() {
				Expect(err).NotTo(HaveOccurred())
				Expect(appIdResult).To(Equal(testAppId))
			})
		})
		Context("when binding for bindingId does not exist", func() {
			It("should return error", func() {
				Expect(err).To(Equal(sql.ErrNoRows))
			})
		})

	})

	Describe("GetAppIdsByInstanceId", func() {
		var results []string
		BeforeEach(func() {
			bdb, err = NewBindingSQLDB(dbConfig, logger)
			Expect(err).NotTo(HaveOccurred())

			cleanServiceBindingTable()
			cleanServiceInstanceTable()
		})
		JustBeforeEach(func() {

			results, err = bdb.GetAppIdsByInstanceId(testInstanceId)
		})
		AfterEach(func() {
			cleanServiceBindingTable()
			cleanServiceInstanceTable()
			err = bdb.Close()
			Expect(err).NotTo(HaveOccurred())
		})
		Context("when binding for bindingId exists", func() {
			BeforeEach(func() {
				err = bdb.CreateServiceInstance(models.ServiceInstance{ServiceInstanceId: testInstanceId, OrgId: testOrgGuid, SpaceId: testSpaceGuid, DefaultPolicy: policyJsonStr, DefaultPolicyGuid: policyGuid})
				Expect(err).NotTo(HaveOccurred())
				err = bdb.CreateServiceBinding(testBindingId, testInstanceId, testAppId)
				Expect(err).NotTo(HaveOccurred())
				err = bdb.CreateServiceBinding(testBindingId+"2", testInstanceId, testAppId+"2")
				Expect(err).NotTo(HaveOccurred())

				// other unrelated service instance with bindings
				err = bdb.CreateServiceInstance(models.ServiceInstance{ServiceInstanceId: testInstanceId + "3", OrgId: testOrgGuid, SpaceId: testSpaceGuid, DefaultPolicy: policyJsonStr, DefaultPolicyGuid: policyGuid})
				Expect(err).NotTo(HaveOccurred())
				err = bdb.CreateServiceBinding(testBindingId+"3", testInstanceId+"3", testAppId+"3")
				Expect(err).NotTo(HaveOccurred())
			})
			It("should succeed", func() {
				Expect(err).NotTo(HaveOccurred())
				Expect(results).To(ConsistOf(testAppId, testAppId+"2"))
			})
		})
		Context("when binding for bindingId does not exist", func() {
			It("should not return an error, but an empty result", func() {
				Expect(err).NotTo(HaveOccurred())
				Expect(results).To(BeEmpty())
			})
		})

	})

	Describe("CountServiceInstancesInOrg", func() {
		var serviceInstanceCount int
		BeforeEach(func() {
			bdb, err = NewBindingSQLDB(dbConfig, logger)
			Expect(err).NotTo(HaveOccurred())

			cleanServiceBindingTable()
			cleanServiceInstanceTable()
		})
		JustBeforeEach(func() {
			serviceInstanceCount, err = bdb.CountServiceInstancesInOrg(testOrgGuid)
		})
		AfterEach(func() {
			cleanServiceBindingTable()
			cleanServiceInstanceTable()
			err = bdb.Close()
			Expect(err).NotTo(HaveOccurred())
		})
		Context("when no service instance exist", func() {
			It("returns 0", func() {
				Expect(err).ToNot(HaveOccurred())
				Expect(serviceInstanceCount).To(Equal(0))
			})
		})
		Context("when service instances exist", func() {
			BeforeEach(func() {
				err = bdb.CreateServiceInstance(models.ServiceInstance{testInstanceId, testOrgGuid, testSpaceGuid, policyJsonStr, policyGuid})
				Expect(err).NotTo(HaveOccurred())
				err = bdb.CreateServiceInstance(models.ServiceInstance{testInstanceId + "2", testOrgGuid, testSpaceGuid, policyJsonStr, policyGuid})
				Expect(err).NotTo(HaveOccurred())
				err = bdb.CreateServiceInstance(models.ServiceInstance{testInstanceId + "3", testOrgGuid2, testSpaceGuid, policyJsonStr, policyGuid})
				Expect(err).NotTo(HaveOccurred())

			})
			It("returns the correct count", func() {
				Expect(err).ToNot(HaveOccurred())
				Expect(serviceInstanceCount).To(Equal(2))
			})
		})

	})

	Describe("GetBindingIdsByInstanceId", func() {
		var bindingIds []string
		var serviceInstanceCount int

		BeforeEach(func() {
			bdb, err = NewBindingSQLDB(dbConfig, logger)
			Expect(err).NotTo(HaveOccurred())

			cleanServiceBindingTable()
			cleanServiceInstanceTable()
		})

		JustBeforeEach(func() {
			bindingIds, err = bdb.GetBindingIdsByInstanceId(testInstanceId)
			serviceInstanceCount, err = bdb.CountServiceInstancesInOrg(testOrgGuid)
		})
		AfterEach(func() {
			cleanServiceBindingTable()
			cleanServiceInstanceTable()
			err = bdb.Close()
			Expect(err).NotTo(HaveOccurred())
		})
		Context("when no service instance exist", func() {
			It("returns 0", func() {
				Expect(err).ToNot(HaveOccurred())
				Expect(serviceInstanceCount).To(Equal(0))
			})
		})
		Context("when service instances exist", func() {
			BeforeEach(func() {
				err = bdb.CreateServiceInstance(models.ServiceInstance{testInstanceId, testOrgGuid, testSpaceGuid, policyJsonStr, policyGuid})
				Expect(err).NotTo(HaveOccurred())
				err = bdb.CreateServiceBinding(testBindingId, testInstanceId, testAppId)
				Expect(err).NotTo(HaveOccurred())
				err = bdb.CreateServiceBinding(testBindingId+"2", testInstanceId, testAppId+"2")
				Expect(err).NotTo(HaveOccurred())

				// other unrelated service instance with bindings
				err = bdb.CreateServiceInstance(models.ServiceInstance{testInstanceId + "3", testOrgGuid, testSpaceGuid, policyJsonStr, policyGuid})
				Expect(err).NotTo(HaveOccurred())
				err = bdb.CreateServiceBinding(testBindingId+"3", testInstanceId+"3", testAppId+"3")
				Expect(err).NotTo(HaveOccurred())
			})
			It("should succeed", func() {
				Expect(err).NotTo(HaveOccurred())
				Expect(bindingIds).To(ConsistOf(testBindingId, testBindingId+"2"))
			})
		})
		Context("when binding for bindingId does not exist", func() {
			It("should not return an error, but an empty result", func() {
				Expect(err).NotTo(HaveOccurred())
				Expect(bindingIds).To(BeEmpty())
			})
		})
	})
})
