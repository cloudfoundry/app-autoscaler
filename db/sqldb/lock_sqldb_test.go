package sqldb_test

import (
	"fmt"
	"os"
	"strings"
	"time"

	"code.cloudfoundry.org/app-autoscaler/src/autoscaler/testhelpers"

	"code.cloudfoundry.org/app-autoscaler/src/autoscaler/db"
	. "code.cloudfoundry.org/app-autoscaler/src/autoscaler/db/sqldb"
	"code.cloudfoundry.org/app-autoscaler/src/autoscaler/models"

	"code.cloudfoundry.org/lager/v3"
	"github.com/go-sql-driver/mysql"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("LockSqldb", func() {
	var (
		ldb            *LockSQLDB
		dbConfig       db.DatabaseConfig
		dbHost         = os.Getenv("DB_HOST")
		logger         lager.Logger
		err            error
		lock           *models.Lock
		isLockAcquired bool
		testTTL        time.Duration
		ownerId        string
		ownerId2       string
	)

	dbUrl := testhelpers.GetDbUrl()
	BeforeEach(func() {
		ownerId = fmt.Sprintf("111111%d", GinkgoParallelProcess())
		ownerId2 = fmt.Sprintf("222222%d", GinkgoParallelProcess())
		logger = lager.NewLogger("lock-sqldb-test")
		dbConfig = db.DatabaseConfig{
			URL:                   dbUrl,
			MaxOpenConnections:    10,
			MaxIdleConnections:    5,
			ConnectionMaxLifetime: 10 * time.Second,
			ConnectionMaxIdleTime: 10 * time.Second,
		}
		testTTL = 1 * time.Second

		ldb, err = NewLockSQLDB(dbConfig, lockTable, logger)
		DeferCleanup(func() {
			if ldb != nil {
				err = ldb.Close()
				Expect(err).NotTo(HaveOccurred())
			}
		})

	})

	AfterEach(func() {
		err = cleanLockTable()
		Expect(err).NotTo(HaveOccurred())
	})

	Describe("NewLockSQLDB", func() {
		JustBeforeEach(func() {
			_ = ldb.Close()
			ldb, err = NewLockSQLDB(dbConfig, lockTable, logger)
		})

		Context("when db url is not correct", func() {
			BeforeEach(func() {
				if !strings.Contains(dbUrl, "postgres") {
					Skip("Postgres only test")
				}
				dbConfig.URL = "postgres://not-exist-user:not-exist-password@localhost/autoscaler?sslmode=disable"
			})
			It("should throw an error", func() {
				Expect(err).To(HaveOccurred())
			})
		})

		Context("when mysql db url is not correct", func() {
			BeforeEach(func() {
				if strings.Contains(dbUrl, "postgres") {
					Skip("Mysql test")
				}
				dbConfig.URL = "not-exist-user:not-exist-password@tcp(" + dbHost + ")/autoscaler?tls=false"
			})
			It("should throw an error", func() {
				Expect(err).To(BeAssignableToTypeOf(&mysql.MySQLError{}))
			})
		})

		Context("when lock db url is correct", func() {
			It("should not error", func() {
				Expect(err).NotTo(HaveOccurred())
				Expect(ldb).NotTo(BeNil())
			})
		})
	})

	Describe("Lock", func() {
		Context("when the lock does not exist", func() {
			Context("because the row does not exist", func() {
				BeforeEach(func() {
					lock = createLock(ownerId, testTTL)
				})

				It("insert the lock for the owner", func() {
					isLockAcquired, err = ldb.Lock(lock)
					Expect(err).NotTo(HaveOccurred())
					Expect(isLockAcquired).To(BeTrue())
					Expect(validateLockInDB(ownerId, lock)).To(Succeed())
				})
			})
		})

		Context("when the lock exist", func() {
			Context("and the owner is same", func() {
				BeforeEach(func() {
					lock = createLock(ownerId, testTTL)
					result, err := insertLockDetails(lock)
					Expect(err).NotTo(HaveOccurred())
					Expect(result.RowsAffected()).To(BeEquivalentTo(1))
					Expect(validateLockInDB(ownerId, lock)).To(Succeed())
				})
				It("should successfully renew the lock", func() {
					lock = createLock(ownerId, testTTL)
					isLockAcquired, err = ldb.Lock(lock)
					Expect(err).NotTo(HaveOccurred())
					Expect(isLockAcquired).To(BeTrue())
				})
			})

			Context("and the owner is different", func() {
				Context("and lock recently renewed by owner", func() {
					BeforeEach(func() {
						lock = createLock(ownerId, testTTL)
						isLockAcquired, err = ldb.Lock(lock)
						Expect(err).NotTo(HaveOccurred())
						Expect(isLockAcquired).To(BeTrue())
						Expect(validateLockInDB(ownerId, lock)).To(Succeed())
					})
					It("competing instance should fail to get the lock", func() {
						lock = createLock(ownerId2, testTTL)
						isLockAcquired, err = ldb.Lock(lock)
						Expect(isLockAcquired).To(BeFalse())
						Expect(validateLockInDB(ownerId2, lock)).NotTo(Succeed())
					})
				})

				Context("and lock expired", func() {
					BeforeEach(func() {
						lock = createLock(ownerId, testTTL)
						isLockAcquired, err = ldb.Lock(lock)
						Expect(err).NotTo(HaveOccurred())
						Expect(isLockAcquired).To(BeTrue())
						Expect(validateLockInDB(ownerId, lock)).To(Succeed())
					})
					It("competing instance should successfully acquire the lock", func() {
						time.Sleep(testTTL + 50*time.Millisecond) //waiting for the ttl to expire
						lock = createLock(ownerId, testTTL)
						isLockAcquired, err = ldb.Lock(lock)
						Expect(err).NotTo(HaveOccurred())
						Expect(isLockAcquired).To(BeTrue())
						Expect(validateLockInDB(ownerId, lock)).To(Succeed())
					})

				})
			})

		})

		Context("when the lock table disappears", func() {
			BeforeEach(func() {
				err = dropLockTable()
				Expect(err).NotTo(HaveOccurred())
			})

			AfterEach(func() {
				err = createLockTable()
				Expect(err).NotTo(HaveOccurred())
			})

			It("should fail to acquire lock", func() {
				lock = createLock(ownerId, testTTL)
				isLockAcquired, err = ldb.Lock(lock)
				Expect(err).To(HaveOccurred())
				Expect(isLockAcquired).To(BeFalse())
			})
		})
	})

	Describe("Release Lock", func() {
		BeforeEach(func() {
			lock = createLock(ownerId, testTTL)
		})

		Context("when the lock exist", func() {
			BeforeEach(func() {
				result, err := insertLockDetails(lock)
				Expect(err).NotTo(HaveOccurred())
				Expect(result.RowsAffected()).To(BeEquivalentTo(1))
				Expect(validateLockInDB(ownerId, lock)).To(Succeed())
			})
			It("removes the lock from the locks table", func() {
				err = ldb.Release(lock.Owner)
				Expect(err).NotTo(HaveOccurred())
				Expect(validateLockNotInDB(ownerId)).To(Succeed())
			})
		})

		Context("when the lock table disappears", func() {
			BeforeEach(func() {
				err = dropLockTable()
				Expect(err).NotTo(HaveOccurred())
			})

			AfterEach(func() {
				err = createLockTable()
				Expect(err).NotTo(HaveOccurred())
			})

			It("should fail to release lock", func() {
				err = ldb.Release(lock.Owner)
				Expect(err).To(HaveOccurred())
			})
		})
	})
})

func createLock(owner string, testTTL time.Duration) *models.Lock {
	return &models.Lock{Owner: owner, LastModifiedTimestamp: time.Now(), Ttl: testTTL}
}
