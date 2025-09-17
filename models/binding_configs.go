package models

import (
	"encoding/json"
	"fmt"
)

// ================================================================================
// BindingConfig and its associated sub-programs
// ================================================================================

// BindingConfig represents the configuration for a service binding.
//
// ⛔ Do not create `BindingConfig` values directly via `BindingConfig{}` because it can lead to
// undefined behaviour due to bypassing all validations.  Use the constructor-functions instead!
type BindingConfig struct {
	appGUID GUID // This always must be set.

	// Whether to use the default authentication scheme for custom metrics binding.
	useDefaultAuthScheme bool

	// Meaningful only if useDefaultAuthScheme is false.
	customMetricsBindingAuth CustomMetricsBindingAuthScheme
}

// NewBindingConfig creates a new BindingConfig instance with the specified application GUID and authentication scheme.
//
// Parameters:
//   - appGUID: The GUID of the application to associate with this binding configuration.
//     This parameter is required and must not be empty.
//   - customMetricsBindingAuth: The authentication scheme to use for custom metrics binding.
//     If nil, the default authentication scheme will be used.
//
// Returns:
//   - *BindingConfig: A newly initialized BindingConfig instance with the specified settings.
func NewBindingConfig(
	appGUID GUID,
	customMetricsBindingAuth *CustomMetricsBindingAuthScheme,
) *BindingConfig {
	// The validity of appGUID is currently unchecked. But it should not be done here but rather on
	// creation of a(ny) GUID.
	if customMetricsBindingAuth == nil {
		return &BindingConfig{
			appGUID:              appGUID,
			useDefaultAuthScheme: true,
		}
	}

	return &BindingConfig{
		appGUID:                  appGUID,
		useDefaultAuthScheme:     false,
		customMetricsBindingAuth: *customMetricsBindingAuth,
	}
}

// // BindingConfigFromServiceBinding creates a new BindingConfig from an existing ServiceBinding.
// //
// // Parameters:
// //   - serviceBinding: The ServiceBinding instance from which to extract configuration data.
// //     Must not be nil. The AppID field is used as the application GUID, and the
// //     CustomMetricsStrategy field determines the metrics submission strategy.
// //
// // Returns:
// //   - *BindingConfig: A newly created BindingConfig instance with the extracted configuration.
// //     Returns nil if an error occurs during processing.
// //   - error: An InvalidArgumentError if serviceBinding is nil, or a formatting error if the
// //     CustomMetricsStrategy contains an unsupported value.
// //
// // Edge cases:
// //   - If serviceBinding is nil, returns an InvalidArgumentError with detailed parameter information.
// //   - If CustomMetricsStrategy is empty string, defaults to CustomMetricsSameApp strategy.
// //   - If CustomMetricsStrategy contains an unsupported value, returns a descriptive error.
// func BindingConfigFromServiceBinding(serviceBinding *ServiceBinding) (*BindingConfig, error) {
//	var bindingConfig *BindingConfig

//	if serviceBinding == nil {
//		err := InvalidArgumentError{
//			Param: "serviceBinding",
//			Value: serviceBinding,
//			Msg:   "serviceBinding must not be nil, see function-contract;",
//		}
//		return nil, &err
//	}

//	bindingConfig = NewBindingConfig(GUID(serviceBinding.AppID), nil)

//	return bindingConfig, nil
// }

// GetAppGUID returns the GUID of the application associated with this binding.
func (bc *BindingConfig) GetAppGUID() GUID {
	return bc.appGUID
}

// 🚧 To-do: Rename to `GetCustomMetricsBindingAuth` and rewrite documentation!
// GetCustomMetricStrategy returns the custom metrics configuration for this binding.
func (bc *BindingConfig) GetCustomMetricStrategy() *CustomMetricsBindingAuthScheme {
	if bc.useDefaultAuthScheme {
		return nil
	}
	return &bc.customMetricsBindingAuth
}

// ---------- Deserialization and serialization methods for BindingConfig ----------

// Json-Serialized example of a BindingConfig:
// {
//   "app_guid": "550e8400-e29b-41d4-a716-446655440000",
//   "credential-type": "binding-secret"
// }

type bindingConfigJsonRawRepr struct {
	AppGUID GUID `json:"app_guid"`

	// Omit if default strategy is used
	CmAuthScheme *CustomMetricsBindingAuthScheme `json:"credential-type,omitempty"`
}

