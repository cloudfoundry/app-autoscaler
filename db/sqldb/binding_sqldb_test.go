package sqldb_test

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"reflect"
	"strings"
	"time"
	"unsafe"

	"code.cloudfoundry.org/app-autoscaler/src/autoscaler/testhelpers"

	"github.com/go-sql-driver/mysql"

	"code.cloudfoundry.org/app-autoscaler/src/autoscaler/db"
	. "code.cloudfoundry.org/app-autoscaler/src/autoscaler/db/sqldb"
	"code.cloudfoundry.org/app-autoscaler/src/autoscaler/models"

	"code.cloudfoundry.org/lager/v3"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("BindingSqldb", func() {
	var (
		bdb            *BindingSQLDB
		dbConfig       db.DatabaseConfig
		dbHost         = os.Getenv("DB_HOST")
		logger         lager.Logger
		err            error
		testInstanceId = addProcessIdTo("test-instance-id")
		testBindingId  = addProcessIdTo("test-binding-id")
		testAppId      = addProcessIdTo("test-app-id")
		testOrgGuid    = addProcessIdTo("test-org-guid")
		testOrgGuid2   = addProcessIdTo("test-org-guid-2")
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
		policyGuid     = addProcessIdTo("test-policy-guid")
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
		policyGuid2           = addProcessIdTo("test-policy-guid-2")
		testInstanceId2       = testInstanceId + "2"
		testInstanceId3       = testInstanceId + "3"
		testAppId2            = testAppId + "2"
		testAppId3            = testAppId + "3"
		testBindingId3        = testBindingId + "3"
		testBindingId2        = testBindingId + "2"
		customMetricsStrategy = models.DefaultCustomMetricsStrategy
	)

	dbUrl := testhelpers.GetDbUrl()
	BeforeEach(func() {
		logger = lager.NewLogger("binding-sqldb-test")
		logger.RegisterSink(lager.NewWriterSink(GinkgoWriter, lager.DEBUG))
		dbConfig = db.DatabaseConfig{
			URL:                   dbUrl,
			MaxOpenConnections:    10,
			MaxIdleConnections:    5,
			ConnectionMaxLifetime: 10 * time.Second,
			ConnectionMaxIdleTime: 10 * time.Second,
		}
		bdb, err = NewBindingSQLDB(dbConfig, logger)
		Expect(err).NotTo(HaveOccurred())
		DeferCleanup(func() error { return bdb.Close() })

		cleanupInstances := func(instanceIds ...string) {
			for _, instanceId := range instanceIds {
				err := bdb.DeleteServiceInstance(context.Background(), instanceId)
				if err != nil {
					logger.Info(fmt.Sprintf("Cleaning service instance %s failed:%s", instanceId, err.Error()))
				}
			}
		}
		cleanupBindings := func(bindingIds ...string) {
			for _, bindingId := range bindingIds {
				err := bdb.DeleteServiceBinding(context.Background(), bindingId)
				if err != nil {
					logger.Info(fmt.Sprintf("Cleaning service binding %s failed:%s", bindingId, err.Error()))
				}
			}
		}
		cleanupBindings(testBindingId, testBindingId2, testBindingId3)
		cleanupInstances(testInstanceId, testInstanceId2, testInstanceId3)
	})

	Describe("NewBindingSQLDB", func() {

		Context("when db url is not correct", func() {
			BeforeEach(func() {
				if !strings.Contains(dbUrl, "postgres") {
					Skip("Not configured for postgres")
				}
				dbConfig.URL = "postgres://not-exist-user:not-exist-password@" + dbHost + "/autoscaler?sslmode=disable"
			})
			It("should throw an error", func() {
				abdb, err := NewBindingSQLDB(dbConfig, logger)
				Expect(err).To(HaveOccurred())
				if abdb != nil {
					err = bdb.Close()
					Expect(err).NotTo(HaveOccurred())
				}
			})
		})

		Context("when mysql db url is not correct", func() {
			BeforeEach(func() {
				if strings.Contains(dbUrl, "postgres") {
					Skip("Not configured for mysql")
				}
				dbConfig.URL = "not-exist-user:not-exist-password@tcp(" + dbHost + ")/autoscaler?tls=false"
			})
			It("should throw an error", func() {
				abdb, err := NewBindingSQLDB(dbConfig, logger)
				Expect(err).To(BeAssignableToTypeOf(&mysql.MySQLError{}))
				if abdb != nil {
					err = bdb.Close()
					Expect(err).NotTo(HaveOccurred())
				}
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
			createdPolicyGuid = policyJsonStr
			createdPolicyGuid = policyGuid
		})

		JustBeforeEach(func() {
			err = bdb.CreateServiceInstance(context.Background(), models.ServiceInstance{ServiceInstanceId: testInstanceId, OrgId: testOrgGuid, SpaceId: testSpaceGuid, DefaultPolicy: createdPolicyJsonStr, DefaultPolicyGuid: createdPolicyGuid})
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
				err = bdb.CreateServiceInstance(context.Background(), models.ServiceInstance{ServiceInstanceId: testInstanceId, OrgId: testOrgGuid, SpaceId: testSpaceGuid, DefaultPolicy: createdPolicyJsonStr, DefaultPolicyGuid: createdPolicyGuid})
				Expect(err).NotTo(HaveOccurred())
			})
			It("should error", func() {
				Expect(err).To(HaveOccurred())
				Expect(err).To(Equal(db.ErrAlreadyExists))
			})
		})
		Context("When a conflicting instance exists", func() {
			BeforeEach(func() {
				err = bdb.CreateServiceInstance(context.Background(), models.ServiceInstance{ServiceInstanceId: testInstanceId, OrgId: testOrgGuid2, SpaceId: testSpaceGuid, DefaultPolicy: createdPolicyJsonStr, DefaultPolicyGuid: createdPolicyGuid})
				DeferCleanup(func() error { return bdb.DeleteServiceInstance(context.Background(), testInstanceId) })
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
			updatedPolicyJsonStr = policyJsonStr2
			updatedPolicyGuid = policyGuid2

		})
		AfterEach(func() {
			err = bdb.Close()
			Expect(err).NotTo(HaveOccurred())
		})
		JustBeforeEach(func() {
			err = bdb.UpdateServiceInstance(context.Background(), models.ServiceInstance{ServiceInstanceId: testInstanceId, OrgId: testOrgGuid, SpaceId: testSpaceGuid, DefaultPolicy: updatedPolicyJsonStr, DefaultPolicyGuid: updatedPolicyGuid})
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
				err = bdb.CreateServiceInstance(context.Background(), models.ServiceInstance{ServiceInstanceId: testInstanceId, OrgId: testOrgGuid, SpaceId: testSpaceGuid, DefaultPolicy: policyJsonStr, DefaultPolicyGuid: policyGuid})
				Expect(err).NotTo(HaveOccurred())
			})
			Context("and the default policy is updated", func() {
				It("should save the update to the database", func() {
					Expect(err).NotTo(HaveOccurred())
					serviceInstance, err := bdb.GetServiceInstance(context.Background(), testInstanceId)
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
					err = bdb.CreateServiceInstance(context.Background(), models.ServiceInstance{ServiceInstanceId: testInstanceId2, OrgId: testOrgGuid2, SpaceId: testSpaceGuid, DefaultPolicy: policyJsonStr, DefaultPolicyGuid: policyGuid + "-other"})
					Expect(err).NotTo(HaveOccurred())
				})
				It("should not change the other service instance", func() {
					Expect(err).NotTo(HaveOccurred())
					serviceInstance, err := bdb.GetServiceInstance(context.Background(), testInstanceId2)
					Expect(err).NotTo(HaveOccurred())
					expectServiceInstancesToEqual(serviceInstance, &models.ServiceInstance{ServiceInstanceId: testInstanceId2, OrgId: testOrgGuid2, SpaceId: testSpaceGuid, DefaultPolicy: policyJsonStr, DefaultPolicyGuid: policyGuid + "-other"})
				})
			})
		})

	})

	Describe("DeleteServiceInstance", func() {
		JustBeforeEach(func() {
			err = bdb.DeleteServiceInstance(context.Background(), testInstanceId)
		})
		Context("When instance doesn't exists", func() {
			It("should error", func() {
				Expect(err).To(HaveOccurred())
				Expect(err).To(Equal(db.ErrDoesNotExist))
			})
		})
		Context("When instance is present", func() {
			BeforeEach(func() {
				err = bdb.CreateServiceInstance(context.Background(), models.ServiceInstance{ServiceInstanceId: testInstanceId, OrgId: testOrgGuid, SpaceId: testSpaceGuid, DefaultPolicy: policyJsonStr, DefaultPolicyGuid: policyGuid})
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
		JustBeforeEach(func() {
			retrievedServiceInstance, err = bdb.GetServiceInstance(context.Background(), testInstanceId)
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
				err = bdb.CreateServiceInstance(context.Background(), models.ServiceInstance{ServiceInstanceId: testInstanceId, OrgId: testOrgGuid, SpaceId: testSpaceGuid, DefaultPolicy: policyJsonStr, DefaultPolicyGuid: policyGuid})
				Expect(err).NotTo(HaveOccurred())
			})
			It("should return what was created", func() {
				expectServiceInstancesToEqual(retrievedServiceInstance, &models.ServiceInstance{ServiceInstanceId: testInstanceId, OrgId: testOrgGuid, SpaceId: testSpaceGuid, DefaultPolicy: policyJsonStr, DefaultPolicyGuid: policyGuid})
			})
		})
		Context("when the service instance doesn't have a default policy", func() {
			BeforeEach(func() {
				err = bdb.CreateServiceInstance(context.Background(), models.ServiceInstance{ServiceInstanceId: testInstanceId, OrgId: testOrgGuid, SpaceId: testSpaceGuid})
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
				err = bdb.CreateServiceInstance(context.Background(), models.ServiceInstance{ServiceInstanceId: testInstanceId, OrgId: testOrgGuid, SpaceId: testSpaceGuid, DefaultPolicy: policyJsonStr, DefaultPolicyGuid: policyGuid})
				Expect(err).NotTo(HaveOccurred())
				err = bdb.CreateServiceBinding(context.Background(), testBindingId, testInstanceId, models.GUID(testAppId), models.CustomMetricsSameApp)
			})
			It("should return what was created", func() {
				expectServiceInstancesToEqual(retrievedServiceInstance, &models.ServiceInstance{ServiceInstanceId: testInstanceId, OrgId: testOrgGuid, SpaceId: testSpaceGuid, DefaultPolicy: policyJsonStr, DefaultPolicyGuid: policyGuid})
			})
		})
	})

	Describe("CreateServiceBinding", func() {

		JustBeforeEach(func() {

			err = bdb.CreateServiceBinding(context.Background(), testBindingId, testInstanceId, models.GUID(testAppId), customMetricsStrategy)
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
				err = bdb.CreateServiceInstance(context.Background(), models.ServiceInstance{ServiceInstanceId: testInstanceId, OrgId: testOrgGuid, SpaceId: testSpaceGuid, DefaultPolicy: policyJsonStr, DefaultPolicyGuid: policyGuid})
				Expect(err).NotTo(HaveOccurred())
				customMetricsStrategy = models.CustomMetricsSameApp
			})

			Context("When service binding is being created first time", func() {
				It("should succeed", func() {
					Expect(err).NotTo(HaveOccurred())
					Expect(hasServiceBinding(testBindingId, testInstanceId)).To(BeTrue())
				})
			})
			Context("When service binding already exists", func() {
				It("should error", func() {
					err = bdb.CreateServiceBinding(context.Background(), testBindingId, testInstanceId, models.GUID(testAppId), models.CustomMetricsSameApp)
					Expect(err).To(HaveOccurred())
					Expect(err).To(Equal(db.ErrAlreadyExists))
				})
			})
			Context("When service binding is created with custom metrics strategy 'bound_app'", func() {
				BeforeEach(func() {
					customMetricsStrategy = models.CustomMetricsBoundApp
				})
				It("should succeed", func() {
					Expect(err).NotTo(HaveOccurred())
					Expect(hasServiceBindingWithCustomMetricStrategy(testBindingId, testInstanceId, customMetricsStrategy.String())).To(BeTrue())
				})
			})
			Context("When service binding is created with custom metrics strategy 'same_app'", func() {
				BeforeEach(func() {
					customMetricsStrategy = models.CustomMetricsSameApp
				})
				It("should succeed", func() {
					Expect(err).NotTo(HaveOccurred())
					Expect(hasServiceBindingWithCustomMetricStrategy(testBindingId, testInstanceId, customMetricsStrategy.String())).To(BeTrue())
				})
			})

			When("service binding is created with invalid custom metrics strategy", func() {
				BeforeEach(func() {
					// 🚸 The only invalid strategy one could create without dangerous methodologies
					// like using reflection, is this one:
					//
					// `customMetricsStrategy = models.CustomMetricsStrategy{}`
					//
					// However this is not “invalid enough” for our tests. The reason is that we
					// generate the empty-string as a strategy which gets mapped to the value `NULL`
					// in the database. This is perfectly legal because we allow `NULL` values in
					// the table. Therefore we follow this violent approach here:
					valueField := reflect.ValueOf(&customMetricsStrategy).Elem().FieldByName("value")
					reflect.NewAt(valueField.Type(), unsafe.Pointer(valueField.UnsafeAddr())).
						Elem().SetString("totally_invalid")
					// 🤔 We could as well think about eliminating this test and “similar” ones as
					// well as a benefit resulting of this re-design.
				})
				It("should throw an error with foreign key violation", func() {
					Expect(err).To(HaveOccurred())
					Expect(err.Error()).To(ContainSubstring("foreign key constraint"))
				})
			})

			// // 🚧 To-do: Remove me because I am not relevant anymore!
			// When("service binding is created with empty/nil custom metrics strategy", func() {
			//	BeforeEach(func() {
			//		customMetricsStrategy = ""
			//	})
			//	It("should return custom metrics strategy as null", func() {
			//		Expect(hasServiceBindingWithCustomMetricStrategyIsNull(testBindingId, testInstanceId)).To(BeTrue())
			//	})
			// })
		})
	})

	Describe("GetServiceBinding", func() {
		var retrievedServiceBinding *models.ServiceBinding
		JustBeforeEach(func() {
			retrievedServiceBinding, err = bdb.GetServiceBinding(context.Background(), testBindingId)
		})
		Context("when the service Binding does not exist", func() {
			It("should return an error", func() {
				Expect(err).To(HaveOccurred())
				Expect(err).To(Equal(db.ErrDoesNotExist))
				Expect(retrievedServiceBinding).To(BeNil())
			})
		})

		Context("when the service Binding exists", func() {
			BeforeEach(func() {
				err = bdb.CreateServiceInstance(context.Background(), models.ServiceInstance{ServiceInstanceId: testInstanceId, OrgId: testOrgGuid, SpaceId: testSpaceGuid, DefaultPolicy: policyJsonStr, DefaultPolicyGuid: policyGuid})
				Expect(err).NotTo(HaveOccurred())
				err = bdb.CreateServiceBinding(context.Background(), testBindingId, testInstanceId, models.GUID(testAppId), models.CustomMetricsSameApp)
				Expect(err).NotTo(HaveOccurred())
			})
			It("should return what was created", func() {
				Expect(err).NotTo(HaveOccurred())
				Expect(retrievedServiceBinding).To(Equal(&models.ServiceBinding{
					ServiceBindingID:      testBindingId,
					ServiceInstanceID:     testInstanceId,
					AppID:                 testAppId,
					CustomMetricsStrategy: "same_app",
				}))
			})
		})

		// // 🚧 To-do: Probably can be removed. The case that an existing custom-metrics-strategy
		// // is null should be excluded by table-constraints.
		// Context("with existing custom metrics strategy is null and binding already exists", func() {
		//	BeforeEach(func() {
		//		customMetricsStrategy = ""
		//		err = bdb.CreateServiceInstance(context.Background(), models.ServiceInstance{ServiceInstanceId: testInstanceId, OrgId: testOrgGuid, SpaceId: testSpaceGuid, DefaultPolicy: policyJsonStr, DefaultPolicyGuid: policyGuid})
		//		Expect(err).NotTo(HaveOccurred())
		//		err = bdb.CreateServiceBinding(context.Background(), testBindingId, testInstanceId, models.GUID(testAppId), models.DefaultCustomMetricsStrategy)
		//		Expect(err).NotTo(HaveOccurred())
		//	})
		//	It("should get the custom metrics strategy as empty", func() {
		//		Expect(retrievedServiceBinding).To(Equal(&models.ServiceBinding{
		//			ServiceBindingID:      testBindingId,
		//			ServiceInstanceID:     testInstanceId,
		//			AppID:                 testAppId,
		//			CustomMetricsStrategy: "",
		//		}))
		//		Expect(hasServiceBindingWithCustomMetricStrategyIsNull(testBindingId, testInstanceId)).To(BeTrue())
		//	})
		// })
	})

	Describe("DeleteServiceBinding", func() {
		JustBeforeEach(func() {
			err = bdb.DeleteServiceBinding(context.Background(), testBindingId)
		})
		Context("When service instance doesn't exist", func() {
			It("should error", func() {
				Expect(err).To(HaveOccurred())
				Expect(err).To(Equal(db.ErrDoesNotExist))
			})
		})
		Context("When service instance exists", func() {
			BeforeEach(func() {
				err = bdb.CreateServiceInstance(context.Background(), models.ServiceInstance{ServiceInstanceId: testInstanceId, OrgId: testOrgGuid, SpaceId: testSpaceGuid, DefaultPolicy: policyJsonStr, DefaultPolicyGuid: policyGuid})
				Expect(err).NotTo(HaveOccurred())
			})

			Context("When service binding doesn't exists", func() {
				It("should error", func() {
					Expect(err).To(HaveOccurred())
					Expect(err).To(Equal(db.ErrDoesNotExist))
				})
			})
			Context("When service binding is present", func() {
				BeforeEach(func() {
					err = bdb.CreateServiceBinding(context.Background(), testBindingId, testInstanceId, models.GUID(testAppId), models.CustomMetricsSameApp)
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
			err = bdb.CreateServiceInstance(context.Background(), models.ServiceInstance{ServiceInstanceId: testInstanceId, OrgId: testOrgGuid, SpaceId: testSpaceGuid, DefaultPolicy: policyJsonStr, DefaultPolicyGuid: policyGuid})
			Expect(err).NotTo(HaveOccurred())
			err = bdb.CreateServiceBinding(context.Background(), testBindingId, testInstanceId, models.GUID(testAppId), models.CustomMetricsSameApp)
			Expect(err).NotTo(HaveOccurred())
			err = bdb.DeleteServiceBindingByAppId(context.Background(), testAppId)
		})
		It("should succeed", func() {
			Expect(err).NotTo(HaveOccurred())
			Expect(hasServiceBinding(testAppId, testInstanceId)).NotTo(BeTrue())
		})
	})

	Describe("CheckServiceBinding", func() {
		var bindingExists bool
		JustBeforeEach(func() {
			bindingExists = bdb.CheckServiceBinding(testAppId)
		})
		Context("when binding for bindingId exists", func() {
			BeforeEach(func() {
				err = bdb.CreateServiceInstance(context.Background(), models.ServiceInstance{ServiceInstanceId: testInstanceId, OrgId: testOrgGuid, SpaceId: testSpaceGuid, DefaultPolicy: policyJsonStr, DefaultPolicyGuid: policyGuid})
				Expect(err).NotTo(HaveOccurred())
				err = bdb.CreateServiceBinding(context.Background(), testBindingId, testInstanceId, models.GUID(testAppId), models.CustomMetricsSameApp)
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
		JustBeforeEach(func() {
			appIdResult, err = bdb.GetAppIdByBindingId(context.Background(), testBindingId)
		})
		Context("when binding for bindingId exists", func() {
			BeforeEach(func() {
				err = bdb.CreateServiceInstance(context.Background(), models.ServiceInstance{ServiceInstanceId: testInstanceId, OrgId: testOrgGuid, SpaceId: testSpaceGuid, DefaultPolicy: policyJsonStr, DefaultPolicyGuid: policyGuid})
				Expect(err).NotTo(HaveOccurred())
				err = bdb.CreateServiceBinding(context.Background(), testBindingId, testInstanceId, models.GUID(testAppId), models.CustomMetricsSameApp)
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
		JustBeforeEach(func() {
			results, err = bdb.GetAppIdsByInstanceId(context.Background(), testInstanceId)
		})
		Context("when binding for bindingId exists", func() {
			BeforeEach(func() {
				err = bdb.CreateServiceInstance(context.Background(), models.ServiceInstance{ServiceInstanceId: testInstanceId, OrgId: testOrgGuid, SpaceId: testSpaceGuid, DefaultPolicy: policyJsonStr, DefaultPolicyGuid: policyGuid})
				Expect(err).NotTo(HaveOccurred())
				err = bdb.CreateServiceBinding(context.Background(), testBindingId, testInstanceId, models.GUID(testAppId), models.CustomMetricsSameApp)
				Expect(err).NotTo(HaveOccurred())
				err = bdb.CreateServiceBinding(context.Background(), testBindingId2, testInstanceId, models.GUID(testAppId2), models.CustomMetricsSameApp)
				Expect(err).NotTo(HaveOccurred())

				// other unrelated service instance with bindings
				err = bdb.CreateServiceInstance(context.Background(), models.ServiceInstance{ServiceInstanceId: testInstanceId3, OrgId: testOrgGuid, SpaceId: testSpaceGuid, DefaultPolicy: policyJsonStr, DefaultPolicyGuid: policyGuid})
				Expect(err).NotTo(HaveOccurred())
				err = bdb.CreateServiceBinding(context.Background(), testBindingId3, testInstanceId3, models.GUID(testAppId3), models.CustomMetricsSameApp)
				Expect(err).NotTo(HaveOccurred())
			})
			It("should succeed", func() {
				Expect(err).NotTo(HaveOccurred())
				Expect(results).To(ConsistOf(testAppId, testAppId2))
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
		JustBeforeEach(func() {
			serviceInstanceCount, err = bdb.CountServiceInstancesInOrg(testOrgGuid)
		})
		Context("when no service instance exist", func() {
			It("returns 0", func() {
				Expect(err).ToNot(HaveOccurred())
				Expect(serviceInstanceCount).To(Equal(0))
			})
		})
		Context("when service instances exist", func() {
			BeforeEach(func() {
				err = bdb.CreateServiceInstance(context.Background(), models.ServiceInstance{testInstanceId, testOrgGuid, testSpaceGuid, policyJsonStr, policyGuid})
				Expect(err).NotTo(HaveOccurred())
				err = bdb.CreateServiceInstance(context.Background(), models.ServiceInstance{testInstanceId2, testOrgGuid, testSpaceGuid, policyJsonStr, policyGuid})
				Expect(err).NotTo(HaveOccurred())
				err = bdb.CreateServiceInstance(context.Background(), models.ServiceInstance{testInstanceId3, testOrgGuid2, testSpaceGuid, policyJsonStr, policyGuid})
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

		JustBeforeEach(func() {
			bindingIds, err = bdb.GetBindingIdsByInstanceId(context.Background(), testInstanceId)
			serviceInstanceCount, err = bdb.CountServiceInstancesInOrg(testOrgGuid)
		})
		Context("when no service instance exist", func() {
			It("returns 0", func() {
				Expect(err).ToNot(HaveOccurred())
				Expect(serviceInstanceCount).To(Equal(0))
			})
		})
		Context("when service instances exist", func() {
			BeforeEach(func() {
				err = bdb.CreateServiceInstance(context.Background(), models.ServiceInstance{testInstanceId, testOrgGuid, testSpaceGuid, policyJsonStr, policyGuid})
				Expect(err).NotTo(HaveOccurred(), fmt.Sprintf("CreateServiceInstance, failed: testInstanceId %s procId %d", testInstanceId, GinkgoParallelProcess()))

				err = bdb.CreateServiceBinding(context.Background(), testBindingId, testInstanceId, models.GUID(testAppId), models.CustomMetricsSameApp)
				Expect(err).NotTo(HaveOccurred())

				err = bdb.CreateServiceBinding(context.Background(), testBindingId2, testInstanceId, models.GUID(testAppId2), models.CustomMetricsSameApp)
				Expect(err).NotTo(HaveOccurred())

				// other unrelated service instance with bindings
				err = bdb.CreateServiceInstance(context.Background(), models.ServiceInstance{testInstanceId3, testOrgGuid, testSpaceGuid, policyJsonStr, policyGuid})
				Expect(err).NotTo(HaveOccurred())

				err = bdb.CreateServiceBinding(context.Background(), testBindingId3, testInstanceId3, models.GUID(testAppId3), models.CustomMetricsSameApp)
				Expect(err).NotTo(HaveOccurred())

			})
			It("should succeed", func() {
				Expect(err).NotTo(HaveOccurred())
				Expect(bindingIds).To(ConsistOf(testBindingId, testBindingId2))
			})
		})
		Context("when binding for bindingId does not exist", func() {
			It("should not return an error, but an empty result", func() {
				Expect(err).NotTo(HaveOccurred())
				Expect(bindingIds).To(BeEmpty())
			})
		})
	})

	Describe("isAppBoundToSameAutoscaler", func() {
		var isTestApp1Bounded bool
		JustBeforeEach(func() {
			isTestApp1Bounded, _ = bdb.IsAppBoundToSameAutoscaler(context.Background(), testAppId, testAppId2)
			Expect(err).NotTo(HaveOccurred())
		})
		When("apps are bounded to same autoscaler instance", func() {
			BeforeEach(func() {
				err = bdb.CreateServiceInstance(context.Background(), models.ServiceInstance{ServiceInstanceId: testInstanceId, OrgId: testOrgGuid, SpaceId: testSpaceGuid, DefaultPolicy: policyJsonStr, DefaultPolicyGuid: policyGuid})
				Expect(err).NotTo(HaveOccurred())
				err = bdb.CreateServiceBinding(context.Background(), testBindingId, testInstanceId, models.GUID(testAppId), models.CustomMetricsSameApp)
				Expect(err).NotTo(HaveOccurred())
				err = bdb.CreateServiceBinding(context.Background(), testBindingId2, testInstanceId, models.GUID(testAppId2), models.CustomMetricsSameApp)
				Expect(err).NotTo(HaveOccurred())
			})
			It("should return true", func() {
				Expect(isTestApp1Bounded).To(BeTrue())
			})
		})
		Context("when neighbouring app is bounded to different autoscaler instance", func() {
			BeforeEach(func() {
				err = bdb.CreateServiceInstance(context.Background(), models.ServiceInstance{ServiceInstanceId: testInstanceId, OrgId: testOrgGuid, SpaceId: testSpaceGuid, DefaultPolicy: policyJsonStr, DefaultPolicyGuid: policyGuid})
				Expect(err).NotTo(HaveOccurred())
				err = bdb.CreateServiceInstance(context.Background(), models.ServiceInstance{ServiceInstanceId: testInstanceId2, OrgId: testOrgGuid, SpaceId: testSpaceGuid, DefaultPolicy: policyJsonStr, DefaultPolicyGuid: policyGuid})
				Expect(err).NotTo(HaveOccurred())
				err = bdb.CreateServiceBinding(context.Background(), testBindingId, testInstanceId, models.GUID(testAppId), models.CustomMetricsSameApp)
				Expect(err).NotTo(HaveOccurred())
				err = bdb.CreateServiceBinding(context.Background(), testBindingId2, testInstanceId2, models.GUID(testAppId2), models.CustomMetricsSameApp)
				Expect(err).NotTo(HaveOccurred())
			})
			It("should return false", func() {
				Expect(err).NotTo(HaveOccurred())
				Expect(isTestApp1Bounded).To(BeFalse())
			})
		})

	})

	Describe("GetCustomMetricStrategyByAppId", func() {
		BeforeEach(func() {
			err = bdb.CreateServiceInstance(context.Background(), models.ServiceInstance{ServiceInstanceId: testInstanceId, OrgId: testOrgGuid, SpaceId: testSpaceGuid, DefaultPolicy: policyJsonStr, DefaultPolicyGuid: policyGuid})
			Expect(err).NotTo(HaveOccurred())
		})
		Context("When service instance and binding exists with custom metrics strategy 'bound_app'", func() {
			BeforeEach(func() {
				err = bdb.CreateServiceBinding(context.Background(), testBindingId, testInstanceId, models.GUID(testAppId), models.CustomMetricsBoundApp)
				Expect(err).NotTo(HaveOccurred())
			})
			It("should get the custom metrics strategy from the database", func() {
				customMetricStrategy, _ := bdb.GetCustomMetricStrategyByAppId(context.Background(), testAppId)
				Expect(customMetricStrategy.String()).To(Equal("bound_app"))
			})
		})
		Context("When service instance and binding exists with custom metrics strategy 'same_app'", func() {
			BeforeEach(func() {
				err = bdb.CreateServiceBinding(context.Background(), testBindingId, testInstanceId, models.GUID(testAppId), models.CustomMetricsSameApp)
				Expect(err).NotTo(HaveOccurred())
			})
			It("should get the custom metrics strategy from the database", func() {
				customMetricStrategy, _ := bdb.GetCustomMetricStrategyByAppId(context.Background(), testAppId)
				Expect(customMetricStrategy.String()).To(Equal("same_app"))
			})
		})

	})

	Describe("SetOrUpdateCustomMetricStrategy", func() {
		BeforeEach(func() {
			err = bdb.CreateServiceInstance(context.Background(), models.ServiceInstance{ServiceInstanceId: testInstanceId, OrgId: testOrgGuid, SpaceId: testSpaceGuid, DefaultPolicy: policyJsonStr, DefaultPolicyGuid: policyGuid})
			Expect(err).NotTo(HaveOccurred())
		})
		Context("Update Custom Metrics Strategy", func() {
			Context("With binding does not exist'", func() {
				JustBeforeEach(func() {
					err = bdb.SetOrUpdateCustomMetricStrategy(context.Background(), testAppId, models.CustomMetricsBoundApp, "update")
				})
				It("should not save the custom metrics strategy and fails ", func() {
					Expect(err).To(HaveOccurred())
				})
			})
			Context("With binding exists'", func() {
				JustBeforeEach(func() {
					err = bdb.SetOrUpdateCustomMetricStrategy(context.Background(), testAppId, customMetricsStrategy, "update")
				})
				When("custom metrics strategy is not present (already null)", func() {
					BeforeEach(func() {
						customMetricsStrategy = models.CustomMetricsBoundApp
						err = bdb.CreateServiceBinding(context.Background(), testBindingId, testInstanceId, models.GUID(testAppId), models.DefaultCustomMetricsStrategy)
						Expect(err).NotTo(HaveOccurred())
					})
					It("should save the custom metrics strategy", func() {
						customMetricStrategy, _ := bdb.GetCustomMetricStrategyByAppId(context.Background(), testAppId)
						Expect(customMetricStrategy.String()).To(Equal("bound_app"))
					})
				})
				When("custom metrics strategy is not present (already null)", func() {
					BeforeEach(func() {
						customMetricsStrategy = models.CustomMetricsSameApp
						err = bdb.CreateServiceBinding(context.Background(), testBindingId, testInstanceId, models.GUID(testAppId), models.DefaultCustomMetricsStrategy)
						Expect(err).NotTo(HaveOccurred())
					})
					It("should save the custom metrics strategy", func() {
						customMetricStrategy, _ := bdb.GetCustomMetricStrategyByAppId(context.Background(), testAppId)
						Expect(customMetricStrategy.String()).To(Equal("same_app"))
					})
				})
				When("custom metrics strategy is already present", func() {
					BeforeEach(func() {
						customMetricsStrategy = models.CustomMetricsBoundApp
						err = bdb.CreateServiceBinding(context.Background(), testBindingId, testInstanceId, models.GUID(testAppId), models.CustomMetricsSameApp)
						Expect(err).NotTo(HaveOccurred())
					})
					It("should update the custom metrics strategy to bound_app", func() {
						Expect(err).NotTo(HaveOccurred())
						customMetricStrategy, _ := bdb.GetCustomMetricStrategyByAppId(context.Background(), testAppId)
						Expect(customMetricStrategy.String()).To(Equal("bound_app"))
					})
				})
				When("custom metrics strategy is already present as same_app", func() {
					BeforeEach(func() {
						customMetricsStrategy = models.CustomMetricsSameApp
						err = bdb.CreateServiceBinding(context.Background(), testBindingId, testInstanceId, models.GUID(testAppId), models.CustomMetricsSameApp)
						Expect(err).NotTo(HaveOccurred())
					})
					It("should not update the same custom metrics strategy", func() {
						Expect(err).NotTo(HaveOccurred())
						customMetricStrategy, _ := bdb.GetCustomMetricStrategyByAppId(context.Background(), testAppId)
						Expect(customMetricStrategy.String()).To(Equal("same_app"))
					})
				})
				When("custom metrics strategy unknown value", func() {
					BeforeEach(func() {
						// 🚸 The only invalid strategy one could create without dangerous methodologies
						// like using reflection, is this one:
						//
						// `customMetricsStrategy = models.CustomMetricsStrategy{}`
						//
						// However this is not “invalid enough” for our tests. The reason is that we
						// generate the empty-string as a strategy which gets mapped to the value `NULL`
						// in the database. This is perfectly legal because we allow `NULL` values in
						// the table. Therefore we follow this violent approach here:
						valueField := reflect.ValueOf(&customMetricsStrategy).Elem().FieldByName("value")
						reflect.NewAt(valueField.Type(), unsafe.Pointer(valueField.UnsafeAddr())).
							Elem().SetString("totally_invalid")
						// 🤔 We could as well think about eliminating this test and “similar” ones as
						// well as a benefit resulting of this re-design.

						err = bdb.CreateServiceBinding(context.Background(), testBindingId, testInstanceId, models.GUID(testAppId), models.DefaultCustomMetricsStrategy)
						Expect(err).NotTo(HaveOccurred())
					})
					It("should throw an error", func() {
						Expect(err.Error()).To(ContainSubstring("foreign key constraint"))
					})
				})
			})
		})
		Context("Delete Custom Metrics Strategy", func() {
			Context("With binding exists'", func() {
				JustBeforeEach(func() {
					err = bdb.SetOrUpdateCustomMetricStrategy(context.Background(), testAppId, customMetricsStrategy, "delete")
					Expect(err).NotTo(HaveOccurred())
				})
				When("custom metrics strategy is already present", func() {
					BeforeEach(func() {
						customMetricsStrategy = models.DefaultCustomMetricsStrategy
						err = bdb.CreateServiceBinding(context.Background(), testBindingId, testInstanceId, models.GUID(testAppId), models.CustomMetricsBoundApp)
						Expect(err).NotTo(HaveOccurred())
					})
					It("should update the custom metrics strategy with the value of the default one", func() {
						customMetricStrategy, _ := bdb.GetCustomMetricStrategyByAppId(context.Background(), testAppId)
						Expect(customMetricStrategy.String()).To(Equal(models.DefaultCustomMetricsStrategy.String()))
					})
				})
			})
		})
	})

})

func addProcessIdTo(id string) string {
	return fmt.Sprintf("%s-%d", id, GinkgoParallelProcess())
}
