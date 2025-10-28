package broker_test

import (
	"acceptance/config"
	"acceptance/helpers"
	"encoding/json"
	"fmt"
	"net/url"
	"os"
	"slices"
	"strings"

	"github.com/cloudfoundry/cf-test-helpers/v2/generator"
	"github.com/cloudfoundry/cf-test-helpers/v2/workflowhelpers"

	"github.com/onsi/gomega/gbytes"

	"github.com/cloudfoundry/cf-test-helpers/v2/cf"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	. "github.com/onsi/gomega/gexec"
)

type serviceInstance string

func createService(onPlan string) serviceInstance {
	instanceName := generator.PrefixedRandomName(cfg.Prefix, cfg.InstancePrefix)
	helpers.FailOnError(helpers.CreateServiceWithPlan(cfg, onPlan, instanceName))
	return serviceInstance(instanceName)
}

func createServiceWithParameters(onPlan string, parameters string) serviceInstance {
	instanceName := generator.PrefixedRandomName(cfg.Prefix, cfg.InstancePrefix)
	helpers.FailOnError(helpers.CreateServiceWithPlanAndParameters(cfg, onPlan, parameters, instanceName))
	return serviceInstance(instanceName)
}

func (s serviceInstance) updatePlan(toPlan string) {
	updateService := s.updatePlanRaw(toPlan)
	ExpectWithOffset(1, updateService).To(Exit(0), "failed updating service")
	Expect(strings.Contains(string(updateService.Out.Contents()), "The service does not support changing plans.")).To(BeFalse())
}

func (s serviceInstance) updatePlanRaw(toPlan string) *Session {
	By(fmt.Sprintf("update service plan to %s", toPlan))
	updateService := cf.Cf("update-service", string(s), "-p", toPlan).Wait(cfg.DefaultTimeoutDuration())
	return updateService
}

func (s serviceInstance) unbind(fromApp string) {
	unbindService := cf.Cf("unbind-service", fromApp, s.name()).Wait(cfg.DefaultTimeoutDuration())
	Expect(unbindService).To(Exit(0), "failed unbinding service instance %s from app %s", s.name(), fromApp)
}

func (s serviceInstance) delete() {
	deleteService := cf.Cf("delete-service", string(s), "-f").Wait(cfg.DefaultTimeoutDuration())
	Expect(deleteService).To(Exit(0), "failed deleting service instance %s", s.name())
}

func (s serviceInstance) name() string {
	return string(s)
}

