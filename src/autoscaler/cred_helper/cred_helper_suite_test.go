package cred_helper_test

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"testing"
)

func TestCredHelper(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Credentials Helper Suite")
}
