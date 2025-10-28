package helpers

import (
	"acceptance/config"
	"bytes"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	url2 "net/url"
	"strings"
	"time"

	"github.com/cloudfoundry/cf-test-helpers/v2/workflowhelpers"

	"github.com/cloudfoundry/cf-test-helpers/v2/generator"

	"github.com/cloudfoundry/cf-test-helpers/v2/cf"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	. "github.com/onsi/gomega/gexec"
)

const (
	DaysOfMonth Days = "days_of_month"
	DaysOfWeek  Days = "days_of_week"

	TestBreachDurationSeconds = 60
	TestCoolDownSeconds       = 60

	PolicyPath = "/v1/apps/{appId}/policy"
)

type Days string

type BindingConfig struct {
	Configuration Configuration `json:"configuration"`
	ScalingPolicy
}
type Configuration struct {
	CustomMetrics CustomMetricsConfig `json:"custom_metrics"`
}

type CustomMetricsConfig struct {
	MetricSubmissionStrategy MetricsSubmissionStrategy `json:"metric_submission_strategy"`
}

type MetricsSubmissionStrategy struct {
	AllowFrom string `json:"allow_from"`
}

func (b *BindingConfig) GetCustomMetricsStrategy() string {
	return b.Configuration.CustomMetrics.MetricSubmissionStrategy.AllowFrom
}

func (b *BindingConfig) SetCustomMetricsStrategy(allowFrom string) {
	b.Configuration.CustomMetrics.MetricSubmissionStrategy.AllowFrom = allowFrom
}

type ScalingPolicy struct {
	InstanceMin    int               `json:"instance_min_count"`
	InstanceMax    int               `json:"instance_max_count"`
	ScalingRules   []*ScalingRule    `json:"scaling_rules,omitempty"`
	Schedules      *ScalingSchedules `json:"schedules,omitempty"`
	CredentialType *string           `json:"credential-type,omitempty"`
}

type ScalingPolicyWithExtraFields struct {
	IsAdmin      bool                           `json:"is_admin"`
	IsSSO        bool                           `json:"is_sso"`
	Role         string                         `json:"role"`
	InstanceMin  int                            `json:"instance_min_count"`
	InstanceMax  int                            `json:"instance_max_count"`
	ScalingRules []*ScalingRulesWithExtraFields `json:"scaling_rules,omitempty"`
	Schedules    *ScalingSchedules              `json:"schedules,omitempty"`
}

type ScalingRule struct {
	MetricType            string `json:"metric_type"`
	BreachDurationSeconds int    `json:"breach_duration_secs"`
	Threshold             int64  `json:"threshold"`
	Operator              string `json:"operator"`
	CoolDownSeconds       int    `json:"cool_down_secs"`
	Adjustment            string `json:"adjustment"`
}

type ScalingRulesWithExtraFields struct {
	StatsWindowSeconds int `json:"stats_window_secs"`
	ScalingRule
}

type ScalingSchedules struct {
	Timezone              string                  `json:"timezone,omitempty"`
	RecurringSchedules    []*RecurringSchedule    `json:"recurring_schedule,omitempty"`
	SpecificDateSchedules []*SpecificDateSchedule `json:"specific_date,omitempty"`
}

type RecurringSchedule struct {
	StartTime             string `json:"start_time"`
	EndTime               string `json:"end_time"`
	DaysOfWeek            []int  `json:"days_of_week,omitempty"`
	DaysOfMonth           []int  `json:"days_of_month,omitempty"`
	ScheduledInstanceMin  int    `json:"instance_min_count"`
	ScheduledInstanceMax  int    `json:"instance_max_count"`
	ScheduledInstanceInit int    `json:"initial_min_instance_count"`
}

type SpecificDateSchedule struct {
	StartDateTime         string `json:"start_date_time"`
	EndDateTime           string `json:"end_date_time"`
	ScheduledInstanceMin  int    `json:"instance_min_count"`
	ScheduledInstanceMax  int    `json:"instance_max_count"`
	ScheduledInstanceInit int    `json:"initial_min_instance_count"`
}

