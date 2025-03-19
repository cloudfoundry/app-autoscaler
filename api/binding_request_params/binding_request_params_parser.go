package main // 🚧 To-do: Rename package to `bindrequestparser`

import (
	"encoding/json"
	"fmt"
	"github.com/xeipuuv/gojsonschema"
)

// 🚧 To-do: Remove this debug-code once, finished!
var Unimplemented error = fmt.Errorf("🚧 To-do: This is still uninmplemented!")


type BindingRequestParameters struct {
	Property1 string `json:"property1"`
	Property2 int    `json:"property2"`
}

type BindReqParamParser struct {
	schema *gojsonschema.Schema
}

func (p BindReqParamParser) Parse(bindingReqParams string) (BindingRequestParameters, error) {
	documentLoader := gojsonschema.NewStringLoader(bindingReqParams)
	_, err := p.schema.Validate(documentLoader)

	var result BindingRequestParameters
	err = json.Unmarshal([]byte(bindingReqParams), &result)
	if err != nil {
		return BindingRequestParameters{}, err
	} else {
		return result, nil
	}
}

func New(bindReqParamSchemaPath string) (BindReqParamParser, error) {
	// 🚧 To-do: Refine on error-type to provide specific one!

	// Type for parameter `bindReqParamSchemaPath` is same type as used in golang's std-library
	schemaLoader := gojsonschema.NewReferenceLoader(bindReqParamSchemaPath)
	schema, err := gojsonschema.NewSchema(schemaLoader)
	if err != nil {
		return BindReqParamParser{}, err
	} else {
		return BindReqParamParser{schema: schema}, nil
	}
}



// 🚧 To-do: Debug-code!
func main() {

	// JSON string to be parsed
	jsonString := `{
		"property1": "example string",
		"property2": 42
	}`

	parser, err := New("file://./binding_request.schema.json")
	if err != nil {
		fmt.Println(err)
		return
	}

	result, err := parser.Parse(jsonString)
	if err != nil {
		fmt.Println(err)
		return
	}

	// Print the parsed data
	fmt.Printf("Property1: %s\n", result.Property1)
	fmt.Printf("Property2: %d\n", result.Property2)
}
