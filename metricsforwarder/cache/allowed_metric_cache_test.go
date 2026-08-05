package cache_test

import (
	"sync"
	"time"

	mfcache "code.cloudfoundry.org/app-autoscaler/src/autoscaler/metricsforwarder/cache"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("AllowedMetricCache", func() {
	var c *mfcache.AllowedMetricCache

	BeforeEach(func() {
		c = mfcache.New(10*time.Minute, -1)
	})

	Describe("Get", func() {
		It("returns a clone so mutating the result does not affect the cached value", func() {
			c.Set("app", map[string]struct{}{"metric-a": {}}, 10*time.Minute)

			got, found := c.Get("app")
			Expect(found).To(BeTrue())
			Expect(got).To(HaveKey("metric-a"))

			// Mutate the returned map.
			got["metric-b"] = struct{}{}

			// The cached value must be unaffected.
			again, found := c.Get("app")
			Expect(found).To(BeTrue())
			Expect(again).To(HaveKey("metric-a"))
			Expect(again).NotTo(HaveKey("metric-b"))
		})

		It("returns (nil, false) on a miss", func() {
			got, found := c.Get("missing")
			Expect(found).To(BeFalse())
			Expect(got).To(BeNil())
		})
	})

	Describe("Set", func() {
		It("stores a clone so mutating the input after Set does not affect the cached value", func() {
			in := map[string]struct{}{"metric-a": {}}
			c.Set("app", in, 10*time.Minute)

			// Mutate the input after Set.
			in["metric-b"] = struct{}{}

			got, found := c.Get("app")
			Expect(found).To(BeTrue())
			Expect(got).To(HaveKey("metric-a"))
			Expect(got).NotTo(HaveKey("metric-b"))
		})
	})

	Describe("Replace", func() {
		It("stores a clone so mutating the input after Replace does not affect the cached value", func() {
			c.Set("app", map[string]struct{}{"metric-a": {}}, 10*time.Minute)

			in := map[string]struct{}{"metric-b": {}}
			Expect(c.Replace("app", in, 10*time.Minute)).To(Succeed())

			// Mutate the input after Replace.
			in["metric-c"] = struct{}{}

			got, found := c.Get("app")
			Expect(found).To(BeTrue())
			Expect(got).To(HaveKey("metric-b"))
			Expect(got).NotTo(HaveKey("metric-c"))
			Expect(got).NotTo(HaveKey("metric-a"))
		})
	})

	Describe("AppIDs", func() {
		It("returns the cached app IDs", func() {
			c.Set("app-a", map[string]struct{}{}, 10*time.Minute)
			c.Set("app-b", map[string]struct{}{}, 10*time.Minute)

			Expect(c.AppIDs()).To(ConsistOf("app-a", "app-b"))
		})
	})

	Describe("Delete", func() {
		It("removes the entry", func() {
			c.Set("app", map[string]struct{}{"metric-a": {}}, 10*time.Minute)
			c.Delete("app")

			_, found := c.Get("app")
			Expect(found).To(BeFalse())
		})
	})

	Describe("Flush", func() {
		It("removes all entries", func() {
			c.Set("app-a", map[string]struct{}{}, 10*time.Minute)
			c.Set("app-b", map[string]struct{}{}, 10*time.Minute)
			c.Flush()

			Expect(c.AppIDs()).To(BeEmpty())
		})
	})

	Describe("concurrent Get and Set", func() {
		It("does not race", func() {
			var wg sync.WaitGroup
			wg.Add(2)

			go func() {
				defer wg.Done()
				for i := 0; i < 1000; i++ {
					c.Set("app", map[string]struct{}{"metric-a": {}}, 10*time.Minute)
				}
			}()

			go func() {
				defer wg.Done()
				for i := 0; i < 1000; i++ {
					if got, found := c.Get("app"); found {
						// Mutate the clone to prove readers own their copy.
						got["reader-local"] = struct{}{}
					}
				}
			}()

			wg.Wait()
		})
	})
})
