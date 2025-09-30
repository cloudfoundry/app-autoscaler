package configutil

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/url"
	"os"
	"slices"
	"strconv"
	"strings"

	"code.cloudfoundry.org/app-autoscaler/src/autoscaler/db"
	"code.cloudfoundry.org/app-autoscaler/src/autoscaler/helpers"
	"code.cloudfoundry.org/app-autoscaler/src/autoscaler/models"
	"github.com/cloud-gov/go-cfenv"
	"go.yaml.in/yaml/v4"
)

var (
	ErrServiceConfigNotFound = errors.New("vcap_services config not found")
	ErrDbServiceNotFound     = errors.New("failed to get service by name")
	ErrMissingCredential     = errors.New("failed to get required credential from service")
	AvailableDatabases       = []string{db.PolicyDb, db.BindingDb, db.AppMetricsDb, db.LockDb, db.ScalingEngineDb, db.SchedulerDb}
)

type VCAPConfigurationReader interface {
	MaterializeDBFromService(dbName string) (string, error)
	MaterializeTLSConfigFromService(serviceTag string) (models.TLSCerts, error)
	GetServiceCredentialContent(serviceTag string, credentialKey string) ([]byte, error)

	GetInstanceTLSCerts() models.TLSCerts

	GetPort() int
	GetSpaceGuid() string
	GetOrgGuid() string
	GetInstanceIndex() int
	IsRunningOnCF() bool

	ConfigureDatabases(confDb *map[string]db.DatabaseConfig, storedProcedureConfig *models.StoredProcedureConfig, credHelperImpl string) error
}

type VCAPConfiguration struct {
	appEnv *cfenv.App
}

func NewVCAPConfigurationReader() (result *VCAPConfiguration, err error) {
	var appEnv *cfenv.App
	result = &VCAPConfiguration{}

	if cfenv.IsRunningOnCF() {
		appEnv, err = cfenv.Current()
		result.appEnv = appEnv
	} else {
		fmt.Println("VCAPConfigurationReader: Not running on CF")
	}

	return result, err
}

func (vc *VCAPConfiguration) GetPort() int {
	return vc.appEnv.Port
}

func (vc *VCAPConfiguration) GetInstanceIndex() int {
	instanceIndex, err := strconv.Atoi(os.Getenv("CF_INSTANCE_INDEX"))
	if err == nil {
		return instanceIndex
	}

	return 0
}

func (vc *VCAPConfiguration) IsRunningOnCF() bool {
	return cfenv.IsRunningOnCF()
}

func (vc *VCAPConfiguration) GetOrgGuid() string {
	var vcap map[string]any
	if err := json.Unmarshal([]byte(os.Getenv("VCAP_APPLICATION")), &vcap); err != nil {
		return ""
	}
	if orgID, ok := vcap["organization_id"].(string); ok {
		return orgID
	}
	return ""
}

func (vc *VCAPConfiguration) GetSpaceGuid() string {
	return vc.appEnv.SpaceID
}

func (vc *VCAPConfiguration) GetInstanceTLSCerts() models.TLSCerts {
	result := models.TLSCerts{}
	result.CACertFile = os.Getenv("CF_INSTANCE_CA_CERT")
	result.CertFile = os.Getenv("CF_INSTANCE_CERT")
	result.KeyFile = os.Getenv("CF_INSTANCE_KEY")

	return result
}

func (vc *VCAPConfiguration) GetServiceCredentialContent(serviceTag, credentialKey string) ([]byte, error) {
	service, err := vc.getServiceByTag(serviceTag)
	if err != nil {
		return []byte(""), err
	}

	content, ok := service.Credentials[credentialKey]
	if !ok {
		return []byte(""), fmt.Errorf("%w: %s", ErrMissingCredential, credentialKey)
	}

	rawJSON, err := json.Marshal(content)
	if err != nil {
		return []byte(""), err
	}

	return rawJSON, nil
}

func (vc *VCAPConfiguration) MaterializeTLSConfigFromService(serviceTag string) (models.TLSCerts, error) {
	service, err := vc.getServiceByTag(serviceTag)
	if err != nil {
		return models.TLSCerts{}, err
	}

	tlsCerts, err := vc.buildTLSCerts(service, serviceTag)
	if err != nil {
		return models.TLSCerts{}, err
	}

	return tlsCerts, nil
}

func (vc *VCAPConfiguration) MaterializeDBFromService(dbName string) (string, error) {
	service, err := vc.getServiceByTag(dbName)
	if err != nil {
		return "", nil
	}

	dbURL, err := vc.buildDatabaseURL(service, dbName)
	if err != nil {
		return "", err
	}

	return dbURL.String(), nil
}

