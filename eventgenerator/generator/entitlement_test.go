package generator

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"code.cloudfoundry.org/app-autoscaler/src/autoscaler/db"
	"code.cloudfoundry.org/app-autoscaler/src/autoscaler/eventgenerator/config"
	"code.cloudfoundry.org/app-autoscaler/src/autoscaler/models"
	"code.cloudfoundry.org/clock/fakeclock"
	"code.cloudfoundry.org/lager/v3/lagertest"
	circuit "github.com/rubyist/circuitbreaker"
)

func TestEntitlementBreachCoverage(t *testing.T) {
	end := time.Unix(1000, 0)
	start := end.Add(-time.Minute)
	points := func(offsets ...int) []*models.AppMetric {
		result := make([]*models.AppMetric, 0, len(offsets))
		for _, offset := range offsets {
			result = append(result, &models.AppMetric{Timestamp: end.Add(time.Duration(offset)*time.Second).UnixNano(), Value: "90", Unit: models.UnitPercentage})
		}
		return result
	}
	for _, tc := range []struct {
		name string
		metrics []*models.AppMetric
		want bool
	}{
		{"complete duration including boundary", points(-80, -60, -40, -20, -1), true},
		{"nearby boundary", points(-61, -40, -20, -1), true},
		{"missing start", points(-40, -20, -1), false},
		{"missing middle", points(-61, -1), false},
		{"stale tail", points(-61, -41), false},
		{"stale boundary", points(-100, -40, -20, -1), false},
		{"future point", points(-61, -40, -20, 1), false},
		{"duplicate timestamp", points(-61, -40, -40, -1), false},
		{"out of order", points(-61, -20, -40, -1), false},
		{"single value", points(-61), false},
		{"empty", nil, false},
		{"nil point", []*models.AppMetric{nil}, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			result := entitlementBreachMetrics(tc.metrics, start, end, 30*time.Second)
			if (len(result) > 0) != tc.want {
				t.Fatalf("unexpected coverage decision: %+v", result)
			}
		})
	}
	metrics := points(-61, -40, -20, -1)
	metrics[1].Value = ""
	result := entitlementBreachMetrics(metrics, start, end, 30*time.Second)
	e := &Evaluator{logger: lagertest.NewTestLogger("coverage")}
	if breached, _ := checkForBreach(result, e, &models.Trigger{}, ">", 80); breached {
		t.Fatal("missing input marker was skipped during breach evaluation")
	}
	metrics[1].Value = "malformed"
	if breached, _ := checkForBreach(result, e, &models.Trigger{}, ">", 80); breached {
		t.Fatal("malformed measurement was skipped")
	}
}

func TestTriggerOrderingAndAppWideCooldown(t *testing.T) {
	now := time.Now()
	manager, err := NewAppEvaluationManager(lagertest.NewTestLogger("trigger-order"), time.Second, fakeclock.NewFakeClock(now), nil, nil, config.CircuitBreakerConfig{})
	if err != nil { t.Fatal(err) }
	rule := &models.ScalingRule{MetricType: models.MetricNameCPUUtil, Threshold: 80, Operator: ">", Adjustment: "+1"}
	vertical := &models.ScalingRule{MetricType: models.MetricNameEntitlementUtil, Threshold: 80, Operator: ">", Adjustment: "+128"}
	policy := &models.PolicyDefinition{ScalingRules: []*models.ScalingRule{rule}, VerticalScalingRules: []*models.ScalingRule{vertical}}
	policies := map[string]*models.AppPolicy{"app": {AppId: "app", ScalingPolicy: policy}}
	for _, priority := range []models.ScalingDimension{models.ScalingDimensionHorizontal, models.ScalingDimensionVertical} {
		policy.ScalingPriority = priority
		triggers := manager.getTriggers(policies)["app"]
		if len(triggers) != 2 { t.Fatalf("missing rule dimension: %+v", triggers) }
		if priority == models.ScalingDimensionVertical {
			if triggers[0].ScalingDimension != models.ScalingDimensionVertical || triggers[1].ScalingDimension != "" { t.Fatalf("wrong vertical ordering: %+v", triggers) }
		} else if triggers[0].ScalingDimension != "" || triggers[1].ScalingDimension != models.ScalingDimensionVertical {
			t.Fatalf("legacy horizontal trigger was changed: %+v", triggers)
		}
	}
	manager.SetCoolDownExpired("app", now.Add(time.Minute).UnixNano())
	if triggers := manager.getTriggers(policies); len(triggers) != 0 { t.Fatalf("cooldown must block both dimensions: %+v", triggers) }
	manager.SetCoolDownExpired("app", now.Add(-time.Second).UnixNano())
	if triggers := manager.getTriggers(policies)["app"]; len(triggers) != 2 { t.Fatalf("expired cooldown kept a dimension blocked: %+v", triggers) }
}

