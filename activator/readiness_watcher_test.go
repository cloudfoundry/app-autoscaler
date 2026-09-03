package activator_test

import (
	"context"
	"errors"
	"time"

	. "code.cloudfoundry.org/app-autoscaler/src/autoscaler/activator"

	"code.cloudfoundry.org/lager/v3/lagertest"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

// fakeStream / fakeSubscriber feed scripted route events to the watcher.
type fakeStream struct {
	events chan RouteEvent
	closed chan struct{}
}

func newFakeStream() *fakeStream {
	return &fakeStream{events: make(chan RouteEvent, 8), closed: make(chan struct{})}
}

func (s *fakeStream) Next() (RouteEvent, error) {
	select {
	case e := <-s.events:
		return e, nil
	case <-s.closed:
		return RouteEvent{}, errors.New("closed")
	}
}

func (s *fakeStream) Close() error {
	select {
	case <-s.closed:
	default:
		close(s.closed)
	}
	return nil
}

type fakeSubscriber struct{ stream *fakeStream }

func (f *fakeSubscriber) Subscribe() (RouteEventStream, error) { return f.stream, nil }

// recordingParker records Unpark calls.
type recordingParker struct{ unparked chan string }

func (p *recordingParker) Park(context.Context, string) error { return nil }
func (p *recordingParker) Unpark(_ context.Context, appID string) error {
	p.unparked <- appID
	return nil
}

var _ = Describe("ReadinessWatcher", func() {
	var (
		stream   *fakeStream
		registry Registry
		parker   *recordingParker
		watcher  ReadinessWatcher
		cancel   context.CancelFunc
	)

	BeforeEach(func() {
		stream = newFakeStream()
		registry = NewInMemoryRegistry()
		parker = &recordingParker{unparked: make(chan string, 1)}
		logger := lagertest.NewTestLogger("watcher-test")
		watcher = NewReadinessWatcher(logger, &fakeSubscriber{stream: stream}, registry, parker)

		var ctx context.Context
		ctx, cancel = context.WithCancel(context.Background())
		go func() { _ = watcher.Run(ctx) }()
	})

	AfterEach(func() {
		cancel()
	})

	Context("when a parked app's route is upserted", func() {
		BeforeEach(func() {
			registry.MarkParked("app-1", []string{"https://app-1.example.com"})
		})

		It("wakes a waiter and unparks the app", func() {
			ready := watcher.WaitForReady(context.Background(), "app-1")

			stream.events <- RouteEvent{RouteURL: "https://app-1.example.com", Action: "Upsert"}

			Eventually(ready).Should(Receive(BeNil()))
			Eventually(parker.unparked).Should(Receive(Equal("app-1")))
		})
	})

	Context("when an unrelated route is upserted", func() {
		BeforeEach(func() {
			registry.MarkParked("app-1", []string{"https://app-1.example.com"})
		})

		It("does not unpark", func() {
			stream.events <- RouteEvent{RouteURL: "https://other.example.com", Action: "Upsert"}
			Consistently(parker.unparked, 200*time.Millisecond).ShouldNot(Receive())
		})
	})

	Context("when a Delete event arrives", func() {
		BeforeEach(func() {
			registry.MarkParked("app-1", []string{"https://app-1.example.com"})
		})

		It("is ignored", func() {
			stream.events <- RouteEvent{RouteURL: "https://app-1.example.com", Action: "Delete"}
			Consistently(parker.unparked, 200*time.Millisecond).ShouldNot(Receive())
		})
	})

	Context("WaitForReady with a cancelled context", func() {
		It("returns the context error", func() {
			ctx, c := context.WithCancel(context.Background())
			ready := watcher.WaitForReady(ctx, "app-unknown")
			c()
			Eventually(ready).Should(Receive(Equal(context.Canceled)))
		})
	})
})
