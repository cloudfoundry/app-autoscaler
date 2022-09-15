package cf_test

import (
	"code.cloudfoundry.org/app-autoscaler/src/autoscaler/cf"
	. "code.cloudfoundry.org/app-autoscaler/src/autoscaler/testhelpers"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	. "github.com/onsi/gomega/ghttp"

	"net/http"
)

var _ = Describe("Cf client Roles", func() {
	BeforeEach(login)
	Describe("Roles.HasRole", func() {
		When("The role is present", func() {
			roles := cf.Roles{{Type: cf.RoleSpaceDeveloper}}
			It("Should return true", func() {
				Expect(roles.HasRole("space_developer")).To(BeTrue())
			})
		})
		When("The role is not present", func() {
			roles := cf.Roles{
				{Type: cf.RoleSpaceManager},
				{Type: cf.RoleOrganizationManager},
				{Type: cf.RoleOrganizationBillingManager},
				{Type: cf.RoleOrganisationUser},
				{Type: cf.RoleOrganizationAuditor},
				{Type: cf.RoleSpaceSupporter},
			}
			It("should return false", func() {
				Expect(roles.HasRole(cf.RoleSpaceDeveloper)).To(BeFalse())

			})

		})
		When("the roles is nil", func() {})
	})

	Describe("GetRoles", func() {

		When("get roles succeeds", func() {
			BeforeEach(func() {
				fakeCC.AppendHandlers(
					CombineHandlers(
						VerifyRequest("GET", "/v3/roles", "types=space_developer&space_guids=some_space_id&user_guids=someUserId"),
						VerifyHeaderKV("Authorization", "Bearer test-access-token"),
						RespondWith(http.StatusCreated, LoadFile("roles.json"), http.Header{"Content-Type": []string{"application/json"}}),
					),
				)
			})

			It("returns correct struct", func() {
				spaceId := cf.SpaceId("some_space_id")
				userId := cf.UserId("someUserId")
				roles, err := cfc.GetSpaceDeveloperRoles(spaceId, userId)
				Expect(err).NotTo(HaveOccurred())
				Expect(roles).To(Equal(cf.Roles{
					{
						Guid: "40557c70-d1bd-4976-a2ab-a85f5e882418",
						Type: "organization_auditor",
					},
					{
						Guid: "12347c70-d1bd-4976-a2ab-a85f5e882418",
						Type: "space_auditor",
					},
					{
						Guid: "12347c70-d1bd-4976-a2ab-a85f5e882418",
						Type: "space_auditor",
					}}))
			})
		})

		When("get app returns a 500 code", func() {
			BeforeEach(func() {
				fakeCC.AppendHandlers(
					CombineHandlers(
						VerifyRequest("GET", "/v3/roles", "types=space_developer&space_guids=some_space_id&user_guids=someUserId"),
						RespondWithJSONEncoded(http.StatusInternalServerError, cf.CfInternalServerError),
					),
				)
			})

			It("should return correct error", func() {
				spaceId := cf.SpaceId("some_space_id")
				userId := cf.UserId("someUserId")
				_, err := cfc.GetSpaceDeveloperRoles(spaceId, userId)
				Expect(err).To(HaveOccurred())
				Expect(err).To(MatchError(MatchRegexp(`failed GetSpaceDeveloperRoles spaceId\(some_space_id\) userId\(someUserId\):.*cf.Response\[.*cf.Role\].*GET.*'UnknownError'.*`)))
			})
		})

	})

})
