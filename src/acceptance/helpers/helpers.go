package helpers

import (
	"acceptance/config"
	"fmt"
	"strconv"
	"strings"

	"github.com/cloudfoundry-incubator/cf-test-helpers/cf"
	"github.com/cloudfoundry-incubator/cf-test-helpers/runner"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/gexec"
)

func Curl(args ...string) (int, []byte, error) {
	curlCmd := runner.Curl(append([]string{"--output", "/dev/stderr", "--write-out", "%{http_code}"}, args...)...).Wait(config.DEFAULT_TIMEOUT)
	if curlCmd.ExitCode() != 0 {
		return 0, curlCmd.Err.Contents(), fmt.Errorf("curl failed: exit code %d", curlCmd.ExitCode())
	}
	statusCode, err := strconv.Atoi(string(curlCmd.Out.Contents()))
	if err != nil {
		return 0, curlCmd.Err.Contents(), err
	}
	return statusCode, curlCmd.Err.Contents(), nil
}

func OauthToken() string {
	cmd := cf.Cf("oauth-token")
	Expect(cmd.Wait(config.DEFAULT_TIMEOUT)).To(gexec.Exit(0))
	return strings.TrimSpace(string(cmd.Out.Contents()))
}
