package server_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"code.cloudfoundry.org/app-autoscaler/src/autoscaler/fakes"
	"code.cloudfoundry.org/app-autoscaler/src/autoscaler/models"
	"code.cloudfoundry.org/app-autoscaler/src/autoscaler/scalingengine/server"
	"code.cloudfoundry.org/lager/v3/lagertest"
)

func TestVerticalScalingResponseDistinguishesAcceptance(t *testing.T) {
	for _, status := range []models.ScalingStatus{models.ScalingStatusPending, models.ScalingStatusUnknown} {
		engine := &fakes.FakeScalingEngine{}
		engine.ScaleStub = func(_ context.Context, appID string, trigger *models.Trigger) (*models.AppScalingResult, error) {
			if appID != "app" || trigger.ScalingDimension != models.ScalingDimensionVertical || trigger.Adjustment != "+128" {
				t.Fatalf("incorrect vertical trigger: app=%s, trigger=%+v", appID, trigger)
			}
			return &models.AppScalingResult{AppId: appID, Status: status, ScalingDimension: models.ScalingDimensionVertical}, nil
		}
		handler := server.NewScalingHandler(lagertest.NewTestLogger("vertical-handler"), &fakes.FakeScalingEngineDB{}, engine)
		recorder := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodPost, "/v1/apps/app/scale", strings.NewReader(`{"scaling_dimension":"vertical","metric_type":"entitlementutil","adjustment":"+128"}`))
		handler.Scale(recorder, req, map[string]string{"appid": "app"})
		if recorder.Code != http.StatusOK {
			t.Fatalf("unexpected HTTP status: %d", recorder.Code)
		}
		var result models.AppScalingResult
		if err := json.Unmarshal(recorder.Body.Bytes(), &result); err != nil {
			t.Fatal(err)
		}
		if result.Status != status || result.Status == models.ScalingStatusSucceeded || result.CooldownExpiredAt != 0 {
			t.Fatalf("acceptance incorrectly represented as completion: %+v", result)
		}
	}
}
