package brokerserver

import (
	"fmt"
	"net/http"

	"autoscaler/api/config"
	"autoscaler/db"
	"autoscaler/healthendpoint"
	"autoscaler/routes"

	"code.cloudfoundry.org/cfhttp"
	"code.cloudfoundry.org/lager"
	"github.com/gorilla/mux"
	"github.com/tedsuo/ifrit"
	"github.com/tedsuo/ifrit/http_server"
	"golang.org/x/crypto/bcrypt"
)

type VarsFunc func(w http.ResponseWriter, r *http.Request, vars map[string]string)

func (vh VarsFunc) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	vh(w, r, vars)
}

type basicAuthenticationMiddleware struct {
	usernameHash []byte
	passwordHash []byte
}

func (bam *basicAuthenticationMiddleware) Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		username, password, authOK := r.BasicAuth()

		if authOK == false || bcrypt.CompareHashAndPassword(bam.usernameHash, []byte(username)) != nil || bcrypt.CompareHashAndPassword(bam.passwordHash, []byte(password)) != nil {
			http.Error(w, http.StatusText(http.StatusUnauthorized), http.StatusUnauthorized)
			return
		}
		next.ServeHTTP(w, r)
	})
}

func NewBrokerServer(logger lager.Logger, conf *config.Config, bindingdb db.BindingDB, policydb db.PolicyDB, httpStatusCollector healthendpoint.HTTPStatusCollector) (ifrit.Runner, error) {

	var usernameHash []byte
	if conf.BrokerUsernameHash != "" {
		usernameHash = []byte(conf.BrokerUsernameHash)
	} else {
		var err error
		usernameHash, err = bcrypt.GenerateFromPassword([]byte(conf.BrokerUsername), bcrypt.MinCost) // use MinCost as the config already provided it as cleartext
		if err != nil {
			logger.Error("failed-new-server-hashing-broker-username", err)
			return nil, err
		}
	}

	var passwordHash []byte
	if conf.BrokerPasswordHash != "" {
		passwordHash = []byte(conf.BrokerPasswordHash)
	} else {
		var err error
		passwordHash, err = bcrypt.GenerateFromPassword([]byte(conf.BrokerPassword), bcrypt.MinCost) // use MinCost as the config already provided it as cleartext
		if err != nil {
			logger.Error("failed-new-server-hashing-broker-password", err)
			return nil, err
		}
	}

	basicAuthentication := &basicAuthenticationMiddleware{
		usernameHash: usernameHash,
		passwordHash: passwordHash,
	}
	httpStatusCollectMiddleware := healthendpoint.NewHTTPStatusCollectMiddleware(httpStatusCollector)
	ah := NewBrokerHandler(logger, conf, bindingdb, policydb)

	r := routes.BrokerRoutes()

	r.Use(basicAuthentication.Middleware)
	r.Use(httpStatusCollectMiddleware.Collect)
	r.Get(routes.BrokerCatalogRouteName).Handler(VarsFunc(ah.GetBrokerCatalog))
	r.Get(routes.BrokerCreateInstanceRouteName).Handler(VarsFunc(ah.CreateServiceInstance))
	r.Get(routes.BrokerUpdateInstanceRouteName).Handler(VarsFunc(ah.UpdateServiceInstance))
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
