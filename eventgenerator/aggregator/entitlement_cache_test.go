package aggregator

import (
	"testing"
	"time"

	"code.cloudfoundry.org/app-autoscaler/src/autoscaler/collection"
	"code.cloudfoundry.org/app-autoscaler/src/autoscaler/models"
)

func TestEntitlementCacheInsertsCohortBoundaries(t *testing.T) {
	manager := &AppManager{
		metricCache: map[string]*collection.TSDCache{"app": collection.NewTSDCache(20)},
		entitlementSources: make(map[string]string),
	}
	newMetric := func(source string) *models.AppMetric {
		return &models.AppMetric{AppId: "app", MetricType: models.MetricNameEntitlementUtil, Value: "90", Unit: models.UnitPercentage, Timestamp: time.Now().UnixNano(), SourceID: source}
	}
	first := newMetric("old-cohort")
	if !manager.SaveMetricToCache(first) || first.Value != "" { t.Fatalf("first post-start point must be invalid: %+v", first) }
	stable := newMetric("old-cohort")
	if !manager.SaveMetricToCache(stable) || stable.Value != "90" { t.Fatalf("stable cohort was discarded: %+v", stable) }
	newContainer := newMetric("new-cohort")
	if !manager.SaveMetricToCache(newContainer) || newContainer.Value != "" { t.Fatalf("rollout boundary was hidden: %+v", newContainer) }
	stable = newMetric("new-cohort")
	if !manager.SaveMetricToCache(stable) || stable.Value != "90" { t.Fatalf("new cohort did not stabilize: %+v", stable) }
	missingIdentity := newMetric("")
	manager.SaveMetricToCache(missingIdentity)
	if missingIdentity.Value != "" { t.Fatal("missing source metadata became valid in the cache") }
	manager.refreshMetricCache(nil)
	if len(manager.entitlementSources) != 0 { t.Fatal("deleted policy kept obsolete cohort metadata") }
}

func TestMonitorsIncludeVerticalOnlyRulesAndDeduplicateMetrics(t *testing.T) {
	aggregator := &Aggregator{defaultStatWindowSecs: 60}
	rule := &models.ScalingRule{MetricType: models.MetricNameEntitlementUtil}
	policy := &models.PolicyDefinition{VerticalScalingRules: []*models.ScalingRule{rule, rule}}
	policies := map[string]*models.AppPolicy{"app": {AppId: "app", ScalingPolicy: policy}}
	monitors := aggregator.getAppMonitors(policies)
	if len(monitors) != 1 || monitors["app-entitlementutil"].MetricType != models.MetricNameEntitlementUtil { t.Fatalf("vertical-only rule was not monitored: %+v", monitors) }
	policy.ScalingRules = []*models.ScalingRule{{MetricType: models.MetricNameCPUUtil}}
	if monitors = aggregator.getAppMonitors(policies); len(monitors) != 2 { t.Fatalf("horizontal/vertical monitors were not combined: %+v", monitors) }
}
