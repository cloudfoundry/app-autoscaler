package combined_parser

import (
	"encoding/json"

	"github.com/xeipuuv/gojsonschema"

	"code.cloudfoundry.org/app-autoscaler/src/autoscaler/models"
	"code.cloudfoundry.org/app-autoscaler/src/autoscaler/api/binding_request"
)

// Combined parser that tries out all other parser that are associated to it in order
// and returns the first successful result.
//
// In case all associated parser fail, it returns the error of the first one.
type CombinedBindingRequestParser struct {
	parsers []binding_request.Parser
}

func (p CombinedBindingRequestParser) Parse(
	bindingReqParams string,
) (binding_request.Parameters, error) {
	return binding_request.Parameters{}, models.ErrUnimplemented // 🚧 To-do
}
