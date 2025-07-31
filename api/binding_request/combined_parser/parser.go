package combined_parser

import (
	"code.cloudfoundry.org/app-autoscaler/src/autoscaler/api/binding_request"
)

// Combined parser that tries out all other parser that are associated to it in order
// and returns the first successful result.
//
// In case all associated parser fail, it returns the error of the first one.
type CombinedBindingRequestParser struct {
	parsers []binding_request.Parser
}

// Ensure CombinedBindingRequestParser implements the binding_request.Parser interface.
var _ binding_request.Parser = CombinedBindingRequestParser{}

func New(parsers []binding_request.Parser) CombinedBindingRequestParser {
	return CombinedBindingRequestParser{parsers: parsers}
}

func (p CombinedBindingRequestParser) Parse(
	bindingReqParams string,
) (binding_request.Parameters, error) {
	var firstErr error
	for i, parser := range p.parsers {
		params, err := parser.Parse(bindingReqParams)
		if i == 0 && err != nil {
			firstErr = err // Store the first error to return later if no parser is successful.
		} else if err == nil {
			return params, nil
		}
	}
	return binding_request.Parameters{}, firstErr
}
