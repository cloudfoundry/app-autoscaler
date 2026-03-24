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
// Top-level parser
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

type schemaVersion int

const (
	schemaVersionLegacy schemaVersion = iota
	schemaVersionV0_1
)

var supportedSchemaVersions = []schemaVersion{
	schemaVersionLegacy,
	schemaVersionV0_1,
}

func (sv schemaVersion) String() string {
	switch sv {
	case schemaVersionLegacy:
		return "legacy"
	case schemaVersionV0_1:
		return "0.1"
	default:
		return fmt.Sprintf("unknown(%d)", sv)
	}
}

var _ fmt.Stringer = schemaVersion(1)

func (p BindRequestParser) Parse(
	bindingReqParams string, ccAppGuid models.GUID,
) (models.AppScalingConfig, error) {
	schemaVersion, err := extractSchemaVersion(bindingReqParams)
	if err != nil {
		// Default to legacy schema if schema-version cannot be extracted. The legacy-schema is the
		// only one without a version-identifier.
		schemaVersion = schemaVersionLegacy
	}

	switch schemaVersion {
	case schemaVersionV0_1:
		return p.v0_1Parser.Parse(bindingReqParams, ccAppGuid)
	case schemaVersionLegacy:
		return p.legacyParser.Parse(bindingReqParams, ccAppGuid)
	default:
		return models.AppScalingConfig{}, fmt.Errorf(
			`unsupported schema-version '%d' referenced in binding-request\npossible values: %v`,
			schemaVersion, supportedSchemaVersions)
	}
}

func extractSchemaVersion(bindingReqParams string) (schemaVersion, error) {
	// We return that in case of an error. This constant is just for better readability and has no
	// special meaning outside of this function's scope.
	const errSchemaVersion = -1

	// Empty or whitespace-only input is valid and indicates legacy schema (no policy)
	if len(bindingReqParams) == 0 {
		return schemaVersionLegacy, nil
	}

	type schemaHolder struct {
		Schema string `json:"schema-version,omitempty"`
	}

	var holder schemaHolder
	err := json.Unmarshal([]byte(bindingReqParams), &holder)
	if err != nil {
		return errSchemaVersion, fmt.Errorf("failed to extract schema-version from binding-request: %w", err)
	}

	var schemaVersion schemaVersion
	switch holder.Schema {
	case "0.1":
		schemaVersion = schemaVersionV0_1
	case "":
		schemaVersion = schemaVersionLegacy
	default:
		return errSchemaVersion, fmt.Errorf("unsupported schema-version '%s' referenced in binding-request", holder.Schema)
	}

	return schemaVersion, nil
}