func (vc *VCAPConfiguration) getServiceByTag(serviceTag string) (*cfenv.Service, error) {
	services, err := vc.appEnv.Services.WithTag(serviceTag)
	if err != nil || len(services) == 0 {
		return nil, fmt.Errorf("%w: %s", ErrDbServiceNotFound, serviceTag)
	}
	return &services[0], nil
}

func (vc *VCAPConfiguration) buildTLSCerts(service *cfenv.Service, serviceTag string) (models.TLSCerts, error) {
	certs := models.TLSCerts{}

	if err := vc.createCertFile(service, "client_cert", "sslcert", serviceTag, &certs.CertFile); err != nil {
		return models.TLSCerts{}, err
	}

	if err := vc.createCertFile(service, "client_key", "sslkey", serviceTag, &certs.KeyFile); err != nil {
		return models.TLSCerts{}, err
	}

	if err := vc.createCertFile(service, "server_ca", "sslrootcert", serviceTag, &certs.CACertFile); err != nil {
		return models.TLSCerts{}, err
	}

	return certs, nil
}

func (vc *VCAPConfiguration) createCertFile(service *cfenv.Service, credentialKey, fileSuffix, serviceTag string, certFile *string) error {
	content, ok := service.CredentialString(credentialKey)
	if !ok {
		return fmt.Errorf("%w: %s", ErrMissingCredential, credentialKey)
	}
	fileName := fmt.Sprintf("%s.%s", credentialKey, fileSuffix)
	createdFile, err := MaterializeContentInFile(serviceTag, fileName, content)
	if err != nil {
		return err
	}
	*certFile = createdFile
	return nil
}

func (vc *VCAPConfiguration) buildDatabaseURL(service *cfenv.Service, dbName string) (*url.URL, error) {
	dbURI, ok := service.CredentialString("uri")
	if !ok {
		return nil, fmt.Errorf("%w: uri", ErrMissingCredential)
	}

	dbURL, err := url.Parse(dbURI)
	if err != nil {
		return nil, err
	}

	parameters, err := url.ParseQuery(dbURL.RawQuery)
	if err != nil {
		return nil, err
	}

	if err := vc.addConnectionParams(service, dbName, parameters); err != nil {
		return nil, err
	}

	dbURL.RawQuery = parameters.Encode()
	return dbURL, nil
}

func (vc *VCAPConfiguration) addPostgresConnectionParams(service *cfenv.Service, dbName string, parameters url.Values) error {
	keys := []struct {
		binding, connection string
	}{
		{"client_cert", "sslcert"},
		{"client_key", "sslkey"},
		{"server_ca", "sslrootcert"},
	}

	for _, key := range keys {
		if err := vc.addConnectionParam(service, dbName, key.binding, key.connection, parameters); err != nil {
			return err
		}
	}
	return nil
}

func (vc *VCAPConfiguration) addConnectionParams(service *cfenv.Service, dbName string, parameters url.Values) error {
	// if service.Tags contains "postgres" then add the connection parameters
	if slices.Contains(service.Tags, "mysql") {
		return nil
		// TODO: add support for MySQL TLS enabled connections
		// return vc.addMySQLConnectionParams(service, dbName, parameters)
	} else {
		return vc.addPostgresConnectionParams(service, dbName, parameters)
	}
}

func (vc *VCAPConfiguration) addConnectionParam(service *cfenv.Service, dbName, bindingKey, connectionParam string, parameters url.Values) error {
	content, ok := service.CredentialString(bindingKey)
	if ok {
		fileName := fmt.Sprintf("%s.%s", bindingKey, connectionParam)
		createdFile, err := MaterializeContentInFile(dbName, fileName, content)
		if err != nil {
			return err
		}
		parameters.Set(connectionParam, createdFile)
	}
	return nil
}

func (vc *VCAPConfiguration) configureStoredProcedureDb(dbName string, confDb *map[string]db.DatabaseConfig, storedProcedureConfig *models.StoredProcedureConfig) error {
	if err := vc.configureDb(dbName, confDb); err != nil {
		return err
	}

	currentStoredProcedureDb := (*confDb)[dbName]
	parsedUrl, err := url.Parse(currentStoredProcedureDb.URL)
	if err != nil {
		return err
	}

	if storedProcedureConfig != nil {
		if storedProcedureConfig.Username != "" {
			currentStoredProcedureDb.URL = strings.Replace(currentStoredProcedureDb.URL, parsedUrl.User.Username(), storedProcedureConfig.Username, 1)
		}
		if storedProcedureConfig.Password != "" {
			bindingPassword, _ := parsedUrl.User.Password()
			currentStoredProcedureDb.URL = strings.Replace(currentStoredProcedureDb.URL, bindingPassword, storedProcedureConfig.Password, 1)
		}
	}
	(*confDb)[dbName] = currentStoredProcedureDb

	return nil
}

