package healthendpoint

import (
	"os"

	"code.cloudfoundry.org/lager"
	"github.com/prometheus/client_golang/prometheus"
)

func RegisterCollectors(registrar prometheus.Registerer, collectors []prometheus.Collector, includeDefault bool, logger lager.Logger) {

	if includeDefault {
		err := registrar.Register(prometheus.NewProcessCollector(os.Getpid(), ""))
		if err != nil {
			logger.Error("Failed to register process collector", err)
		}
		err = registrar.Register(prometheus.NewGoCollector())
		if err != nil {
			logger.Error("Failed to register go collector", err)
		}
	}

	for _, c := range collectors {
		err := registrar.Register(c)
		if err != nil {
			logger.Error("Failed to register collector", err, lager.Data{"collector": c})
		}
	}
}
