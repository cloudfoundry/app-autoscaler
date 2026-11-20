package appstate_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"code.cloudfoundry.org/app-autoscaler/src/autoscaler/appstate"
	"code.cloudfoundry.org/app-autoscaler/src/autoscaler/cf"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

func TestAppState(t *testing.T) { RegisterFailHandler(Fail); RunSpecs(t, "App State Suite") }

type source struct {
	pages   []appstate.Changes
	err     error
	cursors []int64
}

func (s *source) Changes(_ context.Context, cursor int64) (appstate.Changes, error) {
	s.cursors = append(s.cursors, cursor)
	if s.err != nil {
		return appstate.Changes{}, s.err
	}
	page := s.pages[0]
	s.pages = s.pages[1:]
	return page, nil
}
func (*source) Invalidate(context.Context, string) error { return nil }

var _ = Describe("No-op state cache", func() {
	var cache *appstate.Cache
	var now time.Time
	var app *cf.AppAndProcesses
	BeforeEach(func() {
		now = time.Now()
		cache = appstate.NewCache(30 * time.Second)
		app = &cf.AppAndProcesses{App: &cf.App{State: "STARTED"}, Processes: cf.Processes{{Instances: 1}}}
		Expect(cache.Poll(context.Background(), &source{pages: []appstate.Changes{{CheckedAt: now}}})).To(Succeed())
	})
	It("uses a verified count until its original expiry, not sliding expiry", func() {
		cache.Remember("app", app, now, cache.Token())
		count, ok := cache.Instances("app", now.Add(20*time.Second))
		Expect(ok).To(BeTrue())
		Expect(count).To(Equal(1))
		_, ok = cache.Instances("app", now.Add(31*time.Second))
		Expect(ok).To(BeFalse())
	})
	It("invalidates on manual changes and does not restore stale observer counts", func() {
		cache.Remember("app", app, now, cache.Token())
		Expect(cache.Poll(context.Background(), &source{pages: []appstate.Changes{{CheckedAt: now, Cursor: 1,
			Items: []appstate.Observation{{AppID: "app", Instances: 0}}}}})).To(Succeed())
		_, ok := cache.Instances("app", now)
		Expect(ok).To(BeFalse())
	})
	It("fences a CF read that overlaps an invalidation", func() {
		token := cache.Token()
		cache.Invalidate("app")
		cache.Remember("app", app, now, token)
		_, ok := cache.Instances("app", now)
		Expect(ok).To(BeFalse())
	})
	It("does not cache stopped or disabled apps", func() {
		app.App.State = "STOPPED"
		cache.Remember("app", app, now, cache.Token())
		_, ok := cache.Instances("app", now)
		Expect(ok).To(BeFalse())
		app.App.State = "STARTED"
		label := ""
		app.App.DisableAutoscaling = &label
		cache.Remember("app", app, now, cache.Token())
		_, ok = cache.Instances("app", now)
		Expect(ok).To(BeFalse())
	})
	It("fails open on observer failure or reader lag", func() {
		cache.Remember("app", app, now, cache.Token())
		Expect(cache.Poll(context.Background(), &source{err: errors.New("offline")})).To(HaveOccurred())
		_, ok := cache.Instances("app", now)
		Expect(ok).To(BeFalse())
		Expect(cache.Poll(context.Background(), &source{pages: []appstate.Changes{{CheckedAt: now.Add(-time.Minute)}}})).To(Succeed())
		cache.Remember("app", app, now, cache.Token())
		_, ok = cache.Instances("app", now)
		Expect(ok).To(BeFalse())
	})
	It("drains all pages before enabling cached decisions", func() {
		s := &source{pages: []appstate.Changes{{Cursor: 7, More: true, CheckedAt: now}, {Cursor: 9, CheckedAt: now}}}
		Expect(cache.Poll(context.Background(), s)).To(Succeed())
		Expect(s.cursors).To(Equal([]int64{0, 7}))
		cache.Remember("app", app, now, cache.Token())
		_, ok := cache.Instances("app", now)
		Expect(ok).To(BeTrue())
	})
})
