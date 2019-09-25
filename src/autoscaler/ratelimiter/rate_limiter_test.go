package ratelimiter_test

import (
	. "time"
	. "autoscaler/ratelimiter"

	. "code.cloudfoundry.org/lager"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("RateLimiter", func() {

	const (
		bucketCapacity      = 20
		fillInterval        = 1 * Second
		expireDuration      = 5 * Second
		expireCheckInterval = 1 * Second
	)

	var (
		limiter *RateLimiter
	)

	Describe("ExceedsLimit", func() {

		BeforeEach(func() {
			limiter = NewRateLimiter(bucketCapacity, fillInterval, expireDuration, expireCheckInterval, NewLogger("ratelimiter"))
		})

		It("reports if rate exceeded", func() {
			key := "192.168.1.100"
			for i := 0; i < bucketCapacity; i++ {
				Expect(limiter.ExceedsLimit(key)).To(BeFalse())
			}
			Expect(limiter.ExceedsLimit(key)).To(BeTrue())
		})
	})

	Describe("GetStats", func() {
		BeforeEach(func() {
			limiter = NewRateLimiter(bucketCapacity, fillInterval, expireDuration, expireCheckInterval, NewLogger("ratelimiter"))
		})

		It("reports stats ", func() {
			for i := 5; i < bucketCapacity; i++ {
				key := "192.168.1.100"
				Expect(limiter.ExceedsLimit(key)).To(BeFalse())
			}
			for i := 7; i < bucketCapacity; i++ {
				key := "192.168.1.101"
				Expect(limiter.ExceedsLimit(key)).To(BeFalse())
			}

			stats := limiter.GetStats()
			Expect(len(stats)).To(Equal(2))
		})
	})

	Describe("Expire", func() {
		BeforeEach(func() {
			limiter = NewRateLimiter(bucketCapacity, fillInterval, expireDuration, expireCheckInterval, NewLogger("ratelimiter"))
		})

		It("clean the bucket after expire ", func() {
			key := "192.168.1.100"
			for i := 0; i < bucketCapacity; i++ {
				Expect(limiter.ExceedsLimit(key)).To(BeFalse())
			}
			Expect(limiter.ExceedsLimit(key)).To(BeTrue())
			Expect(len(limiter.GetStats())).To(Equal(1))

			Sleep(expireDuration + expireCheckInterval)
			Expect(len(limiter.GetStats())).To(Equal(0))
			Expect(limiter.ExceedsLimit(key)).To(BeFalse())
		})
	})
})
