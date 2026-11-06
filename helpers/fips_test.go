package helpers_test

import (
	"crypto/fips140"
	"fmt"
	"os"
	"os/exec"
	"testing"

	"code.cloudfoundry.org/app-autoscaler/src/autoscaler/helpers"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	dto "github.com/prometheus/client_model/go"
)

var _ = Describe("FIPS Mode", func() {
	Describe("AssertFIPSMode", func() {
		Context("when FIPS is not required", func() {
			It("reports runtime state without enforcing it", func() {
				helpers.AssertFIPSMode(false)

				var metric dto.Metric
				Expect(helpers.FipsEnabledGauge.Write(&metric)).To(Succeed())
				expected := float64(0)
				if fips140.Enabled() {
					expected = 1
				}
				Expect(metric.GetGauge().GetValue()).To(Equal(expected))
				Expect(helpers.FipsEnforcedGauge.Write(&metric)).To(Succeed())
				Expect(metric.GetGauge().GetValue()).To(Equal(float64(0)))
			})
		})
	})

	DescribeTable("runtime state and enforcement are independent", func(runtimeMode, required, expected string, exitCode int) {
		cmd := exec.Command(os.Args[0], "-test.run=^TestFipsModeProcess$")
		cmd.Env = append(os.Environ(), "TEST_FIPS_MODE_PROCESS=true", "TEST_FIPS_REQUIRED="+required, "GODEBUG=fips140="+runtimeMode)
		output, err := cmd.CombinedOutput()
		if exitCode == 0 {
			Expect(err).NotTo(HaveOccurred(), "%s", output)
		} else {
			var exitErr *exec.ExitError
			Expect(err).To(BeAssignableToTypeOf(exitErr), "%s", output)
			Expect(err.(*exec.ExitError).ExitCode()).To(Equal(exitCode), "%s", output)
		}
		Expect(string(output)).To(ContainSubstring(expected))
	},
		Entry("disabled, not required", "off", "false", "enabled=0 enforced=0", 0),
		Entry("enabled, not required", "on", "false", "enabled=1 enforced=0", 0),
		Entry("enabled and required", "on", "true", "enabled=1 enforced=1", 0),
		Entry("required but disabled fails closed", "off", "true", "FIPS 140-3 mode is required but not enabled", 140),
	)

	Describe("FipsEnabledGauge", func() {
		It("has the correct description", func() {
			Expect(helpers.FipsEnabledGauge.Desc().String()).To(ContainSubstring("autoscaler_fips_enabled"))
		})
	})

	Describe("FipsEnforcedGauge", func() {
		It("has the correct description", func() {
			Expect(helpers.FipsEnforcedGauge.Desc().String()).To(ContainSubstring("autoscaler_fips_enforced"))
		})
	})
})

// FIPS runtime mode is fixed at process startup, so exercise each mode in a
// subprocess. This test is otherwise a no-op during normal suite execution.
func TestFipsModeProcess(t *testing.T) {
	if os.Getenv("TEST_FIPS_MODE_PROCESS") != "true" {
		return
	}
	helpers.AssertFIPSMode(os.Getenv("TEST_FIPS_REQUIRED") == "true")
	var enabled, enforced dto.Metric
	if err := helpers.FipsEnabledGauge.Write(&enabled); err != nil {
		t.Fatal(err)
	}
	if err := helpers.FipsEnforcedGauge.Write(&enforced); err != nil {
		t.Fatal(err)
	}
	fmt.Printf("enabled=%.0f enforced=%.0f\n", enabled.GetGauge().GetValue(), enforced.GetGauge().GetValue())
}
