package quota_test

import (
	"autoscaler/api/config"
	"autoscaler/api/quota"
	"autoscaler/helpers"
	"net/http"

	"code.cloudfoundry.org/lager/lagertest"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/ghttp"
)

var _ = Describe("Quota", func() {
	var (
		quotaConfig *config.Config
		qmc         *quota.Client
		quotaSize   int
		err         error
		quotaServer *ghttp.Server
	)
	BeforeEach(func() {
		quotaConfig = &config.Config{Logging: helpers.LoggingConfig{Level: "debug"}}
	})
	Context("GetQuota", func() {
		Context("when not configured", func() {
			BeforeEach(func() {
				qmc = quota.NewClient(quotaConfig, lagertest.NewTestLogger("Quota"))
			})
			It("returns -1", func() {
				quotaSize, err = qmc.GetQuota("test-org", "test-service", "test-plan")
				Expect(err).ToNot(HaveOccurred())
				Expect(quotaSize).To(Equal(-1))
			})
		})
		Context("when configured", func() {
			BeforeEach(func() {
				quotaServer = ghttp.NewServer()
				quotaConfig.QuotaManagement = &config.QuotaManagementConfig{}
				quotaConfig.QuotaManagement.API = quotaServer.URL()
				qmc = quota.NewClient(quotaConfig, lagertest.NewTestLogger("Quota"))
				qmc.SetClient(&http.Client{Transport: helpers.NewTransport(nil)})

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
				quotaSize, err = qmc.GetQuota("test-org", "test-service", "test-plan")
				Expect(err).ToNot(HaveOccurred())
				Expect(quotaSize).To(Equal(23))
			})
		})
	})
})
