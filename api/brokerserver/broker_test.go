package brokerserver_test

import (
	"context"

	"code.cloudfoundry.org/app-autoscaler/src/autoscaler/api/brokerserver"
	"code.cloudfoundry.org/app-autoscaler/src/autoscaler/api/config"
	"code.cloudfoundry.org/app-autoscaler/src/autoscaler/fakes"
	"code.cloudfoundry.org/lager/lagertest"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/pivotal-cf/brokerapi/v8/domain"
)

var _ = Describe("Broker", func() {
	var (
		broker          *brokerserver.Broker
		err             error
		bindingdb       *fakes.FakeBindingDB
		policydb        *fakes.FakePolicyDB
		fakecfClient    *fakes.FakeCFClient
		fakeCredentials *fakes.FakeCredentials
		conf            *config.Config
		testLogger      = lagertest.NewTestLogger("test")
	)

	JustBeforeEach(func() {
		broker = brokerserver.NewBroker(testLogger, conf, bindingdb, policydb, services, fakecfClient, fakeCredentials)
	})

	Describe("Services", func() {
		var services []domain.Service
		JustBeforeEach(func() {
			services, err = broker.Services(context.TODO())
		})
		Context("returns the services", func() {
			Expect(err).ToNot(HaveOccurred())
			Expect(services).To(Equal(services))
		})
	})
})