func OauthToken(cfg *config.Config) string {
	cmd := cf.CfSilent("oauth-token")
	Expect(cmd.Wait(cfg.DefaultTimeoutDuration())).To(Exit(0))
	return strings.TrimSpace(string(cmd.Out.Contents()))
}

func EnableServiceAccess(setup *workflowhelpers.ReproducibleTestSuiteSetup, cfg *config.Config, orgName string) {
	if cfg.ShouldEnableServiceAccess() {
		workflowhelpers.AsUser(setup.AdminUserContext(), cfg.DefaultTimeoutDuration(), func() {
			if orgName == "" {
				Fail(fmt.Sprintf("Org must not be an empty string. Using broker:%s, serviceName:%s", cfg.ServiceBroker, cfg.ServiceName))
			}
			enableServiceAccess := cf.Cf("enable-service-access", cfg.ServiceName, "-b", cfg.ServiceBroker, "-o", orgName).Wait(cfg.DefaultTimeoutDuration())
			Expect(enableServiceAccess).To(Exit(0), fmt.Sprintf("Failed to enable service %s for org %s: %s", cfg.ServiceName, orgName, enableServiceAccess.Err.Contents()))
		})
	}
}

func DisableServiceAccess(cfg *config.Config, setup *workflowhelpers.ReproducibleTestSuiteSetup) {
	if cfg.ShouldEnableServiceAccess() {
		workflowhelpers.AsUser(setup.AdminUserContext(), cfg.DefaultTimeoutDuration(), func() {
			orgName := setup.GetOrganizationName()
			disableServiceAccess := cf.Cf("disable-service-access", cfg.ServiceName, "-b", cfg.ServiceBroker, "-o", orgName).Wait(cfg.DefaultTimeoutDuration())
			Expect(disableServiceAccess).To(Exit(0), fmt.Sprintf("Failed to disable service %s for org %s", cfg.ServiceName, orgName))
		})
	}
}

func CheckServiceExists(cfg *config.Config, spaceName, serviceName string) {
	spaceCmd := cf.Cf("space", spaceName, "--guid").Wait(cfg.DefaultTimeoutDuration())
	Expect(spaceCmd).To(Exit(0), fmt.Sprintf("Space, %s, does not exist", spaceName))
	spaceGuid := strings.TrimSpace(strings.Trim(string(spaceCmd.Out.Contents()), "\n"))

	serviceCmd := cf.CfSilent("curl", "-f", ServicePlansUrl(cfg, spaceGuid)).Wait(cfg.DefaultTimeoutDuration())
	if serviceCmd.ExitCode() != 0 {
		Fail(fmt.Sprintf("Failed get broker information for serviceName=%s spaceName=%s", cfg.ServiceName, spaceName))
	}

	var services = struct {
		Included struct {
			ServiceOfferings []struct{ Name string } `json:"service_offerings"`
		}
	}{}
	contents := serviceCmd.Out.Contents()
	err := json.Unmarshal(contents, &services)
	if err != nil {
		AbortSuite(fmt.Sprintf("Failed to parse service plan json: %s\n\n'%s'", err.Error(), string(contents)))
	}
	GinkgoWriter.Printf("\nFound services: %s\n", services.Included.ServiceOfferings)
	for _, service := range services.Included.ServiceOfferings {
		if service.Name == serviceName {
			return
		}
	}

	cf.Cf("marketplace", "-e", cfg.ServiceName).Wait(cfg.DefaultTimeoutDuration())
	Fail(fmt.Sprintf("Could not find service %s in space %s", serviceName, spaceName))
}

func ServicePlansUrl(cfg *config.Config, spaceGuid string) string {
	values := url2.Values{
		"available": []string{"true"},
		"fields[service_offering.service_broker]": []string{"name,guid"},
		"include":                []string{"service_offering"},
		"per_page":               []string{"5000"},
		"service_broker_names":   []string{cfg.ServiceBroker},
		"service_offering_names": []string{cfg.ServiceName},
		"space_guids":            []string{spaceGuid},
	}
	url := &url2.URL{Path: "/v3/service_plans", RawQuery: values.Encode()}
	return url.String()
}

