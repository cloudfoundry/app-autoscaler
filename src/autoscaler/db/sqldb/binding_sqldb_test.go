package sqldb_test

import (
	"autoscaler/db"
	. "autoscaler/db/sqldb"
	"database/sql"
	"os"
	"time"
	"github.com/lib/pq"
	"github.com/go-sql-driver/mysql"
	"code.cloudfoundry.org/lager"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("BindingSqldb", func() {
	var (
		bdb            *BindingSQLDB
		dbConfig       db.DatabaseConfig
		logger         lager.Logger
		err            error
		testInstanceId string = "test-instance-id"
		testBindingId  string = "test-binding-id"
		testAppId      string = "test-app-id"
		testOrgGuid    string = "test-org-guid"
		testOrgGuid2   string = "test-org-guid-2"
		testSpaceGuid  string = "test-space-guid"
	)

	BeforeEach(func() {
		logger = lager.NewLogger("binding-sqldb-test")
		dbConfig = db.DatabaseConfig{
			URL:                   os.Getenv("DBURL"),
			MaxOpenConnections:    10,
			MaxIdleConnections:    5,
			ConnectionMaxLifetime: 10 * time.Second,
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
				Expect(bdb).NotTo(BeNil())
			})
		})
	})

	Describe("CreateServiceInstance", func() {
		BeforeEach(func() {
			bdb, err = NewBindingSQLDB(dbConfig, logger)
			Expect(err).NotTo(HaveOccurred())

			cleanServiceInstanceTable()
		})
		AfterEach(func() {
			err = bdb.Close()
			Expect(err).NotTo(HaveOccurred())
		})
		JustBeforeEach(func() {
			err = bdb.CreateServiceInstance(testInstanceId, testOrgGuid, testSpaceGuid)
		})
		Context("When instance is being created first time", func() {
			It("should succeed", func() {
				Expect(err).NotTo(HaveOccurred())
				Expect(hasServiceInstance(testInstanceId)).To(BeTrue())
			})
		})
		Context("When instance already exists", func() {
			BeforeEach(func() {
				err = bdb.CreateServiceInstance(testInstanceId, testOrgGuid, testSpaceGuid)
				Expect(err).NotTo(HaveOccurred())
			})
			It("should error", func() {
				Expect(err).To(HaveOccurred())
				Expect(err).To(Equal(db.ErrAlreadyExists))
			})
		})
		Context("When a conflicting instance exists", func() {
			BeforeEach(func() {
				err = bdb.CreateServiceInstance(testInstanceId, testOrgGuid2, testSpaceGuid)
				Expect(err).NotTo(HaveOccurred())
			})
			It("should error", func() {
				Expect(err).To(HaveOccurred())
				Expect(err).To(Equal(db.ErrConflict))
			})
		})

	})

	Describe("DeleteServiceInstance", func() {
		BeforeEach(func() {
			bdb, err = NewBindingSQLDB(dbConfig, logger)
			Expect(err).NotTo(HaveOccurred())

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
				err = bdb.CreateServiceInstance(testInstanceId, testOrgGuid, testSpaceGuid)
				Expect(err).NotTo(HaveOccurred())
			})
			It("should succeed", func() {
				Expect(err).NotTo(HaveOccurred())
				Expect(hasServiceInstance(testInstanceId)).NotTo(BeTrue())
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
				err = bdb.CreateServiceInstance(testInstanceId, testOrgGuid, testSpaceGuid)
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
				BeforeEach(func() {
					err = bdb.CreateServiceBinding(testBindingId, testInstanceId, testAppId)
					Expect(err).NotTo(HaveOccurred())
				})
				It("should error", func() {
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
				err = bdb.CreateServiceInstance(testInstanceId, testOrgGuid, testSpaceGuid)
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

			err = bdb.CreateServiceInstance(testInstanceId, testOrgGuid, testSpaceGuid)
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
				err = bdb.CreateServiceInstance(testInstanceId, testOrgGuid, testSpaceGuid)
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
				err = bdb.CreateServiceInstance(testInstanceId, testOrgGuid, testSpaceGuid)
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

})
