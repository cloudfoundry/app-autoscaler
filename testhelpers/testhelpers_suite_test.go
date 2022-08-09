package testhelpers_test

import (
	_ "github.com/go-sql-driver/mysql"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"testing"
)

func TestTestHelpers(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Testhelpers Suite")
}