func GenerateBindingsWithScalingPolicy(allowFrom string, instanceMin, instanceMax int, metricName string, scaleInThreshold, scaleOutThreshold int64) string {
	bindingConfig := &BindingConfig{
		Configuration: Configuration{CustomMetrics: CustomMetricsConfig{
			MetricSubmissionStrategy: MetricsSubmissionStrategy{AllowFrom: allowFrom},
		}},
		ScalingPolicy: buildScaleOutScaleInPolicy(instanceMin, instanceMax, metricName, scaleInThreshold, scaleOutThreshold),
	}
	marshalledBinding, err := MarshalWithoutHTMLEscape(bindingConfig)
	Expect(err).NotTo(HaveOccurred())
	return string(marshalledBinding)
}

func GenerateDynamicScaleOutPolicy(instanceMin, instanceMax int, metricName string, threshold int64) string {
	policy := buildScalingPolicy(instanceMin, instanceMax, metricName, threshold)
	marshaled, err := MarshalWithoutHTMLEscape(policy)
	Expect(err).NotTo(HaveOccurred())

	return string(marshaled)
}

func buildScalingPolicy(instanceMin int, instanceMax int, metricName string, threshold int64) ScalingPolicy {
	scalingOutRule := ScalingRule{
		MetricType:            metricName,
		BreachDurationSeconds: TestBreachDurationSeconds,
		Threshold:             threshold,
		Operator:              ">=",
		CoolDownSeconds:       TestCoolDownSeconds,
		Adjustment:            "+1",
	}
	policy := ScalingPolicy{
		InstanceMin:  instanceMin,
		InstanceMax:  instanceMax,
		ScalingRules: []*ScalingRule{&scalingOutRule},
	}
	return policy
}

func GenerateDynamicScaleOutPolicyWithExtraFields(instanceMin, instanceMax int, metricName string, threshold int64) (string, string) {
	scalingOutRule := ScalingRule{
		MetricType:            metricName,
		BreachDurationSeconds: TestBreachDurationSeconds,
		Threshold:             threshold,
		Operator:              ">=",
		CoolDownSeconds:       TestCoolDownSeconds,
		Adjustment:            "+1",
	}

	scalingOutRuleWithExtraFields := ScalingRulesWithExtraFields{
		StatsWindowSeconds: 666,
		ScalingRule:        scalingOutRule,
	}

	policy := ScalingPolicy{
		InstanceMin:  instanceMin,
		InstanceMax:  instanceMax,
		ScalingRules: []*ScalingRule{&scalingOutRule},
	}

	policyWithExtraFields := ScalingPolicyWithExtraFields{
		IsAdmin:      true,
		IsSSO:        true,
		Role:         "admin",
		InstanceMin:  instanceMin,
		InstanceMax:  instanceMax,
		ScalingRules: []*ScalingRulesWithExtraFields{&scalingOutRuleWithExtraFields},
	}

	validBytes, err := MarshalWithoutHTMLEscape(policy)
	Expect(err).NotTo(HaveOccurred())

	extraBytes, err := MarshalWithoutHTMLEscape(policyWithExtraFields)
	Expect(err).NotTo(HaveOccurred())

	Expect(extraBytes).NotTo(MatchJSON(validBytes))

	return string(extraBytes), string(validBytes)
}

func GenerateDynamicScaleOutAndInPolicy(instanceMin, instanceMax int, metricName string, scaleInWhenBelowThreshold int64, scaleOutWhenGreaterOrEqualThreshold int64) string {
	policy := buildScaleOutScaleInPolicy(instanceMin, instanceMax, metricName, scaleInWhenBelowThreshold, scaleOutWhenGreaterOrEqualThreshold)
	marshaled, err := MarshalWithoutHTMLEscape(policy)
	Expect(err).NotTo(HaveOccurred())

	return string(marshaled)
}

func GeneratePolicyWithCredentialType(instanceMin, instanceMax int, metricName string, scaleInWhenBelowThreshold int64, scaleOutWhenGreaterOrEqualThreshold int64, credentialType *string) string {
	policyWithCredentialType := buildScaleOutScaleInPolicy(instanceMin, instanceMax, metricName, scaleInWhenBelowThreshold, scaleOutWhenGreaterOrEqualThreshold)
	policyWithCredentialType.CredentialType = credentialType
	marshaled, err := MarshalWithoutHTMLEscape(policyWithCredentialType)
	Expect(err).NotTo(HaveOccurred())

	return string(marshaled)
}

