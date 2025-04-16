package binding_requests

import (
	"fmt"

	"github.com/xeipuuv/gojsonschema"

	"code.cloudfoundry.org/app-autoscaler/src/autoscaler/models"
)

type Parameters struct {
	Configuration *models.BindingConfig `json:"configuration"`
	ScalingPolicy *models.ScalingPolicy `json:"scaling-policy"`
}

type JsonSchemaError []gojsonschema.ResultError

func (e JsonSchemaError) Error() string {
	var errors []gojsonschema.ResultError = e
	return fmt.Sprintf("%s", errors)
}

type Parser interface {
	// NewFromString(jsonSchema string) (BindingRequestParser, error)
	// NewFromFile(pathToSchemaFile string) (BindingRequestParser, error)
	Parse(bindingReqParams string) (Parameters, error)
}
