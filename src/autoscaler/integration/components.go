package integration

import (
	"github.com/tedsuo/ifrit"
	"github.com/tedsuo/ifrit/ginkgomon"
	"os/exec"
	"time"
)

type Executables map[string]string
type Ports map[string]int
type Components struct {
	Executables Executables
	Ports       Ports
}

func (components *Components) ServiceBroker(confPath string, argv ...string) ifrit.Runner {

	return ginkgomon.New(ginkgomon.Config{
		Name:              "broker",
		AnsiColorCode:     "32m",
		StartCheck:        "Service broker app is running",
		StartCheckTimeout: 10 * time.Second,
		Command: exec.Command(
			"node", append([]string{components.Executables["serviceBroker"], "-c", confPath}, argv...)...,
		),
		Cleanup: func() {
		},
	})
}

func (components *Components) ApiServer(confPath string, argv ...string) ifrit.Runner {

	return ginkgomon.New(ginkgomon.Config{
		Name:              "api",
		AnsiColorCode:     "33m",
		StartCheck:        "Autoscaler API server started",
		StartCheckTimeout: 10 * time.Second,
		Command: exec.Command(
			"node", append([]string{components.Executables["apiServer"], "-c", confPath}, argv...)...,
		),
		Cleanup: func() {
		},
	})
}