func buildScaleOutScaleInPolicy(instanceMin int, instanceMax int, metricName string, scaleInWhenBelowThreshold int64, scaleOutWhenGreaterOrEqualThreshold int64) ScalingPolicy {
	scalingOutRule := ScalingRule{
		MetricType:            metricName,
		BreachDurationSeconds: TestBreachDurationSeconds,
		Threshold:             scaleOutWhenGreaterOrEqualThreshold,
		Operator:              ">=",
		CoolDownSeconds:       TestCoolDownSeconds,
		Adjustment:            "+1",
	}
	scalingInRule := ScalingRule{
		MetricType:            metricName,
		BreachDurationSeconds: TestBreachDurationSeconds,
		Threshold:             scaleInWhenBelowThreshold,
		Operator:              "<",
		CoolDownSeconds:       TestCoolDownSeconds,
		Adjustment:            "-1",
	}
	policy := ScalingPolicy{
		InstanceMin:  instanceMin,
		InstanceMax:  instanceMax,
		ScalingRules: []*ScalingRule{&scalingOutRule, &scalingInRule},
	}
	return policy
}

// GenerateDynamicScaleInPolicyBetween creates a scaling policy that scales down from 2 instances to 1, if the metric value is in a range of [upper, lower].
// Example how the scaling rules must be defined to achieve a "scale down if value is in range"-behaviour:
//
//	val <  10  ➡  +1  ➡ don't do anything if below 10 because there are already 2 instances
//	val >  30  ➡  +1  ➡ don't do anything if above 30 because there are already 2 instances
//	val <= 30  ➡  -1  ➡ scale down if less than or equal 30
func GenerateDynamicScaleInPolicyBetween(metricName string, scaleInLowerThreshold int64, scaleInUpperThreshold int64) string {
	noDownscalingWhenBelowLower := ScalingRule{
		MetricType:            metricName,
		BreachDurationSeconds: TestBreachDurationSeconds,
		Threshold:             scaleInLowerThreshold,
		Operator:              "<",
		CoolDownSeconds:       TestCoolDownSeconds,
		Adjustment:            "+1",
	}

	noDownscalingWhenAboveUpper := ScalingRule{
		MetricType:            metricName,
		BreachDurationSeconds: TestBreachDurationSeconds,
		Threshold:             scaleInUpperThreshold,
		Operator:              ">",
		CoolDownSeconds:       TestCoolDownSeconds,
		Adjustment:            "+1",
	}

	downscalingWhenBelowOrEqualUpper := ScalingRule{
		MetricType:            metricName,
		BreachDurationSeconds: TestBreachDurationSeconds,
		Threshold:             scaleInUpperThreshold,
		Operator:              "<=",
		CoolDownSeconds:       TestCoolDownSeconds,
		Adjustment:            "-1",
	}

	policy := ScalingPolicy{
		InstanceMin:  1,
		InstanceMax:  2,
		ScalingRules: []*ScalingRule{&noDownscalingWhenBelowLower, &noDownscalingWhenAboveUpper, &downscalingWhenBelowOrEqualUpper},
	}

	marshaled, err := MarshalWithoutHTMLEscape(policy)
	Expect(err).NotTo(HaveOccurred())

	return string(marshaled)
}

