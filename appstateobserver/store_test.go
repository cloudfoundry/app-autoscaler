//nolint:testpackage // Tests initialize the real migration through the observer's private database connection.
package appstateobserver

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"net/url"
	"os"
	"testing"
	"time"

	"code.cloudfoundry.org/app-autoscaler/src/autoscaler/appstate"
	"code.cloudfoundry.org/app-autoscaler/src/autoscaler/cf"
	"code.cloudfoundry.org/app-autoscaler/src/autoscaler/db"
	"code.cloudfoundry.org/app-autoscaler/src/autoscaler/fakes"
	"code.cloudfoundry.org/lager/v3/lagertest"
	"github.com/google/uuid"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"go.yaml.in/yaml/v4"
)

func TestObserver(t *testing.T) { RegisterFailHandler(Fail); RunSpecs(t, "App State Observer Suite") }

type observerCF struct{ *fakes.FakeCFClient }

func (*observerCF) ListAuditEvents(context.Context, time.Time, time.Time) ([]cf.AuditEvent, error) {
	return nil, nil
}

var _ = Describe("Durable observer journal", func() {
	var store *Store
	var admin *sql.DB
	var schema string
	var ctx context.Context
	BeforeEach(func() {
		databaseURL := os.Getenv("APPSTATE_TEST_DB_URL")
		if databaseURL == "" {
			Skip("set APPSTATE_TEST_DB_URL to run isolated PostgreSQL integration tests")
		}
		ctx = context.Background()
		var err error
		admin, err = sql.Open("pgx", databaseURL)
		Expect(err).NotTo(HaveOccurred())
		schema = "appstate_test_" + fmt.Sprintf("%x", uuid.New())
		_, err = admin.ExecContext(ctx, "CREATE SCHEMA "+schema)
		Expect(err).NotTo(HaveOccurred())
		u, err := url.Parse(databaseURL)
		Expect(err).NotTo(HaveOccurred())
		query := u.Query()
		query.Set("search_path", schema)
		u.RawQuery = query.Encode()
		store, err = NewStore(ctx, db.DatabaseConfig{URL: u.String()})
		Expect(err).NotTo(HaveOccurred())
		// Execute the actual deployment migration, not a duplicated test schema.
		data, err := os.ReadFile("db/appstateobserver.db.changelog.yml")
		Expect(err).NotTo(HaveOccurred())
		var migration struct {
			DatabaseChangeLog []struct {
				ChangeSet struct {
					Changes []struct {
						SQL struct {
							SQL string `yaml:"sql"`
						} `yaml:"sql"`
					} `yaml:"changes"`
				} `yaml:"changeSet"`
			} `yaml:"databaseChangeLog"`
		}
		Expect(yaml.Unmarshal(data, &migration)).To(Succeed())
		_, err = store.db.ExecContext(ctx, migration.DatabaseChangeLog[0].ChangeSet.Changes[0].SQL.SQL)
		Expect(err).NotTo(HaveOccurred())
	})
	AfterEach(func() {
		if store != nil {
			Expect(store.Close()).To(Succeed())
		}
		if admin != nil {
			_, err := admin.ExecContext(context.Background(), "DROP SCHEMA "+schema+" CASCADE")
			Expect(err).NotTo(HaveOccurred())
			Expect(admin.Close()).To(Succeed())
		}
	})
	read := func() appstate.Changes {
		result, err := store.Changes(ctx, 0)
		Expect(err).NotTo(HaveOccurred())
		return result
	}
	poll := func(events ...cf.AuditEvent) {
		Expect(store.Poll(ctx, func(time.Time) (Window, error) {
			return Window{Until: time.Now(), Managed: map[string]bool{"app": true}, Events: events}, nil
		})).To(Succeed())
	}
	It("rolls back membership, deduplication and checkpoint on a failed window", func() {
		before := read()
		Expect(store.Poll(ctx, func(time.Time) (Window, error) { return Window{}, errors.New("CF offline") })).To(HaveOccurred())
		after := read()
		Expect(after.CheckedAt).To(Equal(before.CheckedAt))
		Expect(after.Items).To(BeEmpty())
	})
	It("deduplicates overlap events and fences stale refresh writes", func() {
		poll()
		item := read().Items[0]
		item.Dirty = false
		item.VerifiedAt = time.Now()
		Expect(store.Save(ctx, item.Revision, item)).To(Succeed())
		old := read().Items[0]
		event := cf.AuditEvent{GUID: "scale-event", CreatedAt: time.Now(), TargetGUID: "app", TargetType: "app"}
		poll(event)
		invalidated := read().Items[0]
		Expect(invalidated.Dirty).To(BeTrue())
		Expect(store.Save(ctx, old.Revision, old)).To(Succeed())
		Expect(read().Items[0].Revision).To(Equal(invalidated.Revision))
		poll(event)
		Expect(read().Items[0].Revision).To(Equal(invalidated.Revision))
	})
	It("publishes unbinding tombstones that remain visible after restart", func() {
		poll()
		cursor := read().Cursor
		Expect(store.Poll(ctx, func(time.Time) (Window, error) { return Window{Until: time.Now(), Managed: map[string]bool{}}, nil })).To(Succeed())
		changes, err := store.Changes(ctx, cursor)
		Expect(err).NotTo(HaveOccurred())
		Expect(changes.Items).To(HaveLen(1))
		Expect(changes.Items[0].Managed).To(BeFalse())
		Expect(read().Items[0].Managed).To(BeFalse())
	})
	It("uses only one foundation reader across replicas", func() {
		entered := make(chan struct{})
		release := make(chan struct{})
		done := make(chan error, 1)
		go func() {
			done <- store.Poll(ctx, func(time.Time) (Window, error) {
				close(entered)
				<-release
				return Window{Until: time.Now(), Managed: map[string]bool{}}, nil
			})
		}()
		<-entered
		called := false
		err := store.Poll(ctx, func(time.Time) (Window, error) { called = true; return Window{}, nil })
		close(release)
		Expect(err).NotTo(HaveOccurred())
		Expect(called).To(BeFalse())
		Expect(<-done).To(Succeed())
	})
	It("provides a paginated compacted feed with no lost cursor entries", func() {
		members := make(map[string]bool)
		for i := range 501 {
			members[fmt.Sprintf("app-%03d", i)] = true
		}
		Expect(store.Poll(ctx, func(time.Time) (Window, error) { return Window{Until: time.Now(), Managed: members}, nil })).To(Succeed())
		first := read()
		Expect(first.Items).To(HaveLen(500))
		Expect(first.More).To(BeTrue())
		second, err := store.Changes(ctx, first.Cursor)
		Expect(err).NotTo(HaveOccurred())
		Expect(second.Items).To(HaveLen(1))
		Expect(second.More).To(BeFalse())
	})
	It("resets a cursor ahead of a restored journal", func() {
		poll()
		changes, err := store.Changes(ctx, 99999)
		Expect(err).NotTo(HaveOccurred())
		Expect(changes.Reset).To(BeTrue())
		Expect(changes.Items).To(HaveLen(1))
		Expect(changes.Cursor).To(Equal(read().Cursor))
	})
	It("refreshes manual zero and scale-back-up changes but not unchanged apps", func() {
		poll()
		cfc := &observerCF{FakeCFClient: &fakes.FakeCFClient{}}
		cfc.GetAppAndProcessesReturns(&cf.AppAndProcesses{App: &cf.App{State: "STARTED"}, Processes: cf.Processes{{Instances: 0}}}, nil)
		cfc.GetAppRoutesReturns([]cf.Route{{URL: "app.example.com"}}, nil)
		observer := &Observer{Store: store, CF: cfc, Logger: lagertest.NewTestLogger("observer"), ReconcileInterval: time.Minute}
		Expect(observer.Refresh(ctx)).To(Succeed())
		Expect(read().Items[0].Instances).To(BeZero())
		Expect(read().Items[0].Dirty).To(BeFalse())
		poll()
		Expect(observer.Refresh(ctx)).To(Succeed())
		Expect(cfc.GetAppAndProcessesCallCount()).To(Equal(1))
		poll(cf.AuditEvent{GUID: "manual-up", CreatedAt: time.Now(), TargetGUID: "app", TargetType: "app"})
		cfc.GetAppAndProcessesReturns(&cf.AppAndProcesses{App: &cf.App{State: "STARTED"}, Processes: cf.Processes{{Instances: 2}}}, nil)
		Expect(observer.Refresh(ctx)).To(Succeed())
		Expect(read().Items[0].Instances).To(Equal(2))
		Expect(cfc.GetAppAndProcessesCallCount()).To(Equal(2))
	})
})
