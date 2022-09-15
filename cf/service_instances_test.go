package cf_test

import (
	"code.cloudfoundry.org/app-autoscaler/src/autoscaler/cf"
	. "code.cloudfoundry.org/app-autoscaler/src/autoscaler/testhelpers"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	. "github.com/onsi/gomega/ghttp"

	"net/http"
)

var _ = Describe("Cf client Service Instances", func() {
	BeforeEach(login)
	Describe("Get Service Instances", func() {

		When("get service instances succeeds", func() {
			BeforeEach(func() {
				fakeCC.AppendHandlers(
					CombineHandlers(
						VerifyRequest("GET", "/v3/service_instances/test_guid"),
						VerifyHeaderKV("Authorization", "Bearer test-access-token"),
						RespondWith(http.StatusOK, LoadFile("service_instance.json"), http.Header{"Content-Type": []string{"application/json"}}),
					),
				)
			})

			It("returns correct struct", func() {
				serviceInstance, err := cfc.GetServiceInstance("test_guid")
				Expect(err).NotTo(HaveOccurred())
				Expect(serviceInstance).To(Equal(&cf.ServiceInstance{
					Guid: "c89b3280-fe8d-4aa0-a42e-44465bb1c61c",
					Type: "managed",
					Relationships: cf.ServiceInstanceRelationships{
						ServicePlan: cf.ServicePlanRelation{
							Data: cf.ServicePlanData{
								Guid: "5358d122-638e-11ea-afca-bf6e756684ac",
							},
						},
					}}))
			})
		})

		When("get service instance returns a 500 code", func() {
			BeforeEach(func() {
				fakeCC.AppendHandlers(
					CombineHandlers(
						VerifyRequest("GET", "/v3/service_instances/test_guid"),
						RespondWithJSONEncoded(http.StatusInternalServerError, cf.CfInternalServerError),
					),
				)
			})

			It("should return correct error", func() {
				_, err := cfc.GetServiceInstance("test_guid")
				Expect(err).To(HaveOccurred())
				Expect(err).To(MatchError(MatchRegexp(`failed GetServiceInstance guid\(test_guid\):.*cf.ServiceInstance.*GET.*'UnknownError'.*`)))
			})
		})

	})

})
