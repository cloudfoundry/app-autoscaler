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
	if os.Getenv("DEBUG") == "" || cfg.ASApiEndpoint == "" {
		return
	}
	if os.Getenv("CF_PLUGIN_HOME") == "" {
		_ = os.Setenv("CF_PLUGIN_HOME", os.Getenv("HOME"))
	}
	output := new(strings.Builder)
	_, _ = fmt.Fprintf(output, "\n=============== DEBUG ===============\n")

	// autoscaling-api writes the plugin's local config file that the other
	// autoscaling-* commands read, so it must run first.
	commands := [][]string{
		{"autoscaling-api", cfg.ASApiEndpoint},
		{"app", anApp},
		{"events", anApp},
		{"logs", "--recent", anApp},
		{"autoscaling-policy", anApp},
		{"autoscaling-history", anApp},
		{"autoscaling-metrics", anApp, "memoryused"},
		{"autoscaling-metrics", anApp, "memoryutil"},
		{"autoscaling-metrics", anApp, "responsetime"},
		{"autoscaling-metrics", anApp, "throughput"},
		{"autoscaling-metrics", anApp, "cpu"},
		{"autoscaling-metrics", anApp, "cpuutil"},
		{"autoscaling-metrics", anApp, "disk"},
		{"autoscaling-metrics", anApp, "diskutil"},
		{"autoscaling-metrics", anApp, "test_metric"},
	}
	for _, args := range commands {
		runAndPrint(output, "cf", args...)
	}
	_, _ = fmt.Fprintf(output, "\n=====================================\n")
	GinkgoWriter.Print(output.String())
}

// runAndPrint runs the command to completion and appends its invocation and
// stdout/stderr to output.
func runAndPrint(output *strings.Builder, name string, args ...string) {
	session, err := Start(exec.Command(name, args...), nil, nil)
	if err != nil {
		GinkgoWriter.Println(err.Error())
		return
	}
	session.Wait(30 * time.Second)
	_, _ = fmt.Fprintln(output, strings.Join(session.Command.Args, " ")+":")
	_, _ = fmt.Fprintln(output, string(session.Out.Contents()))
	_, _ = fmt.Fprintln(output, string(session.Err.Contents()))
}
