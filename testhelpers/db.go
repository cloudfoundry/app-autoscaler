package testhelpers

import (
	"os"

	. "github.com/onsi/ginkgo/v2"
)

func GetDbUrl() string {
	dbUrl := os.Getenv("DBURL")
	if dbUrl == "" {
		Fail("environment variable $DBURL is not set")
	}
	return dbUrl
}
