package main_test

import (
	"fmt"
	"os"
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/gexec"
)

var cmdPath string

func TestGorouterproxy(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Gorouterproxy Suite")
}

var _ = BeforeSuite(func() {
	var err error
	gorouterproxyPath, err := gexec.Build(".")
	Expect(err).NotTo(HaveOccurred())

	// Remove execute permissions
	cmdPath = fmt.Sprintf("%s/gorouterproxy", gorouterproxyPath)
	err = os.Chmod(cmdPath, 0770)
	Expect(err).NotTo(HaveOccurred())

	DeferCleanup(gexec.CleanupBuildArtifacts)
})
