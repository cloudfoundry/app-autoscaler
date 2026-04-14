package helpers

import (
	"crypto/fips140"
	"fmt"
	"os"

	"github.com/prometheus/client_golang/prometheus"
)

// FipsEnabledGauge reports FIPS 140-3 mode status: 1 if enabled, 0 if disabled.
var FipsEnabledGauge = prometheus.NewGauge(prometheus.GaugeOpts{
	Namespace: "autoscaler",
	Name:      "fips_enabled",
	Help:      "Indicates whether FIPS 140-3 mode is active (1=enabled, 0=disabled).",
})

// AssertFIPSMode conditionally checks and enforces FIPS 140-3 mode.
// When fipsEnabled is false, FIPS enforcement is skipped.
// When fipsEnabled is true, the process exits with code 140 if FIPS is not active.
// The FipsEnabledGauge is set accordingly.
func AssertFIPSMode(fipsEnabled bool) {
	if !fipsEnabled {
		_, _ = fmt.Fprintf(os.Stdout, "FIPS 140-3 mode is not enabled\n")
		FipsEnabledGauge.Set(0)
		return
	}

	if !fips140.Enabled() {
		FipsEnabledGauge.Set(0)
		_, _ = fmt.Fprintf(os.Stdout, "FIPS 140-3 mode is required but not enabled. Check https://go.dev/doc/security/fips140 for how to enable it. Exiting.\n")
		os.Exit(140)
	}

	_, _ = fmt.Fprintf(os.Stdout, "FIPS 140-3 mode enabled\n")
	FipsEnabledGauge.Set(1)
}
