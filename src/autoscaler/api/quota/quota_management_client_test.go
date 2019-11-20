package quota

import (
	"autoscaler/api/config"
	"net/http"

	"code.cloudfoundry.org/lager/lagertest"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/ghttp"
)

var _ = Describe("Quota", func() {
	var (
		qmc         *QuotaManagementClient
		quota       int
		err         error
		quotaServer *ghttp.Server
	)
	BeforeEach(func() {
	})
	Context("GetQuota", func() {
		Context("when not configured", func() {
			BeforeEach(func() {
				qmc = NewQuotaManagementClient(nil, lagertest.NewTestLogger("Quota"), nil)
			})
			It("returns -1", func() {
				quota, err = qmc.GetQuota("test-org", "test-service", "test-plan")
				Expect(err).ToNot(HaveOccurred())
				Expect(quota).To(Equal(-1))
			})
		})
		Context("when configured", func() {
			BeforeEach(func() {
				quotaServer = ghttp.NewServer()
				quotaConfig := &config.QuotaManagementConfig{}
				quotaConfig.API = quotaServer.URL()
				qmc = NewQuotaManagementClient(quotaConfig, lagertest.NewTestLogger("Quota"), nil)

				quotaServer.AppendHandlers(
					ghttp.CombineHandlers(
						ghttp.VerifyRequest("GET", "/api/v2.0/orgs/test-org/services/test-service/plan/test-plan"),
						ghttp.RespondWithJSONEncoded(http.StatusOK, struct {
							Quota int `json:"quota"`
						}{Quota: 23}),
					),
				)
			})
			It("calls the server and returns the quota", func() {
				quota, err = qmc.GetQuota("test-org", "test-service", "test-plan")
				Expect(err).ToNot(HaveOccurred())
				Expect(quota).To(Equal(23))
			})
		})
	})
})
