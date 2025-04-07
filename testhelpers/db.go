package testhelpers

import (
	"os"

	"code.cloudfoundry.org/app-autoscaler/src/autoscaler/db"
	"code.cloudfoundry.org/app-autoscaler/src/autoscaler/fakes"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

func GetDbUrl() string {
	dbUrl := os.Getenv("DBURL")
	if dbUrl == "" {
		Fail("environment variable $DBURL is not set")
	}
	return dbUrl
}

func ExpectConfigureDatabasesCalledOnce(err error, fakeVcapReader *fakes.FakeVCAPConfigurationReader, expectedCredHelperImpl string) {
	Expect(err).NotTo(HaveOccurred())
	Expect(fakeVcapReader.ConfigureDatabasesCallCount()).To(Equal(1))
	receivedDbConfig, receivedStoredProcedureConfig, receivedCredHelperImpl :=
		fakeVcapReader.ConfigureDatabasesArgsForCall(0)
	Expect(*receivedDbConfig).To(Equal(map[string]db.DatabaseConfig{}))
	Expect(receivedStoredProcedureConfig).To(BeNil())
	Expect(receivedCredHelperImpl).To(Equal(expectedCredHelperImpl))
}