var _ = Describe("AutoScaler Service Broker", func() {
	var appName string

	BeforeEach(func() {
		appName = helpers.CreateTestApp(cfg, "broker-test", 1)
	})

	AfterEach(func() {
		if os.Getenv("SKIP_TEARDOWN") == "true" {
			fmt.Println("Skipping Teardown...")
		} else {
			Eventually(cf.Cf("app", appName, "--guid"), cfg.DefaultTimeoutDuration()).Should(Exit())
			Eventually(cf.Cf("logs", appName, "--recent"), cfg.DefaultTimeoutDuration()).Should(Exit())
			Expect(cf.Cf("delete", appName, "-f", "-r").Wait(cfg.CfPushTimeoutDuration())).To(Exit(0))
		}
	})

	Context("performs lifecycle operations", func() {

		var instance serviceInstance

		BeforeEach(func() {
			instance = createService(cfg.ServicePlan)
		})

		It("fails to bind with invalid policies", func() {
			bindService := cf.Cf("bind-service", appName, instance.name(), "-c", "../assets/file/policy/invalid.json").Wait(cfg.DefaultTimeoutDuration())
			Expect(bindService).To(Exit(1))
			combinedBuffer := gbytes.BufferWithBytes(append(bindService.Out.Contents(), bindService.Err.Contents()...))
			Eventually(string(combinedBuffer.Contents())).Should(ContainSubstring(`[{"context":"(root).scaling_rules.1.adjustment","description":"Does not match pattern '^[-+][1-9]+[0-9]*%?$'"}]`))
		})

		It("binds&unbinds with policy", func() {
			policyFile := "../assets/file/policy/all.json"
			policy, err := os.ReadFile(policyFile)
			Expect(err).NotTo(HaveOccurred())

			err = helpers.BindServiceToAppWithPolicy(cfg, appName, instance.name(), policyFile)
			Expect(err).NotTo(HaveOccurred())

			bindingParameters := helpers.GetServiceCredentialBindingParameters(cfg, instance.name(), appName)
			Expect(bindingParameters).Should(MatchJSON(policy))

			instance.unbind(appName)
		})

		It("binds&unbinds with policy having credential-type as x509", func() {
			policyFile := "../assets/file/policy/policy-with-credential-type.json"
			_, err := os.ReadFile(policyFile)
			Expect(err).NotTo(HaveOccurred())

			err = helpers.BindServiceToAppWithPolicy(cfg, appName, instance.name(), policyFile)
			Expect(err).NotTo(HaveOccurred())

			By("checking broker bind response does not have username/password/url but mtls_url only ")
			appEnvCmd := cf.Cf("env", appName).Wait(cfg.DefaultTimeoutDuration())
			Expect(appEnvCmd).To(Exit(0), "failed getting app env")

			appEnvCmdOutput := appEnvCmd.Out.Contents()
			Expect(appEnvCmdOutput).NotTo(ContainSubstring("username"))
			Expect(appEnvCmdOutput).NotTo(ContainSubstring("password"))
			Expect(appEnvCmdOutput).NotTo(ContainSubstring("\"url\": \"https://"))
			Expect(appEnvCmdOutput).To(ContainSubstring("\"mtls_url\": \"https://"))

			instance.unbind(appName)
		})

		It("bind&unbinds without policy", func() {
			helpers.BindServiceToApp(cfg, appName, instance.name())
			bindingParameters := helpers.GetServiceCredentialBindingParameters(cfg, instance.name(), appName)
			Expect(bindingParameters).Should(MatchJSON("{}"))
			instance.unbind(appName)
		})

		It("binds&unbinds with configurations and policy", func() {
			policyFile := "../assets/file/policy/policy-with-configuration.json"
			policyWithConfig, err := os.ReadFile(policyFile)
			Expect(err).NotTo(HaveOccurred())

			err = helpers.BindServiceToAppWithPolicy(cfg, appName, instance.name(), policyFile)
			Expect(err).NotTo(HaveOccurred())
			By("checking broker bind parameter response should have policy and configuration")
			bindingParameters := helpers.GetServiceCredentialBindingParameters(cfg, instance.name(), appName)
			Expect(bindingParameters).Should(MatchJSON(policyWithConfig))

			instance.unbind(appName)
		})

		AfterEach(func() {
			instance.delete()
		})
	})

	Describe("allows setting default policies", func() {
		var instance serviceInstance
		var defaultPolicy []byte
		var policy []byte

		BeforeEach(func() {
			instance = createServiceWithParameters(cfg.ServicePlan, "../assets/file/policy/default_policy.json")
			Expect(instance).NotTo(BeEmpty())
			var err error
			defaultPolicy, err = os.ReadFile("../assets/file/policy/default_policy.json")
			Expect(err).NotTo(HaveOccurred())

			var serviceParameters = struct {
				DefaultPolicy interface{} `json:"default_policy"`
			}{}

			err = json.Unmarshal(defaultPolicy, &serviceParameters)
			Expect(err).NotTo(HaveOccurred())

			policy, err = json.Marshal(serviceParameters.DefaultPolicy)
			Expect(err).NotTo(HaveOccurred())
		})

		It("allows retrieving the default policy using the Cloud Controller", func() {
			instanceParameters := helpers.GetServiceInstanceParameters(cfg, instance.name())
			Expect(instanceParameters).To(MatchJSON(defaultPolicy))
		})

		It("sets the default policy if no policy is set during binding and allows retrieving the policy via the binding parameters", func() {
			helpers.BindServiceToApp(cfg, appName, instance.name())
			By("checking broker bind parameter response should have default policy")
			bindingParameters := helpers.GetServiceCredentialBindingParameters(cfg, instance.name(), appName)
			Expect(bindingParameters).Should(MatchJSON(policy))

			unbindService := cf.Cf("unbind-service", appName, instance.name()).Wait(cfg.DefaultTimeoutDuration())
			Expect(unbindService).To(Exit(0), "failed unbinding service from app")

		})

		AfterEach(func() {
			if os.Getenv("SKIP_TEARDOWN") == "true" {
				fmt.Println("Skipping Teardown...")
			} else {
				instance.delete()
			}
		})
	})

	Describe("allows updating service plans", func() {
		var instance serviceInstance
		It("should update a service instance from one plan to another plan", func() {
			servicePlans := GetServicePlans(cfg)
			source, target, err := servicePlans.getSourceAndTargetForPlanUpdate()
			Expect(err).NotTo(HaveOccurred(), "failed getting source and target service plans")
			instance = createService(source.Name)
			instance.updatePlan(target.Name)
		})

		AfterEach(func() {
			instance.delete()
		})
	})
})

type ServicePlans []ServicePlan

type (
	ServicePlan struct {
		Guid          string        `json:"guid"`
		Name          string        `json:"name"`
		BrokerCatalog BrokerCatalog `json:"broker_catalog"`
	}
	BrokerCatalog struct {
		Id       string   `json:"id"`
		Features Features `json:"features"`
	}
	Features struct {
		PlanUpdateable bool `json:"plan_updateable"`
	}
)

func (p ServicePlans) length() int { return len(p) }

func GetServicePlans(cfg *config.Config) ServicePlans {
	values := url.Values{
		"per_page":               []string{"5000"},
		"service_broker_names":   []string{cfg.ServiceBroker},
		"service_offering_names": []string{cfg.ServiceName},
	}
	servicePlansURL := &url.URL{Path: "/v3/service_plans", RawQuery: values.Encode()}

	var result ServicePlans

	// This should also work as normal user - for some reason, if we use the normal user
	// plan_updateable is returned as integer instead of boolean
	workflowhelpers.AsUser(setup.AdminUserContext(), cfg.DefaultTimeoutDuration(), func() {
		serviceCmd := cf.CfSilent("curl", "-f", servicePlansURL.String()).Wait(cfg.DefaultTimeoutDuration())
		Expect(serviceCmd).To(Exit(0), "failed getting service plans")

		plansResult := &struct{ Resources []ServicePlan }{}
		err := json.Unmarshal(serviceCmd.Out.Contents(), plansResult)
		Expect(err).NotTo(HaveOccurred())

		result = plansResult.Resources
	})

	return result
}

func (p ServicePlan) isUpdatable() bool {
	return p.BrokerCatalog.Features.PlanUpdateable
}

func (p ServicePlans) getSourceAndTargetForPlanUpdate() (source, target ServicePlan, err error) {
	if p.length() < 2 {
		return ServicePlan{}, ServicePlan{}, fmt.Errorf("two service plans needed, only one plan available")
	}
	updatablePlanIndex := slices.IndexFunc(p, func(n ServicePlan) bool {
		return n.isUpdatable()
	})
	if updatablePlanIndex == -1 {
		return ServicePlan{}, ServicePlan{}, fmt.Errorf("no updatable plan found")
	}
	source = p[updatablePlanIndex]
	target = p[(updatablePlanIndex+1)%p.length()] // simply update to any other plan
	return source, target, nil
}
