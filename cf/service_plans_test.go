package cf_test

import (
	"code.cloudfoundry.org/app-autoscaler/src/autoscaler/cf"
	. "code.cloudfoundry.org/app-autoscaler/src/autoscaler/testhelpers"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	. "github.com/onsi/gomega/ghttp"

	"net/http"
)

var _ = Describe("Cf client Service Plans", func() {
	BeforeEach(login)

	Describe("GetServicePlan", func() {

		When("get service plans succeeds", func() {
			BeforeEach(func() {
				fakeCC.AppendHandlers(
					CombineHandlers(
						VerifyRequest("GET", "/v3/service_plans/test_guid"),
						VerifyHeaderKV("Authorization", "Bearer test-access-token"),
						RespondWith(http.StatusOK, LoadFile("service_plan.json"), http.Header{"Content-Type": []string{"application/json"}}),
					),
				)
			})

			It("returns correct struct", func() {
				servicePlan, err := cfc.GetServicePlan("test_guid")
				Expect(err).NotTo(HaveOccurred())
				Expect(servicePlan).To(Equal(&cf.ServicePlan{Guid: "d67b2fe4-665c-4bf2-9ccc-e080c49d48d4",
					BrokerCatalog: cf.BrokerCatalog{
						Id: "autoscaler-free-plan-id",
					}}))
			})
		})

		When("get service plans returns a 500 code", func() {
			BeforeEach(func() {
				fakeCC.AppendHandlers(
					CombineHandlers(
						VerifyRequest("GET", "/v3/service_plans/test_guid"),
						RespondWithJSONEncoded(http.StatusInternalServerError, cf.CfInternalServerError),
					),
				)
			})

			It("should return correct error", func() {
				_, err := cfc.GetServicePlan("test_guid")
				Expect(err).To(HaveOccurred())
				Expect(err).To(MatchError(MatchRegexp(`failed GetServicePlan\(test_guid\):.*cf.ServicePlan.*GET.*'UnknownError'.*`)))
			})
		})

	})

})