func (bc BindingConfig) ToRawJSON() (json.RawMessage, error) {
	var authScheme *CustomMetricsBindingAuthScheme
	if bc.useDefaultAuthScheme {
		authScheme = nil // The default-scheme does not need to be serialized.
	} else {
		authScheme = &bc.customMetricsBindingAuth
	}

	bindingConfigRaw := bindingConfigJsonRawRepr{
		AppGUID:      bc.appGUID,
		CmAuthScheme: authScheme,
	}

	data, err := json.Marshal(bindingConfigRaw)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal BindingConfig: %w", err)
	}
	return json.RawMessage(data), nil
}

func BindingConfigFromRawJSON(data json.RawMessage) (*BindingConfig, error) {
	if len(data) <= 0 {
		msg := "data must not be empty, see function-contract;"
		return nil, &InvalidArgumentError{
			Param: "data",
			Value: data,
			Msg:   msg,
		}
	}

	var bindingConfigRaw bindingConfigJsonRawRepr
	if err := json.Unmarshal(data, &bindingConfigRaw); err != nil {
		return nil, fmt.Errorf("failed to unmarshal BindingConfig: %w", err)
	}

	bindingConfig := NewBindingConfig(bindingConfigRaw.AppGUID, bindingConfigRaw.CmAuthScheme)
	return bindingConfig, nil
}

func (bc BindingConfig) String() string {
	return fmt.Sprintf("BindingConfig{appGUID: %s, useDefaultAuthScheme: %t}",
		bc.appGUID, bc.useDefaultAuthScheme)
}

var _ fmt.Stringer = BindingConfig{}

// ================================================================================
// Nested configuration objects for BindingConfig
// ================================================================================

// CustomMetricsBindingAuthScheme represents an authentication scheme configuration for custom
// metrics binding operations.
//
// There are only twosupported authentication schemes:
//   - X509Certificate: Uses x.509 certificates for authentication.
//   - BindingSecret: Uses a password-based authentication scheme (similar to Basic Auth).
//
// The struct enforces immutability by making the credentialType field private, which prevents
// direct field access and modification after instantiation.  This design pattern ensures that only
// valid, predefined credential types can be set through controlled constructor functions or
// methods.
//
// ⛔ Do not create `CustomMetricsBindingAuthScheme` values directly via
// `CustomMetricsBindingAuthScheme{}` because it can lead to undefined behaviour due to bypassing
// all validations. Use the constructor-functions instead!
type CustomMetricsBindingAuthScheme struct {
	credentialType string // Private, to make it immutable and enforce predefined values only.
}

// Predefined CustomMetricsBindingAuthScheme values
var (
	X509Certificate = CustomMetricsBindingAuthScheme{credentialType: "x509"}

	// BasicAuth-Variant
	BindingSecret = CustomMetricsBindingAuthScheme{credentialType: "binding-secret"}
)

// String returns a string representation of the authentication scheme
func (c CustomMetricsBindingAuthScheme) String() string {
	return c.credentialType
}

var _ fmt.Stringer = CustomMetricsBindingAuthScheme{} // Ensure CustomMetricsBindingAuthScheme implements fmt.Stringer

// MarshalJSON implements the json.Marshaler interface for CustomMetricsBindingAuthScheme
func (c CustomMetricsBindingAuthScheme) MarshalJSON() ([]byte, error) {
	return json.Marshal(c.credentialType)
}

var _ json.Marshaler = CustomMetricsBindingAuthScheme{} // Ensure CustomMetricsBindingAuthScheme implements json.Marshaler

// UnmarshalJSON implements the json.Unmarshaler interface for CustomMetricsBindingAuthScheme
func (c *CustomMetricsBindingAuthScheme) UnmarshalJSON(data []byte) error {
	var credentialType string
	if err := json.Unmarshal(data, &credentialType); err != nil {
		return fmt.Errorf("failed to unmarshal CustomMetricsBindingAuthScheme: %w", err)
	}

	scheme, err := ParseCustomMetricsBindingAuthScheme(credentialType)
	if err != nil {
		return fmt.Errorf("invalid CustomMetricsBindingAuthScheme value: %w", err)
	}
	*c = *scheme

	return nil
}

var _ json.Unmarshaler = &CustomMetricsBindingAuthScheme{} // Ensure *CustomMetricsBindingAuthScheme implements json.Unmarshaler

func ParseCustomMetricsBindingAuthScheme(
	credentialType string,
) (c *CustomMetricsBindingAuthScheme, err error) {
	switch credentialType {
	case "x509":
		c = &X509Certificate
	case "binding-secret":
		c = &BindingSecret
	default:
		return nil, fmt.Errorf("unknown credential type: %s", credentialType)
	}
	return c, nil
}
