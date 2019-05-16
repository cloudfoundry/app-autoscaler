package brokerserver

import (
	"fmt"
	"net/http"

	"autoscaler/api/config"
	"autoscaler/db"
	"autoscaler/routes"

	"code.cloudfoundry.org/cfhttp"
	"code.cloudfoundry.org/lager"
	"github.com/gorilla/mux"
	"github.com/tedsuo/ifrit"
	"github.com/tedsuo/ifrit/http_server"
)

type VarsFunc func(w http.ResponseWriter, r *http.Request, vars map[string]string)

func (vh VarsFunc) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	vh(w, r, vars)
}

type basicAuthenticationMiddleware struct {
	username string
	password string
}

func (bam *basicAuthenticationMiddleware) Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		username, password, authOK := r.BasicAuth()
		if authOK == false || username != bam.username || password != bam.password {
			http.Error(w, http.StatusText(http.StatusUnauthorized), http.StatusUnauthorized)
			return
		}
		next.ServeHTTP(w, r)
	})
}

func NewBrokerServer(logger lager.Logger, conf *config.Config, bindingdb db.BindingDB, policydb db.PolicyDB) (ifrit.Runner, error) {

	basicAuthentication := &basicAuthenticationMiddleware{
		username: conf.BrokerUsername,
		password: conf.BrokerPassword,
	}

	ah := NewBrokerHandler(logger, conf, bindingdb, policydb)

	r := routes.BrokerRoutes()

	r.Use(basicAuthentication.Middleware)
	r.Get(routes.BrokerCatalogRouteName).Handler(VarsFunc(ah.GetBrokerCatalog))
	r.Get(routes.BrokerCreateInstanceRouteName).Handler(VarsFunc(ah.CreateServiceInstance))
	r.Get(routes.BrokerDeleteInstanceRouteName).Handler(VarsFunc(ah.DeleteServiceInstance))
	r.Get(routes.BrokerCreateBindingRouteName).Handler(VarsFunc(ah.BindServiceInstance))
	r.Get(routes.BrokerDeleteBindingRouteName).Handler(VarsFunc(ah.UnbindServiceInstance))

	addr := fmt.Sprintf("0.0.0.0:%d", conf.BrokerServer.Port)

	var runner ifrit.Runner
	if (conf.BrokerServer.TLS.KeyFile == "") || (conf.BrokerServer.TLS.CertFile == "") {
		runner = http_server.New(addr, r)
	} else {
		tlsConfig, err := cfhttp.NewTLSConfig(conf.BrokerServer.TLS.CertFile, conf.BrokerServer.TLS.KeyFile, conf.BrokerServer.TLS.CACertFile)
		if err != nil {
			logger.Error("failed-new-server-new-tls-config", err, lager.Data{"tls": conf.BrokerServer.TLS})
			return nil, err
		}
		runner = http_server.NewTLSServer(addr, r, tlsConfig)
	}

	logger.Info("broker-http-server-created", lager.Data{"serverConfig": conf.BrokerServer})
	return runner, nil
}

func GetDashboardURL(conf *config.Config, instanceId string) string {
	return fmt.Sprintf("%s/manage/%s", conf.DashboardRedirectURI, instanceId)
}
