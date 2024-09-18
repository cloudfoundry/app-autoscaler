package models

type StoredProcedureConfig struct {
	SchemaName                             string `yaml:"schema_name"`
	CreateBindingCredentialProcedureName   string `yaml:"create_binding_credential_procedure_name"`
	DropBindingCredentialProcedureName     string `yaml:"drop_binding_credential_procedure_name"`
	DropAllBindingCredentialProcedureName  string `yaml:"drop_all_binding_credential_procedure_name"`
	ValidateBindingCredentialProcedureName string `yaml:"validate_binding_credential_procedure_name"`
	Username                               string `yaml:"username"`
	Password                               string `yaml:"password"`
}
