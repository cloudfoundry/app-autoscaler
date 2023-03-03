package ratelimiter_test

import (
	. "time"

	. "code.cloudfoundry.org/app-autoscaler/src/autoscaler/ratelimiter"

	. "code.cloudfoundry.org/lager/v3"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("Store", func() {
	const (
		bucketCapacity      = 20
		maxAmount           = 2
		validDuration       = 1 * Second
		expireDuration      = 5 * Second
		expireCheckInterval = 1 * Second

		moreMaxAmount       = 10
		longerValidDuration = 2 * Second
	)

	var (
		store            Store
		exceedErrMessage = "empty bucket"
	)

	Describe("Increment", func() {
		Describe("with test default config", func() {
			BeforeEach(func() {
				store = NewStore(bucketCapacity, maxAmount, validDuration, expireDuration, expireCheckInterval, NewLogger("ratelimiter"))
			})

			It("shows available", func() {
				for i := 1; i < bucketCapacity+1; i++ {
					err := store.Increment("foo")
					Expect(err).ToNot(HaveOccurred())
				}
				err := store.Increment("foo")
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring(exceedErrMessage))

				Sleep(validDuration)
				for i := 1; i < maxAmount+1; i++ {
					err := store.Increment("foo")
					Expect(err).ToNot(HaveOccurred())
				}
				err = store.Increment("foo")
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring(exceedErrMessage))
			})
		})

		Describe("with moreMaxAmount and longerValidDuration", func() {
			BeforeEach(func() {
				store = NewStore(bucketCapacity, moreMaxAmount, longerValidDuration, expireDuration, expireCheckInterval, NewLogger("ratelimiter"))
			})

			It("shows available", func() {
				for i := 1; i < bucketCapacity+1; i++ {
					err := store.Increment("foo")
					Expect(err).ToNot(HaveOccurred())
				}
				err := store.Increment("foo")
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring(exceedErrMessage))

				Sleep(longerValidDuration)
				for i := 1; i < moreMaxAmount+1; i++ {
					err := store.Increment("foo")
					Expect(err).ToNot(HaveOccurred())
				}
				err = store.Increment("foo")
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring(exceedErrMessage))
			})
		})
	})

	Describe("expiryCycle", func() {
		BeforeEach(func() {
			store = NewStore(bucketCapacity, maxAmount, validDuration, expireDuration, expireCheckInterval, NewLogger("ratelimiter"))
		})

		It("clean the bucket after expire ", func() {
			for i := 0; i < bucketCapacity; i++ {
				err := store.Increment("foo")
				Expect(err).ToNot(HaveOccurred())
			}
			err := store.Increment("foo")
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring(exceedErrMessage))

			Sleep(expireDuration + expireCheckInterval)
			err = store.Increment("foo")
			Expect(err).ToNot(HaveOccurred())
		})
	})

})
