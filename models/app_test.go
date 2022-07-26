package models_test

import (
	"encoding/json"
	"testing"

	"code.cloudfoundry.org/app-autoscaler/src/autoscaler/models"

	"github.com/stretchr/testify/assert"
)

func TestMarshalAppEntity(t *testing.T) {
	appEntity := models.AppEntity{Instances: 3}
	jsonBytes, err := json.Marshal(appEntity)
	assert.NoError(t, err)
	assert.JSONEqf(t, `{"instances":3}`, string(jsonBytes), "state is not omitted")
}
