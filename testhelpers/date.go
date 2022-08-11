package testhelpers

import (
	"time"

	. "github.com/onsi/gomega"
)

func ParseDate(date string) time.Time {
	updated, err := time.Parse(time.RFC3339, date)
	Expect(err).NotTo(HaveOccurred())
	return updated
}
