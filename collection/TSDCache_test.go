package collection_test

import (
	"time"

	. "autoscaler/collection"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("TSDCache", func() {
	var (
		cache    *TSDCache
		capacity int
		err      interface{}
		labels   map[string]string
	)

	JustBeforeEach(func() {
		defer func() {
			err = recover()
		}()
		cache = NewTSDCache(capacity)
	})

	Describe("NewTSDCache", func() {
		Context("when creating TSDCache with invalid capacity", func() {
			BeforeEach(func() {
				capacity = -1
			})
			It("panics", func() {
				Expect(err).To(Equal("invalid TSDCache capacity"))
			})
		})
		Context("when creating TSDCache with valid capacity", func() {
			BeforeEach(func() {
				capacity = 10

			})
			It("returns the TSDCache", func() {
				Expect(err).To(BeNil())
				Expect(cache).NotTo(BeNil())
			})
		})
	})

	Describe("Put", func() {
		Context("when cache capacity is 1", func() {
			BeforeEach(func() {
				capacity = 1
			})
			It("only caches the latest data", func() {
				cache.Put(TestTSD{10, nil})
				Expect(cache.String()).To(Equal("[{10 map[]}]"))
				cache.Put(TestTSD{20, nil})
				Expect(cache.String()).To(Equal("[{20 map[]}]"))
				cache.Put(TestTSD{15, nil})
				Expect(cache.String()).To(Equal("[{20 map[]}]"))
				cache.Put(TestTSD{30, nil})
				Expect(cache.String()).To(Equal("[{30 map[]}]"))
			})
		})
		Context("when data put to cache do not execeed the capacity", func() {
			BeforeEach(func() {
				capacity = 5
			})
			It("cache all data in ascending order", func() {
				cache.Put(TestTSD{20, nil})
				cache.Put(TestTSD{10, nil})
				cache.Put(TestTSD{40, nil})
				cache.Put(TestTSD{50, nil})
				cache.Put(TestTSD{30, nil})
				Expect(cache.String()).To(Equal("[{10 map[]} {20 map[]} {30 map[]} {40 map[]} {50 map[]}]"))
			})
		})
		Context("when data put to cache execeed the capacity", func() {
			BeforeEach(func() {
				capacity = 3
			})
			It("caches latest data in ascending order", func() {
				cache.Put(TestTSD{20, nil})
				Expect(cache.String()).To(Equal("[{20 map[]}]"))
				cache.Put(TestTSD{10, nil})
				Expect(cache.String()).To(Equal("[{10 map[]} {20 map[]}]"))
				cache.Put(TestTSD{40, nil})
				Expect(cache.String()).To(Equal("[{10 map[]} {20 map[]} {40 map[]}]"))
				cache.Put(TestTSD{50, nil})
				Expect(cache.String()).To(Equal("[{20 map[]} {40 map[]} {50 map[]}]"))
				cache.Put(TestTSD{30, nil})
				Expect(cache.String()).To(Equal("[{30 map[]} {40 map[]} {50 map[]}]"))
				cache.Put(TestTSD{50, nil})
				Expect(cache.String()).To(Equal("[{40 map[]} {50 map[]} {50 map[]}]"))
			})
		})

	})

	Describe("Query", func() {
		Context("when cache is empty", func() {
			BeforeEach(func() {
				capacity = 3
			})
			It("return empty results", func() {
				result, ok := cache.Query(0, time.Now().UnixNano(), labels)
				Expect(ok).To(BeTrue())
				Expect(result).To(BeEmpty())
			})
		})
		Context("when data put to cache do not execeeds the capacity", func() {
			BeforeEach(func() {
				capacity = 5
			})
			It("returns the data in [start, end)", func() {
				cache.Put(TestTSD{20, nil})
				result, ok := cache.Query(10, 40, labels)
				Expect(ok).To(BeTrue())
				Expect(result).To(Equal([]TSD{TestTSD{20, nil}}))

				cache.Put(TestTSD{10, nil})
				result, ok = cache.Query(10, 40, labels)
				Expect(ok).To(BeTrue())
				Expect(result).To(Equal([]TSD{TestTSD{10, nil}, TestTSD{20, nil}}))

				cache.Put(TestTSD{40, nil})
				result, ok = cache.Query(10, 40, labels)
				Expect(ok).To(BeTrue())
				Expect(result).To(Equal([]TSD{TestTSD{10, nil}, TestTSD{20, nil}}))

				cache.Put(TestTSD{30, nil})
				result, ok = cache.Query(10, 40, labels)
				Expect(ok).To(BeTrue())
				Expect(result).To(Equal([]TSD{TestTSD{10, nil}, TestTSD{20, nil}, TestTSD{30, nil}}))

				cache.Put(TestTSD{50, nil})
				result, ok = cache.Query(10, 40, labels)
				Expect(ok).To(BeTrue())
				Expect(result).To(Equal([]TSD{TestTSD{10, nil}, TestTSD{20, nil}, TestTSD{30, nil}}))
			})
		})

		Context("when data put to cache execeed the capacity", func() {
			BeforeEach(func() {
				capacity = 3
			})

			Context("when all queried data are guarenteed  in cache", func() {
				It("returns the data in [start, end)", func() {
					cache.Put(TestTSD{20, nil})
					cache.Put(TestTSD{10, nil})
					cache.Put(TestTSD{40, nil})
					cache.Put(TestTSD{30, nil})

					result, ok := cache.Query(30, 50, labels)
					Expect(ok).To(BeTrue())
					Expect(result).To(Equal([]TSD{TestTSD{30, nil}, TestTSD{40, nil}}))

					cache.Put(TestTSD{50, nil})
					result, ok = cache.Query(35, 50, labels)
					Expect(ok).To(BeTrue())
					Expect(result).To(Equal([]TSD{TestTSD{40, nil}}))
				})
				Context("when querying with labels", func() {
					BeforeEach(func() {
						capacity = 5
					})
					It("returns the data with all the labels", func() {
						cache.Put(TestTSD{20, map[string]string{"tom": "cat", "pig": "pepper"}})
						cache.Put(TestTSD{10, nil})
						cache.Put(TestTSD{40, map[string]string{"jerry": "mouse", "tom": "cat", "peppa": "pig"}})
						cache.Put(TestTSD{30, map[string]string{"jerry": "mouse"}})
						cache.Put(TestTSD{50, nil})

						result, ok := cache.Query(20, 60, map[string]string{"jerry": "mouse", "tom": "cat"})
						Expect(ok).To(BeTrue())
						Expect(result).To(Equal(([]TSD{TestTSD{40, map[string]string{"jerry": "mouse", "tom": "cat", "peppa": "pig"}}})))
					})

				})

			})
			Context("when queried data are possibly not in cache", func() {
				It("returns false", func() {
					cache.Put(TestTSD{20, nil})
					cache.Put(TestTSD{10, nil})
					cache.Put(TestTSD{40, nil})
					cache.Put(TestTSD{30, nil})

					_, ok := cache.Query(10, 50, labels)
					Expect(ok).To(BeFalse())

					cache.Put(TestTSD{50, nil})
					_, ok = cache.Query(30, 50, labels)
					Expect(ok).To(BeFalse())
				})

			})

		})

	})
})
