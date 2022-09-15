package cf_test

import (
	"code.cloudfoundry.org/app-autoscaler/src/autoscaler/cf"
	. "code.cloudfoundry.org/app-autoscaler/src/autoscaler/testhelpers"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	. "github.com/onsi/gomega/ghttp"

	"net/http"
)

var _ = Describe("Cf client Endpoints", func() {
	Describe("GetEndpoints", func() {

		When("returns 200", func() {
			BeforeEach(func() {
				fakeCC.AppendHandlers(
					CombineHandlers(
						VerifyRequest("GET", "/"),
						RespondWith(http.StatusOK, LoadFile("endpoints.json"), http.Header{"Content-Type": []string{"application/json"}}),
					),
				)
			})

			It("returns correct struct", func() {
				endpoints, err := cfc.GetEndpoints()
				Expect(err).ToNot(HaveOccurred())
				Expect(endpoints).To(Equal(cf.Endpoints{
					CloudControllerV3: cf.Href{Url: "https://api.autoscaler.ci.cloudfoundry.org/v3"},
					NetworkPolicyV0:   cf.Href{Url: "https://api.autoscaler.ci.cloudfoundry.org/networking/v0/external"},
					NetworkPolicyV1:   cf.Href{Url: "https://api.autoscaler.ci.cloudfoundry.org/networking/v1/external"},
					Login:             cf.Href{Url: "https://login.autoscaler.ci.cloudfoundry.org"},
					Uaa:               cf.Href{Url: "https://uaa.autoscaler.ci.cloudfoundry.org"},
					Routing:           cf.Href{Url: "https://api.autoscaler.ci.cloudfoundry.org/routing"},
					Logging:           cf.Href{Url: "wss://doppler.autoscaler.ci.cloudfoundry.org:443"},
					LogCache:          cf.Href{Url: "https://log-cache.autoscaler.ci.cloudfoundry.org"},
					LogStream:         cf.Href{Url: "https://log-stream.autoscaler.ci.cloudfoundry.org"},
					AppSsh:            cf.Href{Url: "ssh.autoscaler.ci.cloudfoundry.org:2222"},
				}))
			})

		})

		When("returns 500 code", func() {
			BeforeEach(func() {
				fakeCC.AppendHandlers(
					CombineHandlers(
						VerifyRequest("GET", "/"),
						RespondWithJSONEncoded(http.StatusInternalServerError, cf.CfInternalServerError),
					),
				)
			})

			It("should return correct error", func() {
				_, err := cfc.GetEndpoints()
				Expect(err).To(HaveOccurred())
				Expect(err).To(MatchError(MatchRegexp(`failed GetEndpoints: .*cf.EndpointsResponse.*GET.*'UnknownError'.*`)))
			})
		})

	})

})
