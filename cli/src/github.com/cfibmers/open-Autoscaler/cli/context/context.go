package context

import (
	"github.com/cloudfoundry/cli/plugin"
	"github.com/cloudfoundry/cli/plugin/models"
	"github.ibm.com/Bluemix/bluemix-cli-common/bluemix/terminal"
)

var context *CommandContext

type CommandContext struct {
	cliConn plugin.CliConnection
	ui      terminal.UI
}

func Init(cliConnection plugin.CliConnection) {
	context = &CommandContext{
		cliConn: cliConnection,
		ui:      terminal.StdUI,
	}
}

func UI() terminal.UI {
	return context.ui
}

func CheckPrerequisties() {
	if result, _ := context.cliConn.IsLoggedIn(); !result {
		context.ui.Failed("Please log in before using auto-scaling cli.")
	}
	if result, _ := context.cliConn.HasSpace(); !result {
		context.ui.Failed("Please target a space first.")
	}
}

func AccessToken() string {
	token, err := context.cliConn.AccessToken()
	if err != nil {
		context.ui.Failed(err.Error())
	}
	return token
}

func Org() plugin_models.Organization {
	org, err := context.cliConn.GetCurrentOrg()
	if err != nil {
		context.ui.Failed(err.Error())
	}
	return org
}

func Space() plugin_models.Space {
	space, err := context.cliConn.GetCurrentSpace()
	if err != nil {
		context.ui.Failed(err.Error())
	}
	return space
}
