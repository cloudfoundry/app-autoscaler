package models

// BasicAuthHandlingImplConfig defines how "Basic Authentication" is implemented.
// This is a sealed interface with two implementations:
//   - BasicAuthHandlingNative: native implementation by Application Autoscaler
//   - BasicAuthHandlingStoredProc: custom implementation via a stored procedure
type BasicAuthHandlingImplConfig interface {
	isBasicAuthHandlingImplConfig()
}

// BasicAuthHandlingNative states that "Basic Authentication" is implemented natively
// by Application Autoscaler.
type BasicAuthHandlingNative struct{}

func (b BasicAuthHandlingNative) isBasicAuthHandlingImplConfig() {}

var _ BasicAuthHandlingImplConfig = BasicAuthHandlingNative{}

// BasicAuthHandlingStoredProc configures a custom handling of "Basic Authentication"
// via a stored procedure.
type BasicAuthHandlingStoredProc struct {
	Config StoredProcedureConfig
}

func (b BasicAuthHandlingStoredProc) isBasicAuthHandlingImplConfig() {}

var _ BasicAuthHandlingImplConfig = BasicAuthHandlingStoredProc{}
