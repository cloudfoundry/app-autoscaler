package publicapiserver

import (
	"net/http"

	"autoscaler/db"
	"autoscaler/models"

	"code.cloudfoundry.org/cfhttp/handlers"
	"code.cloudfoundry.org/lager"
	"github.com/gorilla/mux"
)

type CheckBindingMiddleware struct {
	logger    lager.Logger
	bindingDB db.BindingDB
}

func NewCheckBindingMiddleware(logger lager.Logger, bindingDB db.BindingDB) *CheckBindingMiddleware {
	return &CheckBindingMiddleware{
		logger:    logger,
		bindingDB: bindingDB,
	}
}

func (cbm *CheckBindingMiddleware) CheckServiceBinding(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		vars := mux.Vars(r)
		appId := vars["appId"]
		if appId == "" {
			cbm.logger.Error("appId is not present", nil, lager.Data{"url": r.URL.String()})
			handlers.WriteJSONResponse(w, http.StatusBadRequest, models.ErrorResponse{
				Code:    "Bad Request",
				Message: "AppId is required",
			})
			return
		}
		hasBinding := cbm.bindingDB.CheckServiceBinding(appId)
		if hasBinding {
			cbm.logger.Debug("binding is present", nil, lager.Data{"appId": appId})
			next.ServeHTTP(w, r)
			return
		}
		cbm.logger.Error("binding is not present", nil, lager.Data{"appId": appId})
		http.Error(w, "{ \"error\": \"The application is not bound to Auto-Scaling service\" }", http.StatusForbidden)
		return

	})
}
