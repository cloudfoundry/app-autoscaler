package sqldb_test

import (
	"autoscaler/db"
	. "autoscaler/db/sqldb"
	"autoscaler/models"
	"os"

	"code.cloudfoundry.org/lager"
	"github.com/lib/pq"
	"github.com/go-sql-driver/mysql"
	"time"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("LockSqldb", func() {
	var (
		ldb            *LockSQLDB
		dbConfig       db.DatabaseConfig
		logger         lager.Logger
		err            error
		lock           *models.Lock
		isLockAcquired bool
		testTTL        time.Duration
	)

	BeforeEach(func() {
		logger = lager.NewLogger("lock-sqldb-test")
		dbConfig = db.DatabaseConfig{
			URL:                   os.Getenv("DBURL"),
			MaxOpenConnections:    10,
			MaxIdleConnections:    5,
			ConnectionMaxLifetime: 10 * time.Second,
		}
		testTTL = time.Duration(15) * time.Second
	})

	Describe("NewLockSQLDB", func() {
		JustBeforeEach(func() {
			ldb, err = NewLockSQLDB(dbConfig, "test_lock", logger)
		})

		AfterEach(func() {
			if ldb != nil {
				err = ldb.Close()
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
		
		Context("when lock db url is correct", func() {
			It("should not error", func() {
				Expect(err).NotTo(HaveOccurred())
				Expect(ldb).NotTo(BeNil())
			})
		})
	})

	Describe("Lock", func() {
		BeforeEach(func() {
			ldb, err = NewLockSQLDB(dbConfig, "test_lock", logger)
			Expect(err).NotTo(HaveOccurred())
		})

		AfterEach(func() {
			err = ldb.Close()
			Expect(err).NotTo(HaveOccurred())
			cleanLockTable()
		})

		Context("when the lock does not exist", func() {
			Context("because the row does not exist", func() {
				BeforeEach(func() {
					lock = &models.Lock{Owner: "123456", Ttl: testTTL}
				})

				It("insert the lock for the owner", func() {
					isLockAcquired, err = ldb.Lock(lock)
					Expect(err).NotTo(HaveOccurred())
					Expect(isLockAcquired).To(BeTrue())
					Expect(validateLockInDB("123456", lock)).To(Succeed())
				})
			})
		})

		Context("when the lock exist", func() {
			Context("and the owner is same", func() {
				BeforeEach(func() {
					lock = &models.Lock{Owner: "213123313", Ttl: testTTL}
					result, err := insertLockDetails(lock)
					Expect(err).NotTo(HaveOccurred())
					Expect(result.RowsAffected()).To(BeEquivalentTo(1))
					Expect(validateLockInDB("213123313", lock)).To(Succeed())
				})
				It("should successfully renew the lock", func() {
					lock = &models.Lock{Owner: "213123313", Ttl: testTTL}
					isLockAcquired, err = ldb.Lock(lock)
					Expect(err).NotTo(HaveOccurred())
					Expect(isLockAcquired).To(BeTrue())
				})
			})

			Context("and the owner is different", func() {
				Context("and lock recently renewed by owner", func() {
					BeforeEach(func() {
						lock = &models.Lock{Owner: "65432199", Ttl: testTTL}
						isLockAcquired, err = ldb.Lock(lock)
						Expect(err).NotTo(HaveOccurred())
						Expect(isLockAcquired).To(BeTrue())
						Expect(validateLockInDB("65432199", lock)).To(Succeed())
					})
					It("competing instance should fail to get the lock", func() {
						lock = &models.Lock{Owner: "1234567", Ttl: testTTL}
						isLockAcquired, err = ldb.Lock(lock)
						Expect(isLockAcquired).To(BeFalse())
						Expect(validateLockInDB("1234567", lock)).NotTo(Succeed())
					})
				})

				Context("and lock expired", func() {
					BeforeEach(func() {
						lock = &models.Lock{Owner: "24165435", Ttl: testTTL}
						isLockAcquired, err = ldb.Lock(lock)
						Expect(err).NotTo(HaveOccurred())
						Expect(isLockAcquired).To(BeTrue())
						Expect(validateLockInDB("24165435", lock)).To(Succeed())
					})
					It("competing instance should successfully acquire the lock", func() {
						time.Sleep(testTTL + 5*time.Second) //waiting for the ttl to expire
						lock = &models.Lock{Owner: "123456", Ttl: testTTL}
						isLockAcquired, err = ldb.Lock(lock)
						Expect(err).NotTo(HaveOccurred())
						Expect(isLockAcquired).To(BeTrue())
						Expect(validateLockInDB("123456", lock)).To(Succeed())
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
				lock = &models.Lock{Owner: "123456", Ttl: testTTL}
				isLockAcquired, err = ldb.Lock(lock)
				Expect(err).To(HaveOccurred())
				Expect(isLockAcquired).To(BeFalse())
			})
		})
	})

	Describe("Release Lock", func() {
		BeforeEach(func() {
			ldb, err = NewLockSQLDB(dbConfig, "test_lock", logger)
			Expect(err).NotTo(HaveOccurred())
			lock = &models.Lock{Owner: "654321", Ttl: testTTL}
		})

		AfterEach(func() {
			err = ldb.Close()
			Expect(err).NotTo(HaveOccurred())
			cleanLockTable()
		})

		Context("when the lock exist", func() {
			BeforeEach(func() {
				result, err := insertLockDetails(lock)
				Expect(err).NotTo(HaveOccurred())
				Expect(result.RowsAffected()).To(BeEquivalentTo(1))
				Expect(validateLockInDB("654321", lock)).To(Succeed())
			})
			It("removes the lock from the locks table", func() {
				err = ldb.Release(lock.Owner)
				Expect(err).NotTo(HaveOccurred())
				Expect(validateLockNotInDB("654321")).To(Succeed())
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
