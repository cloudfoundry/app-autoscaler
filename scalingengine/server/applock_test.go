package server_test

import (
	. "autoscaler/scalingengine/server"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var (
	ch      chan string
	appLock *AppLock
	err     interface{}
)

var _ = Describe("Applock", func() {
	BeforeEach(func() {
		appLock = NewAppLock()
		ch = make(chan string)
	})

	AfterEach(func() {
		close(ch)
	})

	Describe("Lock", func() {
		JustBeforeEach(func() {
			go lockApp(appLock, "an-app-id")
		})
		Context("when acquiring the lock of an app", func() {
			It("gets the lock", func() {
				Eventually(hasAcquiredLock).Should(BeTrue())
			})
		})

		Context("when acquiring the lock of a locked app", func() {
			BeforeEach(func() {
				appLock.Lock("an-app-id")
			})
			It("waits for the lock", func() {
				Consistently(hasAcquiredLock).Should(BeFalse())
			})
		})

		Context("acquiring the lock of an app when there is another app locked", func() {
			BeforeEach(func() {
				appLock.Lock("another-app-id")
			})

			It("gets the lock", func() {
				Eventually(hasAcquiredLock).Should(BeTrue())
			})
		})
	})

	Describe("Unlock", func() {
		JustBeforeEach(func() {
			defer func() {
				err = recover()
			}()
			appLock.UnLock("an-app-id")
		})
		Context("when the app is new and is not locked", func() {
			It("panics", func() {
				Expect(err).To(Equal("unlock app that was not locked"))
			})
		})

		Context("when an existing app is not locked", func() {
			BeforeEach(func() {
				appLock.Lock("an-app-id")
				appLock.UnLock("an-app-id")
			})
			It("panics", func() {
				Expect(err).To(Equal("sync: unlock of unlocked mutex"))
			})
		})

		Context("when the app is locked", func() {
			BeforeEach(func() {
				appLock.Lock("an-app-id")
			})
			It("releases the lock", func() {
				Expect(err).To(BeNil())
				go lockApp(appLock, "an-app-id")
				Eventually(hasAcquiredLock).Should(BeTrue())
			})
		})
	})
})

func lockApp(aLock *AppLock, appId string) {
	aLock.Lock(appId)
	ch <- "done"
}

func hasAcquiredLock() bool {
	select {
	case <-ch:
		return true
	default:
		return false
	}
}
