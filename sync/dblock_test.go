package sync_test

import (
	"os"
	"time"

	"autoscaler/db"
	"autoscaler/db/sqldb"
	. "autoscaler/sync"

	"code.cloudfoundry.org/lager/lagertest"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/tedsuo/ifrit"
)

var _ = Describe("Dblock", func() {
	var (
		lock1       *DatabaseLock
		lock2       *DatabaseLock
		lockRunner1 ifrit.Runner
		lockRunner2 ifrit.Runner
		lockOwner1  string = "owner1"
		lockOwner2  string = "owner2"
		resultOwner string
		dblogger    *lagertest.TestLogger
		logger1     *lagertest.TestLogger
		logger2     *lagertest.TestLogger
		ldb         *sqldb.LockSQLDB
		dbConfig    = db.DatabaseConfig{
			URL:                   os.Getenv("DBURL"),
			MaxOpenConnections:    10,
			MaxIdleConnections:    5,
			ConnectionMaxLifetime: 10 * time.Second,
		}
		retryInterval          time.Duration = 5 * time.Second
		lockTTL                time.Duration = 15 * time.Second
		signalsChan1           chan os.Signal
		readyChan1             chan struct{}
		signalsChan2           chan os.Signal
		readyChan2             chan struct{}
		lostLockChan1          chan struct{}
		lostLockChan2          chan struct{}
		callbackOnAcquireLock1 func()
		callbackOnAcquireLock2 func()
		callbackOnLostLock1    func()
		callbackOnLostLock2    func()
		emptyCallback          = func() {}
		err                    error
	)
	BeforeEach(func() {
		cleanLockTable()
		dblogger = lagertest.NewTestLogger("lockdb")
		ldb, err = sqldb.NewLockSQLDB(dbConfig, lockTableName, dblogger)
		Expect(err).NotTo(HaveOccurred())
		logger1 = lagertest.NewTestLogger(lockOwner1)
		logger2 = lagertest.NewTestLogger(lockOwner2)
		lock1 = NewDatabaseLock(logger1)
		lock2 = NewDatabaseLock(logger2)
		signalsChan1 = make(chan os.Signal, 5)
		signalsChan2 = make(chan os.Signal, 5)
		readyChan1 = make(chan struct{}, 5)
		readyChan2 = make(chan struct{}, 5)
		callbackOnAcquireLock1 = emptyCallback
		callbackOnAcquireLock2 = emptyCallback
		callbackOnLostLock1 = emptyCallback
		callbackOnLostLock2 = emptyCallback

	})
	JustBeforeEach(func() {
		lockRunner1 = lock1.InitDBLockRunner(retryInterval, lockTTL, lockOwner1, ldb, callbackOnAcquireLock1, callbackOnLostLock1)
		lockRunner2 = lock2.InitDBLockRunner(retryInterval, lockTTL, lockOwner2, ldb, callbackOnAcquireLock2, callbackOnLostLock2)
		go lockRunner1.Run(signalsChan1, readyChan1)
		go lockRunner2.Run(signalsChan2, readyChan2)
		select {

		case <-logger1.Buffer().Detect("lock-acquired-in-first-attempt"):
			resultOwner = lockOwner1
		case <-logger2.Buffer().Detect("lock-acquired-in-first-attempt"):
			resultOwner = lockOwner2
		case <-time.After(2 * time.Second):
		}
		logger1.Buffer().CancelDetects()
		logger2.Buffer().CancelDetects()
	})
	AfterEach(func() {
		signalsChan1 <- os.Kill
		signalsChan2 <- os.Kill
		if ldb != nil {
			err := ldb.Close()
			Expect(err).NotTo(HaveOccurred())
		}
	})
	It("only one runner can get the lock", func() {
		Consistently(getLockOwner, 5*retryInterval).Should(Equal(resultOwner))
	})
	Context("when locker owner expired", func() {
		BeforeEach(func() {
			callbackOnAcquireLock1 = func() {
				timeout := time.After(lockTTL * 2)
				for {
					select {
					case <-timeout:
						return
					default:
					}
					if getLockOwner() == lockOwner2 {
						return
					}
					time.Sleep(1 * time.Second)
				}
			}
			callbackOnAcquireLock2 = func() {
				timeout := time.After(lockTTL * 2)
				for {
					select {
					case <-timeout:
						return
					default:
					}
					if getLockOwner() == lockOwner1 {
						return
					}
					time.Sleep(1 * time.Second)
				}
			}
			lostLockChan1 = make(chan struct{}, 1)
			lostLockChan2 = make(chan struct{}, 1)
			callbackOnLostLock1 = func() {
				lostLockChan1 <- struct{}{}
			}
			callbackOnLostLock2 = func() {
				lostLockChan2 <- struct{}{}
			}
		})
		It("the expired lock owner should lost the lock and the competitor should get the lock", func() {
			Eventually(getLockOwner, 5*retryInterval, 1*time.Second).Should(Equal(resultOwner))
			if resultOwner == lockOwner1 {
				By("lockowner2 gets the lock due to lockowner1 is expired")
				Eventually(getLockOwner, 3*lockTTL, 1*time.Second).Should(Equal(lockOwner2))
				Eventually(lostLockChan1, 3*lockTTL, 1*time.Second).Should(Receive())

				By("lockowner1 gets the lock due to lockowner2 is expired")
				Eventually(getLockOwner, 3*lockTTL, 1*time.Second).Should(Equal(lockOwner1))
				Eventually(lostLockChan2, 3*lockTTL, 1*time.Second).Should(Receive())
			} else {
				By("lockowner1 gets the lock due to lockowner2 is expired")
				Eventually(getLockOwner, 3*lockTTL, 1*time.Second).Should(Equal(lockOwner1))
				Eventually(lostLockChan2, 3*lockTTL, 1*time.Second).Should(Receive())

				By("lockowner2 gets the lock due to lockowner1 is expired")
				Eventually(getLockOwner, 3*lockTTL, 1*time.Second).Should(Equal(lockOwner2))
				Eventually(lostLockChan1, 3*lockTTL, 1*time.Second).Should(Receive())
			}
		})

	})

})
