package main // 🚧 To-do: Rename package to `bindrequestparser`

import (
	"encoding/json"
	"fmt"

	"github.com/xeipuuv/gojsonschema"

	"code.cloudfoundry.org/app-autoscaler/src/autoscaler/models"
)

// 🚧 To-do: Remove this debug-code once, finished!
var Unimplemented error = fmt.Errorf("🚧 To-do: This is still uninmplemented!")


type BindingRequestParameters struct {
	Configuration *models.BindingConfig `json:"configuration"`
	ScalingPolicy *models.ScalingPolicy `json:"scaling-policy"`
}

type BindingRequestParser struct {
	schema *gojsonschema.Schema
}

func (p BindingRequestParser) Parse(bindingReqParams string) (BindingRequestParameters, error) {
	documentLoader := gojsonschema.NewStringLoader(bindingReqParams)
	validationResult, err := p.schema.Validate(documentLoader)
	if err != nil {
		// Defined by the implementation of `Validate`, this only happens, if the provided document
		// (in this context `documentLoader`) can not be loaded.
		return BindingRequestParameters{}, err
	} else if ! validationResult.Valid() {
		// The error contains a description of all detected violations against the schema.
		return BindingRequestParameters{}, fmt.Errorf("%s", validationResult.Errors())
	}

	var result BindingRequestParameters
	err = json.Unmarshal([]byte(bindingReqParams), &result)
	if err != nil {
		return BindingRequestParameters{}, err
	} else {
		return result, nil
	}
}

func New(bindReqParamSchemaPath string) (BindingRequestParser, error) {
	// 🚧 To-do: Refine on error-type to provide specific one!

	// Type for parameter `bindReqParamSchemaPath` is same type as used in golang's std-library
	schemaLoader := gojsonschema.NewReferenceLoader(bindReqParamSchemaPath)
	schema, err := gojsonschema.NewSchema(schemaLoader)
	if err != nil {
		return BindingRequestParser{}, err
	} else {
		return BindingRequestParser{schema: schema}, nil
	}
}



// 🚧 To-do: Debug-code!
func main() {

	// JSON string to be parsed
	// jsonString := `{
	//	"invalid-param": "The whole json does not match the schema."
	// }`
	// jsonString := `{
	//	"configuration": {
	//	   "app_guid": "x342.|"
	//	}
	// }`
	jsonString := `{
		"configuration": {
		   "app_guid": "8d0cee08-23ad-4813-a779-ad8118ea0b91"
		}
	}`

	parser, err := New("file://./binding-request.json")
	if err != nil {
		fmt.Println(err)
		return
	}

	result, err := parser.Parse(jsonString)
	if err != nil {
		fmt.Println(err)
		return
	}

	// // Print the parsed data
	// fmt.Printf("Property1: %s\n", result)
	fmt.Printf("AppGUID = %s", result.Configuration.AppGUID)
}
