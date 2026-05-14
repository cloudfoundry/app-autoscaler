package helpers

import (
	"fmt"
	"net/http"
	"os"

	"code.cloudfoundry.org/app-autoscaler/src/autoscaler/helpers/runner"
	"code.cloudfoundry.org/app-autoscaler/src/autoscaler/models"
	"code.cloudfoundry.org/lager/v3"
)

type ServerConfig struct {
	Port int             `yaml:"port" json:"port"`
	TLS  models.TLSCerts `yaml:"tls" json:"tls"`
	XFCC models.XFCCAuth `yaml:"xfcc" json:"xfcc"`
}

func NewHTTPServer(logger lager.Logger, conf ServerConfig, handler http.Handler) (runner.Runner, error) {
	var addr string
	if os.Getenv("APP_AUTOSCALER_TEST_RUN") == "true" {
		addr = fmt.Sprintf("localhost:%d", conf.Port)
	} else {
		addr = fmt.Sprintf("0.0.0.0:%d", conf.Port)
	}

	logger.Info("new-http-server", lager.Data{"serverConfig": conf})

	if (conf.TLS.KeyFile != "") && (conf.TLS.CertFile != "") {
		tlsConfig, err := conf.TLS.CreateServerConfig()
		if err != nil {
			logger.Error("failed-new-server-new-tls-config", err, lager.Data{"tls": conf.TLS})
			return nil, fmt.Errorf("server tls config error: %w", err)
		}
		return runner.HTTPServer(addr, handler, tlsConfig), nil
	}

	return runner.HTTPServer(addr, handler, nil), nil
}
