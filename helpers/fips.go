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

// FipsEnforcedGauge reports whether configured FIPS enforcement passed the
// runtime check at startup. Runtime FIPS may be enabled without enforcement.
var FipsEnforcedGauge = prometheus.NewGauge(prometheus.GaugeOpts{
	Namespace: "autoscaler",
	Name:      "fips_enforced",
	Help:      "Indicates whether FIPS is required by configuration and verified at startup (1=enforced, 0=not enforced).",
})

// AssertFIPSMode conditionally checks and enforces FIPS 140-3 mode.
// Runtime state is reported independently of the configured requirement.
// When fipsRequired is true, the process exits with code 140 if FIPS is not active.
func AssertFIPSMode(fipsRequired bool) {
	runtimeEnabled := fips140.Enabled()
	FipsEnabledGauge.Set(0)
	if runtimeEnabled {
		FipsEnabledGauge.Set(1)
	}
	FipsEnforcedGauge.Set(0)

	if !fipsRequired {
		_, _ = fmt.Fprintf(os.Stdout, "FIPS 140-3 mode is not enforced\n")
		return
	}

	if !runtimeEnabled {
		_, _ = fmt.Fprintf(os.Stdout, "FIPS 140-3 mode is required but not enabled. Check https://go.dev/doc/security/fips140 for how to enable it. Exiting.\n")
		os.Exit(140)
	}

	_, _ = fmt.Fprintf(os.Stdout, "FIPS 140-3 mode enabled\n")
	FipsEnforcedGauge.Set(1)
}
