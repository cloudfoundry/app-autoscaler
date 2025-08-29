package helpers

import (
	"crypto/fips140"
	"fmt"
	"os"
)

func AssertFIPSMode() {
	if !fips140.Enabled() {
		_, _ = fmt.Fprintf(os.Stdout, "FIPS 140-3 mode is required but not enabled. Check https://go.dev/doc/security/fips140 for how to enable it. Exiting.")
		os.Exit(140)
	}
}
