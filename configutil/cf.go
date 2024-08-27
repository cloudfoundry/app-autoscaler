package configutil

import (
	"errors"
	"fmt"
	"net/url"
	"os"

	"code.cloudfoundry.org/app-autoscaler/src/autoscaler/models"
	"github.com/cloudfoundry-community/go-cfenv"
)

var ErrReadEnvironment = errors.New("failed to read environment variables")
var ErrDbServiceNotFound = errors.New("failed to get service by name")

type VCAPConfigurationReader interface {
	MaterializeDBFromService(dbName string) (string, error)
	MaterializeTLSConfigFromService(serviceName string) (models.TLSCerts, error)
}

type VCAPConfiguration struct {
	VCAPConfigurationReader
	appEnv *cfenv.App
}

func NewVCAPConfigurationReader() (*VCAPConfiguration, error) {
	vcapConfiguration := &VCAPConfiguration{}
	appEnv, err := cfenv.Current()
	if err != nil {
		return nil, fmt.Errorf("%w: %w", ErrReadEnvironment, err)
	}

	vcapConfiguration.appEnv = appEnv
	return vcapConfiguration, nil
}

func (vc *VCAPConfiguration) MaterializeTLSConfigFromService(serviceName string) (models.TLSCerts, error) {
	tlsCerts := models.TLSCerts{}
	services, err := vc.appEnv.Services.WithTag(serviceName)
	if err != nil {
		return tlsCerts, fmt.Errorf("%w: %w", ErrDbServiceNotFound, err)
	}

	service := services[0]

	if clientCertContent, ok := service.CredentialString("client_cert"); ok {
		fileName := fmt.Sprintf("%s.%s", "client_cert", "sslcert")
		createdClientCert, err := materializeServiceProperty(serviceName, fileName, clientCertContent)
		if err != nil {
			return models.TLSCerts{}, err
		}
		tlsCerts.CertFile = createdClientCert
	} else {
		return models.TLSCerts{}, fmt.Errorf(fmt.Sprintf("failed to get %s from db service", "client_cert"))
	}

	if clientKeyContent, ok := service.CredentialString("client_key"); ok {
		fileName := fmt.Sprintf("%s.%s", "client_key", "sslkey")
		createdClientKey, err := materializeServiceProperty(serviceName, fileName, clientKeyContent)
		if err != nil {
			return models.TLSCerts{}, err
		}
		tlsCerts.KeyFile = createdClientKey
	} else {
		return models.TLSCerts{}, fmt.Errorf(fmt.Sprintf("failed to get %s from db service", "client_key"))
	}

	if serverCAContent, ok := service.CredentialString("server_ca"); ok {
		fileName := fmt.Sprintf("%s.%s", "server_ca", "sslrootcert")
		createServerCA, err := materializeServiceProperty(serviceName, fileName, serverCAContent)
		if err != nil {
			return models.TLSCerts{}, err
		}
		tlsCerts.CACertFile = createServerCA
	} else {
		return models.TLSCerts{}, fmt.Errorf(fmt.Sprintf("failed to get %s from db service", "server_ca"))
	}

	return tlsCerts, nil
}

func (vc *VCAPConfiguration) MaterializeDBFromService(dbName string) (string, error) {
	var dbURL *url.URL
	var err error

	service, err := vc.appEnv.Services.WithTag(dbName)
	if err != nil {
		return "", fmt.Errorf("%w: %w", ErrDbServiceNotFound, err)
	}

	dbService := service[0]

	dbURI, ok := dbService.CredentialString("uri")
	if !ok {
		return "", fmt.Errorf("failed to get uri from db service")
	}

	dbURL, err = url.Parse(dbURI)
	if err != nil {
		return "", err
	}

	parameters, err := url.ParseQuery(dbURL.RawQuery)
	if err != nil {
		return "", err
	}

	err = materializeConnectionParameter(dbName, dbService, &parameters, "client_cert", "sslcert")
	if err != nil {
		return "", err
	}

	err = materializeConnectionParameter(dbName, dbService, &parameters, "client_key", "sslkey")
	if err != nil {
		return "", err
	}

	err = materializeConnectionParameter(dbName, dbService, &parameters, "server_ca", "sslrootcert")
	if err != nil {
		return "", err
	}

	dbURL.RawQuery = parameters.Encode()

	return dbURL.String(), nil
}

func materializeConnectionParameter(dbName string, dbService cfenv.Service, parameters *url.Values, bindingProperty string, connectionParameter string) error {
	if content, hasProperty := dbService.CredentialString(bindingProperty); hasProperty {
		fileName := fmt.Sprintf("%s.%s", bindingProperty, connectionParameter)
		createdFile, err := materializeServiceProperty(dbName, fileName, content)
		if err != nil {
			return err
		}
		parameters.Set(connectionParameter, createdFile)
	}
	return nil
}

func materializeServiceProperty(serviceName, fileName, content string) (createdFile string, err error) {
	err = os.MkdirAll(fmt.Sprintf("/tmp/%s", serviceName), 0700)
	if err != nil {
		return "", err
	}
	createdFile = fmt.Sprintf("/tmp/%s/%s", serviceName, fileName)
	err = os.WriteFile(createdFile, []byte(content), 0600)
	if err != nil {
		return "", err
	}
	return
}
