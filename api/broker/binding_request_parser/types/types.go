package types

import (
	"encoding/json"
	"fmt"

	"github.com/xeipuuv/gojsonschema"

	"code.cloudfoundry.org/app-autoscaler/src/autoscaler/models"
)

type Parser interface {
	Parse(bindingReqParams string, ccAppGuid models.GUID) (models.AppScalingConfig, error)
}

// ================================================================================
// Parsing or validation errors
// ================================================================================

type JsonSchemaError []gojsonschema.ResultError

func (e *JsonSchemaError) Error() string {
	var errors []gojsonschema.ResultError = *e
	return fmt.Sprintf("%s", errors)
}

var _ error = &JsonSchemaError{}

// MarshalJSON implements the json.Marshaler interface for JsonSchemaError.
// It serializes the schema validation errors to JSON with "context" and "description" fields.
func (e *JsonSchemaError) MarshalJSON() ([]byte, error) {
	var errors []gojsonschema.ResultError = *e

	type validationError struct {
		Context     string `json:"context"`
		Description string `json:"description"`
	}
	validationErrors := make([]validationError, 0, len(errors))

	for _, err := range errors {
		validationErrors = append(validationErrors, validationError{
			Context:     err.Context().String(),
			Description: err.Description(),
		})
	}

	return json.Marshal(validationErrors)
}

var _ json.Marshaler = &JsonSchemaError{}

// This error type is used, when the passed binding-request fails to validate against the schema.
type BindReqNoAppGuid struct{}

func (e BindReqNoAppGuid) Error() string {
	return "error: service must be bound to an application; Please provide a GUID of an app!"
}

var _ error = BindReqNoAppGuid{}
