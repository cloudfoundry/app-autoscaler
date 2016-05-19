package api

import (
	"acceptance/config"
	"acceptance/template"
	"encoding/json"
	"net/http"
	"strings"

	"github.com/cloudfoundry-incubator/cf-test-helpers/cf"
	"github.com/cloudfoundry-incubator/cf-test-helpers/generator"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	. "github.com/onsi/gomega/gexec"
)

type apiResponse struct {
	Data      []interface{} `json:"data"`
	Timestamp *uint64       `json:"timestamp"`
}

var _ = Describe("AutoScaler API", func() {
	var appName string
	var appGUID string
	var instanceName string

	BeforeEach(func() {
		appName = generator.PrefixedRandomName("autoscaler-APP")
		createApp := cf.Cf("push", appName, "--no-start", "-b", cfg.NodejsBuildpackName, "-m", config.NODE_MEMORY_LIMIT, "-p", config.NODE_APP, "-d", cfg.AppsDomain).Wait(config.DEFAULT_TIMEOUT)
		Expect(createApp).To(Exit(0), "failed creating app")
		// app_helpers.SetBackend(appName)
		guid := cf.Cf("app", appName, "--guid").Wait(config.DEFAULT_TIMEOUT)
		Expect(guid).To(Exit(0))
		appGUID = strings.TrimSpace(string(guid.Out.Contents()))

		instanceName = generator.PrefixedRandomName("scaling-")
		createService := cf.Cf("create-service", cfg.ServiceName, "free", instanceName).Wait(config.DEFAULT_TIMEOUT)
		Expect(createService).To(Exit(0), "failed creating service")

		bindService := cf.Cf("bind-service", appName, instanceName).Wait(config.DEFAULT_TIMEOUT)
		Expect(bindService).To(Exit(0), "failed binding app to service")
	})

	AfterEach(func() {
		if instanceName != "" {
			unbindService := cf.Cf("unbind-service", appName, instanceName).Wait(config.DEFAULT_TIMEOUT)
			Expect(unbindService).To(Exit(0), "failed unbinding app to service")

			deleteService := cf.Cf("delete-service", instanceName, "-f").Wait(config.DEFAULT_TIMEOUT)
			Expect(deleteService).To(Exit(0))
		}

		Expect(cf.Cf("delete", appName, "-f", "-r").Wait(config.CF_PUSH_TIMEOUT)).To(Exit(0))
	})

	Describe("Policy API", func() {
		var policy Policy

		BeforeEach(func() {
			policy = NewPolicy(cfg.APIUrl, appGUID)
		})

		Context("with a valid policy", func() {
			updatePolicy := func(policyTemplateFile string, expectedCode int) {

				policy_text, err := template.GeneratePolicy(policyTemplateFile, nil)
				Expect(err).ToNot(HaveOccurred())

				statusCode, err := policy.UpdateWithText(string(policy_text))
				Expect(err).ToNot(HaveOccurred())
				Expect(statusCode).To(Equal(expectedCode))
			}

			BeforeEach(func() {
				updatePolicy("../assets/file/policy/all.json.template", http.StatusCreated)
			})

			AfterEach(func() {
				statusCode, err := policy.Delete()
				Expect(err).ToNot(HaveOccurred())
				Expect(statusCode).To(Equal(http.StatusOK))
			})

			It("retrieves the policy", func() {
				statusCode, err := policy.Get()
				Expect(err).ToNot(HaveOccurred())
				Expect(statusCode).To(Equal(http.StatusOK))
			})

			It("updates the policy", func() {
				By("dynamic")
				updatePolicy("../assets/file/policy/dynamic.json.template", http.StatusOK)

				By("recurring")
				updatePolicy("../assets/file/policy/recurringSchedule.json.template", http.StatusOK)

				By("specific date")
				updatePolicy("../assets/file/policy/specificDate.json.template", http.StatusOK)
			})
		})

		Context("with an invalid policy", func() {
			It("fails", func() {
				statusCode, err := policy.UpdateWithText("garbage")
				Expect(err).ToNot(HaveOccurred())
				Expect(statusCode).To(Equal(http.StatusBadRequest))
			})
		})
	})

	Describe("Metric API", func() {
		It("retrieves the scaling history", func() {
			statusCode, out, err := GetMetrics(cfg.APIUrl, appGUID)
			Expect(err).ToNot(HaveOccurred())
			Expect(statusCode).To(Equal(http.StatusOK))

			var m apiResponse
			err = json.Unmarshal(out, &m)
			Expect(err).ToNot(HaveOccurred())
			Expect(m.Data).ToNot(BeNil())
			Expect(m.Data).To(HaveLen(0))
			Expect(m.Timestamp).ToNot(BeNil())
		})
	})

	Describe("History API", func() {
		It("retrieves the scaling history", func() {
			statusCode, out, err := GetHistory(cfg.APIUrl, appGUID)
			Expect(err).ToNot(HaveOccurred())
			Expect(statusCode).To(Equal(http.StatusOK))

			var h apiResponse
			err = json.Unmarshal(out, &h)
			Expect(err).ToNot(HaveOccurred())
			Expect(h.Data).ToNot(BeNil())
			Expect(h.Data).To(HaveLen(0))
			Expect(h.Timestamp).ToNot(BeNil())
		})
	})
})
