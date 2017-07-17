package sqldb_test

import (
	. "autoscaler/db/sqldb"
	"autoscaler/models"
	"database/sql"
	"os"

	"code.cloudfoundry.org/lager"

	"time"

	"github.com/lib/pq"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("LockSqldb", func() {
	var (
		ldb                  *LockSQLDB
		url                  string
		logger               lager.Logger
		err                  error
		fetchedLocks         *models.Lock
		isLockAcquired       bool
		isSecondLockAcquired bool
		secondErr            error
		testTTL              time.Duration
		timestamp            time.Time
	)

	BeforeEach(func() {
		logger = lager.NewLogger("lock-sqldb-test")
		url = os.Getenv("DBURL")
		testTTL = time.Duration(15) * time.Second
	})

	Describe("NewLockSQLDB", func() {
		JustBeforeEach(func() {
			ldb, err = NewLockSQLDB(url, logger)
		})

		AfterEach(func() {
			if ldb != nil {
				err = ldb.Close()
				Expect(err).NotTo(HaveOccurred())
			}
		})

		Context("when lock db url is not correct", func() {
			BeforeEach(func() {
				url = "postgres://not-exist-user:not-exist-password@localhost/autoscaler?sslmode=disable"
			})
			It("should throw an error", func() {
				Expect(err).To(BeAssignableToTypeOf(&pq.Error{}))
			})

		})

		Context("when lock db url is correct", func() {
			It("should not error", func() {
				Expect(err).NotTo(HaveOccurred())
				Expect(ldb).NotTo(BeNil())
			})
		})
	})

	Describe("Fetch Locks", func() {
		BeforeEach(func() {
			ldb, err = NewLockSQLDB(url, logger)
			Expect(err).NotTo(HaveOccurred())
			cleanLockTable()
		})

		AfterEach(func() {
			err = ldb.Close()
			Expect(err).NotTo(HaveOccurred())
		})

		JustBeforeEach(func() {
			fetchedLocks, err = ldb.Fetch()
		})

		Context("when lock table is empty", func() {
			It("should not return any lock", func() {
				Expect(err).To(HaveOccurred())
				Expect(err).To(BeIdenticalTo(sql.ErrNoRows))
			})
		})

		Context("when lock table is not empty", func() {
			BeforeEach(func() {
				lock := models.Lock{Owner: "123456", Ttl: testTTL}
				err = insertLockDetails(lock)
			})

			It("should return the lock", func() {
				Expect(err).NotTo(HaveOccurred())
				Expect(fetchedLocks.Owner).To(Equal("123456"))
				Expect(fetchedLocks.Ttl).To(Equal(testTTL))
			})
		})
	})

	Describe("Acquire Lock", func() {
		BeforeEach(func() {
			ldb, err = NewLockSQLDB(url, logger)
			Expect(err).NotTo(HaveOccurred())
			cleanLockTable()
		})

		AfterEach(func() {
			err = ldb.Close()
			Expect(err).NotTo(HaveOccurred())
		})

		Context("when no lock owner exist", func() {
			BeforeEach(func() {
				newlock := &models.Lock{Owner: "123456", Ttl: testTTL}
				err = ldb.Acquire(newlock)
			})
			It("should successfully acquire the lock", func() {
				Expect(err).NotTo(HaveOccurred())
			})
		})
	})

	Describe("Lock", func() {
		BeforeEach(func() {
			ldb, err = NewLockSQLDB(url, logger)
			Expect(err).NotTo(HaveOccurred())
			cleanLockTable()
		})

		AfterEach(func() {
			err = ldb.Close()
			Expect(err).NotTo(HaveOccurred())
		})

		Context("when no instance owns the lock", func() {
			BeforeEach(func() {
				lock := &models.Lock{Owner: "123456", Ttl: testTTL}
				isLockAcquired, err = ldb.Lock(lock)
			})

			It("competing instance should successfully acquire the lock", func() {
				Expect(err).NotTo(HaveOccurred())
				Expect(isLockAcquired).To(BeTrue())
			})
		})

		Context("when lock owned by an instance", func() {
			BeforeEach(func() {
				lock := &models.Lock{Owner: "654321", Ttl: testTTL}
				isLockAcquired, err = ldb.Lock(lock)
				Expect(err).NotTo(HaveOccurred())
				lock = &models.Lock{Owner: "654321", Ttl: testTTL}
				isLockAcquired, err = ldb.Lock(lock)
			})
			It("same instance should successfully renew it's lock", func() {
				Expect(err).NotTo(HaveOccurred())
				Expect(isLockAcquired).To(BeTrue())
			})
		})
		Context("when lock recently renewed by owner instance", func() {
			BeforeEach(func() {
				lock := &models.Lock{Owner: "654321", Ttl: testTTL}
				isLockAcquired, err = ldb.Lock(lock)
				Expect(err).NotTo(HaveOccurred())
				secondlock := &models.Lock{Owner: "123456", Ttl: testTTL}
				isSecondLockAcquired, secondErr = ldb.Lock(secondlock)
			})
			It("competing instance should fail to acquire the lock", func() {
				Expect(secondErr).NotTo(HaveOccurred())
				Expect(isSecondLockAcquired).NotTo(BeTrue())
			})
		})

		Context("Lock owned by some instance but expired", func() {
			BeforeEach(func() {
				lock := &models.Lock{Owner: "654321", Ttl: time.Duration(3) * time.Second}
				isLockAcquired, err = ldb.Lock(lock)
				Expect(err).NotTo(HaveOccurred())
				time.Sleep(4 * time.Second)
				secondlock := &models.Lock{Owner: "123456", Ttl: testTTL}
				isSecondLockAcquired, secondErr = ldb.Lock(secondlock)
			})
			It("competing instance should successfully acquire the lock", func() {
				Expect(secondErr).NotTo(HaveOccurred())
				Expect(isSecondLockAcquired).To(BeTrue())
			})
		})
	})

	Describe("Renew Lock", func() {
		BeforeEach(func() {
			ldb, err = NewLockSQLDB(url, logger)
			Expect(err).NotTo(HaveOccurred())
			cleanLockTable()
		})

		AfterEach(func() {
			err = ldb.Close()
			Expect(err).NotTo(HaveOccurred())
		})

		Context("when lock owned by an instance", func() {
			BeforeEach(func() {
				lock := &models.Lock{Owner: "123456", Ttl: testTTL}
				err = ldb.Acquire(lock)
				Expect(err).NotTo(HaveOccurred())
				err = ldb.Renew("123456")
			})
			It("same instance should able to renew the lock successfully", func() {
				Expect(err).NotTo(HaveOccurred())
			})
		})
	})

	Describe("Release Lock", func() {
		BeforeEach(func() {
			ldb, err = NewLockSQLDB(url, logger)
			Expect(err).NotTo(HaveOccurred())
			cleanLockTable()
		})

		AfterEach(func() {
			err = ldb.Close()
			Expect(err).NotTo(HaveOccurred())
		})

		Context("when owner instance wants to release the lock", func() {
			BeforeEach(func() {
				lock := &models.Lock{Owner: "123456", Ttl: testTTL}
				err = ldb.Acquire(lock)
				Expect(err).NotTo(HaveOccurred())
				err = ldb.Release("123456")
			})
			It("owner instance should able to release the lock successfully", func() {
				Expect(err).NotTo(HaveOccurred())
			})
		})
	})

	Describe("Get Database Timestamp", func() {
		BeforeEach(func() {
			ldb, err = NewLockSQLDB(url, logger)
			Expect(err).NotTo(HaveOccurred())
			cleanLockTable()
		})

		AfterEach(func() {
			err = ldb.Close()
			Expect(err).NotTo(HaveOccurred())
		})

		Context("when fetching current database timestamp", func() {
			BeforeEach(func() {
				timestamp, err = ldb.GetDatabaseTimestamp()
			})

			It("should not throw an error ", func() {
				Expect(err).NotTo(HaveOccurred())
				Expect(timestamp).Should(BeAssignableToTypeOf(time.Time{}))
			})
		})

	})

})
