package envelopeprocessor

import (
	"math"
	"testing"

	"code.cloudfoundry.org/app-autoscaler/src/autoscaler/models"
	"code.cloudfoundry.org/go-loggregator/v10/rpc/loggregator_v2"
	"code.cloudfoundry.org/lager/v3/lagertest"
)

func entitlementEnvelope(timestamp int64, values map[string]float64) *loggregator_v2.Envelope {
	gauges := make(map[string]*loggregator_v2.GaugeValue)
	for name, value := range values {
		unit := "bytes"
		if name == "cpu_entitlement" {
			unit = "percentage"
		}
		gauges[name] = &loggregator_v2.GaugeValue{Unit: unit, Value: value}
	}
	return &loggregator_v2.Envelope{
		SourceId: "app", InstanceId: "0", Timestamp: timestamp,
		Tags: map[string]string{"process_type": "web", "process_id": "process", "instance_id": "container"},
		Message: &loggregator_v2.Envelope_Gauge{Gauge: &loggregator_v2.Gauge{Metrics: gauges}},
	}
}

func TestEntitlementInputs(t *testing.T) {
	for _, tc := range []struct {
		name string
		memory, quota, cpu float64
		want string
	}{
		{"memory pressure", 90, 100, 20, "90"},
		{"CPU pressure above entitlement", 20, 100, 140.2, "141"},
		{"rounding", 30.1, 100, 10, "31"},
		{"real zero", 0, 100, 0, "0"},
		{"zero quota", 50, 0, 40, ""},
		{"negative CPU", 50, 100, -1, ""},
		{"NaN memory", math.NaN(), 100, 40, ""},
		{"infinite CPU", 50, 100, math.Inf(1), ""},
	} {
		t.Run(tc.name, func(t *testing.T) {
			envelopes := []*loggregator_v2.Envelope{
				entitlementEnvelope(100, map[string]float64{"memory": tc.memory, "memory_quota": tc.quota}),
				entitlementEnvelope(100, map[string]float64{"cpu_entitlement": tc.cpu}),
			}
			metrics := GetEntitlementInstanceMetrics(envelopes, 200)
			if len(metrics) != 1 || metrics[0].Value != tc.want || metrics[0].Timestamp != 100 || metrics[0].CollectedAt != 200 {
				t.Fatalf("unexpected derived metrics: %+v", metrics)
			}
		})
	}
}

func TestEntitlementRejectsUnmatchedInputs(t *testing.T) {
	for _, tc := range []struct {
		name string
		change func(*loggregator_v2.Envelope, *loggregator_v2.Envelope) []*loggregator_v2.Envelope
	}{
		{"missing CPU", func(memory, cpu *loggregator_v2.Envelope) []*loggregator_v2.Envelope { return []*loggregator_v2.Envelope{memory} }},
		{"missing memory", func(memory, cpu *loggregator_v2.Envelope) []*loggregator_v2.Envelope {
			delete(memory.GetGauge().Metrics, "memory")
			return []*loggregator_v2.Envelope{memory, cpu}
		}},
		{"missing quota", func(memory, cpu *loggregator_v2.Envelope) []*loggregator_v2.Envelope {
			delete(memory.GetGauge().Metrics, "memory_quota")
			return []*loggregator_v2.Envelope{memory, cpu}
		}},
		{"different nanosecond", func(memory, cpu *loggregator_v2.Envelope) []*loggregator_v2.Envelope {
			cpu.Timestamp++
			return []*loggregator_v2.Envelope{memory, cpu}
		}},
		{"different container", func(memory, cpu *loggregator_v2.Envelope) []*loggregator_v2.Envelope {
			cpu.Tags["instance_id"] = "new-container"
			return []*loggregator_v2.Envelope{memory, cpu}
		}},
		{"different process", func(memory, cpu *loggregator_v2.Envelope) []*loggregator_v2.Envelope {
			cpu.Tags["process_id"] = "new-process"
			return []*loggregator_v2.Envelope{memory, cpu}
		}},
		{"unknown source", func(memory, cpu *loggregator_v2.Envelope) []*loggregator_v2.Envelope {
			memory.Tags, cpu.Tags = nil, nil
			return []*loggregator_v2.Envelope{memory, cpu}
		}},
		{"invalid index", func(memory, cpu *loggregator_v2.Envelope) []*loggregator_v2.Envelope {
			memory.InstanceId, cpu.InstanceId = "invalid", "invalid"
			return []*loggregator_v2.Envelope{memory, cpu}
		}},
		{"duplicate source even with identical value", func(memory, cpu *loggregator_v2.Envelope) []*loggregator_v2.Envelope {
			return []*loggregator_v2.Envelope{memory, cpu, cpu}
		}},
		{"conflicting modern and deprecated identity", func(memory, cpu *loggregator_v2.Envelope) []*loggregator_v2.Envelope {
			cpu.DeprecatedTags = map[string]*loggregator_v2.Value{"instance_id": {Data: &loggregator_v2.Value_Text{Text: "other-container"}}}
			return []*loggregator_v2.Envelope{memory, cpu}
		}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			memory := entitlementEnvelope(100, map[string]float64{"memory": 90, "memory_quota": 100})
			cpu := entitlementEnvelope(100, map[string]float64{"cpu_entitlement": 10})
			metrics := GetEntitlementInstanceMetrics(tc.change(memory, cpu), 200)
			if len(metrics) == 0 {
				t.Fatal("expected an invalid measurement marker")
			}
			for _, metric := range metrics {
				if metric.Value != "" {
					t.Fatalf("ambiguous/incomplete source became valid: %+v", metric)
				}
			}
		})
	}
}

