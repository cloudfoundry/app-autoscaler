package configutil

import (
	"errors"
	"fmt"
	"net/url"
	"os"

	"code.cloudfoundry.org/app-autoscaler/src/autoscaler/models"
	"github.com/cloudfoundry-community/go-cfenv"
)

var (
	ErrReadEnvironment   = errors.New("failed to read environment variables")
	ErrDbServiceNotFound = errors.New("failed to get service by name")
	ErrMissingCredential = errors.New("failed to get required credential from service")
)

type VCAPConfigurationReader interface {
	MaterializeDBFromService(dbName string) (string, error)
	MaterializeTLSConfigFromService(serviceName string) (models.TLSCerts, error)
	GetServiceCredentialContent(serviceName string, credentialKey string) ([]byte, error)
	GetPort() int
	IsRunningOnCF() bool
}

type VCAPConfiguration struct {
	appEnv *cfenv.App
}

func NewVCAPConfigurationReader() (*VCAPConfiguration, error) {
	appEnv, err := cfenv.Current()
	if err != nil {
		return nil, fmt.Errorf("%w: %w", ErrReadEnvironment, err)
	}
	return &VCAPConfiguration{appEnv: appEnv}, nil
}

func (vc *VCAPConfiguration) GetPort() int {
	return vc.appEnv.Port
}
func (vc *VCAPConfiguration) IsRunningOnCF() bool {
	return cfenv.IsRunningOnCF()
}

func (vc *VCAPConfiguration) GetServiceCredentialContent(serviceName, credentialKey string) ([]byte, error) {
	service, err := vc.getServiceByName(serviceName)
	if err != nil {
		return []byte(""), err
	}

	content, ok := service.CredentialString(credentialKey)
	if !ok {
		return []byte(""), fmt.Errorf("%w: %s", ErrMissingCredential, credentialKey)
	}

	return []byte(content), nil
}

func (vc *VCAPConfiguration) MaterializeTLSConfigFromService(serviceName string) (models.TLSCerts, error) {
	service, err := vc.getServiceByName(serviceName)
	if err != nil {
		return models.TLSCerts{}, err
	}

	tlsCerts, err := vc.buildTLSCerts(service, serviceName)
	if err != nil {
		return models.TLSCerts{}, err
	}

	return tlsCerts, nil
}

func (vc *VCAPConfiguration) MaterializeDBFromService(dbName string) (string, error) {
	service, err := vc.getServiceByName(dbName)
	if err != nil {
		return "", err
	}

	dbURL, err := vc.buildDatabaseURL(service, dbName)
	if err != nil {
		return "", err
	}

	return dbURL.String(), nil
}

func (vc *VCAPConfiguration) getServiceByName(serviceName string) (*cfenv.Service, error) {
	services, err := vc.appEnv.Services.WithTag(serviceName)
	if err != nil || len(services) == 0 {
		return nil, fmt.Errorf("%w: %s", ErrDbServiceNotFound, serviceName)
	}
	return &services[0], nil
}

func (vc *VCAPConfiguration) buildTLSCerts(service *cfenv.Service, serviceName string) (models.TLSCerts, error) {
	certs := models.TLSCerts{}

	if err := vc.createCertFile(service, "client_cert", "sslcert", serviceName, &certs.CertFile); err != nil {
		return models.TLSCerts{}, err
	}

	if err := vc.createCertFile(service, "client_key", "sslkey", serviceName, &certs.KeyFile); err != nil {
		return models.TLSCerts{}, err
	}

	if err := vc.createCertFile(service, "server_ca", "sslrootcert", serviceName, &certs.CACertFile); err != nil {
		return models.TLSCerts{}, err
	}

	return certs, nil
}

func (vc *VCAPConfiguration) createCertFile(service *cfenv.Service, credentialKey, fileSuffix, serviceName string, certFile *string) error {
	content, ok := service.CredentialString(credentialKey)
	if !ok {
		return fmt.Errorf("%w: %s", ErrMissingCredential, credentialKey)
	}
	fileName := fmt.Sprintf("%s.%s", credentialKey, fileSuffix)
	createdFile, err := materializeServiceProperty(serviceName, fileName, content)
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

func (vc *VCAPConfiguration) addConnectionParams(service *cfenv.Service, dbName string, parameters url.Values) error {
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

func (vc *VCAPConfiguration) addConnectionParam(service *cfenv.Service, dbName, bindingKey, connectionParam string, parameters url.Values) error {
	content, ok := service.CredentialString(bindingKey)
	if ok {
		fileName := fmt.Sprintf("%s.%s", bindingKey, connectionParam)
		createdFile, err := materializeServiceProperty(dbName, fileName, content)
		if err != nil {
			return err
		}
		parameters.Set(connectionParam, createdFile)
	}
	return nil
}

func materializeServiceProperty(serviceName, fileName, content string) (string, error) {
	dirPath := fmt.Sprintf("/tmp/%s", serviceName)
	if err := os.MkdirAll(dirPath, 0700); err != nil {
		return "", err
	}

	filePath := fmt.Sprintf("%s/%s", dirPath, fileName)
	if err := os.WriteFile(filePath, []byte(content), 0600); err != nil {
		return "", err
	}

	return filePath, nil
}