func TestPriorityFallbackOnlyOnDimensionSpecificIgnore(t *testing.T) {
	for _, tc := range []struct {
		name, reason string
		status models.ScalingStatus
		cooldown int64
		wantCalls int
	}{
		{"priority maximum memory", "priority_requires_max_memory", models.ScalingStatusIgnored, 0, 2},
		{"priority minimum memory", "priority_requires_min_memory", models.ScalingStatusIgnored, 0, 2},
		{"priority maximum instances", "priority_requires_max_instances", models.ScalingStatusIgnored, 0, 2},
		{"priority minimum instances", "priority_requires_min_instances", models.ScalingStatusIgnored, 0, 2},
		{"unchanged target", "unchanged_target", models.ScalingStatusIgnored, 0, 2},
		{"active deployment", "active_deployment", models.ScalingStatusIgnored, 0, 1},
		{"cooldown", "cooldown", models.ScalingStatusIgnored, time.Now().Add(time.Minute).UnixNano(), 1},
		{"unknown legacy ignore", "", models.ScalingStatusIgnored, 0, 1},
		{"stopped app", "app_stopped", models.ScalingStatusIgnored, 0, 1},
		{"pending", "", models.ScalingStatusPending, 0, 1},
		{"unknown POST outcome", "", models.ScalingStatusUnknown, 0, 1},
		{"quota failure is not a priority substitute", "quota_exceeded", models.ScalingStatusFailed, 0, 1},
		{"success", "", models.ScalingStatusSucceeded, 0, 1},
	} {
		for _, firstDimension := range []models.ScalingDimension{models.ScalingDimensionHorizontal, models.ScalingDimensionVertical} {
			t.Run(tc.name+"/"+string(firstDimension), func(t *testing.T) {
				var calls []models.ScalingDimension
				server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					var trigger models.Trigger
					if err := json.NewDecoder(r.Body).Decode(&trigger); err != nil { t.Error(err) }
					calls = append(calls, trigger.ScalingDimension)
					result := scalingEngineResult{AppScalingResult: models.AppScalingResult{Status: tc.status, CooldownExpiredAt: tc.cooldown}, Reason: tc.reason}
					if len(calls) > 1 { result.Status = models.ScalingStatusSucceeded }
					if err := json.NewEncoder(w).Encode(result); err != nil { t.Error(err) }
				}))
				defer server.Close()
				query := func(appID, metricType string, _ int64, end int64, _ db.OrderType) ([]*models.AppMetric, error) {
					var result []*models.AppMetric
					for _, seconds := range []int64{70, 50, 30, 10} {
						result = append(result, &models.AppMetric{AppId: appID, MetricType: metricType, Timestamp: end-seconds*int64(time.Second), Value: "90", Unit: models.UnitPercentage})
					}
					return result, nil
				}
				e := NewEvaluator(lagertest.NewTestLogger("fallback"), server.Client(), server.URL, nil, 60, query,
					func(string) *circuit.Breaker { return nil }, func(string, int64) {}, 20*time.Second)
				otherDimension := models.ScalingDimensionVertical
				if firstDimension == models.ScalingDimensionVertical { otherDimension = models.ScalingDimensionHorizontal }
				trigger := func(dimension models.ScalingDimension) *models.Trigger {
					return &models.Trigger{AppId: "app", ScalingDimension: dimension, MetricType: models.MetricNameEntitlementUtil, BreachDurationSeconds: 60, Threshold: 80, Operator: ">", Adjustment: "+1"}
				}
				// A second matched rule from the rejected class must not prevent
				// the other class from being evaluated, or cause another mutation.
				e.doEvaluate([]*models.Trigger{trigger(firstDimension), trigger(firstDimension), trigger(otherDimension)})
				if len(calls) != tc.wantCalls { t.Fatalf("calls=%v, wanted %d", calls, tc.wantCalls) }
				if len(calls) == 2 && calls[1] != otherDimension { t.Fatalf("same dimension retried instead of fallback: %v", calls) }
			})
		}
	}
}

func TestAcceptedAndUnknownDoNotPopulateCooldownCache(t *testing.T) {
	for _, status := range []models.ScalingStatus{models.ScalingStatusPending, models.ScalingStatusUnknown, models.ScalingStatusSuperseded, models.ScalingStatusAborted, models.ScalingStatusFailed, models.ScalingStatusSucceeded, models.ScalingStatusIgnored} {
		for _, httpStatus := range []int{http.StatusOK, http.StatusAccepted} {
			t.Run(time.Duration(status).String()+"/"+http.StatusText(httpStatus), func(t *testing.T) {
				server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
					w.WriteHeader(httpStatus)
					if err := json.NewEncoder(w).Encode(models.AppScalingResult{Status: status, CooldownExpiredAt: 123}); err != nil { t.Error(err) }
				}))
				defer server.Close()
				var cached int64
				e := NewEvaluator(lagertest.NewTestLogger("cache"), server.Client(), server.URL, nil, 60, nil, nil,
					func(_ string, expires int64) { cached = expires })
				if err := e.sendTriggerAlarm(&models.Trigger{AppId: "app", ScalingDimension: models.ScalingDimensionVertical}); err != nil { t.Fatal(err) }
				want := int64(0)
				if status == models.ScalingStatusSucceeded || status == models.ScalingStatusIgnored { want = 123 }
				if cached != want { t.Fatalf("status %d: cached cooldown %d, wanted %d", status, cached, want) }
			})
		}
	}
}
