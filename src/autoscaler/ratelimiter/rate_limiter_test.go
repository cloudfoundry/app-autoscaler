package ratelimiter_test

import (
	. "time"
	. "autoscaler/ratelimiter"

	. "code.cloudfoundry.org/lager"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("RateLimiter", func() {

	var (
		limitPerMinute int
		expireDuration Duration
		limiter        *RateLimiter
	)

	Describe("ExceedsLimit", func() {

		BeforeEach(func() {
			limitPerMinute = 5
			expireDuration = 10 * Minute
			limiter = NewRateLimiter(limitPerMinute, expireDuration, NewLogger("metricsforwarder-ratelimiter"))
		})

		It("reports if rate exceeded", func() {
			ip := "192.168.1.1"
			for i := 0; i < limitPerMinute; i++ {
				Expect(limiter.ExceedsLimit(ip)).To(BeFalse())
			}
			Expect(limiter.ExceedsLimit(ip)).To(BeTrue())
		})
	})

	Describe("Stats", func() {
		BeforeEach(func() {
			limitPerMinute = 10
			expireDuration = 10 * Minute
			limiter = NewRateLimiter(limitPerMinute, expireDuration, NewLogger("metricsforwarder-ratelimiter"))
		})

		It("reports stats ", func() {
			for i := 5; i < limitPerMinute; i++ {
				ip := "192.168.1.100"
				Expect(limiter.ExceedsLimit(ip)).To(BeFalse())
			}
			for i := 7; i < limitPerMinute; i++ {
				ip := "192.168.1.101"
				Expect(limiter.ExceedsLimit(ip)).To(BeFalse())
			}

			stats := limiter.GetStats()
			Expect(len(stats)).To(Equal(2))
		})

	})

})
