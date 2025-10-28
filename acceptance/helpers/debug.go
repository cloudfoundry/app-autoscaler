package helpers

import (
	"acceptance/config"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"time"

	"github.com/cloudfoundry/cf-test-helpers/v2/workflowhelpers"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega/gexec"
)

func DebugInfo(cfg *config.Config, setup *workflowhelpers.ReproducibleTestSuiteSetup, anApp string) {
	if os.Getenv("DEBUG") != "" && cfg.ASApiEndpoint != "" {
		if os.Getenv("CF_PLUGIN_HOME") == "" {
			_ = os.Setenv("CF_PLUGIN_HOME", os.Getenv("HOME"))
		}
		var commands []*Session
		commands = append(commands, command("cf", "app", anApp))
		commands = append(commands, command("cf", "events", anApp))
		commands = append(commands, command("cf", "logs", "--recent", anApp))
		commands = append(commands, command("cf", "autoscaling-api", cfg.ASApiEndpoint))
		commands = append(commands, command("cf", "autoscaling-policy", anApp))
		commands = append(commands, command("cf", "autoscaling-history", anApp))
		commands = append(commands, command("cf", "autoscaling-metrics", anApp, "memoryused"))
		commands = append(commands, command("cf", "autoscaling-metrics", anApp, "memoryutil"))
		commands = append(commands, command("cf", "autoscaling-metrics", anApp, "responsetime"))
		commands = append(commands, command("cf", "autoscaling-metrics", anApp, "throughput"))
		commands = append(commands, command("cf", "autoscaling-metrics", anApp, "cpu"))
		commands = append(commands, command("cf", "autoscaling-metrics", anApp, "cpuutil"))
		commands = append(commands, command("cf", "autoscaling-metrics", anApp, "disk"))
		commands = append(commands, command("cf", "autoscaling-metrics", anApp, "diskutil"))
		commands = append(commands, command("cf", "autoscaling-metrics", anApp, "test_metric"))
		output := new(strings.Builder)
		_, _ = fmt.Fprintf(output, "\n=============== DEBUG ===============\n")
		for _, command := range commands {
			command.Wait(30 * time.Second)
			_, _ = fmt.Fprintln(output, strings.Join(command.Command.Args, " ")+":")
			_, _ = fmt.Fprintln(output, string(command.Out.Contents()))
			_, _ = fmt.Fprintln(output, string(command.Err.Contents()))
		}
		_, _ = fmt.Fprintf(output, "\n=====================================\n")
		GinkgoWriter.Print(output.String())
	}
}

func command(name string, args ...string) *Session {
	cmd := exec.Command(name, args...)
	start, err := Start(cmd, nil, nil)
	if err != nil {
		GinkgoWriter.Println(err.Error())
	}
	return start
}
