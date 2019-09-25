package ratelimiter_test

import (
	. "time"

	. "autoscaler/ratelimiter"

	. "code.cloudfoundry.org/lager"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("Store", func() {
	const (
		bucketCapacity      = 20
		fillInterval        = 1 * Second
		expireDuration      = 5 * Second
		expireCheckInterval = 1 * Second
	)

	var (
		store Store
	)

	Describe("Increment", func() {
		BeforeEach(func() {
			store = NewStore(bucketCapacity, fillInterval, expireDuration, expireCheckInterval, NewLogger("ratelimiter"))
		})

		It("shows available", func() {
			for i := 1; i < bucketCapacity+1; i++ {
				avail, err := store.Increment("foo")
				Expect(err).ToNot(HaveOccurred())
				Expect(avail).To(Equal(bucketCapacity - i))
			}
			avail, err := store.Increment("foo")
			Expect(err).To(HaveOccurred())
			Expect(avail).To(Equal(0))
		})
	})

	Describe("Stats", func() {
		BeforeEach(func() {
			store = NewStore(bucketCapacity, fillInterval, expireDuration, expireCheckInterval, NewLogger("ratelimiter"))
		})

		It("get stats ", func() {
			key1 := "foo"
			key2 := "bar"
			for i := 5; i < bucketCapacity; i++ {
				store.Increment(key1)
			}
			for i := 7; i < bucketCapacity; i++ {
				store.Increment(key2)
			}

			stats := store.Stats()
			Expect(len(stats)).To(Equal(2))
			Expect(stats[key1]).To(Equal(5))
			Expect(stats[key2]).To(Equal(7))
		})
	})

	Describe("expiryCycle", func() {
		BeforeEach(func() {
			store = NewStore(bucketCapacity, fillInterval, expireDuration, expireCheckInterval, NewLogger("ratelimiter"))
		})

		It("clean the bucket after expire ", func() {
			avail := 0
			for i := 0; i < bucketCapacity; i++ {
				avail, _ = store.Increment("foo")
			}
			Expect(avail).To(Equal(0))
			Expect(len(store.Stats())).To(Equal(1))

			Sleep(expireDuration + expireCheckInterval)
			Expect(len(store.Stats())).To(Equal(0))
			avail, _ = store.Increment("foo")
			Expect(avail).To(Equal(bucketCapacity - 1))
		})
	})

})
