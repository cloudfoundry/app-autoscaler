package binding_request_parser

import (
	"fmt"

	"github.com/xeipuuv/gojsonschema"

	"code.cloudfoundry.org/app-autoscaler/src/autoscaler/models"
)

type JsonSchemaError []gojsonschema.ResultError

func (e *JsonSchemaError) Error() string {
	var errors []gojsonschema.ResultError = *e
	return fmt.Sprintf("%s", errors)
}

type Parser interface {
	// Default policies are specified on service-instance-level. Consequently, we need to leave the
	// field Parameters.ScalingPolicy empty when no policy has been specified and instead … let the
	// consumer of the BindingRequest decided what to do with this (i.e. he will use then the
	// default-policy.)
	Parse(bindingReqParams string, ccAppGuid models.GUID) (models.AppScalingConfig, error)
}