func TestEntitlementIgnoresWorkerAndSupportsDeprecatedTags(t *testing.T) {
	web := entitlementEnvelope(100, map[string]float64{"memory": 20, "memory_quota": 100, "cpu_entitlement": 130})
	worker := entitlementEnvelope(100, map[string]float64{"memory": 99, "memory_quota": 100, "cpu_entitlement": 999})
	worker.Tags["process_type"] = "worker"
	web.DeprecatedTags = make(map[string]*loggregator_v2.Value)
	for name, value := range web.Tags {
		web.DeprecatedTags[name] = &loggregator_v2.Value{Data: &loggregator_v2.Value_Text{Text: value}}
	}
	web.Tags = nil
	metrics := GetEntitlementInstanceMetrics([]*loggregator_v2.Envelope{web, worker}, 200)
	if len(metrics) != 1 || metrics[0].Value != "130" || metrics[0].ProcessType != "web" {
		t.Fatalf("worker leaked into the web metric: %+v", metrics)
	}
}

func TestCompactionRetainsIdentityAndDoesNotMutateInputs(t *testing.T) {
	memory := entitlementEnvelope(100, map[string]float64{"memory": 90, "memory_quota": 100})
	cpu := entitlementEnvelope(100, map[string]float64{"cpu_entitlement": 20})
	other := entitlementEnvelope(100, map[string]float64{"cpu_entitlement": 150})
	other.Tags["instance_id"] = "new-container"
	processor := NewProcessor(lagertest.NewTestLogger("entitlement"))
	envelopes := []*loggregator_v2.Envelope{memory, cpu, other}
	if compacted := processor.CompactEnvelopes(envelopes); len(compacted) != 2 {
		t.Fatalf("different rollout containers were compacted: %+v", compacted)
	}
	for i := 0; i < 2; i++ {
		metrics := processor.GetGaugeMetrics(envelopes[:2], 200)
		var derived []models.AppInstanceMetric
		for _, metric := range metrics {
			if metric.Name == models.MetricNameEntitlementUtil {
				derived = append(derived, metric)
			}
		}
		if len(derived) != 1 || derived[0].Value != "90" {
			t.Fatalf("repeated processing changed the inputs: %+v", derived)
		}
	}
	if _, mutated := memory.GetGauge().Metrics["cpu_entitlement"]; mutated {
		t.Fatal("compaction mutated the original memory envelope")
	}
}
