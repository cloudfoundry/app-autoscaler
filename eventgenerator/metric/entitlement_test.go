package metric

import (
	"context"
	"fmt"
	"net/url"
	"testing"
	"time"

	"code.cloudfoundry.org/app-autoscaler/src/autoscaler/envelopeprocessor"
	"code.cloudfoundry.org/app-autoscaler/src/autoscaler/models"
	logcache "code.cloudfoundry.org/go-log-cache/v3"
	"code.cloudfoundry.org/go-log-cache/v3/rpc/logcache_v1"
	"code.cloudfoundry.org/go-loggregator/v10/rpc/loggregator_v2"
	"code.cloudfoundry.org/lager/v3/lagertest"
)

type entitlementLogCache struct {
	envelopes []*loggregator_v2.Envelope
	start time.Time
	options url.Values
}

func (c *entitlementLogCache) Read(_ context.Context, _ string, start time.Time, opts ...logcache.ReadOption) ([]*loggregator_v2.Envelope, error) {
	c.start, c.options = start, make(url.Values)
	for _, option := range opts { option(nil, c.options) }
	return c.envelopes, nil
}

func (*entitlementLogCache) PromQL(context.Context, string, ...logcache.PromQLOption) (*logcache_v1.PromQL_InstantQueryResult, error) {
	return nil, fmt.Errorf("entitlement must not use independently averaged PromQL inputs")
}

func TestEntitlementFetcherUsesOneRawInputRead(t *testing.T) {
	end := time.Now()
	newEnvelope := func(name, unit string, value float64) *loggregator_v2.Envelope {
		return &loggregator_v2.Envelope{
			SourceId: "app", InstanceId: "0", Timestamp: end.Add(-time.Second).UnixNano(),
			Tags: map[string]string{"process_type": "web", "process_id": "process", "instance_id": "container"},
			Message: &loggregator_v2.Envelope_Gauge{Gauge: &loggregator_v2.Gauge{Metrics: map[string]*loggregator_v2.GaugeValue{name: {Unit: unit, Value: value}}}},
		}
	}
	client := &entitlementLogCache{envelopes: []*loggregator_v2.Envelope{
		newEnvelope("memory", "bytes", 10), newEnvelope("memory_quota", "bytes", 100), newEnvelope("cpu_entitlement", "percentage", 150),
	}}
	logger := lagertest.NewTestLogger("entitlement-fetcher")
	fetcher := StandardLogCacheFetcherCreator.NewLogCacheFetcher(logger, client, envelopeprocessor.NewProcessor(logger), 40*time.Second)
	metrics, err := fetcher.FetchMetrics("app", models.MetricNameEntitlementUtil, end.Add(-10*time.Minute), end)
	if err != nil || len(metrics) != 1 || metrics[0].Value != "150" {
		t.Fatalf("failed to fetch the derived metric: metrics=%+v err=%v", metrics, err)
	}
	if client.options.Get("name_filter") != "^(memory|memory_quota|cpu_entitlement)$" || !client.start.Equal(end.Add(-40*time.Second)) {
		t.Fatalf("wrong input/time filter: start=%v options=%v", client.start, client.options)
	}
	client.envelopes = client.envelopes[:2]
	metrics, err = fetcher.FetchMetrics("app", models.MetricNameEntitlementUtil, end.Add(-time.Minute), end)
	if err != nil || len(metrics) != 1 || metrics[0].Value != "" {
		t.Fatalf("missing CPU was replaced with a value: metrics=%+v err=%v", metrics, err)
	}
	client.envelopes = make([]*loggregator_v2.Envelope, 1000)
	if _, err = fetcher.FetchMetrics("app", models.MetricNameEntitlementUtil, end.Add(-time.Minute), end); err == nil {
		t.Fatal("truncated input cohort was accepted")
	}
}
