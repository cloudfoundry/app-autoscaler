package clean_parser

import (
	"encoding/json"

	"github.com/xeipuuv/gojsonschema"

	"code.cloudfoundry.org/app-autoscaler/src/autoscaler/api/binding_request"
)

type CleanBindingRequestParser struct {
	schema *gojsonschema.Schema
}

func new(jsonLoader gojsonschema.JSONLoader) (CleanBindingRequestParser, error) {
	schema, err := gojsonschema.NewSchema(jsonLoader)
	if err != nil {
		return CleanBindingRequestParser{}, err
	} else {
		return CleanBindingRequestParser{schema: schema}, nil
	}
}

func NewFromString(jsonSchema string) (CleanBindingRequestParser, error) {
	schemaLoader := gojsonschema.NewStringLoader(jsonSchema)
	return new(schemaLoader)
}

func NewFromFile(pathToSchemaFile string) (CleanBindingRequestParser, error) {
	// The type for parameter `pathToSchemaFile` is same type as used in golang's std-library
	schemaLoader := gojsonschema.NewReferenceLoader(pathToSchemaFile)
	return new(schemaLoader)
}

func (p CleanBindingRequestParser) Parse(bindingReqParams string) (binding_request.Parameters, error) {
	documentLoader := gojsonschema.NewStringLoader(bindingReqParams)
	validationResult, err := p.schema.Validate(documentLoader)
	if err != nil {
		// Defined by the implementation of `Validate`, this only happens, if the provided document
		// (in this context `documentLoader`) can not be loaded.
		return binding_request.Parameters{}, err
	} else if !validationResult.Valid() {
		// The error contains a description of all detected violations against the schema.
		allErrors := binding_request.JsonSchemaError(validationResult.Errors())
		return binding_request.Parameters{}, allErrors
	}

	var result binding_request.Parameters
	err = json.Unmarshal([]byte(bindingReqParams), &result)
	if err != nil {
		return binding_request.Parameters{}, err
	} else {
		return result, nil
	}
}
