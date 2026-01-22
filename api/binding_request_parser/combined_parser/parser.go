// 🚧 To-do: Rewrite this combined parser in the sense that he has several dedicated parsers and
// delegates the task to the dedicated parser based on "schema-version".

package combined_parser

// import (
//	"code.cloudfoundry.org/app-autoscaler/src/autoscaler/api/binding_request_parser"
// )

// // Combined parser that tries out all other parser that are associated to it in order
// // and returns the first successful result.
// //
// // In case all associated parser fail, it returns the error of the first one.
// type CombinedBindingRequestParser struct {
//	parsers []binding_request_parser.Parser
// }

// // Ensure CombinedBindingRequestParser implements the binding_request_parser.Parser interface.
// var _ binding_request_parser.Parser = CombinedBindingRequestParser{}

// func New(parsers []binding_request_parser.Parser) CombinedBindingRequestParser {
//	return CombinedBindingRequestParser{parsers: parsers}
// }

// func (p CombinedBindingRequestParser) Parse(
//	bindingReqParams string,
// ) (binding_request_parser.Parameters, error) {
//	var firstErr error
//	for i, parser := range p.parsers {
//		params, err := parser.Parse(bindingReqParams)
//		if i == 0 && err != nil {
//			firstErr = err // Store the first error to return later if no parser is successful.
//		} else if err == nil {
//			return params, nil
//		}
//	}
//	return binding_request_parser.Parameters{}, firstErr
// }
