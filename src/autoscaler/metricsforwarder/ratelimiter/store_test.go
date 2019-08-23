package ratelimiter_test

import (
	. "autoscaler/metricsforwarder/ratelimiter"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("Store", func() {
	var (
		store Store
		limit int
	)

	Describe("Increment", func() {
		BeforeEach(func() {
			limit = 10
			store = NewStore(limit)
		})

		It("shows available", func() {
			for i := 1; i < limit+1; i++ {
				avail, err := store.Increment("foo")
				Expect(err).ToNot(HaveOccurred())
				Expect(avail).To(Equal(limit - i))
			}
			avail, err := store.Increment("foo")
			Expect(err).To(HaveOccurred())
			Expect(avail).To(Equal(0))

		})
	})

})