func GenerateSpecificDateSchedulePolicy(startDateTime, endDateTime time.Time, scheduledInstanceMin, scheduledInstanceMax, scheduledInstanceInit int) string {
	scalingInRule := ScalingRule{
		MetricType:            "cpu",
		BreachDurationSeconds: TestBreachDurationSeconds,
		Threshold:             80,
		Operator:              "<",
		CoolDownSeconds:       TestCoolDownSeconds,
		Adjustment:            "-1",
	}
	specificDateSchedule := SpecificDateSchedule{
		StartDateTime:         startDateTime.Round(1 * time.Minute).Format("2006-01-02T15:04"),
		EndDateTime:           endDateTime.Round(1 * time.Minute).Format("2006-01-02T15:04"),
		ScheduledInstanceMin:  scheduledInstanceMin,
		ScheduledInstanceMax:  scheduledInstanceMax,
		ScheduledInstanceInit: scheduledInstanceInit,
	}
	policy := ScalingPolicy{
		InstanceMin:  1,
		InstanceMax:  4,
		ScalingRules: []*ScalingRule{&scalingInRule},
		Schedules: &ScalingSchedules{
			Timezone:              "UTC",
			SpecificDateSchedules: []*SpecificDateSchedule{&specificDateSchedule},
		},
	}

	marshaled, err := MarshalWithoutHTMLEscape(policy)
	Expect(err).NotTo(HaveOccurred())

	return strings.TrimSpace(string(marshaled))
}

func GenerateDynamicAndRecurringSchedulePolicy(instanceMin, instanceMax int, threshold int64,
	timezone string, startTime, endTime time.Time, daysOfMonthOrWeek Days,
	scheduledInstanceMin, scheduledInstanceMax, scheduledInstanceInit int) string {
	scalingInRule := ScalingRule{
		MetricType:            "cpu",
		BreachDurationSeconds: TestBreachDurationSeconds,
		Threshold:             threshold,
		Operator:              "<",
		CoolDownSeconds:       TestCoolDownSeconds,
		Adjustment:            "-1",
	}

	recurringSchedule := RecurringSchedule{
		StartTime:             startTime.Format("15:04"),
		EndTime:               endTime.Format("15:04"),
		ScheduledInstanceMin:  scheduledInstanceMin,
		ScheduledInstanceMax:  scheduledInstanceMax,
		ScheduledInstanceInit: scheduledInstanceInit,
	}

	if daysOfMonthOrWeek == DaysOfMonth {
		day := startTime.Day()
		recurringSchedule.DaysOfMonth = []int{day}
	} else {
		day := int(startTime.Weekday())
		if day == 0 {
			day = 7
		}
		recurringSchedule.DaysOfWeek = []int{day}
	}

	policy := ScalingPolicy{
		InstanceMin:  instanceMin,
		InstanceMax:  instanceMax,
		ScalingRules: []*ScalingRule{&scalingInRule},
		Schedules: &ScalingSchedules{
			Timezone:           timezone,
			RecurringSchedules: []*RecurringSchedule{&recurringSchedule},
		},
	}

	marshaled, err := MarshalWithoutHTMLEscape(policy)
	Expect(err).NotTo(HaveOccurred())

	return string(marshaled)
}

func RunningInstances(appGUID string, timeout time.Duration) (int, error) {
	GinkgoHelper()
	defer GinkgoRecover()
	var cmd *Session
	getAppProcesses := func() error {
		var err error
		cmd = cf.CfSilent("curl", fmt.Sprintf("/v3/apps/%s/processes/web", appGUID)).Wait(timeout)
		if cmd.ExitCode() != 0 {
			err = fmt.Errorf("failed to curl cloud controller api for app: %s  %s", appGUID, string(cmd.Err.Contents()))
		}
		return err
	}

	err := Retry(defaultRetryAttempt, defaultRetryAfter, getAppProcesses)
	if err != nil {
		return 0, err
	}

	var process = struct {
		Instances int `json:"instances"`
	}{}

	err = json.Unmarshal(cmd.Out.Contents(), &process)
	Expect(err).ToNot(HaveOccurred())
	webInstances := process.Instances
	GinkgoWriter.Printf("\nFound %d app instances for app %s \n", webInstances, appGUID)
	return webInstances, nil
}

func WaitForNInstancesRunning(appGUID string, instances int, timeout time.Duration, optionalDescription ...interface{}) {
	GinkgoHelper()
	By(fmt.Sprintf("Waiting for %d instances of app: %s", instances, appGUID))
	Eventually(getAppInstances(appGUID, 8*time.Second)).
		WithTimeout(timeout).
		WithPolling(10*time.Second).
		Should(Equal(instances), optionalDescription...)
}

func getAppInstances(appGUID string, timeout time.Duration) func() int {
	return func() int {
		instances, err := RunningInstances(appGUID, timeout)
		if err != nil {
			fmt.Println("error while computing running instances count: %w", err)
		}
		return instances
	}
}

