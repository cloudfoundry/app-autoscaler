package ratelimiter_test

import (
	. "time"

	. "code.cloudfoundry.org/app-autoscaler/src/autoscaler/ratelimiter"

	. "code.cloudfoundry.org/lager/v3"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("RateLimiter", func() {

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
		limiter *RateLimiter
	)

	Describe("ExceedsLimit", func() {

		Describe("with test default config", func() {
			BeforeEach(func() {
				limiter = NewRateLimiter(bucketCapacity, maxAmount, validDuration, expireDuration, expireCheckInterval, NewLogger("ratelimiter"))
			})

			It("reports if rate exceeded", func() {
				key := "192.168.1.100"
				for i := 0; i < bucketCapacity; i++ {
					Expect(limiter.ExceedsLimit(key)).To(BeFalse())
				}
				Expect(limiter.ExceedsLimit(key)).To(BeTrue())

				Sleep(validDuration)
				for i := 0; i < maxAmount; i++ {
					Expect(limiter.ExceedsLimit(key)).To(BeFalse())
				}
				Expect(limiter.ExceedsLimit(key)).To(BeTrue())
			})
		})

		Describe("with moreMaxAmount and longerValidDuration", func() {
			BeforeEach(func() {
				limiter = NewRateLimiter(bucketCapacity, moreMaxAmount, longerValidDuration, expireDuration, expireCheckInterval, NewLogger("ratelimiter"))
			})

			It("reports if rate exceeded", func() {
				key := "192.168.1.100"
				for i := 0; i < bucketCapacity; i++ {
					Expect(limiter.ExceedsLimit(key)).To(BeFalse())
				}
				Expect(limiter.ExceedsLimit(key)).To(BeTrue())

				Sleep(longerValidDuration)
				for i := 0; i < moreMaxAmount; i++ {
					Expect(limiter.ExceedsLimit(key)).To(BeFalse())
				}
				Expect(limiter.ExceedsLimit(key)).To(BeTrue())
			})
		})

	})

	Describe("Expire", func() {
		BeforeEach(func() {
			limiter = NewRateLimiter(bucketCapacity, maxAmount, validDuration, expireDuration, expireCheckInterval, NewLogger("ratelimiter"))
		})

		It("clean the bucket after expire ", func() {
			key := "192.168.1.100"
			for i := 0; i < bucketCapacity; i++ {
				Expect(limiter.ExceedsLimit(key)).To(BeFalse())
			}
			Expect(limiter.ExceedsLimit(key)).To(BeTrue())

			Sleep(expireDuration + expireCheckInterval)
			Expect(limiter.ExceedsLimit(key)).To(BeFalse())
		})
	})
})
