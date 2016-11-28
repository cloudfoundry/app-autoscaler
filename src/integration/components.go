package integration

import (
	"github.com/tedsuo/ifrit/ginkgomon"
	"os/exec"
	"strconv"
	"time"
)

const (
	APIServer     = "apiServer"
	ServiceBroker = "serviceBroker"
	Scheduler     = "scheduler"
)

type Executables map[string]string
type Ports map[string]int

type Components struct {
	Executables Executables
	Ports       Ports
}

type DBConfig struct {
	URI            string `json:"uri"`
	MinConnections int    `json:"minConnections"`
	MaxConnections int    `json:"maxConnections"`
	IdleTimeout    int    `json:"idleTimeout"`
}

type ServiceBrokerConfig struct {
	Port int `json:"port"`

	Username string `json:"username"`
	Password string `json:"password"`

	DB DBConfig `json:"db"`

	APIServerUri       string `json:"apiServerUri"`
	HttpRequestTimeout int    `json:"httpRequestTimeout"`
}

type APIServerConfig struct {
	Port int `json:"port"`

	DB DBConfig `json:"db"`

	SchedulerUri string `json:"schedulerUri"`
}

func (components *Components) ServiceBroker(confPath string, argv ...string) *ginkgomon.Runner {
	return ginkgomon.New(ginkgomon.Config{
		Name:              ServiceBroker,
		AnsiColorCode:     "32m",
		StartCheck:        "Service broker app is running",
		StartCheckTimeout: 10 * time.Second,
		Command: exec.Command(
			"node", append([]string{components.Executables[ServiceBroker], "-c", confPath}, argv...)...,
		),
		Cleanup: func() {
		},
	})
}

func (components *Components) ApiServer(confPath string, argv ...string) *ginkgomon.Runner {
	return ginkgomon.New(ginkgomon.Config{
		Name:              APIServer,
		AnsiColorCode:     "33m",
		StartCheck:        "Autoscaler API server started",
		StartCheckTimeout: 10 * time.Second,
		Command: exec.Command(
			"node", append([]string{components.Executables[APIServer], "-c", confPath}, argv...)...,
		),
		Cleanup: func() {
		},
	})
}
func (components *Components) Scheduler(confPath string, argv ...string) *ginkgomon.Runner {

	return ginkgomon.New(ginkgomon.Config{
		Name:              Scheduler,
		AnsiColorCode:     "34m",
		StartCheck:        "Started SchedulerApplication in",
		StartCheckTimeout: 60 * time.Second,
		Command: exec.Command(
			"java", append([]string{"-jar", "-Dspring.config.location=" + confPath, "-Dserver.port=" + strconv.FormatInt(int64(components.Ports[Scheduler]), 10), components.Executables[Scheduler]}, argv...)...,
		),
		Cleanup: func() {
		},
	})
}