func MarshalWithoutHTMLEscape(v interface{}) ([]byte, error) {
	var b bytes.Buffer
	enc := json.NewEncoder(&b)
	enc.SetEscapeHTML(false)
	err := enc.Encode(v)
	if err != nil {
		return nil, err
	}

	return b.Bytes(), nil
}

func CreatePolicy(cfg *config.Config, appName, appGUID, policy string) string {
	GinkgoHelper()
	instanceName, _ := createPolicy(cfg, appName, policy)
	return instanceName
}

func CreatePolicyWithErr(cfg *config.Config, appName, policy string) (string, error) {
	return createPolicy(cfg, appName, policy)
}

func createPolicy(cfg *config.Config, appName, policy string) (string, error) {
	GinkgoHelper()
	instanceName := generator.PrefixedRandomName(cfg.Prefix, cfg.InstancePrefix)
	err := Retry(defaultRetryAttempt, defaultRetryAfter, func() error { return CreateServiceWithPlan(cfg, cfg.ServicePlan, instanceName) })
	if err != nil {
		return instanceName, err
	}
	err = Retry(defaultRetryAttempt, defaultRetryAfter, func() error { return BindServiceToAppWithPolicy(cfg, appName, instanceName, policy) })
	return instanceName, err
}

func BindServiceToApp(cfg *config.Config, appName string, instanceName string) {
	err := BindServiceToAppWithPolicy(cfg, appName, instanceName, "")
	Expect(err).ToNot(HaveOccurred())
}

func BindServiceToAppWithPolicy(cfg *config.Config, appName string, instanceName string, policy string) error {
	var err error

	args := []string{"bind-service", appName, instanceName}
	if policy != "" {
		args = append(args, "-c", policy)
	}
	bindService := cf.Cf(args...).Wait(cfg.DefaultTimeoutDuration())

	if bindService.ExitCode() != 0 {
		err = fmt.Errorf("failed binding service %s to app %s. \n Command Error: %s %s",
			instanceName, appName, bindService.Buffer().Contents(), bindService.Err.Contents())
	}

	return err
}

func UnbindServiceFromApp(cfg *config.Config, appName string, instanceName string) {
	unbindService := cf.Cf("unbind-service", appName, instanceName).Wait(cfg.DefaultTimeoutDuration())
	Expect(unbindService).To(Exit(0), fmt.Sprintf("Failed to unbind service %s from app %s \n CLI Output:\n %s %s", instanceName, appName, unbindService.Buffer().Contents(), unbindService.Err.Contents()))
}

func CreateService(cfg *config.Config) string {
	instanceName := generator.PrefixedRandomName(cfg.Prefix, cfg.InstancePrefix)
	FailOnError(CreateServiceWithPlan(cfg, cfg.ServicePlan, instanceName))
	return instanceName
}

func CreateServiceWithPlan(cfg *config.Config, servicePlan string, instanceName string) error {
	return CreateServiceWithPlanAndParameters(cfg, servicePlan, "", instanceName)
}

func CreateServiceWithPlanAndParameters(cfg *config.Config, servicePlan string, defaultPolicy string, instanceName string) (err error) {
	cfCommand := []string{"create-service", cfg.ServiceName, servicePlan, instanceName, "-b", cfg.ServiceBroker}
	if defaultPolicy != "" {
		cfCommand = append(cfCommand, "-c", defaultPolicy)
	}
	createService := cf.Cf(cfCommand...).Wait(cfg.DefaultTimeoutDuration())

	if createService.ExitCode() != 0 {
		err = fmt.Errorf("Failed to create service instance %s on service %s \n Command Error: %s %s",
			instanceName, cfg.ServiceName, createService.Buffer().Contents(), createService.Err.Contents())
	}

	return err
}

func GetServiceInstanceGuid(cfg *config.Config, instanceName string) string {
	guid := cf.Cf("service", instanceName, "--guid").Wait(cfg.DefaultTimeoutDuration())
	Expect(guid).To(Exit(0), fmt.Sprintf("Failed to find service instance guid for service instance: %s \n CLI Output:\n %s", instanceName, guid.Out.Contents()))
	return strings.TrimSpace(string(guid.Out.Contents()))
}

