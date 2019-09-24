package ratelimiter_test

import (
	. "time"

	. "autoscaler/ratelimiter"

	. "code.cloudfoundry.org/lager"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("Store", func() {
	var (
		store          Store
		limitPerMinute int
		expireDuration Duration
	)

	Describe("Increment", func() {
		BeforeEach(func() {
			limitPerMinute = 10
			expireDuration = 10 * Minute
			store = NewStore(limitPerMinute, expireDuration, NewLogger("metricsforwarder-ratelimiter"))
		})

		It("shows available", func() {
			for i := 1; i < limitPerMinute+1; i++ {
				avail, err := store.Increment("foo")
				Expect(err).ToNot(HaveOccurred())
				Expect(avail).To(Equal(limitPerMinute - i))
			}
			avail, err := store.Increment("foo")
			Expect(err).To(HaveOccurred())
			Expect(avail).To(Equal(0))

		})
	})

})