func (vc *VCAPConfiguration) configureDb(dbName string, confDb *map[string]db.DatabaseConfig) error {
	currentDb, ok := (*confDb)[dbName]
	if !ok {
		(*confDb)[dbName] = db.DatabaseConfig{}
	}

	dbURL, err := vc.MaterializeDBFromService(dbName)
	currentDb.URL = dbURL
	if err != nil {
		return err
	}
	(*confDb)[dbName] = currentDb

	return nil
}

func LoadConfig[T any](conf *T, vcapReader VCAPConfigurationReader, credentialName string) error {
	data, err := vcapReader.GetServiceCredentialContent(credentialName, credentialName)
	if err != nil {
		return fmt.Errorf("%w: %v", ErrServiceConfigNotFound, err)
	}

	var raw string
	if json.Unmarshal(data, &raw) == nil {
		return yaml.Unmarshal([]byte(raw), conf)
	}
	return yaml.Unmarshal(data, conf)
}

func (vc *VCAPConfiguration) ConfigureDatabases(confDb *map[string]db.DatabaseConfig, storedProcedureConfig *models.StoredProcedureConfig, credHelperImpl string) error {
	for _, dbName := range AvailableDatabases {
		if err := vc.configureDb(dbName, confDb); err != nil {
			return err
		}
	}

	if storedProcedureConfig != nil && credHelperImpl == "stored_procedure" {
		if err := vc.configureStoredProcedureDb(db.StoredProcedureDb, confDb, storedProcedureConfig); err != nil {
			return err
		}
	}

	return nil
}

func ToJSON(v any) (string, error) {
	b, err := json.Marshal(v)
	if err != nil {
		return "", fmt.Errorf("failed to marshal to json: %w", err)
	}
	return string(b), nil
}

// CommonVCAPConfiguration applies common VCAP configuration settings to any config struct.
// It handles the repeated patterns found in component LoadVcapConfig functions.
type CommonVCAPConfig interface {
	SetLoggingPlainText()
	SetPortsForCF(cfPort int)
	SetXFCCValidation(spaceGuid, orgGuid string)
	GetDatabaseConfig() *map[string]db.DatabaseConfig
}

// BaseConfig contains common configuration fields and methods shared across components
type BaseConfig struct {
	Logging  helpers.LoggingConfig        `yaml:"logging" json:"logging"`
	Server   helpers.ServerConfig         `yaml:"server" json:"server"`
	CFServer helpers.ServerConfig         `yaml:"cf_server" json:"cf_server"`
	Health   helpers.HealthConfig         `yaml:"health" json:"health"`
	Db       map[string]db.DatabaseConfig `yaml:"db" json:"db"`
}

// SetLoggingLevel implements configutil.Configurable
func (c *BaseConfig) SetLoggingLevel() {
	c.Logging.Level = strings.ToLower(c.Logging.Level)
}

// GetLogging returns the logging configuration
func (c *BaseConfig) GetLogging() *helpers.LoggingConfig {
	return &c.Logging
}

// SetLoggingPlainText implements configutil.CommonVCAPConfig
func (c *BaseConfig) SetLoggingPlainText() {
	c.Logging.PlainTextSink = true
}

// SetPortsForCF implements configutil.CommonVCAPConfig
func (c *BaseConfig) SetPortsForCF(cfPort int) {
	c.CFServer.Port = cfPort
	c.Server.Port = 0
}

// SetXFCCValidation implements configutil.CommonVCAPConfig
func (c *BaseConfig) SetXFCCValidation(spaceGuid, orgGuid string) {
	c.CFServer.XFCC.ValidSpaceGuid = spaceGuid
	c.CFServer.XFCC.ValidOrgGuid = orgGuid
}

// GetDatabaseConfig implements configutil.CommonVCAPConfig
func (c *BaseConfig) GetDatabaseConfig() *map[string]db.DatabaseConfig {
	return &c.Db
}

// ApplyCommonVCAPConfiguration handles the common VCAP configuration steps
func ApplyCommonVCAPConfiguration[T any, PT interface {
	*T
	CommonVCAPConfig
}](conf PT, vcapReader VCAPConfigurationReader, serviceName string) error {
	// enable plain text logging. See src/autoscaler/helpers/logger.go
	conf.SetLoggingPlainText()

	// Avoid port conflict: assign actual port to CF server, set BOSH server port to 0 (unused)
	conf.SetPortsForCF(vcapReader.GetPort())

	if err := LoadConfig(conf, vcapReader, serviceName); err != nil {
		return err
	}

	if err := vcapReader.ConfigureDatabases(conf.GetDatabaseConfig(), nil, ""); err != nil {
		return err
	}

	conf.SetXFCCValidation(vcapReader.GetSpaceGuid(), vcapReader.GetOrgGuid())

	return nil
}