func GetServiceInstanceParameters(cfg *config.Config, instanceName string) string {
	instanceGuid := GetServiceInstanceGuid(cfg, instanceName)

	cmd := cf.CfSilent("curl", fmt.Sprintf("/v3/service_instances/%s/parameters", instanceGuid)).Wait(cfg.DefaultTimeoutDuration())
	Expect(cmd).To(Exit(0))
	return strings.TrimSpace(string(cmd.Out.Contents()))
}

func GetServiceCredentialBindingGuid(cfg *config.Config, instanceGuid string, appName string) string {
	appGuid, err := GetAppGuid(cfg, appName)
	Expect(err).NotTo(HaveOccurred())
	guid := cf.CfSilent("curl", fmt.Sprintf("/v3/service_credential_bindings?service_instance_guids=%s&app_guids=%s", instanceGuid, appGuid)).Wait(cfg.DefaultTimeoutDuration())

	Expect(guid).To(Exit(0), fmt.Sprintf("Failed to find service credential binding guid for service instance guid : %s and app name %s \n CLI Output:\n %s", instanceGuid, appName, guid.Out.Contents()))

	contents := guid.Out.Contents()

	type ServiceCredentialBinding struct {
		GUID string `json:"guid"`
	}

	var serviceCredentialBindings = struct {
		Resources []ServiceCredentialBinding `json:"resources"`
	}{}
	err = json.Unmarshal(contents, &serviceCredentialBindings)
	Expect(err).ShouldNot(HaveOccurred())

	return serviceCredentialBindings.Resources[0].GUID
}

func GetServiceCredentialBindingParameters(cfg *config.Config, instanceName string, appName string) string {
	instanceGuid := GetServiceInstanceGuid(cfg, instanceName)
	serviceCredentialBindingGuid := GetServiceCredentialBindingGuid(cfg, instanceGuid, appName)

	cmd := cf.CfSilent("curl", fmt.Sprintf("/v3/service_credential_bindings/%s/parameters", serviceCredentialBindingGuid)).Wait(cfg.DefaultTimeoutDuration())
	Expect(cmd).To(Exit(0))
	return strings.TrimSpace(string(cmd.Out.Contents()))
}

func GetHTTPClient(cfg *config.Config) *http.Client {
	return &http.Client{
		Transport: &http.Transport{
			Proxy: http.ProxyFromEnvironment,
			DialContext: (&net.Dialer{
				Timeout:   30 * time.Second,
				KeepAlive: 30 * time.Second,
			}).DialContext,
			TLSHandshakeTimeout: 10 * time.Second,
			DisableCompression:  true,
			DisableKeepAlives:   true,
			//nolint:gosec // #nosec G402 -- due https://github.com/securego/gosec/issues/11051
			TLSClientConfig: &tls.Config{
				InsecureSkipVerify: cfg.SkipSSLValidation,
			},
		},
		Timeout: 30 * time.Second,
	}
}

func GetAppGuid(cfg *config.Config, appName string) (string, error) {
	getAppGuid := func() (string, error) {
		guid := cf.Cf("app", appName, "--guid").Wait(cfg.DefaultTimeoutDuration())
		if guid.ExitCode() == 0 {
			return strings.TrimSpace(string(guid.Out.Contents())), nil
		}

		return "", fmt.Errorf("Failed to find app guid for app: %s \n CLI Output:\n %s", appName, guid.Err.Contents())
	}

	appGuid, err := TRetry(3, 60, getAppGuid)
	return appGuid, err
}

func SetLabel(cfg *config.Config, appGUID string, labelKey string, labelValue string) {
	GinkgoHelper()
	cmd := cf.Cf("curl", "--fail", fmt.Sprintf("/v3/apps/%s", appGUID), "-X", "PATCH", "-d", fmt.Sprintf(`{"metadata": {"labels": {"%s": "%s"}}}`, labelKey, labelValue)).Wait(cfg.DefaultTimeoutDuration())
	Expect(cmd).To(Exit(0))
}
