package startup

import (
	"fmt"

	"code.cloudfoundry.org/app-autoscaler/src/autoscaler/helpers/runner"
	"code.cloudfoundry.org/lager/v3"
)

// ServerBuilder helps build servers with consistent error handling
type ServerBuilder struct {
	Name       string
	CreateFunc func() (runner.Runner, error)
}

// CreateServers creates multiple servers with standardized error handling
func CreateServers(builders []ServerBuilder, logger lager.Logger) []runner.Member {
	var members []runner.Member
	for _, builder := range builders {
		server, err := builder.CreateFunc()
		ExitOnError(err, logger, fmt.Sprintf("failed to create %s", builder.Name))
		members = append(members, runner.Member{Name: builder.Name, Runner: server})
	}
	return members
}

// Server creates a ServerBuilder with the given name and creation function
func Server(name string, createFunc func() (runner.Runner, error)) ServerBuilder {
	return ServerBuilder{
		Name:       name,
		CreateFunc: createFunc,
	}
}

// StartService is a convenience function that creates servers and starts them
func StartService(logger lager.Logger, servers ...ServerBuilder) {
	members := CreateServers(servers, logger)
	err := StartServices(logger, members)
	if err != nil {
		ExitOnError(err, logger, "service startup failed")
	}
}
