/**
*
* The command line tool for CF open-autoscaler, built as a Cloudfoundry CLI plugin
*
**/
package main

import (
	"github.com/cloudfoundry/cli/plugin"
)


func main() {
	plugin.Start(new(PluginAutoScaler))
}
