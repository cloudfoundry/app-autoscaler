package configutil

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/url"
	"os"

	"code.cloudfoundry.org/app-autoscaler/src/autoscaler/models"
	"github.com/cloud-gov/go-cfenv"
)

var (
	ErrDbServiceNotFound = errors.New("failed to get service by name")
	ErrMissingCredential = errors.New("failed to get required credential from service")
)

type VCAPConfigurationReader interface {
	MaterializeDBFromService(dbName string) (string, error)
	MaterializeTLSConfigFromService(serviceTag string) (models.TLSCerts, error)
	GetServiceCredentialContent(serviceTag string, credentialKey string) ([]byte, error)
	GetPort() int
	IsRunningOnCF() bool
}

type VCAPConfiguration struct {
	appEnv *cfenv.App
}

func NewVCAPConfigurationReader() (*VCAPConfiguration, error) {
	appEnv, err := cfenv.Current()
	if err != nil {
		fmt.Println("failed to read VCAP_APPLICATION environment variable")
	}
	return &VCAPConfiguration{appEnv: appEnv}, nil
}

func (vc *VCAPConfiguration) GetPort() int {
	return vc.appEnv.Port
}
func (vc *VCAPConfiguration) IsRunningOnCF() bool {
	return cfenv.IsRunningOnCF()
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
		return "", err
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
	createdFile, err := materializeServiceProperty(serviceTag, fileName, content)
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

func contains(elems []string, v string) bool {
	for _, s := range elems {
		if s == v {
			return true
		}
	}
	return false
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
	if contains(service.Tags, "mysql") {
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
		createdFile, err := materializeServiceProperty(dbName, fileName, content)
		if err != nil {
			return err
		}
		parameters.Set(connectionParam, createdFile)
	}
	return nil
}

func materializeServiceProperty(serviceTag, fileName, content string) (string, error) {
	dirPath := fmt.Sprintf("/tmp/%s", serviceTag)
	if err := os.MkdirAll(dirPath, 0700); err != nil {
		return "", err
	}

	filePath := fmt.Sprintf("%s/%s", dirPath, fileName)
	if err := os.WriteFile(filePath, []byte(content), 0600); err != nil {
		return "", err
	}

	return filePath, nil
}
