package healthendpoint

import (
	"os"

	"code.cloudfoundry.org/lager/v3"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/collectors"
)

func RegisterCollectors(registrar prometheus.Registerer, col []prometheus.Collector, includeDefault bool, logger lager.Logger) {
	if includeDefault {
		err := registrar.Register(collectors.NewProcessCollector(collectors.ProcessCollectorOpts{
			PidFn: func() (int, error) {
				return os.Getpid(), nil
			},
			Namespace: "",
		}))
		if err != nil {
			logger.Error("Failed to register process collector", err)
		}
		err = registrar.Register(collectors.NewGoCollector())
		if err != nil {
			logger.Error("Failed to register go collector", err)
		}
	}

	for _, c := range col {
		err := registrar.Register(c)
		if err != nil {
			logger.Error("Failed to register collector", err, lager.Data{"collector": c})
		}
	}
}
