package ratelimiter_test

import (
	. "autoscaler/metricsforwarder/ratelimiter"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("RateLimiter", func() {

	var (
		limit   int
		limiter *RateLimiter
	)

	Describe("ExceedsLimit", func() {

		BeforeEach(func() {
			limit = 5
			limiter = NewRateLimiter(limit)
		})

		It("reports if rate exceeded", func() {
			ip := "192.168.1.1"
			for i := 0; i < limit; i++ {
				Expect(limiter.ExceedsLimit(ip)).To(BeFalse())
			}
			Expect(limiter.ExceedsLimit(ip)).To(BeTrue())
		})
	})

	Describe("Stats", func() {
		BeforeEach(func() {
			limit = 10
			limiter = NewRateLimiter(limit)
		})

		It("reports stats ", func() {
			for i := 5; i < limit; i++ {
				ip := "192.168.1.100"
				Expect(limiter.ExceedsLimit(ip)).To(BeFalse())
			}
			for i := 7; i < limit; i++ {
				ip := "192.168.1.101"
				Expect(limiter.ExceedsLimit(ip)).To(BeFalse())
			}

			stats := limiter.GetStats()
			Expect(len(stats)).To(Equal(2))
		})

	})

})
