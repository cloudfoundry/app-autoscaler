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
		maxAmount           = 2
		validDuration       = 1 * Second
		expireDuration      = 5 * Second
		expireCheckInterval = 1 * Second

		moreMaxAmount       = 10
		longerValidDuration = 2 * Second
	)

	var (
		store Store
	)

	Describe("Increment", func() {
		Describe("with test default config", func() {
			BeforeEach(func() {
				store = NewStore(bucketCapacity, maxAmount, validDuration, expireDuration, expireCheckInterval, NewLogger("ratelimiter"))
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

				Sleep(validDuration)
				for i := 1; i < maxAmount+1; i++ {
					avail, err := store.Increment("foo")
					Expect(err).ToNot(HaveOccurred())
					Expect(avail).To(Equal(maxAmount - i))
				}
				avail, err = store.Increment("foo")
				Expect(err).To(HaveOccurred())
				Expect(avail).To(Equal(0))
			})
		})

		Describe("with moreMaxAmount and longerValidDuration", func() {
			BeforeEach(func() {
				store = NewStore(bucketCapacity, moreMaxAmount, longerValidDuration, expireDuration, expireCheckInterval, NewLogger("ratelimiter"))
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

				Sleep(longerValidDuration)
				for i := 1; i < moreMaxAmount+1; i++ {
					avail, err := store.Increment("foo")
					Expect(err).ToNot(HaveOccurred())
					Expect(avail).To(Equal(moreMaxAmount - i))
				}
				avail, err = store.Increment("foo")
				Expect(err).To(HaveOccurred())
				Expect(avail).To(Equal(0))
			})
		})
	})

	Describe("Stats", func() {
		BeforeEach(func() {
			store = NewStore(bucketCapacity, maxAmount, validDuration, expireDuration, expireCheckInterval, NewLogger("ratelimiter"))
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
			stats1 := store.Stats()
			Expect(len(stats1)).To(Equal(2))
			Expect(stats1[key1]).To(Equal(5))
			Expect(stats1[key2]).To(Equal(7))

			// should increase maxAmount * 2 tokens in each bucket
			Sleep(validDuration * 2)
			stats2 := store.Stats()
			Expect(len(stats2)).To(Equal(2))
			Expect(stats2[key1]).To(Equal(5 + maxAmount * 2))
			Expect(stats2[key2]).To(Equal(7 + maxAmount * 2))
		})
	})

	Describe("expiryCycle", func() {
		BeforeEach(func() {
			store = NewStore(bucketCapacity, maxAmount, validDuration, expireDuration, expireCheckInterval, NewLogger("ratelimiter"))
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
