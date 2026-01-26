package binding_request_parser

import (
	"encoding/json"
	"fmt"

	"code.cloudfoundry.org/app-autoscaler/src/autoscaler/api/broker/binding_request_parser/legacy"
	brp "code.cloudfoundry.org/app-autoscaler/src/autoscaler/api/broker/binding_request_parser/types"
	"code.cloudfoundry.org/app-autoscaler/src/autoscaler/api/broker/binding_request_parser/v0_1"
	"code.cloudfoundry.org/app-autoscaler/src/autoscaler/models"
)

// The parser for bind-requests needs to deal with a multi-version-schema. It therefore is a
// aggregation of multiple dedicated parsers where each one is specialised for precisely one
// particular schema-version. The top-level-parser then just extracts the schema that is referenced
// within a binding-request and dispatches the parsing to the responsible one of the specialised
// parsers.
//
// All the specialised parsers as well as the top-level one share the same interface `Parser` (as
// well as some other dedicated types) within the sub-package `types`.

// ================================================================================
// To-level parser
// ================================================================================

type BindRequestParser struct {
	legacyParser legacy.BindingRequestParser
	v0_1Parser   v0_1.BindingRequestParser
}

var _ brp.Parser = BindRequestParser{}

func NewFromParsers(
	legacyParser legacy.BindingRequestParser, v0_1Parser v0_1.BindingRequestParser,
) BindRequestParser {
	return BindRequestParser{
		legacyParser: legacyParser,
		v0_1Parser:   v0_1Parser,
	}
}

func New(
	legacySchemaPath, v0_1SchemaPath string,
	defaultCustomMetricsCredentialType models.CustomMetricsBindingAuthScheme,
) (BindRequestParser, error) {
	legacyParser, err := legacy.New(legacySchemaPath, defaultCustomMetricsCredentialType)
	if err != nil {
		return BindRequestParser{}, fmt.Errorf("failed to create legacy binding-request-parser: %w", err)
	}

	v0_1Parser, err := v0_1.NewFromFile(v0_1SchemaPath, defaultCustomMetricsCredentialType)
	if err != nil {
		return BindRequestParser{}, fmt.Errorf("failed to create v0.1 binding-request-parser: %w", err)
	}

	return NewFromParsers(legacyParser, v0_1Parser), nil
}

func (p BindRequestParser) Parse(
	bindingReqParams string, ccAppGuid models.GUID,
) (models.AppScalingConfig, error) {
	schemaVersion, err := extractSchemaVersion(bindingReqParams)
	if err != nil {
		return models.AppScalingConfig{}, err
	}

	switch schemaVersion {
	case "0.1":
		return p.v0_1Parser.Parse(bindingReqParams, ccAppGuid)
	case "": // The legacy-schema is the only one without a version-identifier;
		return p.legacyParser.Parse(bindingReqParams, ccAppGuid)
	default:
		return models.AppScalingConfig{}, fmt.Errorf(
			"unsupported schema-version '%s' referenced in binding-request", schemaVersion)
	}
}

func extractSchemaVersion(bindingReqParams string) (string, error) {
	type schemaHolder struct {
		Schema string `json:"schema-version,omitempty"`
	}

	var holder schemaHolder
	err := json.Unmarshal([]byte(bindingReqParams), &holder)
	if err != nil {
		return "", fmt.Errorf("failed to extract schema-version from binding-request: %w", err)
	}

	return holder.Schema, nil
}
